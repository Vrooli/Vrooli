package manifestvalidation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// fakeProbe is a programmable RuntimeProbe test double.
type fakeProbe struct {
	obs    RuntimeObservation
	err    error
	called bool
}

func (f *fakeProbe) Probe(_ context.Context, _ string) (RuntimeObservation, error) {
	f.called = true
	return f.obs, f.err
}

// newServiceWithProbe builds a service whose static path always passes (the
// validManifest group "g1"/command "do" binds Svc.Do, matched by the proto
// surface) so runtime-probe findings are the only ones a test sees.
func newServiceWithProbe(probe RuntimeProbe) *Service {
	return New(Deps{
		Manifests:    stubLoader{raw: []byte(validManifest), path: "cli/manifest.json"},
		Schema:       stubSchema{},
		Protos:       stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
		RuntimeProbe: probe,
	})
}

func findingsWithCode(fs []Finding, code string) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Code == code {
			out = append(out, f)
		}
	}
	return out
}

// matchingRuntime mirrors the validManifest surface so the cross-check is clean.
func matchingRuntime() RuntimeObservation {
	return RuntimeObservation{
		Resolved: true,
		Binary:   "/usr/bin/fixture",
		Commands: []RuntimeCommand{{Group: "g1", Name: "do"}},
	}
}

// Degradation: probe does NOT run when execution is not requested, even with a
// probe wired and a resolve failure queued up.
func TestRuntimeProbe_NotRunWhenExecutionNotRequested(t *testing.T) {
	probe := &fakeProbe{obs: RuntimeObservation{Resolved: false}}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(context.Background(), "fixture") // no WithIncludeExecution
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if probe.called {
		t.Fatalf("probe ran without an execution request")
	}
	if got := findingsWithCode(rep.Findings, CodeCLIBinaryUnrunnable); len(got) != 0 {
		t.Fatalf("expected no runtime findings, got %+v", got)
	}
}

// Degradation: a nil probe seam disables runtime probing entirely.
func TestRuntimeProbe_NilSeamDisablesProbe(t *testing.T) {
	svc := newServiceWithProbe(nil)
	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, code := range []string{CodeCLIBinaryUnrunnable, CodeCLIHelpFailed, CodeCLICommandMissing, CodeCLICommandUndeclared} {
		if got := findingsWithCode(rep.Findings, code); len(got) != 0 {
			t.Fatalf("nil probe should emit no %s, got %+v", code, got)
		}
	}
}

// Binary unresolved (execution requested) -> WARNING, never a hard error.
func TestRuntimeProbe_BinaryUnresolvedDegradesToWarning(t *testing.T) {
	probe := &fakeProbe{obs: RuntimeObservation{Resolved: false}}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findingsWithCode(rep.Findings, CodeCLIBinaryUnrunnable)
	if len(got) != 1 {
		t.Fatalf("want one binary_unrunnable, got %+v", got)
	}
	if got[0].Severity != SeverityWarning {
		t.Fatalf("binary_unrunnable must degrade to warning, got %q", got[0].Severity)
	}
	if !rep.Passed {
		t.Fatalf("an absent binary must not fail the scenario (passed=false)")
	}
}

func TestRuntimeProbe_ProjectEmptyHelpIsNotClean(t *testing.T) {
	findings := runtimeFindingsForTarget(
		RuntimeObservation{Resolved: true, Binary: "/usr/bin/vrooli"},
		&cliapp.Manifest{},
		"cli/manifest.json",
		ProjectTargetID,
	)
	got := findingsWithCode(findings, CodeProjectCLIEmpty)
	if len(got) != 1 || got[0].Severity != SeverityWarning {
		t.Fatalf("empty project help must emit a required clean warning, got %+v", findings)
	}
}

// Help broken (binary present) -> ERROR.
func TestRuntimeProbe_HelpFailedIsError(t *testing.T) {
	probe := &fakeProbe{obs: RuntimeObservation{Resolved: true, Binary: "/usr/bin/fixture", HelpFailed: true, HelpError: "exit status 1"}}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findingsWithCode(rep.Findings, CodeCLIHelpFailed)
	if len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("want one help_failed error, got %+v", got)
	}
	if rep.Passed {
		t.Fatalf("a broken --help must fail the scenario")
	}
}

// Clean match -> no runtime findings.
func TestRuntimeProbe_MatchingSurfaceNoFindings(t *testing.T) {
	probe := &fakeProbe{obs: matchingRuntime()}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, code := range []string{CodeCLIBinaryUnrunnable, CodeCLIHelpFailed, CodeCLICommandMissing, CodeCLICommandUndeclared} {
		if got := findingsWithCode(rep.Findings, code); len(got) != 0 {
			t.Fatalf("clean surface should emit no %s, got %+v", code, got)
		}
	}
	if !rep.Passed {
		t.Fatalf("clean runtime surface should pass")
	}
}

