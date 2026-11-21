// Package envx provides a substitute-able process environment reader.
package envx

import "os"

// Reader reads one environment value.
type Reader interface {
	LookupEnv(string) (string, bool)
}

// OS reads from the current process environment.
type OS struct{}

func (OS) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

var _ Reader = OS{}
