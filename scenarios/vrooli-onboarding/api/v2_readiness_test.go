package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2ReadinessReportsOnlyMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	oldPath := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = oldPath })
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "alpha", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"},"dependencies":{"resources":{"demo":{}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "demo", "resource.json"), []byte(`{"credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"api-key","label":"Demo API key","required":true}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"alpha":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) {
		return []byte(`{"identity":"vrooli/demo","field":"api-key","configured":true,"provider":"native-secure-store"}`), nil
	}
	t.Cleanup(func() { credentialStatusCommand = prior })
	releasePrior := releaseAuthorityStatusCommand
	releaseAuthorityStatusCommand = func(context.Context, string) ([]byte, error) {
		return []byte(`{"configured":true,"trust_anchor_match":true,"provider":"native-secure-store"}`), nil
	}
	t.Cleanup(func() { releaseAuthorityStatusCommand = releasePrior })
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"status":"ready"`) || !strings.Contains(body, `"logical_id":"vrooli/demo"`) || !strings.Contains(body, `"name":"release-authority"`) || strings.Contains(body, "secret-value") {
		t.Fatalf("readiness response = %s", body)
	}
}

func TestReleaseAuthorityReadinessNamesRemediation(t *testing.T) {
	prior := releaseAuthorityStatusCommand
	t.Cleanup(func() { releaseAuthorityStatusCommand = prior })
	cases := []struct {
		name   string
		output string
		status string
		want   string
	}{
		{name: "missing", output: `{"configured":false}`, status: "missing", want: "vrooli release-authority init"},
		{name: "mismatched", output: `{"configured":true,"trust_anchor_match":false}`, status: "degraded", want: "--replace-trust-anchor"},
		{name: "ready", output: `{"configured":true,"trust_anchor_match":true}`, status: "ready", want: "synchronized"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			releaseAuthorityStatusCommand = func(context.Context, string) ([]byte, error) { return []byte(tc.output), nil }
			item := releaseAuthorityReadiness(t.TempDir())
			if item.Status != tc.status || !strings.Contains(item.Detail+item.Remediation, tc.want) {
				t.Fatalf("item = %+v, want status %q and %q", item, tc.status, tc.want)
			}
		})
	}
}

func TestV2ReadinessReportsMissingRequiredHostTool(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	prior := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = prior })
	for _, path := range []string{filepath.Join(root, "scenarios", "alpha", ".vrooli"), filepath.Join(root, "internal", "tools", "missing-tool"), filepath.Join(root, ".vrooli")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"},"hostTools":[{"name":"missing_tool","required":true,"reason":"required for test"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "tools", "missing-tool", "tool.json"), []byte(`{"name":"missing_tool","commands":["vrooli-test-intentionally-missing-command"],"platforms":["linux","macos","windows"],"bundling":"host-required"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"alpha":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"status":"missing"`, `"name":"missing_tool"`, `"kind":"tool"`, `"name":"release-authority"`} {
		if !strings.Contains(body, want) {
			t.Errorf("readiness missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "integration-hub") {
		t.Fatalf("readiness fabricated an integration provider: %s", body)
	}
}

func TestV2ReadinessReportsDeclaredIntegrationsWithoutFabricatingProviders(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	oldPath := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = oldPath })
	for _, path := range []string{filepath.Join(root, "scenarios", "alpha", ".vrooli"), filepath.Join(root, ".vrooli")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"},"integrations":[{"connector":"github-oauth","scopes":["repo:read"],"purpose":"Read project issues","required":true}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"alpha":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	releasePrior := releaseAuthorityStatusCommand
	releaseAuthorityStatusCommand = func(context.Context, string) ([]byte, error) {
		return []byte(`{"configured":true,"trust_anchor_match":true}`), nil
	}
	t.Cleanup(func() { releaseAuthorityStatusCommand = releasePrior })
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"alpha/github-oauth"`) || !strings.Contains(w.Body.String(), "Read project issues") || !strings.Contains(w.Body.String(), "repo:read") {
		t.Fatalf("readiness = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2ReadinessIncludesScenarioCredentialDeclarations(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	oldPath := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = oldPath })
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name   string
		fields []string
	}{
		{name: "landing-page-business-suite", fields: []string{"session-secret", "service-secret", "api-key-encryption-key", "remote-profile-encryption-key", "admin-default-password"}},
		{name: "demo-alpha", fields: []string{"first-id", "second-id", "operator-token"}},
	} {
		dir := filepath.Join(root, "scenarios", scenario.name, ".vrooli")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		descriptors := make([]string, 0, len(scenario.fields))
		for _, field := range scenario.fields {
			descriptors = append(descriptors, fmt.Sprintf(`{"logical_id":"vrooli/%s","field":"%s","label":"%s","required":true}`, scenario.name, field, field))
		}
		manifest := fmt.Sprintf(`{"service":{"name":"%s"},"credentials":{"descriptors":[%s]}}`, scenario.name, strings.Join(descriptors, ","))
		if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"landing-page-business-suite":{"enabled":true},"demo-alpha":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) { return []byte(`{"configured":true}`), nil }
	t.Cleanup(func() { credentialStatusCommand = prior })
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, field := range []string{"session-secret", "service-secret", "api-key-encryption-key", "remote-profile-encryption-key", "admin-default-password", "first-id", "second-id", "operator-token"} {
		if !strings.Contains(body, `"field":"`+field+`"`) {
			t.Errorf("readiness missing scenario credential field %q: %s", field, body)
		}
	}
}

