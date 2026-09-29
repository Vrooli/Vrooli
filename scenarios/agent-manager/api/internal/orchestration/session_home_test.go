package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

var liveCodexWorkerPolicy = flag.Bool("live-codex-worker-policy", false, "Check installed Codex policy composition without invoking a model")

func TestLiveCodexWorkerPolicy(t *testing.T) {
	if !*liveCodexWorkerPolicy {
		t.Skip("pass -live-codex-worker-policy for the no-model installed CLI check")
	}
	root := t.TempDir()
	data, err := compileCodexWorkerPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	file, err := writeRunnerPolicy(root, uuid.New(), data)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "codex")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[mcp_servers.fixture]\ncommand='/bin/false'\nenabled=true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binary, err := osexec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	run := func(policy bool, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// A private /etc keeps this synthetic fixture independent of host
		// policy. Production composition retains inherited admin requirements.
		base := []string{"--unshare-user", "--unshare-pid", "--unshare-net", "--die-with-parent", "--ro-bind", "/", "/", "--tmpfs", "/etc", "--bind", root, root, "--proc", "/proc", "--dev", "/dev", "--chdir", root, "--setenv", "CODEX_HOME", home}
		if policy {
			base = append(base, "--ro-bind", file.Source, file.Source, "--ro-bind", file.Source, file.Target)
		}
		base = append(base, "--", binary)
		cmd := osexec.CommandContext(ctx, "bwrap", append(base, args...)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Codex policy probe failed: %v\n%s", err, output)
		}
		return output
	}
	// The fixture's MCP server must be visible before applying the ceiling;
	// otherwise an empty inventory could give a false positive.
	for _, policy := range []bool{false, true} {
		output := run(policy, "-c", "mcp_servers.fixture.enabled=true", "mcp", "list", "--json")
		var servers []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(output, &servers); err != nil || len(servers) != 1 || servers[0].Name != "fixture" || servers[0].Enabled == policy {
			t.Fatalf("MCP policy=%v: %s (%v)", policy, output, err)
		}
	}
	args := []string{}
	for _, feature := range codexWorkerDisabledFeatures {
		args = append(args, "--enable", feature)
	}
	output := run(true, append(args, "features", "list")...)
	values := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			values[fields[0]] = fields[len(fields)-1]
		}
	}
	for _, feature := range codexWorkerDisabledFeatures {
		if values[feature] != "false" {
			t.Errorf("managed feature %s must remain disabled against command-line enable; got %q", feature, values[feature])
		}
	}
	// Remote-tool denial must not disable the local runtime that dispatches
	// Luna/Sol's shell and patch calls. A remote-inventory-only canary missed this.
	if values["code_mode_host"] != "true" {
		t.Errorf("local execution host must remain usable; got %q", values["code_mode_host"])
	}
}

