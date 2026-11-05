// Package envreader isolates process-environment access from domain services.
package envreader

import "os"

// Reader is the small environment seam consumed by production services.
type Reader interface {
	Getenv(string) string
	LookupEnv(string) (string, bool)
}

// System reads the process environment.
type System struct{}

func (System) Getenv(key string) string {
	return os.Getenv(key)
}

func (System) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Func adapts a deterministic getter for tests. LookupEnv treats an empty
// value as absent, matching the behavior needed by tunnel-manager settings.
type Func func(string) string

func (f Func) Getenv(key string) string {
	return f(key)
}

func (f Func) LookupEnv(key string) (string, bool) {
	value := f(key)
	return value, value != ""
}
