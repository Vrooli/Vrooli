// Package envx contains the environment boundary used by production code.
package envx

// Reader is the small portion of process environment access that configuration
// consumers need. It lets callers load and validate configuration without
// mutating the process-wide environment.
type Reader interface {
	LookupEnv(string) (string, bool)
}

// OS reads the current process environment.
type OS struct{}

func (OS) LookupEnv(key string) (string, bool) { return lookupEnv(key) }
