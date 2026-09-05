package credentials

import (
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// CommandContext is the narrow command boundary used by the credential
// service. It deliberately carries streams and global output options, not the
// root CLI application, so the domain package remains independently testable.
type CommandContext struct {
	Root    string
	Globals rootcli.GlobalOptions
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// App owns the credential command service. The value is stateless; it exists
// to keep the command entry points grouped behind one domain boundary.
type App struct{}

// Input returns the configured input stream, falling back to the process
// standard input for the root CLI.
func (ctx *CommandContext) Input() io.Reader {
	if ctx.Stdin != nil {
		return ctx.Stdin
	}
	return os.Stdin
}

// Run dispatches the credentials command group.
func (app *App) Run(ctx *CommandContext, args []string) error {
	return app.runCredentialsCommand(ctx, args)
}

// RunBreakGlass dispatches the break-glass command group.
func (app *App) RunBreakGlass(ctx *CommandContext, args []string) error {
	return app.runBreakGlassCommandWithInput(ctx, args, ctx.Input())
}
