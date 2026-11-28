package codecs

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
)

// fakeCodexAppServer answers the JSON-RPC exchange of a compaction. FAKE_MODE
// selects the compaction outcome; every request line is appended to
// $FAKE_LOG so tests can assert what agent-manager sent.
const fakeCodexAppServer = `#!/bin/bash
[ "$1" = "app-server" ] || { echo "unexpected args: $*" >&2; exit 2; }
echo "CODEX_HOME=$CODEX_HOME" >> "$FAKE_LOG"
while IFS= read -r line; do
  echo "$line" >> "$FAKE_LOG"
  id=$(printf '%s' "$line" | sed -n 's/^{"id":\([0-9]*\),.*/\1/p')
  case "$line" in
    *'"method":"initialize"'*) echo "{\"id\":$id,\"result\":{}}" ;;
    *'"method":"thread/resume"'*)
      case "$line" in
        *'"excludeTurns":true'*) ;;
        *) echo "{\"id\":$id,\"error\":{\"code\":-32602,\"message\":\"thread/resume must exclude turns\"}}"; continue ;;
      esac
      echo '{"method":"thread/tokenUsage/updated","params":{"threadId":"thread-1"}}'
      echo "{\"id\":$id,\"result\":{\"thread\":{\"id\":\"thread-1\",\"status\":\"idle\"}}}" ;;
    *'"method":"thread/compact/start"'*)
      echo "{\"id\":$id,\"result\":{}}"
      case "$FAKE_MODE" in
        no-compaction) ;;
        failed) echo '{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"status":"failed","error":{"message":"quota"}}}}'; continue ;;
        *) echo '{"method":"item/completed","params":{"threadId":"thread-1","item":{"type":"contextCompaction"}}}' ;;
      esac
      echo '{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"status":"completed"}}}' ;;
  esac
done
`

func newFakeAppServerCodex(t *testing.T, mode string) (*Codex, string, string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte(fakeCodexAppServer), 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "requests.log")
	t.Setenv("FAKE_LOG", logPath)
	t.Setenv("FAKE_MODE", mode)
	home := filepath.Join(dir, "codex-home")
	return NewCodexForTestWithBinary(bin), home, logPath
}

