// Package envx provides the onboarding API's injectable environment seam.
package envx

import "os"

// Reader is the environment access needed by onboarding path resolution.
type Reader interface {
	Getenv(string) string
}

// OS reads the process environment.
type OS struct{}

func (OS) Getenv(key string) string { return os.Getenv(key) }
