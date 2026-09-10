package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func doReadiness(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	response, err := buildReadinessResponse(context.Background())
	if err != nil {
		t.Fatalf("build readiness: %v", err)
	}
	w := httptest.NewRecorder()
	w.Code = http.StatusOK
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestV2ReadinessReportsOnlyMetadata(t *testing.T) {
	root := newV2Root(t)
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
	w := doReadiness(t)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"status":"ready"`) || !strings.Contains(body, `"logical_id":"vrooli/demo"`) || !strings.Contains(body, `"name":"release-authority"`) || strings.Contains(body, "secret-value") {
		t.Fatalf("readiness response = %s", body)
	}
}

func TestCredentialReadinessInventoryMergesSharedAddressProvenance(t *testing.T) {
	root := newV2Root(t)
	for _, name := range []string{"alpha", "gamma"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", name, ".vrooli"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{
  "service":{"name":"alpha"},
  "credentials":{"descriptors":[{"logical_id":"vrooli/shared","field":"token","required":true}],"consumers":[{"logical_id":"vrooli/shared","field":"token","kind":"delegated","consumer":"alpha broker","source_ref":"api/alpha.go:7"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "gamma", ".vrooli", "service.json"), []byte(`{
  "service":{"name":"gamma"},
  "credentials":{"descriptors":[{"logical_id":"vrooli/shared","field":"token","required":false}],"consumers":[{"logical_id":"vrooli/shared","field":"token","kind":"external","consumer":"gamma provider","source_ref":"api/gamma.go:9"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	previous := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) { return []byte(`{"configured":true}`), nil }
	t.Cleanup(func() { credentialStatusCommand = previous })
	items, err := credentialReadinessInventoryContext(context.Background(), closureResult{
		Scenarios: []closureMember{{Name: "alpha"}, {Name: "gamma"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LogicalID != "vrooli/shared" || !items[0].Required || items[0].Status != "configured" {
		t.Fatalf("shared readiness items = %+v, want one configured aggregate", items)
	}
	if items[0].EvidenceStatus != credentialEvidenceUnverified || items[0].EvidenceDetail == "" {
		t.Fatalf("configured credential evidence = %q/%q, want explicit unverified state", items[0].EvidenceStatus, items[0].EvidenceDetail)
	}
	if len(items[0].Provenance) != 2 || len(items[0].Provenance[0].Consumers) != 1 || len(items[0].Provenance[1].Consumers) != 1 {
		t.Fatalf("shared readiness provenance = %+v, want both consumer declarations", items[0].Provenance)
	}
}

func TestCredentialReadinessDoesNotClaimEvidenceForStoredValue(t *testing.T) {
	previous := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) {
		return []byte(`{"configured":true}`), nil
	}
	t.Cleanup(func() { credentialStatusCommand = previous })

	items := credentialReadinessForRefs(context.Background(), []credentialclient.CredentialRef{{Resource: "demo", LogicalID: "vrooli/demo", Field: "token", Required: true}}, true)
	if len(items) != 1 || items[0].Status != "configured" {
		t.Fatalf("readiness items = %+v, want configured storage state", items)
	}
	if items[0].EvidenceStatus == "verified" || items[0].EvidenceStatus != credentialEvidenceUnverified {
		t.Fatalf("stored value was presented as exercised evidence: %+v", items[0])
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
	root := newV2Root(t)
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

	w := doReadiness(t)
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
	root := newV2Root(t)
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
	w := doReadiness(t)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"alpha/github-oauth"`) || !strings.Contains(w.Body.String(), "Read project issues") || !strings.Contains(w.Body.String(), "repo:read") {
		t.Fatalf("readiness = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2ReadinessIncludesScenarioCredentialDeclarations(t *testing.T) {
	root := newV2Root(t)
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
	w := doReadiness(t)
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

	items := credentialReadinessForRefs(context.Background(), []credentialclient.CredentialRef{
		{Resource: "demo", LogicalID: "vrooli/demo", Field: "operator-key", Required: true, Provisioning: "operator"},
		{Resource: "demo", LogicalID: "vrooli/demo", Field: "derived-key", Required: true, Provisioning: "derived", DerivedFrom: "operator-key"},
	}, true)
	if len(items) != 2 || items[1].Provisioning != "derived" || items[1].DerivedFrom != "operator-key" {
		t.Fatalf("readiness items = %+v", items)
	}
}

func TestCredentialReadinessProbesIndependentDescriptorsConcurrently(t *testing.T) {
	previous := credentialStatusCommand
	t.Cleanup(func() { credentialStatusCommand = previous })

	const descriptorCount = 4
	started := make(chan struct{}, descriptorCount)
	release := make(chan struct{})
	var calls atomic.Int32
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return []byte(`{"configured":false}`), nil
	}

	refs := make([]credentialclient.CredentialRef, descriptorCount)
	for index := range refs {
		refs[index] = credentialclient.CredentialRef{Resource: "demo", LogicalID: fmt.Sprintf("vrooli/demo-%d", index), Field: "key", Required: true}
	}
	done := make(chan []credentialReadiness, 1)
	go func() { done <- credentialReadinessForRefs(context.Background(), refs, true) }()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for range refs {
		select {
		case <-started:
		case <-deadline.C:
			close(release)
			t.Fatal("credential readiness probes did not start concurrently")
		}
	}
	close(release)

	select {
	case items := <-done:
		if len(items) != descriptorCount || calls.Load() != descriptorCount {
			t.Fatalf("credential readiness items = %d, calls = %d", len(items), calls.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("credential readiness probes did not complete")
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
	root := newV2Root(t)
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
	w := doReadiness(t)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"missing"`) || !strings.Contains(w.Body.String(), `vrooli/example:api-key`) {
		t.Fatalf("readiness = %s", w.Body.String())
	}
}
