// Runtime switch for AUTH-01 CreateRun caller enforcement.
package owneridentity

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// CreateRunCallerEnforceEnv is the single runtime switch for AUTH-01 caller
// enforcement. It is off unless set to 1, true, yes or on (operator decision
// P-18, docs/agent-system/DELIVERY_RELIABILITY.md). The strict primitives in
// create_run_caller.go are unchanged; owners that honor the switch call the
// Admit/Require/Carry variants below.
const CreateRunCallerEnforceEnv = "VROOLI_AUTH01_ENFORCE"

// CreateRunCallerEnforced reports the switch. It is read on each call so an
// owner never caches a stale value across its own reconfiguration or tests.
func CreateRunCallerEnforced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(CreateRunCallerEnforceEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// AdmitCreateRunCaller applies AuthorizeCreateRunCaller when enforcement is on.
// When it is off, absent or unverifiable proof is not a refusal: a single
// verified human Bearer is still attached for attribution, and every other
// caller (scheduler, agent identity) continues without a qualified caller.
// No proof is minted, and an offered run identity is not verified here.
func AdmitCreateRunCaller(ctx context.Context, headers http.Header, verifier Validator, now time.Time) (context.Context, error) {
	if CreateRunCallerEnforced() {
		return AuthorizeCreateRunCaller(ctx, headers, verifier, now)
	}
	if values := headers.Values("Authorization"); len(values) == 1 {
		if qualified, err := AuthorizeCreateRunCaller(ctx, http.Header{"Authorization": values}, verifier, now); err == nil {
			return qualified, nil
		}
	}
	return ctx, nil
}

// RequireAdmittedCreateRunCaller is RequireCreateRunCaller when enforcement is
// on. When it is off, an absent or expired caller does not hold work.
func RequireAdmittedCreateRunCaller(ctx context.Context, now time.Time) error {
	if CreateRunCallerEnforced() {
		return RequireCreateRunCaller(ctx, now)
	}
	return nil
}

// CarryAdmittedCreateRunCaller is CarryCreateRunCaller when enforcement is on.
// When it is off, a still-valid caller is carried for attribution and an
// absent one leaves the target context without a caller.
func CarryAdmittedCreateRunCaller(from, to context.Context, now time.Time) (context.Context, error) {
	if CreateRunCallerEnforced() || RequireCreateRunCaller(from, now) == nil {
		return CarryCreateRunCaller(from, to, now)
	}
	return to, nil
}
