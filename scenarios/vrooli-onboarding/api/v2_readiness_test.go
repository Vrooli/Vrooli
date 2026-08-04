package main

import (
	"context"
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
	w := doGet(t, NewServer(), "/api/v2/readiness")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"status":"ready"`) || !strings.Contains(body, `"logical_id":"vrooli/demo"`) || strings.Contains(body, "secret-value") {
		t.Fatalf("readiness response = %s", body)
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
	for _, want := range []string{`"status":"missing"`, `"name":"missing_tool"`, `"kind":"tool"`, `"name":"integration-hub"`, `"status":"deferred"`} {
		if !strings.Contains(body, want) {
			t.Errorf("readiness missing %s: %s", want, body)
		}
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
		{name: "tunnel-manager", fields: []string{"cloudflare-account-id", "cloudflare-tunnel-id", "cloudflare-api-token"}},
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
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"landing-page-business-suite":{"enabled":true},"tunnel-manager":{"enabled":true}}}`), 0o600); err != nil {
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
	for _, field := range []string{"session-secret", "service-secret", "api-key-encryption-key", "remote-profile-encryption-key", "admin-default-password", "cloudflare-account-id", "cloudflare-tunnel-id", "cloudflare-api-token"} {
		if !strings.Contains(body, `"field":"`+field+`"`) {
			t.Errorf("readiness missing scenario credential field %q: %s", field, body)
		}
	}
}
