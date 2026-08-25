package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
)

func hostNames(items []hostItem) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	return names
}

func resolvedNames(requirements []hostreq.ResolvedRequirement) []string {
	names := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		names = append(names, requirement.Name)
	}
	sort.Strings(names)
	return names
}

func requireSameNames(t *testing.T, label string, got, want []string) {
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

// [REQ:ONB-HOST-SCOPE-PARITY]
// Two implementations of host-requirement resolution stay by decision: hostreq
// needs a repository layout, and onboarding must also serve a flat bundle
// catalog. This test is what makes the duplication safe. It converts a silent
// fork into a tested equivalence, so a change to either side that drops or adds
// a host item fails here instead of quietly showing the operator a consent
// screen for a fraction of the host changes setup performs.
func TestOnboardingHostDerivationMatchesHostreq(t *testing.T) {
	root, storageRoot := writeProjectScopeFixture(t)
	home := t.TempDir()

	state, err := loadOperatorState()
	if err != nil {
		t.Fatal(err)
	}
	models, err := selectedScenarioModels()
	if err != nil {
		t.Fatal(err)
	}
	derived, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := hostreq.Resolve(root, home, hostreq.ResolveOptions{
		When:      "setup",
		Resources: "none",
		Scenarios: "alpha",
		Platform:  "linux",
	})
	if err != nil {
		t.Fatal(err)
	}

	requireSameNames(t, "tool names", hostNames(derived.Tools), resolvedNames(resolution.Tools))
	requireSameNames(t, "safeguard names", hostNames(derived.Safeguards), resolvedNames(resolution.Safeguards))
	// Guard against a vacuous pass: the project-declared items must actually be
	// in the compared sets.
	requireSameNames(t, "tool names", hostNames(derived.Tools), []string{"jq", "tmux"})
	requireSameNames(t, "safeguard names", hostNames(derived.Safeguards), []string{"workspace_sandbox_userns"})
	_ = storageRoot
}

// The negative half of the same equivalence: a bundle inherits no project-scope
// host item, and hostreq says the same thing through ExcludeRoot.
func TestBundleHostDerivationMatchesHostreqExcludeRoot(t *testing.T) {
	root, _ := writeProjectScopeFixture(t)
	home := t.TempDir()

	resolution, err := hostreq.Resolve(root, home, hostreq.ResolveOptions{
		ExcludeRoot: true,
		When:        "setup",
		Resources:   "none",
		Scenarios:   "alpha",
		Platform:    "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	requireSameNames(t, "excluded-root tool names", resolvedNames(resolution.Tools), []string{"tmux"})
	if len(resolution.Safeguards) != 0 {
		t.Fatalf("excluded-root safeguards = %v, want none", resolvedNames(resolution.Safeguards))
	}
}

// [REQ:ONB-TIER-BUNDLE-COMPLETENESS]
func TestBundleModeExcludesProjectScopeHostItems(t *testing.T) {
	bundle, storageRoot := writeBundleFixture(t, true)
	writeFixtureFile(t, filepath.Join(bundle, "catalog", ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope that must not be inherited"},
  "hostSafeguards": [{"name": "workspace_sandbox_userns", "required": true, "reason": "must not be inherited"}]
}`)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)

	w := doGet(t, NewServer(), "/api/v2/host-requirements")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var response hostRequirementsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, item := range append(append([]hostItem(nil), response.Tools...), response.Safeguards...) {
		if item.Name == "workspace_sandbox_userns" {
			t.Fatalf("bundle mode inherited a project-scope host item: %s", w.Body.String())
		}
	}
}

// [REQ:ONB-HOST-PROBE-TRUTH]
// A user-scoped safeguard declares its verification path as $USER_HOME/...,
// which is one declaration for every account and platform. Reading it
// literally reported every such safeguard as unapplied. That is a false
// negative on a required item, and a completion gate built on it would block a
// host that is in fact correctly configured.
func TestSafeguardVerificationPathExpandsPortableTokens(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	applied := filepath.Join(home, ".config", "systemd", "user", "probe.service")
	writeFixtureFile(t, applied, "[Unit]\n")

	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "probe_applied", "safeguard.json"),
		`{"name":"probe_applied","description":"Applied","verificationCheck":{"files":["$USER_HOME/.config/systemd/user/probe.service"]}}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "probe_absent", "safeguard.json"),
		`{"name":"probe_absent","description":"Absent","verificationCheck":{"files":["$USER_HOME/.config/systemd/user/never.service"]}}`)

	appliedItem := hostItem{hostRequirement: hostRequirement{Name: "probe_applied", Required: true}, Status: "required"}
	if got := inspectSafeguardReadiness(root, appliedItem).Status; got != "ready" {
		t.Fatalf("applied safeguard status = %q, want ready", got)
	}
	absentItem := hostItem{hostRequirement: hostRequirement{Name: "probe_absent", Required: true}, Status: "required"}
	if got := inspectSafeguardReadiness(root, absentItem).Status; got != "missing" {
		t.Fatalf("absent safeguard status = %q, want missing", got)
	}
}

// [REQ:ONB-HOST-PROBE-TRUTH]
// A handler-owned safeguard has no file to stat, but that never made it
// uncheckable: the control plane owns a read-only inspection half that is
// separate from Apply by interface. This asserts onboarding asks it rather than
// short-circuiting to "state is reported during apply", which told the operator
// a safeguard was unknowable while the answer was one unprivileged call away.
func TestHandlerOwnedSafeguardIsCheckedThroughTheControlPlane(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "handler_owned", "safeguard.json"),
		`{"name":"handler_owned","description":"Handler owned","handler":"internal/safeguards/handler-owned","privilege":"elevated"}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "no_probe", "safeguard.json"),
		`{"name":"no_probe","description":"Neither probe nor handler"}`)

	// This fixture is declared under the test root but is not a member of the
	// control plane's safeguard catalog, so no unprivileged read can reach it.
	// That still may not be reported as a host fault, and it may not be
	// reported with the old canned excuse either: the operator is entitled to
	// know that the check did not run and why.
	handlerItem := hostItem{hostRequirement: hostRequirement{Name: "handler_owned", Required: true}, Status: "required"}
	got := inspectSafeguardReadiness(root, handlerItem)
	if got.Status != "deferred" {
		t.Fatalf("uncatalogued safeguard status = %q, want deferred", got.Status)
	}
	if !strings.Contains(got.Detail, "could not sample") || !strings.Contains(got.Detail, "handler_owned") {
		t.Fatalf("a skipped check must say what failed, got %q", got.Detail)
	}

	unverifiable := hostItem{hostRequirement: hostRequirement{Name: "no_probe", Required: true}, Status: "required"}
	if status := inspectSafeguardReadiness(root, unverifiable).Status; status != "unsupported" {
		t.Fatalf("safeguard with neither probe nor handler status = %q, want unsupported", status)
	}
}

// [REQ:ONB-HOST-PROBE-TRUTH]
// Every handler-owned safeguard in the real catalog must reach a verdict from
// an unprivileged read. "deferred" is reserved for a probe that could not run;
// if one appears here it means onboarding is again reporting a knowable host as
// unknowable, which is the exact defect this path was built to remove.
func TestCatalogHandlerOwnedSafeguardsReachAVerdict(t *testing.T) {
	root := repoRootForTest(t)
	decided := map[string]bool{"ready": true, "missing": true, "degraded": true, "unsupported": true}

	for _, name := range []string{"tpm_credential_access", "remote_desktop_access"} {
		item := hostItem{hostRequirement: hostRequirement{Name: name, Required: true}, Status: "required"}
		got := inspectSafeguardReadiness(root, item)
		if !decided[got.Status] {
			t.Fatalf("%s: status = %q, want a decided verdict; detail = %q", name, got.Status, got.Detail)
		}
		if strings.TrimSpace(got.Detail) == "" {
			t.Fatalf("%s: a decided verdict must carry the handler's reason, got an empty detail", name)
		}
	}
}

// repoRootForTest locates the repository root from the api package directory.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "internal", "safeguards")); statErr != nil {
		t.Fatalf("repository root %q does not contain internal/safeguards: %v", root, statErr)
	}
	return root
}
