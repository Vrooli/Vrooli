package policy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadRightsContextFileRequiresCompleteVersionedOperatorContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "context.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"1","context":{"id":"operator","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRightsContextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "operator" || got.EntityScope != "" || got.RevenueStatus != "pre_revenue" {
		t.Fatalf("context = %+v", got)
	}
	artifact := testConditionalRights()
	decision := EvaluateRights(&artifact, RightsUseLocalInference, got, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if decision.Status != RightsUnknown || decision.Reason != "rights_facts_incomplete" {
		t.Fatalf("context without entity scope must remain unknown: %+v", decision)
	}
	unrestricted := testUnrestrictedRights()
	decision = EvaluateRights(&unrestricted, RightsUseLocalInference, got, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if decision.Status != RightsAllowed || decision.Reason != "verified_unrestricted_grant" {
		t.Fatalf("complete trusted context without a conditional entity scope should allow an unrestricted grant: %+v", decision)
	}
	for _, document := range []string{
		`{"schema_version":"2","context":{"id":"operator","version":"v1","as_of":"2026-11-10","source":"operator_statement"}}`,
		`{"schema_version":"1","context":{"id":"operator","version":"v1","as_of":"2026-11-10","source":"operator_statement"}}`,
		`{"schema_version":"1","context":{"id":"operator","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10"}}`,
	} {
		if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadRightsContextFile(path); err == nil {
			t.Fatalf("accepted incomplete context %s", document)
		}
	}
}

func TestResolveRightsContextPathUsesOnlyConfiguredOrCanonicalOwnerPath(t *testing.T) {
	root := t.TempDir()
	contextPath := filepath.Join(root, "scenarios", "ai-gateway", "config", "model-rights-context.json")
	if err := os.MkdirAll(filepath.Dir(contextPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextPath, []byte(`{"schema_version":"1","context":{"id":"operator","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRightsContextPath(func(key string) string {
		if key == "RESOURCE_ROOT" {
			return filepath.Join(root, "resources", "ollaya")
		}
		return ""
	})
	if err != nil || got != contextPath {
		t.Fatalf("resource-root context path=%q err=%v, want %q", got, err, contextPath)
	}
	override := filepath.Join(t.TempDir(), "managed-context.json")
	got, err = ResolveRightsContextPath(func(key string) string {
		if key == RightsContextPathEnv {
			return override
		}
		return ""
	})
	if err != nil || got != override {
		t.Fatalf("configured context path=%q err=%v, want %q", got, err, override)
	}
}

func TestCanonicalContextRetainsOperatorSuppliedHistoricalMAUCountsAndEvidence(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	contextPath := filepath.Join(repo, "scenarios/ai-gateway/config/model-rights-context.json")
	got, err := LoadRightsContextFile(contextPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2023-06", "2024-06", "2024-08"}
	for _, period := range want {
		if got.MonthlyActiveUsers[period] != "0" || !strings.Contains(got.MonthlyActiveUsersEvidence[period], "OPERATOR-ANSWERS-20261114.md#LM5-RIGHTS-01") {
			t.Errorf("trusted MAU %s = %q, evidence %q", period, got.MonthlyActiveUsers[period], got.MonthlyActiveUsersEvidence[period])
		}
	}
}

func testUnrestrictedRights() ArtifactRights {
	artifact := testConditionalRights()
	artifact.Classification = RightsUnrestricted
	artifact.Conditions = nil
	return artifact
}
