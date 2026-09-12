package vps

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

func ptr(d domain.Deployment) *domain.Deployment { return &d }

func domain0(h *harness) domain.Deployment {
	return domain.Deployment{ID: h.depID, ScenarioID: h.manifest.Scenario.ID, Environment: "certification", Target: targetRef()}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func containsArgSequence(args []string, want ...string) bool {
	for i := 0; i+len(want) <= len(args); i++ {
		matched := true
		for j := range want {
			if args[i+j] != want[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// [REQ:STC-P0-016] The install-scope verbs never touch the live scenario
// tree: staging is the target owner's `release stage` into its releases
// directory, the inventory is the read-only `data inventory` verb, and
// setup runs as argv through the deployment-local binary.
func TestInstallScopeVerbsNeverTouchTheLiveTree(t *testing.T) {
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, newFakeTarget(), w, "install")
	plan := h.compile(execplan.ScopeInstall, execplan.Observations{})
	cc := CommandContext{DeploymentID: h.depID, ScenarioID: h.manifest.Scenario.ID, Identity: Identity{OperationID: "op-1", Fence: 1}, Manifest: h.manifest}
	for _, action := range plan.Actions {
		commands, err := ActionCommands(action, cc)
		if err != nil {
			t.Fatalf("%s: %v", action.ID, err)
		}
		for _, tc := range commands {
			argv := strings.Join(tc.Command.Argv(), " ")
			for _, forbidden := range []string{"rm ", "mv ", "tar ", "xargs", "find ", "/root/Vrooli/scenarios"} {
				if strings.Contains(argv, forbidden) {
					t.Fatalf("%s carries %q: %s", action.ID, forbidden, argv)
				}
			}
		}
		switch action.ID {
		case execplan.OpReleaseStage:
			if commands[0].Command.Verb != "cloud-target release stage" || !commands[0].Command.Effectful {
				t.Fatalf("stage = %+v", commands[0])
			}
		case execplan.OpDataInventory:
			if commands[0].Command.Verb != "cloud-target data inventory" || commands[0].Command.Effectful {
				t.Fatalf("inventory must be a read verb: %+v", commands[0])
			}
		case execplan.OpConfigApply:
			if commands[0].Command.Verb != "setup" || !containsArgSequence(commands[0].Command.Args, "--yes", "yes", "--environment", "production") || !containsArg(commands[0].Command.Args, "--selection-b64") {
				t.Fatalf("setup argv = %v", commands[0].Command.Argv())
			}
		case execplan.OpReleaseActivate:
			t.Fatal("install scope stages; activation belongs to the runtime scope")
		}
	}
}

// [REQ:STC-P0-010] The autoheal scope declaration is a typed document
// delivered as a file, never a printf over the shell, and it names the
// deployment's scenario and dependencies.
func TestAutohealScopeIsATypedDocument(t *testing.T) {
	manifest := domain.CloudManifest{Environment: "production", Scenario: domain.ManifestScenario{ID: "my-app"}, Dependencies: domain.ManifestDependencies{Resources: []string{"postgres", "redis"}, Scenarios: []string{"my-app", "vrooli-autoheal"}}}
	var doc map[string]any
	if err := json.Unmarshal(AutohealScopeJSON(manifest), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["scenario_id"] != "my-app" || doc["desired_state"] != "runtime-owner" {
		t.Fatalf("doc = %v", doc)
	}
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, target, w, "autoheal")
	h.manifest.Bundle.IncludeAutoheal = true
	if _, execErr := h.execute(context.Background(), h.compile(execplan.ScopeInstall, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
		t.Fatalf("install: %v (%s)", execErr, execErr.ActionID)
	}
	var delivered bool
	for _, f := range target.delivered {
		if f.Role == "autoheal_scope" && f.RemotePath == "/root/Vrooli/.vrooli/cloud/autoheal-scope.json" {
			delivered = true
		}
	}
	if !delivered {
		t.Fatalf("autoheal scope must be delivered as a file: %+v", target.delivered)
	}
}
