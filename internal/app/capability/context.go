package capabilityapp

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// CommandContext is the narrow root-to-capability command boundary.
type CommandContext struct {
	Root    string
	Globals rootcli.GlobalOptions
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// App owns capability command orchestration and delegates the portability
// read model to its scenario owner.
type App struct{}
