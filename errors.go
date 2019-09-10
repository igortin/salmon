package salmon
import "errors"

// Message is returned by operation that is performed reliable request.
var (
	ErrShutdown        = errors.New("process could not be stopped : ")
	CircuitBreaker     = errors.New("circuit breaker condition")
	ServiceUnavailable = "service unavailable : 503"
)

// Message is returned by operation that is performed to RabbitMQ Connection Pool.
var  (
	NoSlotPool = errors.New("unable put connection, no empty slots")
	NoConPool = errors.New("unable get connection, no free connection")
)