//nolint:goconst // test data deliberately reuses stable path fixtures.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/guardmodes"
	"github.com/vrooli/vrooli/internal/agentscope"
)

func TestGuardModeRootCommandAllowsOperatorAndAuditsMutation(t *testing.T) {
	previousEvidence, previousActor := guardModeCallerEvidence, guardModeCallerActor
	t.Cleanup(func() { guardModeCallerEvidence, guardModeCallerActor = previousEvidence, previousActor })
	guardModeCallerEvidence = func() string { return agentscope.SessionEvidence(cliutil.CallerKindHuman, "") }
	guardModeCallerActor = func() string { return "local user operator" }
	dir := t.TempDir()
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	var stdout, stderr strings.Builder
	handled, code := runGuardModeCommand([]string{"guard-mode", "enable", "--snapshot-dir", dir, "--check", "scenario-dependency-analyzer:dependency-addition", "--expires-at", expires, "--reason", "reviewed structural decision"}, &stdout, &stderr)
	if !handled || code != 0 {
		t.Fatalf("operator enable handled=%t code=%d stderr=%q", handled, code, stderr.String())
	}
	store := guardmodes.NewStore(dir)
	active, err := store.Active(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Check != "scenario-dependency-analyzer:dependency-addition" {
		t.Fatalf("operator-created modes = %+v", active)
	}
	audit, err := store.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 1 || audit[0].Actor != "local user operator" {
		t.Fatalf("operator audit = %+v", audit)
	}
}

func TestGuardModeRootCommandRejectsAgentScopeForEnableAndDisable(t *testing.T) {
	previousEvidence, previousActor := guardModeCallerEvidence, guardModeCallerActor
	t.Cleanup(func() { guardModeCallerEvidence, guardModeCallerActor = previousEvidence, previousActor })
	guardModeCallerEvidence = func() string {
		return agentscope.SessionEvidence(cliutil.CallerKindHuman, "0::/user.slice/vrooli-agents.slice/vrooli-agent-test.scope")
	}
	actorCalled := false
	guardModeCallerActor = func() string { actorCalled = true; return "operator spoof" }
	dir := t.TempDir()
	var stdout, stderr strings.Builder
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	handled, code := runGuardModeCommand([]string{"guard-mode", "enable", "--snapshot-dir", dir, "--check", "scenario-dependency-analyzer:dependency-addition", "--expires-at", expires, "--reason", "claiming operator"}, &stdout, &stderr)
	if !handled || code == 0 || !strings.Contains(stderr.String(), "agent session scope") {
		t.Fatalf("agent enable handled=%t code=%d stderr=%q", handled, code, stderr.String())
	}
	if actorCalled {
		t.Fatal("operator actor attribution was consulted before session authorization")
	}
	store := guardmodes.NewStore(dir)
	active, err := store.Active(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("agent created modes: %+v", active)
	}
	if entries, err := store.Audit(); err != nil || len(entries) != 0 {
		t.Fatalf("rejected mutation audit entries=%+v err=%v", entries, err)
	}
	// Seed the mode through the core store, then prove the same backstop rejects
	// a disable attempt from an agent scope.
	if err := store.SetLogOnly("scenario-dependency-analyzer:dependency-addition", time.Now().UTC().Add(time.Hour), "seed", "operator"); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	handled, code = runGuardModeCommand([]string{"guard-mode", "disable", "--snapshot-dir", dir, "--check", "scenario-dependency-analyzer:dependency-addition", "--reason", "claiming operator"}, &stdout, &stderr)
	if !handled || code == 0 || !strings.Contains(stderr.String(), "agent session scope") {
		t.Fatalf("agent disable handled=%t code=%d stderr=%q", handled, code, stderr.String())
	}
	active, err = store.Active(time.Now().UTC())
	if err != nil || len(active) != 1 {
		t.Fatalf("agent disable changed modes=%+v err=%v", active, err)
	}
}

func TestGuardModeApplyAnswerVerifiesExactHumanAnswerAndAuditsAnswerID(t *testing.T) {
	previousRead, previousNow := guardModeHubRead, guardModeNow
	t.Cleanup(func() { guardModeHubRead, guardModeNow = previousRead, previousNow })
	now := time.Date(2026, 11, 16, 21, 0, 0, 0, time.UTC)
	guardModeNow = func() time.Time { return now }
	subject := stewardGuardSubjectPrefix + "scenario-dependency-analyzer:dependency-addition"
	guardModeHubRead = func(got string) (map[string]any, error) {
		if got != subject {
			t.Fatalf("Hub subject = %q", got)
		}
		var envelope guardModeHubPayload
		wire := `{"ask":{"id":"ask-42","subject_key":"` + subject + `","state":"answered","question":"` + stewardGuardQuestion + `","answered_by_identity_kind":"verified_human","answered_by_identity_source":"personal_local","template_name":"structural-change","template_data":{"meta":{"title":"` + stewardGuardQuestion + `"},"technical":[{"label":"Evaluator ID","value":"scenario-dependency-analyzer:dependency-addition"}]},"options":[{"key":"loosen_24h","label":"` + stewardGuardLoosenLabel + `"}],"answer":"loosen_24h","answer_changes":{"answer":"loosen_24h"}}}`
		if err := json.Unmarshal([]byte(wire), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Ask, nil
	}
	dir := t.TempDir()
	var stdout, stderr strings.Builder
	handled, code := runGuardModeCommand([]string{"guard-mode", "apply-answer", "--snapshot-dir", dir,
		"--subject-key", subject, "--ask-id", "ask-42"}, &stdout, &stderr)
	if !handled || code != 0 {
		t.Fatalf("apply-answer handled=%t code=%d stderr=%q", handled, code, stderr.String())
	}
	active, err := guardmodes.NewStore(dir).Active(now)
	if err != nil || len(active) != 1 || active[0].Check != "scenario-dependency-analyzer:dependency-addition" || !active[0].ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("active guard modes = %+v, err=%v", active, err)
	}
	audit, err := guardmodes.NewStore(dir).Audit()
	if err != nil || len(audit) != 1 || audit[0].AuthorizationID != "ask-42" || audit[0].Actor != "verified operator via Notification Hub" {
		t.Fatalf("answer audit = %+v, err=%v", audit, err)
	}
	stdout.Reset()
	stderr.Reset()
	_, replayCode := runGuardModeCommand([]string{"guard-mode", "apply-answer", "--snapshot-dir", dir,
		"--subject-key", subject, "--ask-id", "ask-42"}, &stdout, &stderr)
	if replayCode == 0 || !strings.Contains(stderr.String(), "already consumed") {
		t.Fatalf("same-answer replay code=%d stderr=%q", replayCode, stderr.String())
	}
	active, err = guardmodes.NewStore(dir).Active(now)
	if err != nil || len(active) != 1 || !active[0].ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("same-answer replay extended mode: %+v err=%v", active, err)
	}
	now = now.Add(25 * time.Hour)
	stdout.Reset()
	stderr.Reset()
	_, replayCode = runGuardModeCommand([]string{"guard-mode", "apply-answer", "--snapshot-dir", dir,
		"--subject-key", subject, "--ask-id", "ask-42"}, &stdout, &stderr)
	if replayCode == 0 || !strings.Contains(stderr.String(), "already consumed") {
		t.Fatalf("expired-answer replay code=%d stderr=%q", replayCode, stderr.String())
	}
}

