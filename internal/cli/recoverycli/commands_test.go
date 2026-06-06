package recoverycli

import (
	"testing"
	"time"
)

func TestParseCaptureRequest(t *testing.T) {
	req, err := ParseCaptureRequest([]string{"--scenario", "demo", "--slug", "abc", "--source", "/tmp/x", "--no-reflink"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Scenario != "demo" || req.Slug != "abc" || req.Source != "/tmp/x" || !req.NoReflink {
		t.Fatalf("unexpected capture request: %+v", req)
	}
}

func TestParseNamespaceRequestDefaultsToShadow(t *testing.T) {
	req, err := ParseNamespaceRequest([]string{"--scenario", "demo"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Scenario != "demo" || req.Variant != "shadow" {
		t.Fatalf("omitted --variant should default to shadow: %+v", req)
	}
}

func TestParseNamespaceRequestExplicitVariant(t *testing.T) {
	req, err := ParseNamespaceRequest([]string{"--scenario", "demo", "--variant", "live"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Variant != "live" {
		t.Fatalf("variant = %q, want live", req.Variant)
	}
}

func TestParseWriteRequestMapsFlags(t *testing.T) {
	req, err := ParseWriteRequest([]string{
		"--scenario", "demo", "--slug", "abc",
		"--mode", "shadow", "--variant", "shadow",
		"--ttl", "3h", "--ambient-var", "demo",
		"--shadow-instance-key", "demo@shadow", "--anchor", "base-1",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Mode != "shadow" || req.Variant != "shadow" {
		t.Fatalf("mode/variant not mapped: %+v", req)
	}
	if req.TTL != 3*time.Hour {
		t.Fatalf("ttl = %v, want 3h", req.TTL)
	}
	if req.AmbientVar != "demo" || req.ShadowInstanceKey != "demo@shadow" || req.Anchor != "base-1" {
		t.Fatalf("optional fields not mapped: %+v", req)
	}
}

func TestParseWriteRequestOmittedTTLIsZero(t *testing.T) {
	req, err := ParseWriteRequest([]string{"--scenario", "demo", "--slug", "abc", "--mode", "live"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.TTL != 0 {
		t.Fatalf("omitted ttl should be 0, got %v", req.TTL)
	}
}

func TestParseWriteRequestInvalidTTL(t *testing.T) {
	if _, err := ParseWriteRequest([]string{"--scenario", "demo", "--slug", "abc", "--mode", "shadow", "--ttl", "soon"}); err == nil {
		t.Fatal("expected invalid-ttl error")
	}
}

func TestParseRefRequest(t *testing.T) {
	req, err := ParseRefRequest(CommandShow, "recovery show", []string{"--scenario", "demo", "--slug", "abc"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Scenario != "demo" || req.Slug != "abc" {
		t.Fatalf("unexpected ref: %+v", req)
	}
}

func TestParseMigrateRequest(t *testing.T) {
	req, err := ParseMigrateRequest([]string{
		"--scenario", "demo", "--slug", "wip",
		"--engine", "sqlite", "--db-path", "/tmp/live.db",
		"--migrations-dir", "/tmp/m", "--dry-run",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Scenario != "demo" || req.Slug != "wip" || req.Engine != "sqlite" {
		t.Fatalf("core fields not mapped: %+v", req)
	}
	if req.DBPath != "/tmp/live.db" || req.MigrationsDir != "/tmp/m" || !req.DryRun {
		t.Fatalf("optional fields not mapped: %+v", req)
	}
}

func TestParseMigrateRequestDefaults(t *testing.T) {
	req, err := ParseMigrateRequest([]string{"--scenario", "demo", "--slug", "wip"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Engine != "" || req.DBPath != "" || req.DryRun {
		t.Fatalf("omitted flags should be zero-valued: %+v", req)
	}
}

func TestParseSetTTLRequest(t *testing.T) {
	req, err := ParseSetTTLRequest([]string{"--scenario", "demo", "--slug", "abc", "--ttl", "6h"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.TTL != 6*time.Hour {
		t.Fatalf("ttl = %v, want 6h", req.TTL)
	}
}

// TestCommandSpecCoverage guards that every declared command has a spec lookup
// (commandSpec panics otherwise), so a new command can't ship without a spec.
func TestCommandSpecCoverage(t *testing.T) {
	for _, id := range []CommandID{
		CommandCapture, CommandRestore, CommandWrite, CommandShow,
		CommandList, CommandTouch, CommandSetTTL, CommandClean, CommandMigrate,
	} {
		if got := commandSpec(id); got.Handler != id {
			t.Fatalf("spec for %q resolved to %q", id, got.Handler)
		}
	}
}
