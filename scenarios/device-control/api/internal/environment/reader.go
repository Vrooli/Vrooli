// Package environment provides the process-environment boundary used by
// composition roots. Runtime components receive resolved values instead of
// reading mutable process state themselves.
package environment

import "os"

// Reader is the narrow environment lookup seam.
// seam: device-control process environment
type Reader interface {
	LookupEnv(string) (string, bool)
}

// OSReader is the production environment implementation.
type OSReader struct{}

func (OSReader) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

// MapReader is a deterministic environment implementation for configuration
// and composition tests.
type MapReader map[string]string

var (
	_ Reader = OSReader{}
	_ Reader = MapReader{}
)

func (r MapReader) LookupEnv(key string) (string, bool) {
	value, ok := r[key]
	return value, ok
}

// Value returns a trimmed setting or an empty string when it is absent.
func Value(reader Reader, key string) string {
	if reader == nil {
		return ""
	}
	value, _ := reader.LookupEnv(key)
	return value
}