// Undeclared-accepted: the binary exposes a command under a manifest group that
// the manifest does not declare.
func TestRuntimeProbe_UndeclaredCommandIsError(t *testing.T) {
	obs := matchingRuntime()
	obs.Commands = append(obs.Commands, RuntimeCommand{Group: "g1", Name: "sneaky"})
	probe := &fakeProbe{obs: obs}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findingsWithCode(rep.Findings, CodeCLICommandUndeclared)
	if len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("want one command_undeclared error for the undeclared command, got %+v", got)
	}
}

func TestRuntimeProbe_RuntimeOnlyGroupIsUndeclaredAndLowersDiscoveryCoverage(t *testing.T) {
	obs := matchingRuntime()
	obs.Commands = append(obs.Commands, RuntimeCommand{Group: "g2", Name: "one"}, RuntimeCommand{Group: "g2", Name: "two"})
	rep, err := newServiceWithProbe(&fakeProbe{obs: obs}).ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsWithCode(rep.Findings, CodeCLICommandUndeclared); len(got) != 2 {
		t.Fatalf("want two runtime-only group findings, got %+v", got)
	}
	if got := findingsWithCode(rep.Findings, CodeCLIDiscoveryCoverage); len(got) != 1 {
		t.Fatalf("want discovery coverage finding, got %+v", got)
	}
}

func TestRuntimeProbe_OmissionNamingLiveCommandIsRejected(t *testing.T) {
	m := &cliapp.Manifest{Omitted: []cliapp.ManifestOmission{{Service: "Svc", Method: "Do", Reason: "Hand-registered as 'g1 do' through another path."}}}
	if got := omissionContradictionFindings(matchingRuntime(), m, "cli/manifest.json"); len(got) != 1 {
		t.Fatalf("want contradiction finding, got %+v", got)
	}
}

// Built-in commands cli-core injects (help/version/configure/completion) are
// never flagged as undeclared even when they show up under a manifest group.
func TestRuntimeProbe_BuiltinsNotFlagged(t *testing.T) {
	obs := matchingRuntime()
	obs.Commands = append(obs.Commands,
		RuntimeCommand{Group: "g1", Name: "help"},
		RuntimeCommand{Group: "g1", Name: "version"},
	)
	probe := &fakeProbe{obs: obs}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsWithCode(rep.Findings, CodeCLICommandUndeclared); len(got) != 0 {
		t.Fatalf("built-ins must not be flagged, got %+v", got)
	}
}

// Declared-but-missing: the manifest declares a command the binary does not
// expose under a group it otherwise exposes.
func TestRuntimeProbe_DeclaredButMissingIsError(t *testing.T) {
	// Runtime exposes group g1 but only a different command, so "do" is missing.
	obs := RuntimeObservation{
		Resolved: true,
		Binary:   "/usr/bin/fixture",
		Commands: []RuntimeCommand{{Group: "g1", Name: "other"}},
	}
	probe := &fakeProbe{obs: obs}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	missing := findingsWithCode(rep.Findings, CodeCLICommandMissing)
	if len(missing) != 1 {
		t.Fatalf("want one command_missing finding, got %+v", missing)
	}
	undeclared := findingsWithCode(rep.Findings, CodeCLICommandUndeclared)
	if len(undeclared) != 1 {
		t.Fatalf("want one command_undeclared finding, got %+v", undeclared)
	}
}

// A successfully probed scenario binary that omits an entire manifest group is
// the strongest form of false capability absence and must not pass silently.
func TestRuntimeProbe_AbsentGroupReportsEveryDeclaredCommandMissing(t *testing.T) {
	obs := RuntimeObservation{
		Resolved: true,
		Binary:   "/usr/bin/fixture",
		Commands: []RuntimeCommand{}, // binary exposes nothing under g1
	}
	probe := &fakeProbe{obs: obs}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsWithCode(rep.Findings, CodeCLICommandMissing); len(got) != 1 {
		t.Fatalf("absent group must report its declared command missing, got %+v", got)
	}
}

func TestRuntimeProbe_OpaqueLegacyParentDoesNotInventMissingLeaves(t *testing.T) {
	obs := RuntimeObservation{
		Resolved: true,
		Binary:   "/usr/bin/fixture",
		Commands: []RuntimeCommand{{Group: "", Name: "g1"}},
	}
	probe := &fakeProbe{obs: obs}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsWithCode(rep.Findings, CodeCLICommandMissing); len(got) != 0 {
		t.Fatalf("opaque legacy parent must not invent missing leaves, got %+v", got)
	}
}

