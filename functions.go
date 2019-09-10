package salmon

import (
	"context"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	count = &Count{}
)

// GetResponseRoundRobin return response with Round-Robin police.
func GetResponseRoundRobin(index int, config Config, reqId string, logger *logrus.Logger) ([]byte, string ,error) {
	var body []byte
	var resp *http.Response
	var err error
	if count.scorer == len(config.Upstreams[index].Back) {
		count.scorer = 0
	}
	upstream := count.scorer % len(config.Upstreams[index].Back)
	count.IncreaseCount()
	resp, err = http.Get(config.Upstreams[index].Back[upstream]) // NO
	if err != nil {
		logger.Error(err)
		return nil,"", err
	}

	defer resp.Body.Close()
	logger.WithFields(logrus.Fields{
		"request_id": reqId,
		"url": config.Upstreams[index].Back[upstream],
		"proxy_method": config.Upstreams[index].ProxyMethod,
		"context": config.Upstreams[index].Path,
		"code": resp.StatusCode,
	}).Debug("response was successfully completed")

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return body,config.Upstreams[index].Back[upstream], nil
}

// GetResponseAnycast return response with Anycast police.
func GetResponseAnycast(index int, config Config, reqId string, logger *logrus.Logger) ([]byte, string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan []byte, 2)				// buffered chan (body, url)
	f := func(index int, n int) error {
		upstream := config.Upstreams[index].Back[n]
		logEvent := logger.WithFields(logrus.Fields{
			"request_id": reqId,
			"proxy_method": config.Upstreams[index].ProxyMethod,
			"context": config.Upstreams[index].Path,
			"url": config.Upstreams[index].Back[n],
		})
		resp, err := http.Get(upstream)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer resp.Body.Close()

		body,_ := ioutil.ReadAll(resp.Body)
		select {
		case ch <- body:
			ch <- []byte(upstream)	// string(<-ch) send upstream
			defer cancel() 			// send Cancel event to other goroutine
			logEvent.Debug("response was successfully completed")

			return nil
		case <-ctx.Done():	 // return <-chan struct{} and listening for a cancellation event is as easy as waiting for <- ctx.Done().
			logEvent.Debug("goroutine was successfully cancelled")
			return nil
		}
	}
	for i:=0; i < len(config.Upstreams[index].Back);i++ {
		go f(index, i)
	}
	select {
	case b:=<-ch:
		return b, string(<-ch), nil 				// string(<-ch) get upstream
	case <-time.After(25000 * time.Millisecond):
		return []byte("Timeout 25 sec"), string(<-ch), nil
	}
}