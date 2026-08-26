package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/projectstate"
)

func credential(owner, id, field string, required bool, status string) credentialReadiness {
	return credentialReadiness{Resource: owner, LogicalID: id, Field: field, Required: required, Status: status}
}

func host(name, kind, status string, required bool) hostReadiness {
	return hostReadiness{readinessItem: readinessItem{Name: name, Status: status}, Kind: kind, Required: required}
}

func blockerNames(items []completionBlocker) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Kind+"/"+item.Name)
	}
	return names
}

// [REQ:ONB-COMPLETE-REQUIRES-READY]
func TestAssessCompletionSeparatesRequiredFromOptional(t *testing.T) {
	readiness := readinessResponse{
		Credentials: []credentialReadiness{
			credential("calendar", "vrooli/calendar", "jwt-secret", true, "unconfigured"),
			credential("project", "vrooli/remote-desktop", "username", false, "unconfigured"),
			credential("openai", "vrooli/openai", "api-key", true, "configured"),
		},
		Hosts: []hostReadiness{
			host("workspace_sandbox_userns", "safeguard", "missing", true),
			host("docker", "tool", "missing", false),
			host("tmux", "tool", "ready", true),
			host("clock", "safeguard", "deferred", false),
		},
		Recovery: recoveryReadiness{RequiredAbsent: []string{"vrooli/calendar:jwt-secret"}},
	}
	assessment := assessCompletion(readiness, nil)
	wantBlockers := []string{"credential/vrooli/calendar:jwt-secret", "host/workspace_sandbox_userns", "recovery/vrooli/calendar:jwt-secret"}
	requireStrings(t, "blockers", blockerNames(assessment.Blockers), wantBlockers)
	wantDegraded := []string{"credential/vrooli/remote-desktop:username", "host/docker"}
	requireStrings(t, "degraded", blockerNames(assessment.Degraded), wantDegraded)
	for _, blocker := range append(append([]completionBlocker(nil), assessment.Blockers...), assessment.Degraded...) {
		if blocker.Remediation == "" {
			t.Fatalf("blocker %s carries no remediation", blocker.Name)
		}
	}
}

// A derived credential is provided by its owning component after its source is
// available, so it is neither a blocker nor a degraded gap.
func TestAssessCompletionIgnoresDerivedCredentials(t *testing.T) {
	readiness := readinessResponse{Credentials: []credentialReadiness{
		{LogicalID: "vrooli/derived", Field: "token", Required: true, Status: "unconfigured", Provisioning: "derived"},
	}}
	assessment := assessCompletion(readiness, nil)
	if len(assessment.Blockers) != 0 || len(assessment.Degraded) != 0 {
		t.Fatalf("derived credential produced %v / %v", assessment.Blockers, assessment.Degraded)
	}
}

func TestAssessCompletionReportsUnsuccessfulApplyItems(t *testing.T) {
	run := applyRun{Items: []applyItemResult{
		{applyItem: applyItem{Kind: "tool", Name: "docker"}, Outcome: "applied"},
		{applyItem: applyItem{Kind: "safeguard", Name: "kernel_config"}, Outcome: "needs_elevation", Remediation: "run `vrooli setup --sudo-mode=ask`"},
		{applyItem: applyItem{Kind: "scenario", Name: "alpha"}, Outcome: "already_satisfied"},
	}}
	assessment := assessCompletion(readinessResponse{}, &run)
	requireStrings(t, "apply blockers", blockerNames(assessment.Blockers), []string{"apply/safeguard:kernel_config"})
}

