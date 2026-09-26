package privilegebroker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// CaddyMainConfigPath is the only configuration file the caddy actions
// validate. Deployment routes live in per-deployment snippets imported from
// this file; the actions never receive a path.
const CaddyMainConfigPath = "/etc/caddy/Caddyfile"

// caddySubjectPattern is the deployment identifier grammar shared with
// internal/cloudtarget (identifierPattern). The subject is evidence for the
// audit line and receipt; it never reaches argv.
var caddySubjectPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func validateCaddy(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome != nil || req.Log != nil || req.Journal != nil || req.Docker != nil || req.Apt != nil || req.Edge != nil || req.Process != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	if req.Caddy == nil {
		return fmt.Errorf("caddy_subject_required")
	}
	if !caddySubjectPattern.MatchString(strings.TrimSpace(req.Caddy.DeploymentID)) {
		return fmt.Errorf("invalid_deployment_id")
	}
	return nil
}

// CaddyArgs returns the fixed tool and argv for an accepted caddy action.
// validate reads the main Caddyfile (and, through its import line, every
// snippet) without changing the running server; reload asks systemd to apply
// it. Neither argv carries anything from the request.
func CaddyArgs(req Request) (string, []string, error) {
	if err := Validate(req); err != nil {
		return "", nil, err
	}
	switch req.Action {
	case ActionEdgeCaddyValidate:
		return "caddy", []string{"validate", "--config", CaddyMainConfigPath, "--adapter", "caddyfile"}, nil
	case ActionEdgeCaddyReload:
		return "systemctl", []string{"reload", "caddy"}, nil
	default:
		return "", nil, fmt.Errorf("action_not_allowed")
	}
}

func executeCaddy(ctx context.Context, executor Executor, req Request) Result {
	name, args, err := CaddyArgs(req)
	if err != nil {
		return NewFailure(req.RequestID, req.Action, "action_not_allowed")
	}
	out, runErr := executor.Run(ctx, name, args...)
	detail := boundedDetail(out)
	if runErr != nil {
		code := "caddy_validate_failed"
		if req.Action == ActionEdgeCaddyReload {
			code = "caddy_reload_failed"
		}
		result := NewFailure(req.RequestID, req.Action, code)
		result.Evidence = Evidence{Available: true, Detail: detail}
		return result
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: req.Action == ActionEdgeCaddyReload, Evidence: Evidence{Available: true, Detail: detail}}
}
