package validation

import (
	"context"
	"sync"
	"testing"
)

type advisoryIdentityCommander struct {
	version string
}

func (c advisoryIdentityCommander) LookPath(string) (string, error) { return "/logical/tool", nil }

func (c advisoryIdentityCommander) Run(_ context.Context, _ string, name string, _ ...string) ([]byte, []byte, int, error) {
	if name != "govulncheck" {
		return nil, nil, 0, nil
	}
	return []byte(c.version), nil, 0, nil
}

func TestAdvisoryDatabaseIdentityUsesReportedGovulncheckDatabase(t *testing.T) {
	advisoryDatabaseCache = sync.Map{}
	identity, ok := advisoryDatabaseIdentity(context.Background(), advisoryIdentityCommander{version: "DB updated: 2026-08-28 14:47:45 +0000 UTC"}, "govulncheck")
	if !ok || identity != "2026-08-28 14:47:45 +0000 UTC" {
		t.Fatalf("identity = %q, ok=%t", identity, ok)
	}
}

func TestAdvisoryDatabaseIdentityFallsBackForOSVScanner(t *testing.T) {
	advisoryDatabaseCache = sync.Map{}
	if identity, ok := advisoryDatabaseIdentity(context.Background(), advisoryIdentityCommander{}, "osv-scanner"); ok || identity != "" {
		t.Fatalf("osv identity = %q, ok=%t; want fallback", identity, ok)
	}
}
