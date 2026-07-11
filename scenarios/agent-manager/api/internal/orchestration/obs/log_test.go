package obs

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TestStableKeysAreUnique guards the contract that every stable key
// has a single canonical name. A duplicate constant would mean log
// queries silently disagree on which name to grep.
func TestStableKeysAreUnique(t *testing.T) {
	keys := []string{
		KeyRunID, KeyTaskID, KeyConversationID,
		KeyRunMode, KeyRunnerType, KeyLauncherType, KeySandboxID,
		KeyPhase, KeyDuration,
		KeyExitCode, KeyTerminalCode, KeyOutcome,
		KeyQueueDepth, KeyActiveCount, KeyStartingCount,
		KeyError, KeyMessage, KeyComponent,
		KeyPermissionPolicyDigest, KeyPermissionPolicyResourceCount,
		KeyPermissionPolicyDriftCount, KeyPermissionPolicyUnsupportedCount,
		KeyHardEnforcementSatisfied, KeyMissingHardEnforcementRuleIDs,
		KeyPermissionPolicyPartialFailure,
	}
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, dup := seen[k]; dup {
			t.Errorf("duplicate stable key: %q", k)
		}
		seen[k] = struct{}{}
	}
}

// TestInitWithWriterEmitsJSON proves that selecting the json format
// produces a parseable JSON line per record. Asserts on the keys, not
// on the exact bytes, so a slog format tweak doesn't break the test.
func TestInitWithWriterEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("json", "info", &buf)
	defer InitWithWriter("text", "info", &buf) // restore

	Logger().Info("hello", KeyRunID, "00000000-0000-0000-0000-000000000001")

	if buf.Len() == 0 {
		t.Fatal("expected at least one log line")
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", buf.String(), err)
	}
	if record[KeyRunID] != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected runID key, got %v", record)
	}
	if record["msg"] != "hello" {
		t.Errorf("expected msg=hello, got %v", record)
	}
}

// TestInitFallsBackOnInvalidLevel ensures a misconfigured lever can't
// silently disable logging — Init must keep emitting at info level
// when given a bogus value.
func TestInitFallsBackOnInvalidLevel(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("text", "loud-please", &buf)
	defer InitWithWriter("text", "info", &buf)

	Logger().Info("baseline")
	if !strings.Contains(buf.String(), "baseline") {
		t.Fatalf("info-level message should appear under fallback, got %q", buf.String())
	}
}

// TestRunCtxThreadsRunScopedKeys asserts that RunCtx + L produce
// loggers that include runID/runMode/runnerType/sandboxID in every
// emitted record without the call site repeating them.
func TestRunCtxThreadsRunScopedKeys(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("json", "info", &buf)
	defer InitWithWriter("text", "info", &buf)

	runID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	sandboxID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := RunCtx(context.Background(), runID, domain.RunModeSandboxed, domain.RunnerTypeCodex, &sandboxID)

	L(ctx).Info("scoped")

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("expected valid JSON: %v (got %q)", err, buf.String())
	}
	if record[KeyRunID] != runID.String() {
		t.Errorf("expected runID=%s, got %v", runID, record[KeyRunID])
	}
	if record[KeyRunMode] != string(domain.RunModeSandboxed) {
		t.Errorf("expected runMode=sandboxed, got %v", record[KeyRunMode])
	}
	if record[KeyRunnerType] != string(domain.RunnerTypeCodex) {
		t.Errorf("expected runnerType=codex, got %v", record[KeyRunnerType])
	}
	if record[KeySandboxID] != sandboxID.String() {
		t.Errorf("expected sandboxID=%s, got %v", sandboxID, record[KeySandboxID])
	}
}

// TestRunCtxOmitsZeroSandboxID protects against polluting every log
// line with "sandboxID=00000000-…" for in-place runs.
func TestRunCtxOmitsZeroSandboxID(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("json", "info", &buf)
	defer InitWithWriter("text", "info", &buf)

	runID := uuid.New()
	ctx := RunCtx(context.Background(), runID, domain.RunModeInPlace, domain.RunnerTypeClaudeCode, nil)
	L(ctx).Info("no sandbox")

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if _, ok := record[KeySandboxID]; ok {
		t.Errorf("expected sandboxID key absent for nil sandbox, got %v", record)
	}
}

// TestLFallsBackToPackageLogger guards the path where ctx is the bare
// background context. Any call site that forgot to thread RunCtx must
// still produce output.
func TestLFallsBackToPackageLogger(t *testing.T) {
	var buf bytes.Buffer
	InitWithWriter("json", "info", &buf)
	defer InitWithWriter("text", "info", &buf)

	L(context.Background()).Info("fallback")
	if !strings.Contains(buf.String(), "fallback") {
		t.Fatalf("expected fallback message, got %q", buf.String())
	}
}
