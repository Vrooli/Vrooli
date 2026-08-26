package runtimeapp

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// CommandContext is the narrow stream and environment boundary used by the
// runtime application. It has no dependency on the root CLI application.
type CommandContext struct {
	Root      string
	Globals   rootcli.GlobalOptions
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	HomeDirFn func() (string, error)
}

// App owns runtime command orchestration.
type App struct {
	Version       string
	ResolveRootFn func() (string, error)
}

func (ctx *CommandContext) HomeDir() (string, error) {
	if ctx.HomeDirFn == nil {
		return "", io.ErrClosedPipe
	}
	return ctx.HomeDirFn()
}

func (app *App) resolveRoot() (string, error) {
	if app.ResolveRootFn == nil {
		return "", io.ErrClosedPipe
	}
	return app.ResolveRootFn()
}
