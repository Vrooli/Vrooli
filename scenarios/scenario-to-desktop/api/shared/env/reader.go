// Package env provides unified environment variable access utilities.
//
// This package centralizes environment variable reading logic that was previously
// duplicated across signing, distribution, and other packages. It provides both
// an interface for testability and a real implementation using os.Getenv.
package env

import "os"

// Reader abstracts environment variable access for testing.
// Implementations can be injected to control environment behavior in tests.
type Reader interface {
	// GetEnv retrieves an environment variable value.
	// Returns empty string if the variable is not set.
	GetEnv(key string) string

	// LookupEnv retrieves an environment variable and reports if it exists.
	// Returns the value and true if set, or empty string and false if not set.
	LookupEnv(key string) (string, bool)
}

// OSReader implements Reader using the actual OS environment.
// This is the default implementation for production use.
type OSReader struct{}

// NewOSReader creates a new OS-based environment reader.
func NewOSReader() Reader {
	return &OSReader{}
}

// GetEnv retrieves an environment variable value from the OS.
func (r *OSReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// LookupEnv retrieves an environment variable and reports if it exists.
func (r *OSReader) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
