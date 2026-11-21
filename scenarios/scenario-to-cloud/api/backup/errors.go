package backup

import (
	"errors"

	"scenario-to-cloud/apierrors"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// FromEngine maps a recoverypoint error onto the typed API error model. The
// engine's stable codes are reused verbatim; a blocker (the recovery key
// reference) becomes the next action an operator must satisfy.
func FromEngine(err error, fallback string) *apierrors.Error {
	if err == nil {
		return nil
	}
	var typed *apierrors.Error
	if errors.As(err, &typed) {
		return typed
	}
	engine := recoverypoint.AsError(err, fallback)
	code := engine.Code
	switch code {
	case recoverypoint.CodeCaptureFailed:
		code = apierrors.CodeBackupFailed
	case recoverypoint.CodeRestoreFailed, recoverypoint.CodeVerifyFailed:
		code = apierrors.CodeRestoreFailed
	case recoverypoint.CodeInvalidArgument:
		code = apierrors.CodeInvalidRequest
	case recoverypoint.CodeRecoveryPointExists:
		code = apierrors.CodeRecoveryPointConflict
	}
	out := apierrors.New(code, engine.Message)
	for key, value := range engine.Details {
		out = out.WithDetail(key, value)
	}
	if engine.Blocker != "" {
		out = out.WithDetail("blocker", engine.Blocker).WithNextAction(apierrors.NextAction{
			Owner: "operator", Kind: "supply_recovery_key", Reference: engine.Blocker,
			Label: "Make the recovery key resolvable from the reference outside the target failure domain, then retry.",
		})
	}
	return out
}
