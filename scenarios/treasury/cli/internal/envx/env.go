// Package envx provides a substitute-able process environment reader.
package envx

import "os"

// Reader reads one process configuration value.
type Reader interface {
	LookupEnv(string) (string, bool)
}

// OS reads the host process environment.
type OS struct{}

func (OS) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

// Empty is a deterministic environment with no configured values.
type Empty struct{}

func (Empty) LookupEnv(string) (string, bool) { return "", false }
