package vroolicli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/recovery"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestAgentRecoveryListIncludesDurableInvestigations(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`[{"id":"receipt-1","scenario":"web-console","requester":"vrooli-autoheal","outcome":"unconfirmed","runner":"codex","created_at":"2026-11-05T20:00:00Z","deadline_at":"2026-11-05T20:10:00Z"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	ctx := &AppContext{Stdout: &out, Stderr: &strings.Builder{}}
	if err := (&App{}).runAgentRecovery(ctx, []string{"list", "--json"}); err != nil {
		t.Fatal(err)
	}
	var result struct {
		CodingAgents []recovery.CodingAgentRecord `json:"coding_agents"`
	}
	if err := json.Unmarshal([]byte(out.String()), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.CodingAgents) != 1 || result.CodingAgents[0].ID != "receipt-1" || result.CodingAgents[0].Outcome != "unconfirmed" {
		t.Fatalf("missing receipt: %s", out.String())
	}
	// The root parser consumes global --json before invoking this handler.
	out.Reset()
	ctx.Globals.JSON = true
	if err := (&App{}).runAgentRecovery(ctx, []string{"list"}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out.String()), &result); err != nil {
		t.Fatalf("global JSON was ignored: %v, output=%s", err, out.String())
	}
	ctx.Globals.JSON = false
	out.Reset()
	if err := (&App{}).runAgentRecovery(ctx, []string{"list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "kind=coding-agent runner=codex outcome=unconfirmed") {
		t.Fatalf("text listing=%s", out.String())
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (&App{}).runAgentRecovery(ctx, []string{"list", "--json"}); err == nil {
		t.Fatal("listing hid unreadable coding-agent ledger")
	}
}

func TestAgentRecoveryReconcileReportsUncertaintyWithoutExitProof(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`[{"id":"legacy","scenario":"fixture","requester":"fixture","outcome":"running","created_at":"2026-09-10T18:00:00Z"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	ctx := &AppContext{Stdout: &out, Stderr: &strings.Builder{}}
	ctx.Globals.JSON = true
	if err := (&App{}).runAgentRecovery(ctx, []string{"reconcile", "--limit", "1"}); err != nil {
		t.Fatal(err)
	}
	var result recovery.CodingAgentReconciliation
	if err := json.Unmarshal([]byte(out.String()), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 1 || result.Changed[0].Outcome != "unconfirmed" || result.ExitProven || result.ExecutionEffects || !result.Changed[0].FinishedAt.IsZero() {
		t.Fatalf("CLI invented an execution outcome: %s", out.String())
	}
}

func TestAgentLaunchHonorsTimeoutAndCallerCancellation(t *testing.T) {
	// Adopt a disposable earlier session so this test has no live attachment,
	// lease, or cgroup effects. The child is only a shell fixture.
	t.Setenv(cliutil.AgentSessionEnv, "recovery-deadline-fixture")
	t.Setenv(cliutil.AgentManagerIdentityTokenEnv, "")
	t.Setenv(cliutil.ModelSourceEnvironment, "")
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	script := shelltest.POSIXShebang() + "while :; do :; done\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"timeout", "caller"} {
		t.Run(mode, func(t *testing.T) {
			callerCtx := context.Background()
			args := []string{"launch", "--runner", "claude", "--prompt", "fixture"}
			if mode == "timeout" {
				args = append(args, "--timeout", "30ms")
			} else {
				var cancel context.CancelFunc
				callerCtx, cancel = context.WithTimeout(callerCtx, 30*time.Millisecond)
				defer cancel()
			}
			ctx := &AppContext{Context: callerCtx, Stdout: &strings.Builder{}, Stderr: &strings.Builder{}}
			started := time.Now()
			if err := (&App{}).runAgentCommand(ctx, args); err == nil {
				t.Fatal("cancelled fixture reported success")
			}
			if time.Since(started) > 2*time.Second {
				t.Fatal("waiting wrapper ignored cancellation")
			}
		})
	}
}
