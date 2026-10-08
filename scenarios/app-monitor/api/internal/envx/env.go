// Package envx defines the injectable environment seam used by API services.
package envx

import "os"

// Reader is the narrow environment dependency shared by production code and
// tests. Keeping reads behind this interface prevents tests from depending on
// ambient host configuration.
type Reader interface {
	Getenv(string) string
}

type system struct{}

func (system) Getenv(key string) string { return os.Getenv(key) }

func System() Reader { return system{} }
