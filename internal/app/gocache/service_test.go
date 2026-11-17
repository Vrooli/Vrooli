package gocacheapp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/vrooli/internal/gocache"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestPolicyIsTheGoToolBudgetAndTheProtocolRecencyFloor(t *testing.T) {
	policy, err := Service{}.Policy()
	if err != nil {
		t.Fatal(err)
	}
	if policy.BudgetBytes != 40<<30 || policy.MinIdle != gocache.DefaultMinIdle {
		t.Fatalf("policy = %+v, want the 40GiB tool budget and the %s floor", policy, gocache.DefaultMinIdle)
	}
}

// Storage Manager invokes this owner because the repository contract names
// it; both declarations must agree on the command and the budget.
func TestRepositoryContractDelegatesTheGoBuildCacheToThisOwner(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Storage struct {
			Roots []struct {
				ID                  string   `json:"id"`
				Tier                string   `json:"tier"`
				MaxBytes            string   `json:"max_bytes"`
				OwnerReclaimCommand []string `json:"owner_reclaim_command"`
				ToolPruneCommand    []string `json:"tool_prune_command"`
			} `json:"roots"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	policy, err := Service{}.Policy()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, root := range contract.Storage.Roots {
		if root.ID != "go-build-cache" {
			continue
		}
		found = true
		if got := strings.Join(root.OwnerReclaimCommand, " "); got != OwnerCommand || root.Tier != "safe_with_owner" || len(root.ToolPruneCommand) != 0 {
			t.Errorf("go-build-cache root = %+v, want tier safe_with_owner reclaimed only by %q", root, OwnerCommand)
		}
		if declared, err := retention.ParseBytes(root.MaxBytes); err != nil || declared != policy.BudgetBytes {
			t.Errorf("contract max_bytes %q disagrees with the owner budget %d bytes", root.MaxBytes, policy.BudgetBytes)
		}
	}
	if !found {
		t.Fatal("repository contract declares no go-build-cache root")
	}
}

// activationHome is an isolated user: its own home, Vrooli runtime home, and
// go env file. The real user's go env file is never read or written.
type activationHome struct {
	service Service
	program string
	envFile string
}

func newActivationHome(t *testing.T) activationHome {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the build-cache protocol requires Linux")
	}
	if os.Geteuid() == 0 {
		t.Skip("the protocol refuses to run as root")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	home := t.TempDir()
	envFile := filepath.Join(t.TempDir(), "go-env")
	t.Setenv("GOENV", envFile)
	t.Setenv("GOCACHEPROG", "")
	service := Service{Home: home}
	program, err := service.Program()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(program, home+string(filepath.Separator)) {
		t.Fatalf("helper path %s escapes the isolated home %s", program, home)
	}
	return activationHome{service: service, program: program, envFile: envFile}
}

func (h activationHome) installHelper(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the real helper")
	}
	if err := os.MkdirAll(filepath.Dir(h.program), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", h.program, "./cmd/vrooli-gocacheprog")
	build.Dir = repoRoot(t)
	build.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}
}

func (h activationHome) envFileContent(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(h.envFile)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(raw)
}

func TestActivateSelfTestsTheHelperThenPointsTheGoEnvFileAtIt(t *testing.T) {
	home := newActivationHome(t)
	home.installHelper(t)
	ctx := context.Background()

	activation, err := home.service.Activate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !activation.Active || activation.ConfiguredProgram != home.program || activation.EnvFile != home.envFile {
		t.Fatalf("activation = %+v, want active with GOCACHEPROG=%s in %s", activation, home.program, home.envFile)
	}
	if content := home.envFileContent(t); !strings.Contains(content, "GOCACHEPROG="+home.program) {
		t.Fatalf("go env file = %q, want GOCACHEPROG=%s", content, home.program)
	}
	// Activation is idempotent.
	if again, err := home.service.Activate(ctx); err != nil || !again.Active {
		t.Fatalf("second activation = %+v %v", again, err)
	}

	deactivated, err := home.service.Deactivate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deactivated.Active || deactivated.ConfiguredProgram != "" || strings.Contains(home.envFileContent(t), "GOCACHEPROG") {
		t.Fatalf("deactivation = %+v, env file %q; want GOCACHEPROG removed", deactivated, home.envFileContent(t))
	}
	if again, err := home.service.Deactivate(ctx); err != nil || again.Active {
		t.Fatalf("second deactivation = %+v %v", again, err)
	}
}

func TestActivationNeverReplacesAnotherCacheProgram(t *testing.T) {
	home := newActivationHome(t)
	home.installHelper(t)
	foreign := "GOCACHEPROG=/opt/other/cacheprog\n"
	if err := os.WriteFile(home.envFile, []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := home.service.Activate(context.Background()); err == nil || !strings.Contains(err.Error(), "will not replace") {
		t.Fatalf("activation over a foreign GOCACHEPROG: %v", err)
	}
	if _, err := home.service.Deactivate(context.Background()); err == nil || !strings.Contains(err.Error(), "does not manage") {
		t.Fatalf("deactivation of a foreign GOCACHEPROG: %v", err)
	}
	if got := home.envFileContent(t); got != foreign {
		t.Fatalf("go env file changed to %q", got)
	}
}

// Once GOCACHEPROG names the helper every go command depends on it, so a
// missing or broken helper must never be activated.
func TestActivationRefusesAHelperThatIsMissingOrFailsItsSelfTest(t *testing.T) {
	home := newActivationHome(t)
	if _, err := home.service.Activate(context.Background()); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("activation without a helper: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(home.program), 0o755); err != nil {
		t.Fatal(err)
	}
	// Speaks nothing of the protocol and exits.
	if err := os.WriteFile(home.program, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := home.service.Activate(context.Background()); err == nil || !strings.Contains(err.Error(), "self-test failed") {
		t.Fatalf("activation of a broken helper: %v", err)
	}
	if got := home.envFileContent(t); strings.Contains(got, "GOCACHEPROG") {
		t.Fatalf("a failed activation wrote the go env file: %q", got)
	}
}
