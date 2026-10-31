package wiring

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/promptmanager"
	"agent-manager/internal/usageattribution"
)

type recordedBugs []promptmanager.ScenarioQABug

func (r *recordedBugs) PublishScenarioQABug(_ context.Context, bug promptmanager.ScenarioQABug) (string, error) {
	*r = append(*r, bug)
	return "knw-1", nil
}

// A friction report reaches scenario-qa as written, keyed by its fingerprint
// so the inbox files one bug per problem however often it is reported.
func TestFrictionBugsAreFiledOncePerFingerprint(t *testing.T) {
	bugs := &recordedBugs{}
	filer := frictionBugFiler{client: bugs}
	report := usageattribution.FrictionBug{Fingerprint: "fp1", Title: "tool friction (failure): vrooli", Owner: "vrooli", Actual: "it failed.", Description: "evidence", Repro: []string{"run it"}, Context: map[string]string{"owner": "vrooli"}}
	if id, err := filer.FileFrictionBug(context.Background(), report); err != nil || id != "knw-1" {
		t.Fatalf("file = %q, %v", id, err)
	}
	got := (*bugs)[0]
	if got.IdempotencyKey != "agent-manager-friction/fp1" || got.Title != report.Title || got.Actual != report.Actual || got.Context["owner"] != "vrooli" || len(got.Repro) != 1 {
		t.Fatalf("bug = %+v", got)
	}
}

// API ports map to the scenario serving them from the control plane's process
// registry; a UI port or an unknown port names no one.
func TestReadPortRegistryNamesTheServingScenario(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	dir := home + "/processes/scenarios/agent-manager"
	if err := mkdirWrite(dir+"/start-api.json", `{"scenario":"agent-manager","port":18800,"port_key":"API_PORT"}`); err != nil {
		t.Fatal(err)
	}
	if err := mkdirWrite(dir+"/start-ui.json", `{"scenario":"agent-manager","port":35000,"port_key":"UI_PORT"}`); err != nil {
		t.Fatal(err)
	}
	owners := processPortOwners()
	if owners(18800) != "agent-manager" || owners(35000) != "" || owners(1) != "" {
		t.Fatalf("owners: api=%q ui=%q unknown=%q", owners(18800), owners(35000), owners(1))
	}
}

func mkdirWrite(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