func writeRollout(t *testing.T, home, sessionID string, inputTokens ...int) {
	t.Helper()
	dir := filepath.Join(home, "sessions", "2026", "10", "01")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString(`{"type":"session_meta","payload":{}}` + "\n")
	for _, n := range inputTokens {
		b.WriteString(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":` + strconv.Itoa(n) + `}}}}` + "\n")
	}
	b.WriteString(`{"type":"event_msg","payload":{"type":"token_count","info":null}}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "rollout-2026-10-01T00-00-00-"+sessionID+".jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func compactReq(home string, min int64) runner.CompactSessionRequest {
	return runner.CompactSessionRequest{
		SessionID: "thread-1", Model: "gpt-6-luna", WorkingDir: os.TempDir(),
		Env: map[string]string{"CODEX_HOME": home}, MinContextTokens: min,
	}
}

func TestCodexCompactSessionCompactsThroughAppServer(t *testing.T) {
	codec, home, logPath := newFakeAppServerCodex(t, "ok")
	writeRollout(t, home, "thread-1", 20_000, 150_000)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := codec.CompactSession(ctx, compactReq(home, 40_000))
	if err != nil {
		t.Fatalf("CompactSession: %v", err)
	}
	if !res.Compacted || res.ContextTokens != 150_000 {
		t.Fatalf("result = %+v, want compacted with the newest context size", res)
	}
	log, _ := os.ReadFile(logPath)
	for _, want := range []string{
		"CODEX_HOME=" + home,
		`"method":"thread/resume","params":{"excludeTurns":true,"model":"gpt-6-luna","threadId":"thread-1"}`,
		`"method":"thread/compact/start","params":{"threadId":"thread-1"}`,
	} {
		if !strings.Contains(string(log), want) {
			t.Fatalf("app-server never received %s; log:\n%s", want, log)
		}
	}
}

func TestCodexCompactSessionSkipsSmallContexts(t *testing.T) {
	codec, home, logPath := newFakeAppServerCodex(t, "ok")
	writeRollout(t, home, "thread-1", 12_000)

	res, err := codec.CompactSession(context.Background(), compactReq(home, 40_000))
	if err != nil {
		t.Fatalf("CompactSession: %v", err)
	}
	if res.Compacted || res.ContextTokens != 12_000 || res.Reason == "" {
		t.Fatalf("result = %+v, want a skip with its reason", res)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("a skipped compaction must not start the app-server")
	}
}

func TestCodexCompactSessionFailsWithoutACompaction(t *testing.T) {
	for _, mode := range []string{"no-compaction", "failed"} {
		t.Run(mode, func(t *testing.T) {
			codec, home, _ := newFakeAppServerCodex(t, mode)
			writeRollout(t, home, "thread-1", 90_000)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if res, err := codec.CompactSession(ctx, compactReq(home, 40_000)); err == nil {
				t.Fatalf("CompactSession = %+v, want an error when no compaction happened", res)
			}
		})
	}
}

// Compacting resumes the thread in an app-server, which an active goal may not
// survive; a size or goal the rollout cannot show is a reason to leave it alone.
func TestCodexCompactSessionLeavesAloneWhatItCannotClear(t *testing.T) {
	appendRollout := func(t *testing.T, home, line string) {
		t.Helper()
		matches, _ := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "*thread-1*.jsonl"))
		if len(matches) != 1 {
			t.Fatalf("rollouts = %v", matches)
		}
		f, err := os.OpenFile(matches[0], os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	activeEvent := `{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"ship","status":"active"}}}`
	createGoal := `{"type":"response_item","payload":{"type":"custom_tool_call","call_id":"c","name":"exec","input":"await tools.create_goal({objective:\"ship\"})"}}`
	for _, tc := range []struct {
		name   string
		tokens []int
		lines  []string
		noHome bool
		reason string
		size   int64
	}{
		{name: "active goal event", tokens: []int{150_000}, lines: []string{activeEvent}, reason: "active goal", size: 150_000},
		{name: "goal started by a tool", tokens: []int{150_000}, lines: []string{createGoal}, reason: "active goal", size: 150_000},
		{name: "no token count", lines: []string{}, reason: "size is unknown"},
		{name: "zero token count", tokens: []int{0}, reason: "size is unknown"},
		{name: "no rollout", reason: "could not be read"},
		{name: "no home", noHome: true, reason: "could not be read"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, home, logPath := newFakeAppServerCodex(t, "ok")
			if tc.tokens != nil || tc.lines != nil {
				writeRollout(t, home, "thread-1", tc.tokens...)
			}
			for _, line := range tc.lines {
				appendRollout(t, home, line)
			}
			if tc.noHome {
				home = ""
			}
			res, err := codec.CompactSession(context.Background(), compactReq(home, 40_000))
			if err != nil {
				t.Fatalf("CompactSession: %v", err)
			}
			if res.Compacted || res.ContextTokens != tc.size || !strings.Contains(res.Reason, tc.reason) {
				t.Fatalf("result = %+v, want a skip naming %q", res, tc.reason)
			}
			if _, err := os.Stat(logPath); !os.IsNotExist(err) {
				t.Fatal("a skipped compaction must not start the app-server")
			}
		})
	}
}

func TestCodexCompactSessionProceedsAfterAPausedGoal(t *testing.T) {
	codec, home, _ := newFakeAppServerCodex(t, "ok")
	writeRollout(t, home, "thread-1", 150_000)
	matches, _ := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "*thread-1*.jsonl"))
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	goal := `{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"ship","status":"active"}}}` + "\n" +
		`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"ship","status":"paused"}}}` + "\n"
	if err := os.WriteFile(matches[0], append([]byte(goal), data...), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := codec.CompactSession(context.Background(), compactReq(home, 40_000))
	if err != nil || !res.Compacted {
		t.Fatalf("result = %+v, err = %v", res, err)
	}
}