// [REQ:ONB-READY-DEGRADED-CONTINUE]
// The acknowledgement names the exact degraded set it accepted, so accepting
// one gap cannot authorise completion over a different one later.
func TestDegradedAcknowledgementIsScopedToTheSetItAcknowledged(t *testing.T) {
	first := assessCompletion(readinessResponse{Credentials: []credentialReadiness{
		credential("project", "vrooli/remote-desktop", "username", false, "unconfigured"),
	}}, nil)
	second := assessCompletion(readinessResponse{Credentials: []credentialReadiness{
		credential("project", "vrooli/remote-desktop", "username", false, "unconfigured"),
		credential("openai", "vrooli/openai", "api-key", false, "unconfigured"),
	}}, nil)
	if first.DegradedDigest == "" || second.DegradedDigest == "" {
		t.Fatal("a non-empty degraded set must produce a digest")
	}
	if first.DegradedDigest == second.DegradedDigest {
		t.Fatal("two different degraded sets produced the same digest")
	}
	acknowledged := OperatorState{Completion: &operatorCompletion{DegradedAcknowledgement: &operatorDegradedAcknowledgement{ReadinessDigest: first.DegradedDigest}}}
	if !configurationMayComplete(first, acknowledged) {
		t.Fatal("the acknowledged set must be allowed to complete")
	}
	if configurationMayComplete(second, acknowledged) {
		t.Fatal("an acknowledgement of one degraded set authorised completion over a different one")
	}
}

func TestConfigurationMayNotCompleteWithABlocker(t *testing.T) {
	assessment := assessCompletion(readinessResponse{Credentials: []credentialReadiness{
		credential("calendar", "vrooli/calendar", "jwt-secret", true, "unconfigured"),
	}}, nil)
	if configurationMayComplete(assessment, OperatorState{}) {
		t.Fatal("a required unresolved credential must withhold completion")
	}
}

func TestConfigurationMayCompleteWithNothingUnresolved(t *testing.T) {
	assessment := assessCompletion(readinessResponse{Credentials: []credentialReadiness{
		credential("openai", "vrooli/openai", "api-key", true, "configured"),
	}}, nil)
	if !configurationMayComplete(assessment, OperatorState{}) {
		t.Fatal("a clean readiness verdict must be allowed to complete")
	}
}

func TestConfigurationMayNotCompleteWithAnUnacknowledgedDegradedGap(t *testing.T) {
	assessment := assessCompletion(readinessResponse{Credentials: []credentialReadiness{
		credential("project", "vrooli/remote-desktop", "username", false, "unconfigured"),
	}}, nil)
	if configurationMayComplete(assessment, OperatorState{}) {
		t.Fatal("an unacknowledged optional gap must withhold completion")
	}
}

func requireStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", label, got, want)
		}
	}
}

