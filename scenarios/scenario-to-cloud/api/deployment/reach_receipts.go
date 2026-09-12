package deployment

import (
	"context"
	"errors"
	"fmt"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/reach"
)

// ReachTargetReceipts reads the target's receipt for one step through the
// reach seam ("vrooli cloud-target receipt get"). The deployment's target
// binding selects the transport; the verb travels as argv, never as a shell
// fragment, and a missing native binary is reported as ErrNativeCLIAbsent so
// the worker records an unknown effect instead of replaying blind.
type ReachTargetReceipts struct {
	Reach reach.Reach
	Repo  interface {
		GetDeploymentRef(ctx context.Context, id string) (*identity.DeploymentRef, error)
	}
}

var _ operations.TargetReceipts = (*ReachTargetReceipts)(nil)

// Read implements operations.TargetReceipts.
func (r *ReachTargetReceipts) Read(ctx context.Context, op *domain.CloudOperation, step string) (operations.TargetReceipt, error) {
	if err := faultinject.Hit(ctx, faultinject.TransportSend); err != nil {
		return operations.TargetReceipt{}, err
	}
	if r == nil || r.Reach == nil || r.Repo == nil {
		return operations.TargetReceipt{}, operations.ErrNativeCLIAbsent
	}
	ref, err := r.Repo.GetDeploymentRef(ctx, op.DeploymentID)
	if err != nil || ref == nil {
		return operations.TargetReceipt{}, fmt.Errorf("deployment %s unavailable for receipt read", op.DeploymentID)
	}
	for _, v := range []string{op.DeploymentID, op.ID, step} {
		if !safeIdentifier(v) {
			return operations.TargetReceipt{}, fmt.Errorf("unsafe identifier %q in receipt read", v)
		}
	}
	res, err := r.Reach.Exec(ctx, ref.Target, reach.Command{
		Verb:          "cloud-target receipt get",
		Args:          []string{"--deployment", op.DeploymentID, "--operation", op.ID, "--step", step, "--json"},
		RequiredScope: "vrooli:read",
	})
	if errors.Is(err, context.DeadlineExceeded) {
		return operations.TargetReceipt{}, err
	}
	if reach.IsKind(err, reach.KindProtocolUnsupported) {
		return operations.TargetReceipt{}, operations.ErrNativeCLIAbsent
	}
	if err != nil {
		return operations.TargetReceipt{}, err
	}
	if err := faultinject.Hit(ctx, faultinject.TransportReply); err != nil {
		return operations.TargetReceipt{}, err
	}
	var runErr error
	if res.ExitCode != 0 {
		runErr = fmt.Errorf("receipt get exited %d", res.ExitCode)
	}
	return ParseTargetReceipt(res.Stdout, res.ExitCode, runErr)
}
