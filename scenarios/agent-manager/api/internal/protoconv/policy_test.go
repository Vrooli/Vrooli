package protoconv

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/rolepolicy"
)

func TestRolePolicyProjectionsPreserveDiagnosticsAndSortedRoles(t *testing.T) {
	now := time.Now().UTC()
	status := RolePolicyStatusToProto(rolepolicy.Status{
		Path: "/policy.json", Requirement: rolepolicy.Requirement{Required: true, Reason: "required"}, Ready: true, ActiveDigest: "sha256:role", ActivatedAt: &now,
		LastReloadAttempt: &rolepolicy.ReloadAttempt{AttemptedAt: now, Succeeded: false, Digest: "sha256:bad", Diagnostic: &rolepolicy.Diagnostic{Code: "invalid", Message: "bad catalog", Cause: "parse"}},
	})
	if status.GetPath() != "/policy.json" || !status.GetReady() || status.GetLastReloadAttempt().GetDiagnostic().GetCode() != "invalid" {
		t.Fatalf("status=%+v", status)
	}
	if RolePolicyDiagnosticToProto(nil) != nil || RolePolicyCatalogToProto(nil) != nil {
		t.Fatal("nil role policy projections should remain nil")
	}
	catalog := RolePolicyCatalogToProto(&rolepolicy.Catalog{SchemaVersion: 1, Metadata: rolepolicy.Metadata{CatalogID: "roles", UpdatedAt: "2026-07-23"}, DefaultRole: "code.default", Roles: map[string]rolepolicy.Role{
		"code.z": {Intent: "last", Candidates: []rolepolicy.Candidate{{Runner: domain.RunnerTypeCodex, ResourceRole: "code.z"}}},
		"code.a": {Intent: "first", Candidates: []rolepolicy.Candidate{{Runner: domain.RunnerTypeClaudeCode, ResourceRole: "code.a"}}},
	}})
	if len(catalog.GetRoles()) != 2 || catalog.GetRoles()[0].GetRoleRef() != "code.a" || catalog.GetRoles()[1].GetCandidates()[0].GetRunnerType() == 0 {
		t.Fatalf("catalog=%+v", catalog)
	}
}

func TestPermissionPolicyProjectionsPreservePlanAndReconcileEvidence(t *testing.T) {
	now := time.Now().UTC()
	status := PermissionPolicyStatusToProto(permissionpolicy.Status{
		Path: "/permissions.json", Requirement: permissionpolicy.Requirement{Required: true, Reason: "required"}, Ready: true, ActiveDigest: "sha256:permissions", ActivatedAt: &now,
		LastReloadAttempt: &permissionpolicy.ReloadAttempt{AttemptedAt: now, Succeeded: false, Diagnostic: &permissionpolicy.Diagnostic{Code: "invalid", Message: "bad catalog"}},
	})
	if status.GetLastReloadAttempt().GetDiagnostic().GetMessage() != "bad catalog" {
		t.Fatalf("status=%+v", status)
	}
	catalog := PermissionPolicyCatalogToProto(&permissionpolicy.Catalog{SchemaVersion: 1, Metadata: permissionpolicy.Metadata{CatalogID: "permissions", UpdatedAt: "2026-07-23"}, TargetScopes: []string{"workspace"}, Rules: []permissionpolicy.Rule{{ID: "allow-read", Action: "allow", Matcher: permissionpolicy.Matcher{Kind: "tool", Pattern: "read"}, TargetScope: "workspace", RequiresHardEnforcement: true}}})
	if len(catalog.GetRules()) != 1 || catalog.GetRules()[0].GetMatcher().GetPattern() != "read" {
		t.Fatalf("catalog=%+v", catalog)
	}
	plan := PermissionPolicyPlanToProto(permissionpolicy.AggregatePlan{CatalogDigest: "sha256:permissions", HardEnforcementSatisfied: false, MissingHardEnforcementRuleIDs: []string{"allow-read"}})
	if plan.GetHardEnforcementSatisfied() || len(plan.GetMissingHardEnforcementRuleIds()) != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	if PermissionPolicyDiagnosticToProto(nil) != nil || PermissionPolicyCatalogToProto(nil) != nil || PermissionPolicyReconcileResultToProto(nil) != nil {
		t.Fatal("nil permission policy projections should remain nil")
	}
}