// applyFixtureRoot builds a repository whose apply run has one resource and one
// scenario, plus a project manifest that declares the credential under test.
func applyFixtureRoot(t *testing.T, projectCredentials string) string {
	t.Helper()
	root := t.TempDir()
	// The configuration marker lives under the runtime home, so the fixture owns
	// its own home rather than writing into the operator's.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-25T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope"},
  "credentials": {"descriptors": [`+projectCredentials+`]}
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true}}`)
	if err := os.MkdirAll(filepath.Join(root, "resources"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The configuration marker is the second half of first run and refuses to be
	// written before the bootstrap marker, so the fixture has to represent a
	// host on which setup already finished its bootstrap.
	writeBootstrapMarker(t, root)
	stubExternalReadinessProbes(t)
	return root
}

// stubExternalReadinessProbes replaces the two readiness inputs that leave the
// process. The apply gate computes readiness before it decides whether the
// marker may be written, and letting each of these tests spend the credential
// doctor's walk plus the release-authority subprocess timeout would make the
// unit phase's duration a function of host state rather than of the code.
func stubExternalReadinessProbes(t *testing.T) {
	t.Helper()
	previousDoctor := credentialDoctorCommand
	previousRelease := releaseAuthorityStatusCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return []byte(`{"recovery":{"receipt_exists":true,"exported_at":"2026-08-25T00:00:00Z","entry_count":1,"uncovered":[],"required_absent":[]}}`), nil
	}
	releaseAuthorityStatusCommand = func(context.Context, string) ([]byte, error) {
		return []byte(`{"configured":true,"trust_anchor_match":true,"provider":"test"}`), nil
	}
	t.Cleanup(func() {
		credentialDoctorCommand = previousDoctor
		releaseAuthorityStatusCommand = previousRelease
	})
}

func projectStateLocator(t *testing.T, root string) projectstate.Locator {
	t.Helper()
	home, err := config.HomeDir()
	if err != nil {
		t.Fatal(err)
	}
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		t.Fatal(err)
	}
	return locator
}

func writeBootstrapMarker(t *testing.T, root string) {
	t.Helper()
	locator := projectStateLocator(t, root)
	writeFixtureFile(t, locator.BootstrapCompletePath(), `{"setup_version":"2.0.0","phase":"bootstrap_complete"}`)
}

func runApplyToTerminal(t *testing.T) applyRun {
	t.Helper()
	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })
	useInProcessApplyRunner(t)
	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
	}
	var accepted applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	return waitApplyTerminal(t, accepted.ID)
}

func configurationMarkerPresent(t *testing.T, root string) bool {
	t.Helper()
	return projectStateLocator(t, root).HasConfigurationComplete()
}

// [REQ:ONB-COMPLETE-REQUIRES-READY]
// Every apply item succeeding says the host changes were made. It says nothing
// about whether a required credential is present, and the marker is the flow's
// claim that it is.
func TestApplyWithheldMarkerWhenARequiredCredentialIsUnresolved(t *testing.T) {
	root := applyFixtureRoot(t, `{"logical_id": "vrooli/blocked-fixture", "field": "token", "label": "Blocked fixture token", "required": true}`)
	terminal := runApplyToTerminal(t)
	if terminal.Status != "configuration_incomplete" {
		t.Fatalf("status = %q, want configuration_incomplete: %+v", terminal.Status, terminal)
	}
	named := false
	for _, blocker := range terminal.Blockers {
		if blocker.Kind == "credential" && blocker.Name == "vrooli/blocked-fixture:token" {
			named = true
		}
	}
	if !named {
		t.Fatalf("blockers did not name the unresolved required credential: %v", terminal.Blockers)
	}
	if configurationMarkerPresent(t, root) {
		t.Fatal("the configuration-complete marker was written while a required credential was unresolved")
	}
}

// [REQ:ONB-READY-DEGRADED-CONTINUE]
// An optional gap withholds the marker until the operator accepts that exact
// set, and accepting it through the API releases the marker.
func TestApplyWithheldMarkerUntilTheDegradedSetIsAcknowledged(t *testing.T) {
	root := applyFixtureRoot(t, `{"logical_id": "vrooli/optional-fixture", "field": "token", "label": "Optional fixture token", "required": false}`)

	first := runApplyToTerminal(t)
	if first.Status != "configuration_incomplete" {
		t.Fatalf("status = %q, want configuration_incomplete: %+v", first.Status, first)
	}
	if first.DegradedDigest == "" {
		t.Fatalf("an optional gap produced no degraded digest: %+v", first)
	}
	if configurationMarkerPresent(t, root) {
		t.Fatal("the marker was written for an unacknowledged optional gap")
	}

	stale := doRequest(t, NewServer(), http.MethodPost, "/api/v2/readiness/degraded-acknowledgement", `{"readiness_digest":"0000"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale acknowledgement status = %d: %s", stale.Code, stale.Body.String())
	}

	accepted := doRequest(t, NewServer(), http.MethodPost, "/api/v2/readiness/degraded-acknowledgement", `{"readiness_digest":"`+first.DegradedDigest+`"}`)
	if accepted.Code != http.StatusCreated {
		t.Fatalf("acknowledgement status = %d: %s", accepted.Code, accepted.Body.String())
	}

	second := runApplyToTerminal(t)
	if second.Status != "applied" {
		t.Fatalf("status after acknowledgement = %q, want applied: %+v", second.Status, second)
	}
	if !configurationMarkerPresent(t, root) {
		t.Fatal("the marker was withheld after the degraded set was acknowledged")
	}
}