func TestCompileCodexWorkerPolicyPreservesLocalExecution(t *testing.T) {
	for _, input := range []string{"", "[features]\ncode_mode_host = true\ncode_mode = true\n"} {
		data, err := compileCodexWorkerPolicy([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := toml.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		features := got["features"].(map[string]any)
		for _, feature := range []string{"code_mode_host", "code_mode"} {
			value, present := features[feature]
			if input == "" && present || input != "" && value != true {
				t.Fatalf("local execution feature %s must retain its inherited/default value: %s", feature, data)
			}
		}
	}
}

func TestCompileCodexWorkerPolicy(t *testing.T) {
	input := []byte(`allowed_sandbox_modes = ["read-only"]
allowed_web_search_modes = ["live"]
[network]
allowed_domains = ["api.example.test"]
[features]
memories = false
[mcp_servers.admin]
identity = { command = "/bin/false" }
`)
	data, err := compileCodexWorkerPolicy(input)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got["allowed_sandbox_modes"]) != "[read-only]" || fmt.Sprint(got["network"].(map[string]any)["allowed_domains"]) != "[api.example.test]" {
		t.Fatalf("inherited restrictions were lost: %s", data)
	}
	if len(got["mcp_servers"].(map[string]any)) != 0 || len(got["allowed_web_search_modes"].([]any)) != 0 || got["allow_managed_hooks_only"] != true {
		t.Fatalf("remote tools or user hooks remain available: %s", data)
	}
	features := got["features"].(map[string]any)
	for _, name := range append(codexWorkerDisabledFeatures, "memories") {
		if features[name] != false {
			t.Fatalf("feature %s was not disabled", name)
		}
	}
	for _, invalid := range []string{"[broken", "features = 'bad'", "[features]\napps = true", "[hooks]\nmanaged_dir='/owner/hooks'"} {
		if _, err := compileCodexWorkerPolicy([]byte(invalid)); err == nil {
			t.Fatalf("must refuse malformed/conflicting admin policy: %s", invalid)
		}
	}
}

func TestPrepareRunnerPolicyImmutableAcrossResume(t *testing.T) {
	root, id := t.TempDir(), uuid.New()
	first, err := writeRunnerPolicy(root, id, []byte("owner policy"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeRunnerPolicy(root, id, []byte("owner policy"))
	if err != nil || first != second {
		t.Fatalf("same policy lost identity: %v, %v", second, err)
	}
	if first.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte("owner policy"))) || strings.Contains(first.Source, "/runtime/") {
		t.Fatalf("policy identity/location is wrong: %+v", first)
	}
	if err := os.Chmod(first.Source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first.Source, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeRunnerPolicy(root, id, []byte("owner policy")); err == nil {
		t.Fatal("must refuse changed persisted policy")
	}
}

func TestPrepareCodecSessionHome_PersistsCodexRolloutAcrossTurns(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("AM_SQLITE_PATH", filepath.Join(temp, ".vrooli", "data", "agent-manager.db"))
	shared := filepath.Join(temp, ".codex")
	if err := os.MkdirAll(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "auth.json"), []byte(`{"token":"test"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runID := uuid.New()
	runRoot := filepath.Join(temp, "runs")
	env, err := PrepareCodecSessionHome(runRoot, runID, domain.RunnerTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	home := env["CODEX_HOME"]
	if home == "" {
		t.Fatal("CODEX_HOME was not set")
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("seeded auth missing: %v", err)
	}

	rollout := filepath.Join(home, "sessions", "2026", "07", "25", "rollout-thread.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte("turn one"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondTurnEnv, err := PrepareCodecSessionHome(runRoot, runID, domain.RunnerTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if secondTurnEnv["CODEX_HOME"] != home {
		t.Fatalf("second turn CODEX_HOME = %q, want %q", secondTurnEnv["CODEX_HOME"], home)
	}
	if got, err := os.ReadFile(rollout); err != nil || string(got) != "turn one" {
		t.Fatalf("second turn cannot read first rollout: got %q, err=%v", got, err)
	}

	if err := CleanupCodecSessionHomeCredentials(runRoot, runID, domain.RunnerTypeCodex); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); !os.IsNotExist(err) {
		t.Fatalf("seeded auth retained after cleanup: %v", err)
	}
	if _, err := os.Stat(rollout); err != nil {
		t.Fatalf("rollout removed by credential cleanup: %v", err)
	}

	catalog := filepath.Join(home, "cache", "remote_plugin_catalog", "catalog.json")
	bundle := filepath.Join(home, "plugins", "cache", "openai-curated-remote", "reference.pptx")
	for _, path := range []string{catalog, bundle} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("downloaded"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runDir := filepath.Dir(filepath.Dir(home))
	if err := PruneRunnerCaches(runDir); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{catalog, bundle} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("downloaded cache %s survived pruning: %v", path, err)
		}
	}
	if _, err := os.Stat(rollout); err != nil {
		t.Fatalf("rollout removed by cache pruning: %v", err)
	}
}
