package ensure

import (
	"bytes"
	"context"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
)

func testCatalog(t *testing.T) policy.Catalog {
	t.Helper()
	catalog, err := policy.Parse([]byte(`{
      "schema_version": "1",
      "roles": {"chat.default": {"model": "m", "endpoint": "chat", "required_capabilities": ["chat"], "preference": 1, "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}},
      "models": {"m": {"provider": "p", "family": "f", "capabilities": ["chat"], "modalities": {"input": ["text"], "output": ["text"]}, "endpoints": ["chat"], "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}}},
      "constraints": {"endpoints": ["chat"], "capability_vocabulary": ["chat"], "role_preference_order": ["chat.default"], "provenance_source_kinds": ["manual_policy"], "modality_vocabulary": ["text"]},
      "provenance": {"policy": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}
    }`), policy.CloudOptions())
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

type checker struct {
	present map[string]bool
	err     error
}

func (c checker) Present(context.Context, []string, []string) (map[string]bool, error) {
	return c.present, c.err
}

func TestRunValidatesRoles(t *testing.T) {
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}}}
	var out bytes.Buffer
	if err := Run(context.Background(), cfg, testCatalog(t), checker{present: map[string]bool{"m": true}}, &out, "test:"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("resolved role chat.default -> m")) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRunRejectsDirectModel(t *testing.T) {
	cfg := Config{DeprecatedModel: []byte(`"m"`)}
	if err := Run(context.Background(), cfg, testCatalog(t), nil, &bytes.Buffer{}, "test:"); err == nil {
		t.Fatal("expected deprecated field rejection")
	}
}

func TestRunRejectsUnknownRole(t *testing.T) {
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "nope"}}}
	if err := Run(context.Background(), cfg, testCatalog(t), nil, &bytes.Buffer{}, "test:"); err == nil {
		t.Fatal("expected unknown role rejection")
	}
}

func TestRunDegradesCatalogCheck(t *testing.T) {
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}}}
	var out bytes.Buffer
	if err := Run(context.Background(), cfg, testCatalog(t), checker{err: context.DeadlineExceeded}, &out, "test:"); err != nil {
		t.Fatalf("catalog uncertainty must not fail ensure: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("degraded")) {
		t.Fatalf("output = %s", out.String())
	}
}
