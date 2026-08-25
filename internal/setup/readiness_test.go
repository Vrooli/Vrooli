package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/projectstate"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

// readinessFixtureRoot builds a repository whose declared credential
// population is exactly one required address, so the verdict is unambiguous.
func readinessFixtureRootIn(t *testing.T, _ string, requiredCredential bool) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	descriptors := ""
	if requiredCredential {
		descriptors = `"credentials": {"descriptors": [{"logical_id": "vrooli/readiness-fixture", "field": "token", "required": true}]},`
	}
	write(filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Fixture"},
  `+descriptors+`
  "hostTools": []
}`)
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSetupReadinessVerdictReportsReadyWhenNothingIsUnresolved(t *testing.T) {
	verdict := setupReadinessVerdict(credentialinventory.Result{}, nil, vrooliruntime.Report{}, nil)
	if verdict.Status != ReadinessStatusReady || verdict.Source != ReadinessSourceInProcess {
		t.Fatalf("verdict = %+v, want ready from the in-process source", verdict)
	}
	if len(verdict.Blockers) != 0 {
		t.Fatalf("blockers = %v, want none", verdict.Blockers)
	}
}

// A required credential that is declared and absent is the exact condition a
// green "complete" used to hide.
func TestSetupReadinessVerdictNamesAnAbsentRequiredCredential(t *testing.T) {
	verdict := setupReadinessVerdict(
		credentialinventory.Result{RequiredAbsent: []string{"vrooli/calendar:jwt-secret"}}, nil,
		vrooliruntime.Report{}, nil)
	if verdict.Status != ReadinessStatusMissing {
		t.Fatalf("verdict = %+v, want missing", verdict)
	}
	if len(verdict.Blockers) != 1 || verdict.Blockers[0] != "vrooli/calendar:jwt-secret" {
		t.Fatalf("blockers = %v", verdict.Blockers)
	}
}

func TestSetupReadinessVerdictNamesAMissingRequiredHostItem(t *testing.T) {
	verdict := setupReadinessVerdict(credentialinventory.Result{}, nil,
		vrooliruntime.Report{MissingRequired: []string{"workspace_sandbox_userns"}}, nil)
	if verdict.Status != ReadinessStatusMissing {
		t.Fatalf("verdict = %+v, want missing", verdict)
	}
	if len(verdict.Blockers) != 1 || verdict.Blockers[0] != "workspace_sandbox_userns" {
		t.Fatalf("blockers = %v", verdict.Blockers)
	}
}

// An optional item that is unresolved lowers the verdict without blocking it.
func TestSetupReadinessVerdictReportsDegradedForOptionalGapsOnly(t *testing.T) {
	verdict := setupReadinessVerdict(credentialinventory.Result{}, nil,
		vrooliruntime.Report{MissingOptional: []string{"docker"}}, nil)
	if verdict.Status != ReadinessStatusDegraded || len(verdict.Blockers) != 0 {
		t.Fatalf("verdict = %+v, want degraded with no blocker", verdict)
	}
}

// A source that could not be read must never be reported as ready. Reporting
// ready from an unread source is the same failure as trusting the marker.
func TestSetupReadinessVerdictNeverReportsReadyFromAnUnreadSource(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		inventoryErr error
		reportErr    error
	}{
		{"inventory unreadable", errUnreadableSource, nil},
		{"requirements unreadable", nil, errUnreadableSource},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			verdict := setupReadinessVerdict(credentialinventory.Result{}, testCase.inventoryErr, vrooliruntime.Report{}, testCase.reportErr)
			if verdict.Status == ReadinessStatusReady {
				t.Fatalf("verdict = %+v, want a non-ready status", verdict)
			}
			if verdict.Source != ReadinessSourceUnavailable || verdict.Reason == "" {
				t.Fatalf("verdict = %+v, want an unavailable source with a reason", verdict)
			}
		})
	}
}

var errUnreadableSource = &unreadableSourceError{}

type unreadableSourceError struct{}

func (*unreadableSourceError) Error() string { return "source is unreachable" }

// The IO half must actually read the repository it is given. This asserts a
// delta rather than an absolute list, because credentialinventory also
// enumerates live managed instances whose presence depends on the host.
func TestVerifySetupReadinessReadsTheProjectManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	withoutCredential := readinessFixtureRootIn(t, home, false)
	withCredential := readinessFixtureRootIn(t, home, true)

	base := verifySetupReadiness(withoutCredential, vrooliruntime.Report{}, nil)
	verdict := verifySetupReadiness(withCredential, vrooliruntime.Report{}, nil)
	requireBlockerDelta(t, base.Blockers, verdict.Blockers, "vrooli/readiness-fixture:token")
}

// requireBlockerDelta asserts that exactly one named blocker was added.
func requireBlockerDelta(t *testing.T, before, after []string, added string) {
	t.Helper()
	baseline := map[string]struct{}{}
	for _, name := range before {
		baseline[name] = struct{}{}
	}
	extra := make([]string, 0, 1)
	for _, name := range after {
		if _, known := baseline[name]; !known {
			extra = append(extra, name)
		}
	}
	if len(extra) != 1 || extra[0] != added {
		t.Fatalf("added blockers = %v, want exactly [%s]", extra, added)
	}
}

func TestFinalizeReportsConfigurationPendingWhenTheVerdictContradictsTheMarker(t *testing.T) {
	root := readinessFixtureRootIn(t, t.TempDir(), true)
	home := markerHome(t, root)
	base := SetupResult{Version: SetupResultVersion, Status: SetupStatusSuccess, Category: SetupCategorySuccess}
	verdict := SetupReadiness{Status: ReadinessStatusMissing, Blockers: []string{"vrooli/readiness-fixture:token"}, Source: ReadinessSourceInProcess}

	result := finalizeSetupResultConfiguration(base, home, root, nil, &verdict)
	if result.Category != SetupCategoryConfigurationPending || !result.ConfigurationPending {
		t.Fatalf("result = %+v, want configuration_pending", result)
	}
	if !strings.Contains(result.Remediation, "vrooli setup --include-optional --maintenance-window --sudo-mode=ask --onboarding=auto") {
		t.Fatalf("remediation = %q, want the single in-flow command", result.Remediation)
	}
	if result.Readiness == nil || result.Readiness.Status != ReadinessStatusMissing {
		t.Fatalf("readiness = %+v, want the verdict recorded on the result", result.Readiness)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"readiness"`) {
		t.Fatalf("result document does not carry the readiness object: %s", payload)
	}
}

