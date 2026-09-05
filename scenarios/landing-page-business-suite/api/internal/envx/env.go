// Package envx owns process-environment access at the application boundary.
package envx

import "os"

// Reader is the injectable boundary for process configuration.
//
// seam: EnvironmentReader
type Reader interface {
	Get(key string) string
}

// System reads configuration from the process environment.
type System struct{}

// Get returns the process environment value for key.
func (System) Get(key string) string { return os.Getenv(key) }

var _ Reader = System{}

// Get is the production convenience entry point. Dependencies that need
// deterministic configuration should accept a Reader instead.
func Get(key string) string { return System{}.Get(key) }
