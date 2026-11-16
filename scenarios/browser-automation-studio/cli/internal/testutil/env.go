// Package testutil contains dependency-free helpers shared by CLI tests.
package testutil

// MapEnv adapts caller-owned values to the CLI app context's environment seam.
type MapEnv map[string]string

func (e MapEnv) LookupEnv(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}