func TestFinalizeKeepsSuccessAndStatesTheVerdictWhenReady(t *testing.T) {
	root := readinessFixtureRootIn(t, t.TempDir(), false)
	home := markerHome(t, root)
	base := SetupResult{Version: SetupResultVersion, Status: SetupStatusSuccess, Category: SetupCategorySuccess}
	verdict := SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}

	result := finalizeSetupResultConfiguration(base, home, root, nil, &verdict)
	if result.Category != SetupCategorySuccess || result.ConfigurationPending {
		t.Fatalf("result = %+v, want success", result)
	}
	if !strings.Contains(result.Remediation, "verified") {
		t.Fatalf("remediation = %q, want it to state the verified verdict", result.Remediation)
	}
}

func TestFinalizeStillReportsPendingWhenTheMarkerIsAbsent(t *testing.T) {
	root := readinessFixtureRootIn(t, t.TempDir(), false)
	home := t.TempDir()
	base := SetupResult{Version: SetupResultVersion, Status: SetupStatusSuccess, Category: SetupCategorySuccess}
	verdict := SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}

	result := finalizeSetupResultConfiguration(base, home, root, nil, &verdict)
	if result.Category != SetupCategoryConfigurationPending || !result.ConfigurationPending {
		t.Fatalf("result = %+v, want configuration_pending while the marker is absent", result)
	}
}

