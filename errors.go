package salmon
import "errors"

// Errors descriptions for the package
var (
	ErrShutdown        = errors.New("process could not be stopped : ")
	CircuitBreaker     = errors.New("circuit breaker condition")
	ServiceUnavailable = "service unavailable : 503"
)