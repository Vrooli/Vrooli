package platform

import (
	"context"
	"fmt"
	"path"
	"strings"
)

// ServiceScopeRef observes an already-running service in its dedicated native
// scope. It does not invent a historical dispatch intent or launch spec.
// The domain owner must verify resource ownership before and after capture.
type ServiceScopeRef struct {
	Scope              ScopeRef `json:"scope"`
	RootPID            int      `json:"root_pid"`
	RootStartTime      string   `json:"root_start_time"`
	NativeInvocationID string   `json:"native_invocation_id"`
	ScopeDevice        uint64   `json:"scope_device"`
	ScopeInode         uint64   `json:"scope_inode"`
}

func (r ServiceScopeRef) ValidateIdentity() error {
	unit := path.Base(r.Scope.Path)
	if r.Scope.Kind != ScopeKindCgroup || r.Scope.PID != 0 || !path.IsAbs(r.Scope.Path) || path.Clean(r.Scope.Path) != r.Scope.Path || path.Base(path.Dir(r.Scope.Path)) != ServicesSlice || !strings.HasPrefix(unit, "vrooli-service-resource-") || !strings.HasSuffix(unit, ".scope") || strings.TrimSuffix(unit, ".scope") != r.Scope.Name || strings.ContainsAny(unit, " \t\r\n\\") || r.RootPID <= 0 || r.RootStartTime == "" || !operationInvocation.MatchString(r.NativeInvocationID) || r.ScopeDevice == 0 || r.ScopeInode == 0 {
		return fmt.Errorf("%w: invalid dedicated service-scope receipt", ErrOperationIdentity)
	}
	for _, c := range r.RootStartTime {
		if c < '0' || c > '9' {
			return fmt.Errorf("%w: invalid native root start time", ErrOperationIdentity)
		}
	}
	return nil
}

type ServiceScopeStopReceipt struct {
	Ref           ServiceScopeRef `json:"ref"`
	RootGone      bool            `json:"root_gone"`
	TreeQuiescent bool            `json:"tree_quiescent"`
}

type ServiceScopeObservation struct {
	Ref           ServiceScopeRef `json:"ref"`
	RootAlive     bool            `json:"root_alive"`
	TreePopulated bool            `json:"tree_populated"`
	NativePresent bool            `json:"native_present"`
}

// InspectServiceScope preserves unknown manager/filesystem observations as
// errors. A recycled PID or scope is never accepted as the recorded owner.
func InspectServiceScope(ctx context.Context, ref ServiceScopeRef) (ServiceScopeObservation, error) {
	if ctx == nil {
		return ServiceScopeObservation{}, fmt.Errorf("platform: nil service-scope inspection context")
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeObservation{}, err
	}
	if err := ref.ValidateIdentity(); err != nil {
		return ServiceScopeObservation{}, err
	}
	return inspectServiceScope(ctx, ref)
}

// CaptureServiceScope requires a dedicated singleton root with no nested
// groups. Stable native handles fence PID reuse during identity inspection.
// Descendants subsequently created inside the scope remain owned by it.
func CaptureServiceScope(ctx context.Context, scope ScopeRef, pid int, startTime string) (ServiceScopeRef, error) {
	if ctx == nil {
		return ServiceScopeRef{}, fmt.Errorf("platform: nil scope capture context")
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeRef{}, err
	}
	return captureServiceScope(ctx, scope, pid, startTime)
}

// StopServiceScope signals the exact held root and native cgroup handles,
// never a numeric PID or a possibly recycled unit name. It refuses unknown
// ownership and reports quiescence separately from process termination.
func StopServiceScope(ctx context.Context, ref ServiceScopeRef) (ServiceScopeStopReceipt, error) {
	if ctx == nil {
		return ServiceScopeStopReceipt{}, fmt.Errorf("platform: nil service-scope stop context")
	}
	if err := ctx.Err(); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	if err := ref.ValidateIdentity(); err != nil {
		return ServiceScopeStopReceipt{}, err
	}
	return stopServiceScope(ctx, ref)
}
