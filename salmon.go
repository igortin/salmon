// Package salmon provides proxy servers
package salmon

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	rabbit "github.com/rabbitmq-client"
	satori "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	reqCount = 5
)

var (
	p = &ConPool{nil, GetMaxPoolSz()}
)

type IoProxy interface {
	Run(wg *sync.WaitGroup) error
	Shutdown() error
}
// Proxy is main server struct.
type Proxy struct {
	Service         *http.Server
	Stopped         bool
	Router          *mux.Router
	GracefulTimeout time.Duration
	Cfg             Config
	Logger          *logrus.Logger
}


// RequestAmqpMessage is decoded from http request.
type RequestAmqpMessage struct {
	Id      string `json:"id,omitempty"`
	Message string `json:"message"`
}


// Run return listening HTTP server.
func (cmd *Proxy) Run(wg *sync.WaitGroup) error {
	cmd.Logger.Debug("Server started on port: ", cmd.Cfg.Port)
	// Goroutine control and server stop
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-stop
		cmd.Logger.Debug("Get signal to stop server: ", cmd.Cfg.Port)
		if err := cmd.Shutdown(); err != nil {
			cmd.Logger.Error(ErrShutdown, err)
		} else {
			cmd.Logger.Debug("Server was stopped by Ctrl+C: ", cmd.Cfg.Port)
		}
		defer wg.Done()
	}()
	cmd.Stopped = false
	cmd.Router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(map[string]bool{"ok": true}) })
	for i, n := range cmd.Cfg.Upstreams {
		cmd.Router.HandleFunc(n.Path, cmd.makeHandlers(i)).Methods(n.Method)
	}
	return cmd.Service.ListenAndServe()
}

// NewProxy return Proxy object.
func NewProxy(srv *http.Server, config Config, graceToStop time.Duration, Logger *logrus.Logger) IoProxy {
	router := mux.NewRouter()
	srv.Handler = router
	return &Proxy{
		srv,
		true,
		router,
		graceToStop * time.Second,
		config,
		Logger,
	}
}

func (cmd *Proxy) makeHandlers(index int) http.HandlerFunc {
	var body []byte
	var err error
	var reqId string
	var upstream string
	switch cmd.Cfg.Upstreams[index].ProxyMethod {
	case roundrobin:
		// Mechanism of reliable good request
		return func(w http.ResponseWriter, r *http.Request) {
			if cmd.Stopped {
				w.WriteHeader(503)
				return
			}
			//Generate request id
			uid, _ := satori.NewV4()
			reqId = uid.String()
			// Reliable request mechanism
			for i := 0; i < reqCount; i++ {
				body, upstream, err = GetResponseRoundRobin(index, cmd.Cfg, reqId, cmd.Logger)
				if err != nil {
					cmd.Logger.Warn(CircuitBreaker)
					t := rand.Intn(reqCount)
					time.Sleep(time.Duration(i*t) * time.Millisecond)
					if i == reqCount-1 {
						cmd.Logger.Error(ServiceUnavailable)
						w.Write([]byte(ServiceUnavailable))
						break
					}
				} else {
					break
				}
			}
			w.Header().Set("URL", upstream)
			w.Write(body)
		}
	case anycast:
		return func(w http.ResponseWriter, r *http.Request) {
			if cmd.Stopped {
				w.WriteHeader(503)
				return
			}
			for i := 0; i < reqCount; i++ {
				uid, _ := satori.NewV4()
				reqId = uid.String()
				// Reliable request mechanism
				body, upstream, err = GetResponseAnycast(index, cmd.Cfg, reqId, cmd.Logger)
				if err != nil {
					cmd.Logger.Warn(CircuitBreaker)
					t := rand.Intn(reqCount)
					time.Sleep(time.Duration(i*t) * time.Millisecond)
					if i == reqCount {
						w.Write([]byte(ServiceUnavailable))
						break
					}
				} else {
					break
				}
			}
			w.Header().Set("URL", upstream)
			w.Write(body)
		}
	case amqpRabbit:
		return func(w http.ResponseWriter, r *http.Request) {
			cmd.Logger.Debug("Handler return")
			var con *amqp.Connection
			if cmd.Stopped {
				w.WriteHeader(503)
				return
			}
			if r.Body == nil {
				w.WriteHeader(400)
				return
			}
			uid, _ := satori.NewV4()
			reqId = uid.String()
			req := RequestAmqpMessage{}
			logEvent := cmd.Logger.WithFields(logrus.Fields{
				"request_id":   reqId,
				"proxy_method": cmd.Cfg.Upstreams[index].ProxyMethod,
				"context":      cmd.Cfg.Upstreams[index].Path,
				"url":          cmd.Cfg.Upstreams[index].Back[0],
			})
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				logEvent.Error(err)
			}
			req.Id = reqId
			// Create connection Pool for RabbitMQ
			if p.pool == nil {
				for i := 0; i < GetMaxPoolSz(); i++ {
					con, err = rabbit.NewConnect(cmd.Cfg.Upstreams[index].Back[0])
					if err != nil {
						logEvent.Error(ServiceUnavailable," ",err)
					}
					p.PutCon(con)
					cmd.Logger.Debug("Connection append to Rabbit pool")
				}
			}
			logEvent.Debug("Request successfully send to RabbitMQ")
			body, err := GetRespRabbitMQ([]byte(req.Message), p)
			if err != nil {
				logEvent.Error(err)
			}
			logEvent.Debug("Request successfully processed by RabbitMQ")
			w.Write(body)
		}
	}
	return nil
}
// Method to stop servers by Ctrl+C command in background mode.
func (cmd *Proxy) Shutdown() error {
	for _,con := range p.pool {
		con.Close()
	}
	cmd.Stopped = true
	ctx, cancel := context.WithTimeout(context.Background(), cmd.GracefulTimeout)
	defer cancel()

	time.Sleep(cmd.GracefulTimeout)
	return cmd.Service.Shutdown(ctx)
}