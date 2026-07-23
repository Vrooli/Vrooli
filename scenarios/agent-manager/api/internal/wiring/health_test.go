package wiring

import (
	"context"
	"path/filepath"
	"testing"

	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/rolepolicy"
)

func TestPolicyHealthCheckersExposeUnavailableRequiredCatalogs(t *testing.T) {
	if RolePolicyHealthChecker(nil) != nil || PermissionPolicyHealthChecker(nil, nil) != nil {
		t.Fatal("nil policy state should not register a checker")
	}
	role, _ := rolepolicy.NewState(filepath.Join(t.TempDir(), "missing-role.json"), rolepolicy.Requirement{Required: true, Reason: "run safety"})
	roleResult := RolePolicyHealthChecker(role).Check(context.Background())
	if roleResult.Connected || roleResult.Name != "role_policy_catalog" || roleResult.Error == nil {
		t.Fatalf("role health=%+v", roleResult)
	}
	permission, _ := permissionpolicy.NewState(filepath.Join(t.TempDir(), "missing-permission.json"), permissionpolicy.Requirement{Required: true, Reason: "run safety"})
	permissionResult := PermissionPolicyHealthChecker(permission, nil).Check(context.Background())
	if permissionResult.Connected || permissionResult.Name != "permission_policy_catalog" || permissionResult.Error == nil {
		t.Fatalf("permission health=%+v", permissionResult)
	}
}