func TestGuardModeHubReadParsesConversationsGetProtoJSONEnvelope(t *testing.T) {
	previous := guardModeHubCommand
	t.Cleanup(func() { guardModeHubCommand = previous })
	subject := stewardGuardSubjectPrefix + "scenario-dependency-analyzer:dependency-addition"
	guardModeHubCommand = func(got string) ([]byte, error) {
		if got != subject {
			t.Fatalf("Hub subject key = %q, want exact subject %q", got, subject)
		}
		return []byte(`{"ask":{"id":"ask-wire-1","subject_key":"` + subject + `","state":"answered","answer":"loosen_24h","answer_changes":{"answer":"loosen_24h"},"answered_by_identity_kind":"verified_human","answered_by_identity_source":"cloudflare_access"}}`), nil
	}
	ask, err := readGuardModeHubAsk(subject)
	if err != nil {
		t.Fatal(err)
	}
	if ask["id"] != "ask-wire-1" || ask["answer"] != "loosen_24h" || ask["answered_by_identity_kind"] != "verified_human" || ask["answered_by_identity_source"] != "cloudflare_access" {
		t.Fatalf("decoded actual Hub ask fields = %#v", ask)
	}
	changes, ok := ask["answer_changes"].(map[string]any)
	if !ok || changes["answer"] != "loosen_24h" {
		t.Fatalf("decoded Hub page answer changes = %#v", ask["answer_changes"])
	}
}