func TestCredentialReadinessCarriesProvisioningMetadataGenerically(t *testing.T) {
	previous := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) {
		return []byte(`{"configured":false}`), nil
	}
	t.Cleanup(func() { credentialStatusCommand = previous })

	items := credentialReadinessForDescriptors("demo", []readinessCredentialDescriptor{
		{LogicalID: "vrooli/demo", Field: "operator-key", Required: true, Provisioning: "operator"},
		{LogicalID: "vrooli/demo", Field: "derived-key", Required: true, Provisioning: "derived", DerivedFrom: "operator-key"},
	})
	if len(items) != 2 || items[1].Provisioning != "derived" || items[1].DerivedFrom != "operator-key" {
		t.Fatalf("readiness items = %+v", items)
	}
}

func TestRecoveryReadinessCarriesEscrowClassesAndRootCopyIssues(t *testing.T) {
	var diagnosis credentialDiagnosisResponse
	err := json.Unmarshal([]byte(`{
		"recovery": {
			"receipt_exists": true,
			"entry_count": 2,
			"uncovered": ["vrooli/example:stale"],
			"required_absent": ["vrooli/example:required"],
			"required_absent_details": [{"address":"vrooli/example:required","description":"Required integration key."}],
			"root_copy": null,
			"root_copy_issues": ["no encrypted credential-store copy receipt exists"]
		}
	}`), &diagnosis)
	if err != nil {
		t.Fatal(err)
	}
	got := diagnosis.Recovery
	if len(got.Uncovered) != 1 || len(got.RequiredAbsent) != 1 || len(got.RequiredAbsentDetails) != 1 || len(got.RootCopyIssues) != 1 {
		t.Fatalf("recovery projection = %+v", got)
	}
	if got.RequiredAbsentDetails[0].Description != "Required integration key." {
		t.Fatalf("required-absent detail = %+v", got.RequiredAbsentDetails[0])
	}
}

func TestV2ReadinessElevatesRequiredAbsentRecoveryToMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	priorStatePath := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = priorStatePath })
	for _, dir := range []string{filepath.Join(root, "scenarios", "alpha", ".vrooli"), filepath.Join(root, ".vrooli")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"alpha":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	previousDoctor := credentialDoctorCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return []byte(`{"recovery":{"receipt_exists":true,"entry_count":1,"uncovered":[],"required_absent":["vrooli/example:api-key"],"required_absent_details":[{"address":"vrooli/example:api-key","description":"Required key."}],"root_copy_issues":[]}}`), nil
	}
	t.Cleanup(func() { credentialDoctorCommand = previousDoctor })
	previousRelease := releaseAuthorityStatusCommand
	releaseAuthorityStatusCommand = func(context.Context, string) ([]byte, error) {
		return []byte(`{"configured":true,"trust_anchor_match":true}`), nil
	}
	t.Cleanup(func() { releaseAuthorityStatusCommand = previousRelease })
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"missing"`) || !strings.Contains(w.Body.String(), `vrooli/example:api-key`) {
		t.Fatalf("readiness = %s", w.Body.String())
	}
}
