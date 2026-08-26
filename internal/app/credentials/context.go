package credentials

import (
	"encoding/json"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"google.golang.org/protobuf/types/known/structpb"
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

// writeCredentialJSON preserves the established object-shaped JSON contract
// while routing it through protobuf JSON. The source values are status and
// address metadata only; secret material is never accepted by this renderer.
func writeCredentialJSON(w io.Writer, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	payload, err := structpb.NewStruct(object)
	if err != nil {
		return err
	}
	return cliout.WriteProtoJSON(w, payload)
}