func TestRuntimeProbe_DeclaredFrameworkBuiltinIsImplicitlyPresent(t *testing.T) {
	m := &cliapp.Manifest{Groups: []cliapp.ManifestGroup{{
		Name:     "meta",
		Flat:     true,
		Commands: []cliapp.ManifestCommand{{Name: "help"}},
	}}}
	got := commandSurfaceFindings(
		RuntimeObservation{Resolved: true, Binary: "/usr/bin/fixture"},
		m,
		"cli/manifest.json",
	)
	if missing := findingsWithCode(got, CodeCLICommandMissing); len(missing) != 0 {
		t.Fatalf("framework builtin must be treated as present, got %+v", missing)
	}
}

// Probe infrastructure error degrades to a warning (not a scenario defect).
func TestRuntimeProbe_ProbeErrorDegrades(t *testing.T) {
	probe := &fakeProbe{err: errors.New("exec environment broken")}
	svc := newServiceWithProbe(probe)

	rep, err := svc.ValidateScenario(WithIncludeExecution(context.Background(), true), "fixture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findingsWithCode(rep.Findings, CodeCLIBinaryUnrunnable)
	if len(got) != 1 || got[0].Severity != SeverityWarning {
		t.Fatalf("probe error should degrade to one warning, got %+v", got)
	}
	if !rep.Passed {
		t.Fatalf("a probe-infrastructure error must not fail the scenario")
	}
}

// writeScriptOnPath drops an executable shell script named `name` into a fresh
// temp dir, prepends that dir to PATH, and returns. Real resolve+exec proof for
// the production probe (LookPath + ExecRunner). Skips on Windows.
func writeScriptOnPath(t *testing.T, name, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "POSIX shell fixture; not run on Windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, name)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// End-to-end production probe against a real script: resolve on PATH, exec
// `--help`, parse the tree into the observed command surface.
func TestCLIRuntimeProbe_RealBinaryResolveExecParse(t *testing.T) {
	// A cli-core-shaped help: a "Commands:" section with one group + one leaf.
	writeScriptOnPath(t, "probefix", `#!/bin/sh
cat <<'HELP'
probefix CLI

Usage:
  probefix <command>

Commands:
    g1
      do                 Do the thing
HELP
`)
	probe := NewCLIRuntimeProbe(5 * time.Second)
	obs, err := probe.Probe(context.Background(), "probefix")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !obs.Resolved {
		t.Fatalf("expected the script to resolve on PATH")
	}
	if obs.HelpFailed {
		t.Fatalf("help should have run cleanly, got HelpFailed (%s)", obs.HelpError)
	}
	found := false
	for _, c := range obs.Commands {
		if c.Name == "do" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected to observe command 'do', got %+v", obs.Commands)
	}
}

// BAS's scenario-local CLI is the binary selected by the comprehensive
// contracts provider. Keep its help-tree probe honest for the two commands
// that previously disappeared when a stale local binary was left in place.
func TestCLIRuntimeProbe_BASLocalCLIExposesManifestLeaves(t *testing.T) {
	repoRoot := findRepoRoot(t)
	cliDir := filepath.Join(repoRoot, "scenarios", "browser-automation-studio", "cli")
	t.Setenv("PATH", cliDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	obs, err := NewCLIRuntimeProbe(5*time.Second).Probe(context.Background(), "browser-automation-studio")
	if err != nil {
		t.Fatalf("probe BAS CLI: %v", err)
	}
	if !obs.Resolved || obs.HelpFailed {
		t.Fatalf("BAS local CLI must resolve with working help: %+v", obs)
	}
	want := map[string]bool{
		"ai/element-at-coordinate":       false,
		"projects/bulk-delete-workflows": false,
	}
	for _, command := range obs.Commands {
		path := command.Group + "/" + command.Name
		if _, tracked := want[path]; tracked {
			want[path] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("BAS local CLI help tree did not expose %s; commands=%+v", path, obs.Commands)
		}
	}
}

// End-to-end production probe against a binary whose --help exits non-zero:
// resolves but HelpFailed is reported.
func TestCLIRuntimeProbe_RealBinaryHelpFails(t *testing.T) {
	writeScriptOnPath(t, "brokenfix", `#!/bin/sh
echo "boom" 1>&2
exit 1
`)
	probe := NewCLIRuntimeProbe(5 * time.Second)
	obs, err := probe.Probe(context.Background(), "brokenfix")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !obs.Resolved {
		t.Fatalf("expected the script to resolve on PATH")
	}
	if !obs.HelpFailed {
		t.Fatalf("expected HelpFailed for a non-zero --help, got %+v", obs)
	}
}
