package cloudtargethandlers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cloudtarget"
)

type envRunner struct {
	recordingRunner
	env [][]string
}

func (r *envRunner) RunEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	r.env = append(r.env, env)
	return r.Run(ctx, name, args...)
}

// [REQ:STC-P0-024] JSON-shaped flags travel as b64:<base64url> on an
// argv-only transport and decode to the same document; a value that does
// not decode to JSON is refused before any store access.
func TestJSONFlagsAcceptBase64URLSpelling(t *testing.T) {
	workdir := t.TempDir()
	scenarioDir := filepath.Join(workdir, "scenarios", "landing-app", "data")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := cloudtarget.NewStore(filepath.Join(t.TempDir(), "deployments"))
	raw := `[{"id":"records","path":"data"}]`
	for _, spelling := range []string{raw, EncodeJSONArg([]byte(raw))} {
		value, exit := runVerb(t, store, "data", "inventory", "--deployment", "dep", "--workdir", workdir, "--scenario", "landing-app", "--bindings", spelling)
		if exit != 0 || value["uncovered_count"] != float64(0) {
			t.Fatalf("spelling %q: exit=%d value=%v", spelling, exit, value)
		}
		entries, _ := value["entries"].([]any)
		if len(entries) != 1 || entries[0].(map[string]any)["binding_id"] != "records" {
			t.Fatalf("spelling %q: entries=%v", spelling, entries)
		}
	}
	if strings.ContainsAny(EncodeJSONArg([]byte(raw)), `"{}[] ;|&$`) {
		t.Fatalf("encoded flag must be argv-safe: %q", EncodeJSONArg([]byte(raw)))
	}
	value, exit := runVerb(t, store, "data", "inventory", "--deployment", "dep", "--workdir", workdir, "--scenario", "landing-app", "--bindings", "b64:not-json")
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeInvalidArgument {
		t.Fatalf("undecodable value must be refused: exit=%d value=%v", exit, value)
	}
}

// [REQ:STC-P0-028] The activate verb accepts pinned ports, data bindings,
// legacy carries and the legacy root, and refuses malformed ports before
// touching the release store.
func TestActivateFlagsReachTheStore(t *testing.T) {
	store := cloudtarget.NewStore(filepath.Join(t.TempDir(), "deployments"))
	runner := &envRunner{}
	deps := Deps{Store: func() (*cloudtarget.Store, error) { return store, nil }, Runner: runner}
	value, exit := runVerbWith(t, deps, "release", "activate", "--deployment", "dep", "--operation", "op", "--step", "activate", "--fence", "1", "--release", strings.Repeat("a", 64), "--port", "ui=abc")
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeInvalidArgument {
		t.Fatalf("bad port exit=%d value=%v", exit, value)
	}
	value, exit = runVerbWith(t, deps, "release", "activate", "--deployment", "dep", "--operation", "op", "--step", "activate", "--fence", "1", "--release", strings.Repeat("a", 64), "--data-binding", "uploads")
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeInvalidArgument {
		t.Fatalf("bad binding exit=%d value=%v", exit, value)
	}
	// Well-formed flags reach the store, which refuses because nothing is
	// staged: the refusal is receipted (outcome failed), proving the verb
	// parsed every flag and entered the effect.
	value, exit = runVerbWith(t, deps, "release", "activate", "--deployment", "dep", "--operation", "op", "--step", "activate", "--fence", "1", "--release", strings.Repeat("a", 64), "--port", "ui=3000", "--data-binding", "uploads=landing-app/uploads", "--legacy-carry", "landing-app/data", "--legacy-root", "/root/Vrooli", "--restart")
	receipt, _ := value["receipt"].(map[string]any)
	if exit != cloudtarget.ExitRefused || receipt["outcome"] != string(cloudtarget.OutcomeFailed) || value["error"].(map[string]any)["code"] != cloudtarget.CodeReleaseNotStaged {
		t.Fatalf("well-formed activate exit=%d value=%v", exit, value)
	}
}
