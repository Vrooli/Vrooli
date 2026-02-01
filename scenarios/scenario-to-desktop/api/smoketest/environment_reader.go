package smoketest

import "os"

// DefaultEnvironmentReader implements EnvironmentReader using the real environment.
type DefaultEnvironmentReader struct{}

// NewEnvironmentReader creates a new environment reader.
func NewEnvironmentReader() *DefaultEnvironmentReader {
	return &DefaultEnvironmentReader{}
}

// GetEnv retrieves the value of an environment variable.
func (e *DefaultEnvironmentReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// UserHomeDir returns the current user's home directory.
func (e *DefaultEnvironmentReader) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}
