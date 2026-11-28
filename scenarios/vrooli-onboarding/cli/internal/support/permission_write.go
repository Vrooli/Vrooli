package support

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
)

const (
	permissionReadProcedure  = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission"
	permissionWriteProcedure = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission"
)

// WriteCurrentPermission performs a revision-bound write through the shared
// owner registry. The registry derives actor identity from the authenticated
// request; callers cannot supply or override it here.
func WriteCurrentPermission(core *cliapp.ScenarioApp, permissionID, owner, idempotencyPrefix, label string, target *permissionsv1.PermissionTarget, value []byte) (*permissionsv1.PermissionReceipt, error) {
	if core == nil || permissionID == "" || owner == "" || idempotencyPrefix == "" || label == "" || len(value) == 0 {
		return nil, fmt.Errorf("permission write requires a client, permission, owner, idempotency prefix, label, and value")
	}
	read := new(permissionsv1.ReadPermissionResponse)
	readRequest := &permissionsv1.ReadPermissionRequest{PermissionId: permissionID, Owner: owner, Target: target}
	if err := RequestProto(core, permissionReadProcedure, readRequest, read, label); err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if read.GetPermissionId() != permissionID || read.GetOwner() != owner || read.GetRevision() == "" {
		return nil, fmt.Errorf("read %s returned a mismatched permission, owner, or empty revision", label)
	}
	if !samePermissionTarget(read.GetTarget(), target) {
		return nil, fmt.Errorf("read %s returned a mismatched target", label)
	}

	requestID := make([]byte, 16)
	if _, err := rand.Read(requestID); err != nil {
		return nil, fmt.Errorf("create %s request id: %w", label, err)
	}
	idempotencyKey := idempotencyPrefix + ":" + hex.EncodeToString(requestID)
	write := new(permissionsv1.WritePermissionResponse)
	writeRequest := &permissionsv1.WritePermissionRequest{
		PermissionId: permissionID, Owner: owner, Target: target,
		ExpectedRevision: read.GetRevision(), IdempotencyKey: idempotencyKey, ValueJson: append([]byte(nil), value...),
	}
	if err := RequestProto(core, permissionWriteProcedure, writeRequest, write, label); err != nil {
		return nil, fmt.Errorf("write %s: %w", label, err)
	}
	receipt := write.GetReceipt()
	if receipt == nil || receipt.GetPermissionId() != permissionID || receipt.GetOwner() != owner ||
		receipt.GetActorId() == "" || receipt.GetExpectedRevision() != read.GetRevision() ||
		receipt.GetResultRevision() == "" || receipt.GetIdempotencyKey() != idempotencyKey ||
		receipt.GetOutcome() == "" || !samePermissionTarget(receipt.GetTarget(), target) {
		return nil, fmt.Errorf("write %s returned a mismatched or incomplete owner receipt", label)
	}
	return receipt, nil
}

func samePermissionTarget(left, right *permissionsv1.PermissionTarget) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.GetMachineId() == right.GetMachineId() && left.GetNodeId() == right.GetNodeId() && left.GetEnrollmentGeneration() == right.GetEnrollmentGeneration()
}
