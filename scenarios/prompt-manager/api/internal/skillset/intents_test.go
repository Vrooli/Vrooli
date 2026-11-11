package skillset

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scenarioWithSkill(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "scenarios", "demo")
	write(t, filepath.Join(root, "skills", "demo", "SKILL.md"), "---\nname: demo\n---\n")
	return root
}

// Feature: a scenario's own skill must surface for the requests it declares.
func TestSkillThatDoesNotSurfaceForItsDeclaredIntentIsWarned(t *testing.T) {
	root := scenarioWithSkill(t)
	write(t, filepath.Join(root, ".vrooli", "intents.json"), `{"scenario":"demo","cases":[{"id":"a","query":"show my tv"},{"id":"b","query":"record screen"}]}`)
	recall := func(_ context.Context, q string, _ int) ([]string, error) {
		if q == "show my tv" {
			return []string{"unrelated", "demo"}, nil
		}
		return []string{"unrelated"}, nil
	}
	r := Result{Scenario: "demo", Status: "ok"}
	CheckIntents(context.Background(), &r, root, recall)
	if r.Intents == nil || r.Intents.Cases != 2 || r.Intents.Passed != 1 || len(r.Intents.Missed) != 1 || r.Intents.Missed[0] != "record screen" {
		t.Fatalf("intents = %+v", r.Intents)
	}
	if len(r.Warnings) != 1 || r.Warnings[0] != "skill-set.intent_recall:usage" || r.Status != "ok" {
		t.Fatalf("warnings %v status %s: recall is advisory and must not fail the phase", r.Warnings, r.Status)
	}
}

func TestSkillWithoutDeclaredIntentsIsWarned(t *testing.T) {
	r := Result{Scenario: "demo", Status: "ok"}
	CheckIntents(context.Background(), &r, scenarioWithSkill(t), nil)
	if len(r.Warnings) != 1 || r.Warnings[0] != "skill-set.intent_missing:usage" {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestScenarioWithoutSkillIsNotWarned(t *testing.T) {
	r := Result{Scenario: "demo", Status: "ok"}
	CheckIntents(context.Background(), &r, filepath.Join(t.TempDir(), "scenarios", "demo"), nil)
	if len(r.Warnings) != 0 {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestUnavailableSearchIsNotEvidenceAboutTheSkill(t *testing.T) {
	root := scenarioWithSkill(t)
	write(t, filepath.Join(root, ".vrooli", "intents.json"), `{"scenario":"demo","cases":[{"id":"a","query":"q"}]}`)
	r := Result{Scenario: "demo", Status: "ok"}
	CheckIntents(context.Background(), &r, root, func(context.Context, string, int) ([]string, error) { return nil, errors.New("down") })
	if len(r.Warnings) != 0 || r.Intents != nil {
		t.Fatalf("an unavailable index produced %v / %+v", r.Warnings, r.Intents)
	}
}

func TestSharedSearchHubCorpusAndPerCaseOwnerAreHonoured(t *testing.T) {
	root := scenarioWithSkill(t)
	write(t, filepath.Join(filepath.Dir(root), "search-hub", "intents", "x.json"), `{"scenario":"*","cases":[{"id":"a","owner":"demo","query":"mine"},{"id":"b","owner":"other","query":"theirs"}]}`)
	got := intentQueries(root, "demo")
	if len(got) != 1 || got[0] != "mine" {
		t.Fatalf("queries = %v", got)
	}
}