// markerHome returns a runtime home in which both first-run markers are present.
func markerHome(t *testing.T, root string) string {
	t.Helper()
	home := t.TempDir()
	if err := markComplete(home, root); err != nil {
		t.Fatalf("markComplete: %v", err)
	}
	if err := projectstate.MarkConfigurationComplete(home, root, "fixture-digest"); err != nil {
		t.Fatalf("mark configuration complete: %v", err)
	}
	return home
}

func TestRenderSetupReadinessVerdictNeverPrintsAnUnconditionalSuccess(t *testing.T) {
	cases := []struct {
		name    string
		verdict SetupReadiness
		marker  bool
		want    string
	}{
		{"ready", SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}, true, "configuration verified ready"},
		{"degraded", SetupReadiness{Status: ReadinessStatusDegraded, Source: ReadinessSourceInProcess}, true, "optional items unresolved"},
		{"missing", SetupReadiness{Status: ReadinessStatusMissing, Blockers: []string{"vrooli/x:y"}, Source: ReadinessSourceInProcess}, true, "configuration is not verified"},
		{"unavailable", SetupReadiness{Status: ReadinessStatusUnsupported, Source: ReadinessSourceUnavailable, Reason: "inventory unreachable"}, true, "could not be verified"},
		{"no marker", SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}, false, "configuration remains pending"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var out bytes.Buffer
			renderSetupReadinessVerdict(&out, testCase.verdict, testCase.marker)
			if !strings.Contains(out.String(), testCase.want) {
				t.Fatalf("output = %q, want it to contain %q", out.String(), testCase.want)
			}
			if strings.Contains(out.String(), "Setup completed successfully.") {
				t.Fatalf("output still carries the unconditional success line: %q", out.String())
			}
		})
	}
}

// TestPendingConfigurationNeverClaimsCompletion pins a contradiction observed on
// a live run, whose last two lines were:
//
//	[INFO]    Bootstrap setup completed; configuration remains pending until onboarding reports completion.
//	[ACTION]  Setup and onboarding configuration are complete. Optional items remain unresolved; ...
//
// The remediation was printed outside the switch, so the ready/degraded wording
// ("configuration are complete") was emitted even when the marker was absent and
// configuration was by definition pending.
func TestPendingConfigurationNeverClaimsCompletion(t *testing.T) {
	for _, status := range []string{ReadinessStatusReady, ReadinessStatusDegraded, ReadinessStatusMissing} {
		t.Run(status, func(t *testing.T) {
			var out strings.Builder
			renderSetupReadinessVerdict(&out, SetupReadiness{
				Status:   status,
				Source:   ReadinessSourceInProcess,
				Blockers: []string{"onboarding_apply_privileges"},
			}, false)
			rendered := out.String()
			if !strings.Contains(rendered, "remains pending") {
				t.Fatalf("an absent marker must report pending, got %q", rendered)
			}
			if strings.Contains(rendered, "configuration are complete") {
				t.Fatalf("pending configuration claimed completion: %q", rendered)
			}
			if !strings.Contains(rendered, "vrooli-onboarding wizard run") {
				t.Fatalf("pending configuration must name the command that finishes it, got %q", rendered)
			}
		})
	}
}

// TestVerifiedCompletionStillStatesTheVerdict guards the opposite error: once the
// marker is present the verdict must still be reported, so this fix cannot be
// satisfied by silencing the action line everywhere.
func TestVerifiedCompletionStillStatesTheVerdict(t *testing.T) {
	var out strings.Builder
	renderSetupReadinessVerdict(&out, SetupReadiness{Status: ReadinessStatusDegraded, Source: ReadinessSourceInProcess}, true)
	rendered := out.String()
	if !strings.Contains(rendered, "configuration are complete") {
		t.Fatalf("a present marker with a degraded verdict must still state completion, got %q", rendered)
	}
}
