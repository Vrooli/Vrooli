package runs

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestScenariosFromPaths(t *testing.T) {
	got := ScenariosFromPaths([]string{
		"scenarios/test-genie/api/main.go",
		"scenarios/test-genie/cli/runs/command.go", // duplicate scenario
		"scenarios/audio-tools/PRD.md",
		"packages/proto/schemas/x.proto", // shared package: out of digest scope
		"docs/README.md",
		"scenarios/README.md", // file directly under scenarios/, not a scenario
		"./scenarios/web-console/ui/src/App.tsx",
		"",
	})
	want := []string{"audio-tools", "test-genie", "web-console"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ScenariosFromPaths = %v, want %v", got, want)
	}
}

// withFakeGit substitutes the git enumeration seam: rev-parse resolves to a
// fixed root and each enumeration command returns the configured lines.
func withFakeGit(t *testing.T, outputs map[string][]string) {
	t.Helper()
	prev := gitLines
	gitLines = func(_ context.Context, _ string, args ...string) ([]string, error) {
		key := strings.Join(args, " ")
		if key == "rev-parse --show-toplevel" {
			return []string{"/repo"}, nil
		}
		if lines, ok := outputs[key]; ok {
			return lines, nil
		}
		return nil, nil
	}
	t.Cleanup(func() { gitLines = prev })
}

func freshResp(scenario string, statuses map[string]string, cmd string) *runspb.CheckFreshnessResponse {
	resp := &runspb.CheckFreshnessResponse{Scenario: scenario, SuggestedCommand: cmd}
	for phase, status := range statuses {
		resp.Phases = append(resp.Phases, &runspb.PhaseFreshness{Phase: phase, Status: status})
	}
	return resp
}

