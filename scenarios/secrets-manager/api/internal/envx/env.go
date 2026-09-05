// Package envx provides the environment-reader seam used by API domains.
package envx

import "os"

// Reader reads a named process-environment value. Production uses OS and
// tests provide a deterministic reader instead of mutating process globals.
//
// seam: Reader isolates process environment access for Secrets Manager domains.
type Reader interface {
	Getenv(key string) string
}

// OS is the production Reader backed by the process environment.
type OS struct{}

func (OS) Getenv(key string) string { return os.Getenv(key) }

var _ Reader = OS{}
