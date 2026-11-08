//go:build ignore

// Workflow Health runs this only after installing paired test storage.
// The disposable project lives inside Test Genie's SQLite isolation root and
// is removed by that owner's cleanup, including on a failed browser case.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	root := os.Getenv("VROOLI_STORAGE_ROOT")
	scenario := os.Getenv("TEST_GENIE_SCENARIO_DIR")
	lease := os.Getenv("TEST_GENIE_ROUTING_LEASE_ID")
	if !validSeedIdentity(os.Getenv("TEST_GENIE_SEEDS"), root, scenario, lease) {
		log.Fatal("seed requires the installed Agent Manager test lease and disposable storage root")
	}
	project := filepath.Join(root, "browser-spawn-project")
	if err := os.MkdirAll(project, 0700); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "README.txt"), []byte("Disposable read-only browser spawn acceptance workspace.\n"), 0600); err != nil {
		log.Fatal(err)
	}
	// A protected sandbox needs an actual repository, wholly inside this
	// disposable owner root. These commands never touch the shared worktree.
	hooks := filepath.Join(root, "empty-git-hooks")
	if err := os.MkdirAll(hooks, 0700); err != nil {
		log.Fatal(err)
	}
	config := filepath.Join(root, "empty-git-config")
	if err := os.WriteFile(config, nil, 0600); err != nil {
		log.Fatal(err)
	}
	// Inherited repository overrides and user hooks must not redirect this
	// disposable fixture into the caller's shared worktree.
	gitEnv := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GIT_") {
			gitEnv = append(gitEnv, value)
		}
	}
	gitEnv = append(gitEnv, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+config)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "README.txt"}, {"-c", "user.name=Browser acceptance", "-c", "user.email=browser-acceptance@localhost", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Initialize disposable acceptance workspace"}} {
		args = append([]string{"-c", "core.hooksPath=" + hooks}, args...)
		command := exec.CommandContext(ctx, "git", args...)
		command.Env = gitEnv
		command.Dir = project
		if output, err := command.CombinedOutput(); err != nil {
			log.Fatalf("seed git preparation failed: %v: %s", err, output)
		}
	}
	state, err := json.Marshal(map[string]string{"browser_spawn_project": project})
	if err != nil {
		log.Fatal(err)
	}
	// This is the SeedStatePath contract consumed by the provider's seed owner.
	out := filepath.Join(scenario, "coverage", "runtime", "seed-state.json")
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(out, state, 0600); err != nil {
		log.Fatal(err)
	}
	log.Print("seed: disposable browser spawn workspace ready")
}

// The private allocator and paired routing lease use the same exact run ID.
// A similarly named directory cannot establish ownership of this invocation.
func validSeedIdentity(enabled, root, scenario, lease string) bool {
	return enabled == "1" && lease != "" && filepath.Base(lease) == lease &&
		filepath.IsAbs(root) && filepath.Base(root) == "test-genie-playbooks-"+lease && scenario != ""
}
