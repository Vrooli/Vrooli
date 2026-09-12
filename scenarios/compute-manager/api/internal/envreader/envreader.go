// Package envreader provides the environment seam used by configuration.
package envreader

import "os"

type Reader interface {
	Getenv(string) string
}

type System struct{}

func (System) Getenv(key string) string { return os.Getenv(key) }
