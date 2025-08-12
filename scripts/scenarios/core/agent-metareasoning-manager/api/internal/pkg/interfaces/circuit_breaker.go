package interfaces

// CircuitBreakerManager defines the interface for circuit breaker management
type CircuitBreakerManager interface {
	GetBreaker(name string) CircuitBreaker
	GetAllStats() map[string]interface{}
}

// CircuitBreaker defines the interface for circuit breaker operations
type CircuitBreaker interface {
	Call(fn func() error) error
	GetState() CircuitBreakerState
	GetStats() map[string]interface{}
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN" 
	case HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}