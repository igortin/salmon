package salmon

import (
	rabbit "github.com/rabbitmq-client"
	"github.com/streadway/amqp"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	sep = "./"
)

var (
	queueResponse   = "Q_resp"
	queueRequest    = "Q_rpc"
	exchangeRequest = "Ex_" + strings.Trim(os.Args[0], sep)
	rk = ".*" + strings.TrimLeft(os.Args[0], sep)
	poolSz = 5
)

// ConPool represents a pool of connections. Connections objects stored in a field pool. Max capacity is maxSize.
type ConPool struct {
	pool []*amqp.Connection
	maxSize int
}

// GetCon return pointer to amqp connection.
func (p *ConPool) GetCon() (*amqp.Connection, error) {
	if len(p.pool) == 0 {
		return nil, NoConPool
	}
	connection := p.pool[0]
	p.pool = p.pool[1:]
	return connection, nil
}

// PutCon append amqp connection pointer to object of struct ConPool.
func (p *ConPool) PutCon(connection *amqp.Connection) error {
	length := len(p.pool)
	if length + 1 > p.maxSize {
		return NoSlotPool
	}
	p.pool = append(p.pool, connection)
	return nil
}

// GetRespRabbitMQ return response from RPC server.
func GetRespRabbitMQ(message []byte, p *ConPool) (resp []byte, errs error) {
	var wg sync.WaitGroup
	con, err := p.GetCon()
	if err != nil {
		return nil, err
	}
	ch1, err := rabbit.NewChannel(con)
	if err != nil {
		return nil, err
	}
	defer ch1.Close()
	q1 := rabbit.Queue{
		Name: queueResponse,
		Durable: false,
		AutoDelete: false,
		Exclusive: true,
		NoWait: false,
		Args: nil,
	}

	queue, err := rabbit.CreateQueue(ch1, q1)
	if err != nil {
		return nil, err
	}

	consumer := rabbit.Consumer{
		QueueName: queue.Name,
		Name: "",
		AutoAck: true, // acknowledged for every consume msg
		Exclusive: false,
		NoLocal: false,
		NoWait: false,
		Args: nil,
	}

	msgs, err := rabbit.CreateAmqpChannel(ch1, consumer)
	if err != nil {
		return nil, err
	}
	corrId := rabbit.GetCorelId() // corrID for send and response
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		for d := range msgs {
			if corrId == d.CorrelationId {
				resp = d.Body
			}
			wg.Done()
		}
	}(&wg)

	ch2, err := rabbit.NewChannel(con)
	if err != nil {
		return nil, err
	}
	defer ch2.Close()

	e := rabbit.Exchange{
		Name: exchangeRequest,
		Kind: "topic",
		Durable: true,
		AutoDeleted: false,
		Internal: false,
		NoWait: false,
		Args: nil,
	}

	err = rabbit.CreateExchange(ch2, e)
	if err != nil {
		return nil, err
	}

	q2 := rabbit.Queue{
		Name: queueRequest,
		Durable: false,
		AutoDelete: false,
		Exclusive: true,
		NoWait: false,
		Args: nil,
	}
	_, err = rabbit.CreateQueue(ch2, q2)
	if err != nil {
		return nil, err
	}

	b := rabbit.Bind{
		QueueName: queueRequest,
		RoutingKey: rk,
		Exchange: e.Name,
		NoWait: false,
		Args: nil,
	}

	err = rabbit.CreateBind(ch1, b)
	if err != nil {
		return nil, err
	}

	msg := rabbit.Message{
		ExchangeName:e.Name,
		RoutingKey: rk,
		Mandatory: false,
		Immediate: false,
		Publish: amqp.Publishing{
			ContentType:   "text/plain",
			Body:          message,
			ReplyTo:       queue.Name,
			CorrelationId: corrId,
		},
	}

	err = msg.Send(ch1)
	if err != nil {
		return nil, err
	}
	wg.Wait()

	err = p.PutCon(con)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
// GetMaxPoolSz to return connection pool length.
func GetMaxPoolSz() int {
	if os.Getenv("RabbitPoolSize") != "" {
		count, err := strconv.Atoi(os.Getenv("RabbitPoolSize"))
		if err == nil {
			return count
		}
	}
	return poolSz
}