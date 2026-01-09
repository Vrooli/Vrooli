package platforms

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

// NewRealCommandRunner creates a new real command runner.
func NewRealCommandRunner() CommandRunner {
	return &RealCommandRunner{}
}

// Run executes a command and returns stdout, stderr, and any error.
func (r *RealCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// LookPath searches for an executable in the PATH.
func (r *RealCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// RealFileSystem implements FileSystem using the real file system.
type RealFileSystem struct{}

// NewRealFileSystem creates a new real file system.
func NewRealFileSystem() FileSystem {
	return &RealFileSystem{}
}

// Exists checks if a file or directory exists.
func (r *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RealEnvironmentReader implements EnvironmentReader using os.
type RealEnvironmentReader struct{}

// NewRealEnvironmentReader creates a new real environment reader.
func NewRealEnvironmentReader() EnvironmentReader {
	return &RealEnvironmentReader{}
}

// GetEnv retrieves the value of the environment variable.
func (r *RealEnvironmentReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// LookupEnv retrieves the value of the environment variable and reports if it exists.
func (r *RealEnvironmentReader) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
