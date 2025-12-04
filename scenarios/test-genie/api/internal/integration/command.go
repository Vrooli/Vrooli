package integration

import (
	"context"
	"io"
)

// CommandExecutor executes commands and returns any error.
type CommandExecutor func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

// CommandCapture executes commands and captures output.
type CommandCapture func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)

// CommandLookup checks if a command is available.
type CommandLookup func(name string) (string, error)