func TestChangedWarnsOnStaleScenario(t *testing.T) {
	withFakeGit(t, map[string][]string{
		"diff --cached --name-only": {"scenarios/demo/api/main.go"},
	})
	withFakeClient(t, &fakeClient{freshnessFn: func(scenario string) (*runspb.CheckFreshnessResponse, error) {
		return freshResp(scenario, map[string]string{"unit": "stale", "structure": "fresh"}, "test-genie execute demo --preset quick"), nil
	}})

	advisory := adviseChanged(context.Background(), nil)
	if !advisory.Checked {
		t.Fatal("expected checked advisory")
	}
	if len(advisory.Warnings) != 1 || advisory.Warnings[0].Scenario != "demo" {
		t.Fatalf("unexpected warnings: %+v", advisory.Warnings)
	}
	if !reflect.DeepEqual(advisory.Warnings[0].StalePhases, []string{"unit"}) {
		t.Fatalf("unexpected stale phases: %v", advisory.Warnings[0].StalePhases)
	}

	var buf bytes.Buffer
	renderChangedAdvisory(&buf, advisory)
	if !strings.Contains(buf.String(), "test-genie hasn't run [unit] since the latest changes in demo") {
		t.Errorf("unexpected rendering: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "Run: test-genie execute demo --preset quick") {
		t.Errorf("missing command: %q", buf.String())
	}
}

func TestChangedExitCodeAndJSON(t *testing.T) {
	withFakeGit(t, map[string][]string{
		"diff --name-only": {"scenarios/demo/api/main.go"},
	})
	withFakeClient(t, &fakeClient{freshnessFn: func(scenario string) (*runspb.CheckFreshnessResponse, error) {
		return freshResp(scenario, map[string]string{"unit": "stale"}, "test-genie execute demo --preset quick"), nil
	}})

	var buf bytes.Buffer
	err := runFreshness(nil, []string{"--changed", "--json"}, &buf)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitRegression {
		t.Fatalf("stale change-set must exit %d, got %v", exitRegression, err)
	}
	out := buf.String()
	for _, needle := range []string{`"checked": true`, `"scenario": "demo"`, `"stale_phases"`, `"command": "test-genie execute demo --preset quick"`} {
		if !strings.Contains(out, needle) {
			t.Errorf("JSON missing %s: %q", needle, out)
		}
	}
}

func TestChangedSilentWhenFresh(t *testing.T) {
	withFakeGit(t, map[string][]string{
		"diff --name-only": {"scenarios/demo/api/main.go"},
	})
	withFakeClient(t, &fakeClient{freshnessFn: func(scenario string) (*runspb.CheckFreshnessResponse, error) {
		return freshResp(scenario, map[string]string{"unit": "fresh"}, ""), nil
	}})
	advisory := adviseChanged(context.Background(), nil)
	if len(advisory.Warnings) != 0 || !advisory.Checked {
		t.Fatalf("fresh scenario must produce no warnings: %+v", advisory)
	}

	var buf bytes.Buffer
	if err := runFreshness(nil, []string{"--changed"}, &buf); err != nil {
		t.Fatalf("fresh change-set must exit clean, got %v", err)
	}
}

func TestChangedDegradesSilentlyWhenTestGenieDown(t *testing.T) {
	withFakeGit(t, map[string][]string{
		"diff --cached --name-only": {"scenarios/demo/api/main.go"},
	})
	withFakeClient(t, &fakeClient{freshnessFn: func(string) (*runspb.CheckFreshnessResponse, error) {
		return nil, errors.New("connection refused")
	}})
	advisory := adviseChanged(context.Background(), nil)
	if advisory.Checked || len(advisory.Warnings) != 0 {
		t.Fatalf("advisory must degrade silently when test-genie is down: %+v", advisory)
	}

	// The command must still exit 0: skipped is not a failure.
	var buf bytes.Buffer
	if err := runFreshness(nil, []string{"--changed"}, &buf); err != nil {
		t.Fatalf("degraded check must exit clean, got %v", err)
	}
	if !strings.Contains(buf.String(), "skipped") {
		t.Errorf("expected skip notice: %q", buf.String())
	}
}

func TestChangedNoScenariosTouched(t *testing.T) {
	withFakeGit(t, map[string][]string{
		"diff --cached --name-only": {"docs/README.md", "packages/proto/x.proto"},
	})
	called := false
	withFakeClient(t, &fakeClient{freshnessFn: func(string) (*runspb.CheckFreshnessResponse, error) {
		called = true
		return nil, nil
	}})
	advisory := adviseChanged(context.Background(), nil)
	if !advisory.Checked || len(advisory.Warnings) != 0 {
		t.Fatalf("non-scenario changes must produce an empty checked advisory: %+v", advisory)
	}
	if called {
		t.Fatal("CheckFreshness must not be called when no scenario is touched")
	}
}

func TestChangedNotAGitRepo(t *testing.T) {
	prev := gitLines
	gitLines = func(context.Context, string, ...string) ([]string, error) {
		return nil, errors.New("not a git repository")
	}
	t.Cleanup(func() { gitLines = prev })

	advisory := adviseChanged(context.Background(), nil)
	if advisory.Checked {
		t.Fatalf("outside a git repo the advisory must be skipped: %+v", advisory)
	}
}

func TestChangedTruncatesLargeChangeSets(t *testing.T) {
	var lines []string
	for _, s := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		lines = append(lines, "scenarios/"+s+"/file.go")
	}
	withFakeGit(t, map[string][]string{
		"diff --cached --name-only": lines,
	})
	withFakeClient(t, &fakeClient{freshnessFn: func(scenario string) (*runspb.CheckFreshnessResponse, error) {
		return freshResp(scenario, map[string]string{"unit": "fresh"}, ""), nil
	}})
	advisory := adviseChanged(context.Background(), nil)
	if !advisory.Truncated || len(advisory.Scenarios) != maxChangedScenarios {
		t.Fatalf("expected truncation to %d scenarios: %+v", maxChangedScenarios, advisory)
	}
}

func TestChangedRejectsScenarioAndPhases(t *testing.T) {
	var buf bytes.Buffer
	if err := runFreshness(nil, []string{"--changed", "--scenario", "demo"}, &buf); err == nil {
		t.Fatal("expected error combining --changed with --scenario")
	}
	if err := runFreshness(nil, []string{"--changed", "--phases", "unit"}, &buf); err == nil {
		t.Fatal("expected error combining --changed with --phases")
	}
	if err := runFreshness(nil, []string{"--changed", "demo"}, &buf); err == nil {
		t.Fatal("expected error combining --changed with a positional scenario")
	}
}
