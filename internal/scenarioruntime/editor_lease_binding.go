package scenarioruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// EditorLeaseLookup reads one exact editor lease without enumerating the registry.
type EditorLeaseLookup interface {
	GetEditorLease(context.Context, string) (EditorLease, error)
}

// ErrEditorLeaseIdentityMismatch means an exact lookup returned a different
// session. It is not evidence that the requested session is absent.
var ErrEditorLeaseIdentityMismatch = errors.New("editor lease session identity mismatch")

// ResolveAttachedEditorLease selects the lease for an attached Agent Manager
// run. The immutable host harness session is canonical. Older launchers stored
// the run ID instead, so that legacy key is read only when the canonical key is
// absent. An empty harness session permits legacy-only lookup for maintenance.
//
// This proves only the exact key association. Callers must still validate the
// lease's process incarnation, harness, owner, scope, and status as appropriate.
// A conflicting canonical row or unavailable registry must never cause fallback.
func ResolveAttachedEditorLease(ctx context.Context, reader EditorLeaseLookup, runID, harnessSessionID string) (EditorLease, error) {
	if err := ctx.Err(); err != nil {
		return EditorLease{}, err
	}
	if reader == nil {
		return EditorLease{}, errors.New("resolve attached editor lease: reader is required")
	}
	if strings.TrimSpace(runID) == "" {
		return EditorLease{}, errors.New("resolve attached editor lease: run_id is required")
	}
	key := harnessSessionID
	if key == "" {
		key = runID
	}
	if strings.TrimSpace(key) == "" {
		return EditorLease{}, errors.New("resolve attached editor lease: session_id is required")
	}
	lease, err := lookupAttachedEditorLease(ctx, reader, key)
	if errors.Is(err, ErrNotFound) && harnessSessionID != "" && key != runID {
		return lookupAttachedEditorLease(ctx, reader, runID)
	}
	return lease, err
}

func lookupAttachedEditorLease(ctx context.Context, reader EditorLeaseLookup, key string) (EditorLease, error) {
	if err := ctx.Err(); err != nil {
		return EditorLease{}, err
	}
	lease, err := reader.GetEditorLease(ctx, key)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return EditorLease{}, ctxErr
	}
	if err != nil {
		return EditorLease{}, err
	}
	if lease.SessionID != key {
		return EditorLease{}, fmt.Errorf("%w: requested %q, received %q", ErrEditorLeaseIdentityMismatch, key, lease.SessionID)
	}
	return lease, nil
}

// EditorLeaseBindingEqual compares the identity-bearing fields checked during
// an attached-run proof. Advisory editor paths and heartbeat timing do not
// change a binding; the attempt BindingID, CreatedAt and recorded owner evidence do.
func EditorLeaseBindingEqual(a, b EditorLease) bool {
	return a.BindingID == b.BindingID && a.SessionID == b.SessionID && a.Harness == b.Harness && a.Agent == b.Agent && a.PID == b.PID && a.HostBootID == b.HostBootID && a.Scope == b.Scope && a.Status == b.Status && a.CreatedAt.Equal(b.CreatedAt) && a.Owner.Kind == b.Owner.Kind && a.Owner.Ref == b.Owner.Ref && a.Owner.Source == b.Owner.Source && a.Owner.EvidenceJSON == b.Owner.EvidenceJSON
}
