// Package envx defines the API's injectable process-environment boundary.
package envx

import "os"

// Reader is the subset of process environment access used by the API.
type Reader interface {
	Getenv(string) string
	Environ() []string
}

// OS reads the process environment and is wired by production composition roots.
type OS struct{}

func (OS) Getenv(key string) string { return os.Getenv(key) }
func (OS) Environ() []string        { return os.Environ() }
