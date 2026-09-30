package appctx

import (
	"testing"

	"connectrpc.com/connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
)

func TestDecodeIgnoresNilResultAndEmptyBody(t *testing.T) {
	if err := decode([]byte(`{"ok":true}`), nil); err != nil {
		t.Fatalf("nil result should not decode: %v", err)
	}
	var result map[string]bool
	if err := decode(nil, &result); err != nil {
		t.Fatalf("empty body should not decode: %v", err)
	}
}

func TestDecodeUnmarshalsJSON(t *testing.T) {
	var result struct {
		OK bool `json:"ok"`
	}
	if err := decode([]byte(`{"ok":true}`), &result); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !result.OK {
		t.Fatal("expected decoded true value")
	}
}

func TestRPCBodyUnwrapsAnEmptyRepeatedFieldAsJSONArray(t *testing.T) {
	body, handled, err := rpcBody(connect.NewResponse(&skillsv1.ListSkillVariantsResponse{}), "variants", true, nil)
	if err != nil {
		t.Fatalf("unwrap empty variants: %v", err)
	}
	if !handled || string(body) != "[]" {
		t.Fatalf("body=%s handled=%t", body, handled)
	}
}

func TestNormalizeInt64JSONFieldRestoresLegacyNumber(t *testing.T) {
	body, err := normalizeInt64JSONField([]byte(`{"durationMs":"42","status":"dry-run"}`), "durationMs")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"durationMs":42,"status":"dry-run"}` {
		t.Fatalf("body=%s", body)
	}
}

func TestRenameJSONFieldPreservesLegacyDryRunShape(t *testing.T) {
	body, err := renameJSONField([]byte(`{"dryRun":true,"plan":{}}`), "dryRun", "dry_run")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"dry_run":true,"plan":{}}` {
		t.Fatalf("unexpected body: %s", body)
	}
}
