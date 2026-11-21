// Package session exposes the owner-facing interactive-session controls. The
// binary WebSocket is intentionally a UI/terminal concern; the CLI owns the
// destructive kill switch so an operator can terminate a stuck remote session
// without relying on the browser that opened it.
package session

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	return cliapp.SubcommandGroup{
		Name:        "session",
		Description: "Inspect and terminate interactive Bridge sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{{
			Name:        "kill",
			Description: "Terminate an interactive session by id",
			NeedsAPI:    true,
			DryRun:      cliapp.DryRunUnsupported,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Interactive session id"}}},
			RunCtx:      kill,
		}},
	}, nil
}

func kill(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	if id == "" {
		return fmt.Errorf("session id is required")
	}
	if _, err := ctx.Core().Request(http.MethodDelete, "/channel/session/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("kill session %q: %w", id, err)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Interactive session %s terminated.", id)},
		NextSteps: []string{"Use the terminal launcher to open a new session when the node is ready."},
	})
}
