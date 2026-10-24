package recoverycli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/cliout"
)

// TestRenderCaptureJSONContract pins the `recovery capture --json` wire shape:
// bare payload (no envelope), snake_case keys, nested stats, integer fields as
// JSON numbers.
func TestRenderCaptureJSONContract(t *testing.T) {
	resp := recoveryapp.CaptureOutput{
		Scenario:         "swarm-manager",
		Slug:             "feat-x",
		Source:           "/src",
		RestorePointPath: "/rp",
		Stats: baselinefloor.CopyStats{
			Dirs:          3,
			Symlinks:      1,
			ReflinkFiles:  4,
			DeepCopyFiles: 2,
			BytesCopied:   9999,
			Excluded:      5,
		},
	}

	var buf bytes.Buffer
	if err := RenderCapture(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderCapture: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	// Bare payload: no success envelope.
	if _, ok := got["success"]; ok {
		t.Errorf("unexpected success envelope: %v", got)
	}
	if got["scenario"] != "swarm-manager" || got["slug"] != "feat-x" {
		t.Errorf("scenario/slug mismatch: %v", got)
	}
	if got["restore_point_path"] != "/rp" {
		t.Errorf("restore_point_path (snake_case?): %v", got)
	}
	stats, ok := got["stats"].(map[string]any)
	if !ok {
		t.Fatalf("stats missing/wrong type: %v", got["stats"])
	}
	// int32 fields must be JSON numbers (float64), not strings.
	for _, k := range []string{"dirs", "symlinks", "reflink_files", "deep_copy_files", "excluded"} {
		if _, ok := stats[k].(float64); !ok {
			t.Errorf("stats[%q] must be a JSON number, got %T (%v)", k, stats[k], stats[k])
		}
	}
	if stats["dirs"].(float64) != 3 {
		t.Errorf("dirs value: %v", stats["dirs"])
	}
	// bytes_copied is int64 (working tree can exceed 2 GiB); protojson serializes
	// int64 as a JSON string by design.
	if stats["bytes_copied"] != "9999" {
		t.Errorf("bytes_copied: want JSON string \"9999\", got %T (%v)", stats["bytes_copied"], stats["bytes_copied"])
	}
}

// TestRenderEngagementJSONContract pins the embedded-Manifest flattening, the
// duration-string ttl, and the sparse (no-TTL) case where expires_at is "".
func TestRenderEngagementJSONContract(t *testing.T) {
	created := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	view := recoveryapp.EngagementView{
		Manifest: baselinefloor.Manifest{
			Scenario:          "swarm-manager",
			Slug:              "feat-x",
			Variant:           "shadow",
			Mode:              baselinefloor.ModeShadow,
			RestorePointPath:  "/rp",
			ShadowInstanceKey: "swarm-manager@shadow",
			CreatedAt:         created,
			LastTouchedAt:     created,
			TTL:               baselinefloor.Duration(3 * time.Hour),
		},
		Expired: false,
		// ExpiresAt nil -> "" (sparse case).
	}

	var buf bytes.Buffer
	if err := RenderEngagement(&buf, cliout.FormatJSON, view); err != nil {
		t.Fatalf("RenderEngagement: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["mode"] != "shadow" || got["variant"] != "shadow" {
		t.Errorf("mode/variant mismatch: %v", got)
	}
	if got["shadow_instance_key"] != "swarm-manager@shadow" {
		t.Errorf("shadow_instance_key (snake_case?): %v", got)
	}
	if got["ttl"] != "3h0m0s" {
		t.Errorf("ttl: want duration string, got %v", got["ttl"])
	}
	if got["created_at"] != "2026-06-11T10:00:00Z" {
		t.Errorf("created_at: %v", got["created_at"])
	}
	if got["expires_at"] != "" {
		t.Errorf("expires_at: want empty for no-TTL, got %v", got["expires_at"])
	}
	if got["expired"] != false {
		t.Errorf("expired: %v", got["expired"])
	}
}

// TestRenderMigrateJSONContract pins the embedded MigrationResult flattening and
// numeric scripts_seen.
func TestRenderMigrateJSONContract(t *testing.T) {
	resp := recoveryapp.MigrateOutput{
		Scenario:           "swarm-manager",
		Slug:               "feat-x",
		MigrationsDir:      "/m",
		DBPathAutoResolved: true,
		MigrationResult: baselinefloor.MigrationResult{
			Engine:      baselinefloor.EngineSQLite,
			Database:    "/db.sqlite",
			DryRun:      false,
			FastPath:    false,
			ScriptsSeen: 2,
			Applied:     []string{"001.sql"},
			Skipped:     []string{"000.sql"},
		},
	}

	var buf bytes.Buffer
	if err := RenderMigrate(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderMigrate: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["db_path_auto_resolved"] != true {
		t.Errorf("db_path_auto_resolved: %v", got["db_path_auto_resolved"])
	}
	if got["engine"] != "sqlite" {
		t.Errorf("engine: %v", got["engine"])
	}
	if n, ok := got["scripts_seen"].(float64); !ok || n != 2 {
		t.Errorf("scripts_seen must be JSON number 2, got %T (%v)", got["scripts_seen"], got["scripts_seen"])
	}
	applied, ok := got["applied"].([]any)
	if !ok || len(applied) != 1 || applied[0] != "001.sql" {
		t.Errorf("applied: %v", got["applied"])
	}
}

// TestRenderListJSONContract pins the list envelope and nested engagement view.
func TestRenderListJSONContract(t *testing.T) {
	expires := time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC)
	resp := recoveryapp.ListOutput{
		Engagements: []recoveryapp.EngagementView{
			{
				Manifest: baselinefloor.Manifest{
					Scenario: "a", Slug: "s", Variant: "live", Mode: baselinefloor.ModeLive,
				},
				ExpiresAt: &expires,
				Expired:   true,
			},
		},
	}

	var buf bytes.Buffer
	if err := RenderList(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	engs, ok := got["engagements"].([]any)
	if !ok || len(engs) != 1 {
		t.Fatalf("engagements: want 1, got %v", got["engagements"])
	}
	first := engs[0].(map[string]any)
	if first["scenario"] != "a" || first["mode"] != "live" {
		t.Errorf("first engagement mismatch: %v", first)
	}
	if first["expires_at"] != "2026-06-11T13:00:00Z" || first["expired"] != true {
		t.Errorf("expires_at/expired mismatch: %v", first)
	}
}

// TestRenderNamespaceJSONContract pins the namespace addressing payload.
func TestRenderNamespaceJSONContract(t *testing.T) {
	resp := recoveryapp.NamespaceOutput{
		Scenario:         "swarm-manager",
		Variant:          "shadow",
		InstanceKey:      "swarm-manager@shadow",
		PostgresDb:       "vrooli_swarm-manager_shadow",
		DataDir:          "/data/swarm-manager@shadow",
		DataDirName:      "swarm-manager@shadow",
		StorageNamespace: "swarm-manager_shadow",
	}

	var buf bytes.Buffer
	if err := RenderNamespace(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderNamespace: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["instance_key"] != "swarm-manager@shadow" {
		t.Errorf("instance_key (snake_case?): %v", got)
	}
	if got["postgres_db"] != "vrooli_swarm-manager_shadow" {
		t.Errorf("postgres_db: %v", got["postgres_db"])
	}
	if got["storage_namespace"] != "swarm-manager_shadow" {
		t.Errorf("storage_namespace: %v", got["storage_namespace"])
	}
}
