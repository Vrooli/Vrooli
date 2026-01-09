package infra

import "os"

// EnvReader abstracts environment variable access for testing.
type EnvReader interface {
	// Getenv retrieves the value of the environment variable named by the key.
	Getenv(key string) string
	// Environ returns a copy of strings representing the environment.
	Environ() []string
}

// RealEnvReader implements EnvReader using the os package.
type RealEnvReader struct{}

func (RealEnvReader) Getenv(key string) string {
	return os.Getenv(key)
}

func (RealEnvReader) Environ() []string {
	return os.Environ()
}

// Ensure RealEnvReader implements EnvReader.
var _ EnvReader = RealEnvReader{}
