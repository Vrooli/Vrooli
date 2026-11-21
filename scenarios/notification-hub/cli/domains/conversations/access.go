package conversations

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// readError explains a refused read of asks in words the caller can act on.
//
// The Hub shows an ask only to the owner, to a verified agent run for the asks of
// its conversation (the run and the runs recovered from it, spawned by it or
// re-attached to the same harness session), to a run of the same owner that
// names the ask's exact subject key, or to the service that raised it as an
// event. A shell that presents none of those is refused, and a run that asks
// about an ask outside its reach is told it was not found (the Hub answers both
// the same way on purpose). The CLI sends
// the shell's own credential already: the agent identity token when there is
// one, else the API token. This only says what to do when neither is enough.
func readError(action string, err error) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return cliapp.WrapAPIError(action, err, nil)
	}
	if source := strings.TrimSpace(os.Getenv(SendAsEnv)); source != "" && os.Getenv(cliutil.EnvIdentityToken) == "" && os.Getenv("NOTIFICATION_HUB_API_TOKEN") == "" &&
		(connectErr.Code() == connect.CodeUnauthenticated || connectErr.Code() == connect.CodePermissionDenied) {
		return fmt.Errorf("%s: the Hub did not accept the sender token of %s (%s=%s): %s. Check that the principal is declared for the Hub and its credential was issued with `scenario-authenticator service-principals sync --apply`",
			action, hubsender.Principal(source), SendAsEnv, source, connectErr.Message())
	}
	switch connectErr.Code() {
	case connect.CodeUnauthenticated:
		return fmt.Errorf("%s: the Hub shows asks only to the owner, to an agent run (for its own conversation), or to the service that raised them, and this shell presented no verified identity. "+
			"Run the command inside the agent run that sent the ask, or give it the owner's credential (set NOTIFICATION_HUB_API_TOKEN, or run it on this machine in personal_local mode)", action)
	case connect.CodeNotFound:
		if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
			return fmt.Errorf("%s: not found. An agent run reads the asks of its own conversation (this run, and the runs it was recovered from, spawned by or re-attached to). "+
				"If this ask belongs to the same owner but another conversation, name it by its exact --subject-key instead of its id; an ask sent without a verified identity or by another owner is never shown to a run", action)
		}
	}
	return cliapp.WrapAPIError(action, err, nil)
}
