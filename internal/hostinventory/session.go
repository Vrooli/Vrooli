package hostinventory

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/shell"
)

type SessionCommandRunner = shell.Runner

// ActiveSessionUser resolves the user owning seat0 through the shared host
// inventory authority. Callers that need a session bus may separately resolve
// that user's UID without reimplementing session selection.
func ActiveSessionUser(ctx context.Context, commands SessionCommandRunner) string {
	probeCtx, cancel := context.WithTimeout(ctx, tuning.ServiceHealthTimeout)
	defer cancel()
	activeSession, err := commands.Run(probeCtx, "loginctl", "show-seat", "seat0", "-p", "ActiveSession", "--value")
	if err != nil {
		return ""
	}
	sessionID := strings.TrimSpace(string(activeSession))
	if sessionID == "" {
		return ""
	}
	user, err := commands.Run(probeCtx, "loginctl", "show-session", sessionID, "-p", "Name", "--value")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(user))
}
