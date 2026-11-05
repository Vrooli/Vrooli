package archtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/fixtures"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"
)

// shellSyntax is the byte set reach.ValidateArgs refuses. A plan input that
// carries one of these would need quoting to be safe on a transport that
// joins argv, and the plan contract forbids anything that needs quoting.
const shellSyntax = ";|&$`<>\n\r\x00'\"\\"

// fixtureManifest builds the generic manifest a fixture deploys with: the
// same shape api/generality uses, with documentation addresses only.
func fixtureManifest(w *fixtures.Workload) (domain.CloudManifest, *domain.Closure) {
	ports := domain.ManifestPorts{}
	for i, listener := range w.Declaration.Listeners {
		ports[listener.Name] = 3000 + i
	}
	if _, ok := ports["ui"]; !ok {
		ports["ui"] = 3000
	}
	manifest := domain.CloudManifest{
		Version:      "1",
		Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario:     domain.ManifestScenario{ID: w.Declaration.PrimaryScenario},
		Dependencies: domain.ManifestDependencies{Resources: append([]string(nil), w.Declaration.Resources...), Scenarios: append(append([]string(nil), w.Declaration.Scenarios...), w.Declaration.PrimaryScenario)},
		Ports:        ports,
		Edge:         domain.ManifestEdge{Domain: w.ID + ".example.test", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
	c := &domain.Closure{SchemaVersion: "1", ScenarioID: w.Declaration.PrimaryScenario, Digest: "sha256:fixture-" + w.ID}
	c.Components = append(c.Components, domain.ClosureComponent{ID: w.Declaration.PrimaryScenario, Kind: domain.ClosureKindScenario, Required: true, Recovery: &domain.ClosureRecovery{CodeRollback: "any_predecessor", SchemaStrategy: "expand_contract"}})
	return manifest, c
}

// fixtureRelease writes a complete release set (bundle, manifest, native
// CLI, .complete marker) so the release actions derive their target argv.
func fixtureRelease(t *testing.T, dir, content string) string {
	t.Helper()
	bundle := []byte("fixture-bundle-" + content)
	binary := []byte("fixture-native-cli")
	sumBundle := sha256.Sum256(bundle)
	sumBinary := sha256.Sum256(binary)
	manifest := cloudrelease.Manifest{
		SchemaVersion: cloudrelease.ManifestSchemaVersion, BundleSHA256: hex.EncodeToString(sumBundle[:]),
		NativeCLI:     cloudrelease.NativeCLI{SHA256: hex.EncodeToString(sumBinary[:]), GOOS: "linux", GOARCH: "amd64"},
		ClosureDigest: strings.Repeat("c", 64), ConfigurationDigest: strings.Repeat("d", 64),
		Provenance: cloudrelease.Provenance{Builder: "scenario-to-cloud", Policy: "development-local-unsigned"},
		Limits:     cloudrelease.Limits{MaxEntries: 1000, MaxExpandedBytes: 1 << 20, MaxEntryBytes: 1 << 16},
	}
	digest, err := cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseDigest = digest
	releaseDir := filepath.Join(dir, "releases", digest)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(manifest)
	for name, body := range map[string][]byte{cloudrelease.BundleFileName: bundle, cloudrelease.ManifestFileName: raw, cloudrelease.NativeCLIFileName("linux", "amd64"): binary, cloudrelease.CompleteMarker: []byte("")} {
		if err := os.WriteFile(filepath.Join(releaseDir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(releaseDir, cloudrelease.BundleFileName)
}

// TestFixturePlansCarryNoShellSyntax [REQ:STC-P0-043] compiles the full
// plan of every workload fixture and asserts that every action input and
// every derived target argv passes reach.ValidateArgs. The plan is the only
// thing that reaches a target, so this is the boundary at which shell
// composition would enter.
func TestFixturePlansCarryNoShellSyntax(t *testing.T) {
	cat, err := fixtures.Load(filepath.Join(scenarioRoot(t), "fixtures"))
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	if len(cat.Workloads) < 4 {
		t.Fatalf("catalog holds only %d workloads; the fixture root is wrong", len(cat.Workloads))
	}
	ids := make([]string, 0, len(cat.Workloads))
	for id := range cat.Workloads {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		w := cat.Workloads[id]
		t.Run(id, func(t *testing.T) {
			manifest, closure := fixtureManifest(w)
			plan, err := vps.CompilePlan(context.Background(), vps.PlanRequest{
				Manifest:     manifest,
				BundlePath:   fixtureRelease(t, t.TempDir(), w.ID),
				Closure:      closure,
				Scope:        execplan.ScopeFull,
				Observations: execplan.Observations{DeploymentRevision: 4},
			})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if len(plan.Actions) == 0 {
				t.Fatal("plan has no actions")
			}
			cc := vps.CommandContext{DeploymentID: "dep-" + w.ID, ScenarioID: manifest.Scenario.ID, Identity: vps.Identity{OperationID: "op-archtest", Fence: 7}, Manifest: manifest}
			derived := 0
			for _, action := range plan.Actions {
				for key, value := range action.Inputs {
					if strings.ContainsAny(value, shellSyntax) {
						t.Errorf("action %s input %s carries shell syntax: %q", action.ID, key, value)
					}
					if value == "" {
						continue
					}
					if err := reach.ValidateArgs([]string{value}); err != nil {
						t.Errorf("action %s input %s is not a valid argument: %v", action.ID, key, err)
					}
				}
				commands, err := vps.ActionCommands(action, cc)
				if err != nil {
					t.Errorf("action %s: derive commands: %v", action.ID, err)
					continue
				}
				for _, command := range commands {
					derived++
					if err := reach.ValidateArgs(command.Command.Args); err != nil {
						t.Errorf("action %s step %s argv refused: %v (argv %q)", action.ID, command.Step, err, command.Command.Args)
					}
					if strings.ContainsAny(command.Command.Verb, shellSyntax) {
						t.Errorf("action %s verb %q carries shell syntax", action.ID, command.Command.Verb)
					}
				}
			}
			if derived == 0 {
				t.Fatal("no target command derived from the plan; ActionCommands produced nothing")
			}
		})
	}
}
