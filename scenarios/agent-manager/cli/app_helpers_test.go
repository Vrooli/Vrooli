package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/types/known/timestamppb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestAppHelpersNormalizeOperationalInputs(t *testing.T) {
	if formatEnumValue(domainpb.RunMode_RUN_MODE_IN_PLACE, "RUN_MODE_", "-") != "in-place" || formatEnumValue(nil, "", "_") != "" {
		t.Fatal("enum formatting")
	}
	if got := formatTimestamp(timestamppb.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("local", -5*3600)))); got != "2026-01-02T08:04:05Z" || formatTimestamp(nil) != "" {
		t.Fatalf("timestamp=%q", got)
	}
	if trimTimestamp("2026-01-02T03:04:05.123Z") != "2026-01-02T03:04:05" || formatDuration(nil) != "" {
		t.Fatal("timestamp/duration formatting")
	}
	if parseRunMode("in-place") != domainpb.RunMode_RUN_MODE_IN_PLACE || parseRunMode("sandboxed") != domainpb.RunMode_RUN_MODE_SANDBOXED || parseRunMode("other") != domainpb.RunMode_RUN_MODE_UNSPECIFIED {
		t.Fatal("run mode parsing")
	}
	if parseExecutionMode("codec-pipe") != domainpb.ExecutionMode_EXECUTION_MODE_CODEC_PIPE || parseExecutionMode("interactive") != domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE || parseExecutionMode("attached") != domainpb.ExecutionMode_EXECUTION_MODE_ATTACHED || parseExecutionMode("other") != domainpb.ExecutionMode_EXECUTION_MODE_UNSPECIFIED {
		t.Fatal("execution mode parsing")
	}
	if protoString("") != nil || *protoString("value") != "value" || !strings.Contains(marshalProtoJSON(&domainpb.Task{Id: "task"}), "task") || marshalProtoJSON(nil) != "" {
		t.Fatal("proto helpers")
	}
}

func TestAppHelpersValidateSandboxAndAPIErrorContracts(t *testing.T) {
	if cfg, err := parseSandboxConfig("", ""); err != nil || cfg != nil {
		t.Fatalf("empty config=%+v err=%v", cfg, err)
	}
	if _, err := parseSandboxConfig(`{"mode":"SANDBOX_MODE_OFF"}`, ""); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sandbox.json")
	if err := os.WriteFile(path, []byte(`{"mode":"SANDBOX_MODE_PROTECTED"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if cfg, err := parseSandboxConfig("", path); err != nil || cfg.Mode != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED {
		t.Fatalf("file config=%+v err=%v", cfg, err)
	}
	if _, err := parseSandboxConfig("{", ""); err == nil {
		t.Fatal("invalid JSON should fail")
	}
	if _, err := parseSandboxConfig("", filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing file should fail")
	}

	for mode, want := range map[string]domainpb.SandboxMode{"off": domainpb.SandboxMode_SANDBOX_MODE_OFF, "tracking": domainpb.SandboxMode_SANDBOX_MODE_TRACKING, "protected": domainpb.SandboxMode_SANDBOX_MODE_PROTECTED} {
		cfg, err := applySandboxModeOverride(nil, strings.ToUpper(mode))
		if err != nil || cfg.Mode != want {
			t.Fatalf("mode=%s cfg=%+v err=%v", mode, cfg, err)
		}
	}
	if _, err := applySandboxModeOverride(nil, "bad"); err == nil {
		t.Fatal("invalid sandbox mode should fail")
	}
	if profileSandboxMode(nil) != "-" || profileSandboxMode(&domainpb.AgentProfile{SandboxConfig: &domainpb.SandboxConfig{Mode: domainpb.SandboxMode_SANDBOX_MODE_TRACKING}}) != "tracking" {
		t.Fatal("profile sandbox mode")
	}
	retention, err := applySandboxRetention(nil, "delete_on_terminal", "90")
	if err != nil || retention.Lifecycle.GetTtl().AsDuration() != 90*time.Second || len(retention.Lifecycle.DeleteOn) != 1 {
		t.Fatalf("retention=%+v err=%v", retention, err)
	}
	if _, err := applySandboxRetention(nil, "bad", ""); err == nil {
		t.Fatal("invalid retention mode should fail")
	}
	if _, err := applySandboxRetention(nil, "", "bad"); err == nil {
		t.Fatal("invalid retention TTL should fail")
	}

	original := errors.New("transport failed")
	if apiError(nil, nil) != nil || apiError([]byte(`{"message":"bad request","details":{"fields":{"hint":{"string_value":"fix input"}}}}`), original).Error() != "bad request\nhint: fix input" {
		t.Fatal("API error enrichment")
	}
	transport := &cliutil.APIError{RawResponse: []byte(`{"message":"from transport"}`)}
	if got := apiError(nil, transport); got.Error() != "from transport" {
		t.Fatalf("transport error=%v", got)
	}
}
