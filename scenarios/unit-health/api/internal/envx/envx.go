// Package envx owns the process environment implementation used by Unit Health.
// Consumers depend on their narrow EnvReader interfaces; tests substitute a fake.
package envx

import "os"

// OS reads the process environment.
type OS struct{}

func (OS) Getenv(key string) string            { return os.Getenv(key) }
func (OS) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }
func (OS) Environ() []string                   { return os.Environ() }
