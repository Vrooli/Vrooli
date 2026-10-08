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
      echo '{"method":"thread/tokenUsage/updated","params":{"threadId":"thread-1"}}'
      echo "{\"id\":$id,\"result\":{\"thread\":{}}}" ;;
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
		`"method":"thread/resume","params":{"model":"gpt-6-luna","threadId":"thread-1"}`,
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
