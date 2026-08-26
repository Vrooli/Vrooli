package hostapp

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// CommandContext is the narrow root-to-host command boundary.
type CommandContext struct {
	Root    string
	Globals rootcli.GlobalOptions
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// App owns host observation and host-command orchestration.
type App struct{}