func TestGuardModeApplyAnswerRejectsUnverifiedOrMisboundAnswers(t *testing.T) {
	previousRead := guardModeHubRead
	t.Cleanup(func() { guardModeHubRead = previousRead })
	subject := stewardGuardSubjectPrefix + "scenario-dependency-analyzer:dependency-addition"
	guardModeHubRead = func(string) (map[string]any, error) {
		return map[string]any{
			"id": "ask-42", "subject_key": subject, "state": "answered", "question": stewardGuardQuestion,
			"answered_by_identity_kind": "unknown", "template_name": "structural-change",
			"template_data": map[string]any{"meta": map[string]any{"title": stewardGuardQuestion}, "technical": []any{map[string]any{"label": "Evaluator ID", "value": "scenario-dependency-analyzer:dependency-addition"}}},
			"options":       []any{map[string]any{"key": "loosen_24h", "label": stewardGuardLoosenLabel}},
			"answer":        map[string]any{"changes": map[string]any{"answer": "loosen_24h"}},
		}, nil
	}
	dir := t.TempDir()
	var stdout, stderr strings.Builder
	_, code := runGuardModeCommand([]string{"guard-mode", "apply-answer", "--snapshot-dir", dir,
		"--subject-key", subject, "--ask-id", "ask-42"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "verified-human") {
		t.Fatalf("unverified answer code=%d stderr=%q", code, stderr.String())
	}
	if entries, err := guardmodes.NewStore(dir).Audit(); err != nil || len(entries) != 0 {
		t.Fatalf("unverified answer created audit entries=%+v err=%v", entries, err)
	}
}

func usableWorkingDirSeams() workingDirSeams {
	return workingDirSeams{
		getwd:   func() (string, error) { return "/home/example/project", nil },
		stat:    func(string) (os.FileInfo, error) { return nil, nil },
		chdir:   func(string) error { return nil },
		homeDir: func() (string, error) { return "/home/example", nil },
		tempDir: func() string { return "/tmp" },
	}
}

func TestEnsureUsableWorkingDirectoryLeavesAReadableDirectoryAlone(t *testing.T) {
	var stderr bytes.Buffer
	chdirCalls := 0
	seams := usableWorkingDirSeams()
	seams.chdir = func(string) error { chdirCalls++; return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if chdirCalls != 0 {
		t.Fatalf("chdir calls = %d, want 0 when the working directory is usable", chdirCalls)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty when the working directory is usable", stderr.String())
	}
}

// A shell that dropped privileges while sitting in a directory it cannot read
// (the /root case) keeps a resolvable path but fails to stat it.
func TestEnsureUsableWorkingDirectoryRelocatesWhenCWDIsUnreadable(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.stat = func(string) (os.FileInfo, error) {
		return nil, &fs.PathError{Op: "stat", Path: ".", Err: fs.ErrPermission}
	}
	seams.chdir = func(dir string) error { entered = append(entered, dir); return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/home/example" {
		t.Fatalf("entered = %v, want exactly [/home/example]", entered)
	}
	if !strings.Contains(stderr.String(), "/home/example") {
		t.Fatalf("stderr = %q, want the relocation to be announced", stderr.String())
	}
}

// A directory deleted underneath a live shell keeps a stat-able inode, so only
// resolving its path reveals the problem. The Go toolchain refuses to run in
// this state too, so the guard has to catch it.
func TestEnsureUsableWorkingDirectoryRelocatesWhenCWDWasDeleted(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.getwd = func() (string, error) {
		return "", &fs.PathError{Op: "getwd", Path: ".", Err: fs.ErrNotExist}
	}
	seams.chdir = func(dir string) error { entered = append(entered, dir); return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/home/example" {
		t.Fatalf("entered = %v, want exactly [/home/example]", entered)
	}
	if !strings.Contains(stderr.String(), "/home/example") {
		t.Fatalf("stderr = %q, want the relocation to be announced", stderr.String())
	}
}

func TestEnsureUsableWorkingDirectoryFallsBackToTempWhenHomeIsUnusable(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.stat = func(string) (os.FileInfo, error) { return nil, errors.New("permission denied") }
	seams.homeDir = func() (string, error) { return "", errors.New("no home") }
	seams.chdir = func(dir string) error { entered = append(entered, dir); return errors.New("cannot enter") }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/tmp" {
		t.Fatalf("entered = %v, want the temp-dir fallback to be attempted", entered)
	}
	if !strings.Contains(stderr.String(), "no fallback directory") {
		t.Fatalf("stderr = %q, want the unrecoverable case reported", stderr.String())
	}
}
