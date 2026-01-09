package shared

import "time"

// TimeProvider defines the interface for getting the current time.
// This seam allows tests to control time-dependent behavior.
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider returns the actual current time.
type RealTimeProvider struct{}

// Now returns the current time.
func (RealTimeProvider) Now() time.Time {
	return time.Now()
}

// DefaultTimeProvider is the default time provider used throughout the application.
// Tests can override this to control time-dependent behavior.
var DefaultTimeProvider TimeProvider = RealTimeProvider{}

// GetTimeProvider returns the current time provider.
func GetTimeProvider() TimeProvider {
	return DefaultTimeProvider
}

// SetTimeProvider allows overriding the time provider (for testing).
func SetTimeProvider(tp TimeProvider) {
	DefaultTimeProvider = tp
}
