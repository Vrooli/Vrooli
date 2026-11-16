package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/packages/capacity"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

// lockedFixture has express as its only direct dependency and body-parser only as
// a transitive lock entry, the shape of a scenario that inherits a vulnerable
// body-parser through express.
func lockedFixture(t *testing.T) (*connectHandler, *testDiskJobBroker, *recordingInstaller) {
	t.Helper()
	repoRoot := t.TempDir()
	ui := filepath.Join(repoRoot, "scenarios", "demo", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"package.json":   `{"name":"demo-ui","dependencies":{"express":"^4.22.1"}}`,
		"pnpm-lock.yaml": "lockfileVersion: '9.0'\n\npackages:\n\n  body-parser@1.20.5:\n    resolution: {integrity: sha512-x}\n\n  express@4.22.1:\n    resolution: {integrity: sha512-y}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(ui, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installer := &recordingInstaller{}
	h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: installer})
	// No evidence unless a test supplies it: an update never asks the real Security Health.
	h.advisoryEvidence = func(context.Context, *governancev1.ListSecurityGovernanceGapsRequest) ([]*governancev1.SecurityVulnerabilityEvidence, error) {
		return nil, nil
	}
	return h, h.diskBroker.(*testDiskJobBroker), installer
}

func updateRequest(apply bool, names ...string) *connect.Request[governancev1.UpdateLockedDependencyRequest] {
	return connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageNames: names, Apply: apply})
}

func TestUpdateLockedDependencyPreviewsATransitivePackageWithoutRunning(t *testing.T) {
	h, broker, installer := lockedFixture(t)

	resp, err := h.UpdateLockedDependency(context.Background(), updateRequest(false, "body-parser"))
	if err != nil {
		t.Fatal(err)
	}

	msg := resp.Msg
	if !msg.GetDryRun() || msg.GetBlocked() || msg.GetVerdict() != "updatable" || msg.GetUpdated() {
		t.Fatalf("dry run = %v", msg)
	}
	if !strings.Contains(msg.GetCommand(), "pnpm update body-parser") || !strings.Contains(msg.GetMessage(), "--apply") {
		t.Fatalf("a preview must show the exact command and how to apply: %v", msg)
	}
	if installer.called || len(broker.requests) != 0 {
		t.Fatal("a preview must not run the installer or reserve disk")
	}
}

func TestUpdateLockedDependencyApplyReservesDiskAndHandsTheInstallerTheUpdate(t *testing.T) {
	h, broker, installer := lockedFixture(t)

	resp, err := h.UpdateLockedDependency(context.Background(), updateRequest(true, "body-parser"))
	if err != nil {
		t.Fatal(err)
	}

	msg := resp.Msg
	if !msg.GetUpdated() || msg.GetBlocked() || msg.GetDryRun() || len(broker.requests) != 1 {
		t.Fatalf("apply = %v reservations=%d", msg, len(broker.requests))
	}
	if !installer.called || strings.Join(installer.plan.UpdatedPackages, ",") != "body-parser" || len(installer.plan.RemovedPackages) != 0 {
		t.Fatalf("installer was handed %+v", installer.plan)
	}
}

func TestUpdateLockedDependencyBlocksLikeInstallWhenAdmissionRefuses(t *testing.T) {
	h, broker, installer := lockedFixture(t)
	h.diskPlanner = func(context.Context, installgateway.Resolution) (installgateway.DiskAdmissionPlan, error) {
		return installgateway.DiskAdmissionPlan{Unsupported: "pnpm could not resolve the update of body-parser in a staging copy"}, nil
	}
	var verdicts [2]string
	for i, apply := range []bool{false, true} {
		resp, err := h.UpdateLockedDependency(context.Background(), updateRequest(apply, "body-parser"))
		if err != nil {
			t.Fatal(err)
		}
		msg := resp.Msg
		if !msg.GetBlocked() || msg.GetUpdated() || msg.GetVerdict() != "update_blocked" || len(msg.GetNextSteps()) == 0 {
			t.Fatalf("apply=%t response = %v", apply, msg)
		}
		verdicts[i] = msg.GetVerdict()
	}
	if verdicts[0] != verdicts[1] || installer.called || len(broker.requests) != 0 {
		t.Fatalf("dry run and apply must agree and nothing may run: %v", verdicts)
	}
}

func TestUpdateLockedDependencyRejectsAPackageThatIsNotLockedBeforeAnyPlanning(t *testing.T) {
	h, broker, installer := lockedFixture(t)

	_, err := h.UpdateLockedDependency(context.Background(), updateRequest(true, "body-parser", "left-pad"))

	if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "left-pad is not in the lockfile") {
		t.Fatalf("err = %v", err)
	}
	if installer.called || len(broker.requests) != 0 {
		t.Fatal("nothing may run for an invalid update")
	}
}

func TestUpdateLockedDependencyRefusesAVersionedName(t *testing.T) {
	h, _, _ := lockedFixture(t)

	_, err := h.UpdateLockedDependency(context.Background(), updateRequest(false, "body-parser@1.20.9"))

	if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "carries a version") {
		t.Fatalf("err = %v; an update takes the newest in-range version, not a pinned one", err)
	}
}

func TestHighlightedVersionNotesLeadWithTheNamedPackages(t *testing.T) {
	removed := []string{"body-parser@1.20.5", "proxy-addr@2.0.7", "qs@6.15.1"}
	added := []string{"@emnapi/core@1.11.3", "body-parser@1.20.9", "proxy-addr@2.0.8", "zod@3.0.0"}

	got := strings.Join(highlightedVersionNotes([]string{"body-parser", "proxy-addr", "left-pad", "zod"}, removed, added, nil), "|")

	want := "body-parser: 1.20.5 -> 1.20.9|proxy-addr: 2.0.7 -> 2.0.8|left-pad: locked version unchanged|zod: now also locked at 3.0.0"
	if got != want {
		t.Fatalf("notes = %q, want %q", got, want)
	}
	if note := highlightedVersionNotes([]string{"left-pad", "never"}, nil, nil, map[string][]string{"left-pad": {"1.3.0", "2.0.0"}}); len(note) != 2 ||
		note[0] != "left-pad: locked at 1.3.0, 2.0.0 (unchanged)" || note[1] != "never: locked version unchanged" {
		t.Fatalf("an unchanged package must report the version it is locked at: %v", note)
	}
	if note := highlightedVersionNotes([]string{"qs"}, removed, nil, nil); len(note) != 1 || note[0] != "qs: 6.15.1 no longer locked" {
		t.Fatalf("notes = %v", note)
	}
}

// withEvidence gives the handler the vulnerability evidence Security Health would
// report, and records the query so a test can see what was asked.
func withEvidence(h *connectHandler, asked *[]*governancev1.ListSecurityGovernanceGapsRequest, evidence ...*governancev1.SecurityVulnerabilityEvidence) {
	h.advisoryEvidence = func(_ context.Context, req *governancev1.ListSecurityGovernanceGapsRequest) ([]*governancev1.SecurityVulnerabilityEvidence, error) {
		*asked = append(*asked, req)
		return evidence, nil
	}
}

func exposure(pkg, id, scenario string, files ...string) *governancev1.SecurityVulnerabilityEvidence {
	return &governancev1.SecurityVulnerabilityEvidence{
		VulnerabilityId: id, Ecosystem: "npm", PackageName: pkg, Scenarios: []string{scenario}, SourceFiles: files,
	}
}

func TestUpdateByAdvisoryUpdatesThePackagesTheAdvisoryNamesOnThatSurface(t *testing.T) {
	h, _, installer := lockedFixture(t)
	var asked []*governancev1.ListSecurityGovernanceGapsRequest
	alias := exposure("body-parser", "CVE-2026-1", "demo", "scenarios/demo/ui/pnpm-lock.yaml") // a repository-relative path
	alias.Aliases = []string{"GHSA-27r3-jxmh-p828"}
	withEvidence(h, &asked,
		alias,
		exposure("body-parser", "CVE-2026-1", "demo", "ui/pnpm-lock.yaml"),                            // the same package twice is one name
		exposure("express", "GHSA-27r3-jxmh-p828", "demo", "api/pnpm-lock.yaml"),                      // another surface
		exposure("left-pad", "GHSA-27r3-jxmh-p828", "other", "ui/pnpm-lock.yaml"),                     // another scenario
		exposure("qs", "GHSA-other", "demo", "ui/pnpm-lock.yaml"),                                     // another advisory
		exposure("cookie", "GHSA-27r3-jxmh-p828", "demo", "docs/internal/evidence/ui/pnpm-lock.yaml"), // a retained copy, never the surface (cookie is not even locked)
	)

	resp, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: "demo", Surface: "ui", Advisory: "GHSA-27r3-jxmh-p828", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if !resp.Msg.GetUpdated() || strings.Join(installer.plan.UpdatedPackages, ",") != "body-parser" {
		t.Fatalf("an advisory must select exactly its exposed packages on this surface: %v / %+v", resp.Msg, installer.plan)
	}
	if len(resp.Msg.GetNotes()) == 0 || !strings.Contains(resp.Msg.GetNotes()[0], "advisory GHSA-27r3-jxmh-p828 selects body-parser") {
		t.Fatalf("the response must say what the advisory selected: %v", resp.Msg.GetNotes())
	}
	if len(asked) != 1 || asked[0].GetScenario() != "demo" || asked[0].GetEcosystem() != "npm" {
		t.Fatalf("one query, scoped to the scenario's npm evidence: %v", asked)
	}
}

func TestUpdateByAdvisoryWithNoExposureChangesNothing(t *testing.T) {
	h, broker, installer := lockedFixture(t)
	var asked []*governancev1.ListSecurityGovernanceGapsRequest
	withEvidence(h, &asked)

	resp, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: "demo", Surface: "ui", Advisory: "GHSA-gone", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if resp.Msg.GetVerdict() != "not_affected" || resp.Msg.GetBlocked() || resp.Msg.GetUpdated() || installer.called || len(broker.requests) != 0 {
		t.Fatalf("no exposure is a clean no-op: %v", resp.Msg)
	}
}

func TestUpdateByAdvisoryRefusesNamesAndAnUnscopedRequest(t *testing.T) {
	h, _, installer := lockedFixture(t)
	for name, req := range map[string]*governancev1.UpdateLockedDependencyRequest{
		"names and an advisory": {Scenario: "demo", Surface: "ui", Advisory: "GHSA-x", PackageNames: []string{"body-parser"}},
		"no surface":            {Scenario: "demo", Advisory: "GHSA-x"},
	} {
		if _, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if installer.called {
		t.Fatal("nothing may run for an invalid selection")
	}
}

func TestUpdateAcceptsEcosystemQualifiedNames(t *testing.T) {
	h, _, _ := lockedFixture(t)

	resp, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: "demo", Surface: "ui", PackageNames: []string{"npm/body-parser"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Msg.GetCommand(), "pnpm update body-parser") {
		t.Fatalf("npm/body-parser must update body-parser, got %q", resp.Msg.GetCommand())
	}
	if eco, names := lockedPackageSelection("", []string{"npm/@types/node", "@types/react", "left-pad"}); eco != "npm" || strings.Join(names, ",") != "@types/node,@types/react,left-pad" {
		t.Fatalf("selection = %q %v", eco, names)
	}
}

func TestUpdateByAdvisoriesTakesTheUnionInOneResolution(t *testing.T) {
	h, _, installer := lockedFixture(t)
	var asked []*governancev1.ListSecurityGovernanceGapsRequest
	withEvidence(h, &asked,
		exposure("body-parser", "GHSA-a", "demo", "ui/pnpm-lock.yaml"),
		exposure("express", "GHSA-b", "demo", "ui/pnpm-lock.yaml"),
	)

	resp, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: "demo", Surface: "ui", Advisory: "GHSA-a, ghsa-b,GHSA-A", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if !resp.Msg.GetUpdated() || strings.Join(installer.plan.UpdatedPackages, ",") != "body-parser,express" || installer.plan.Argv[len(installer.plan.Argv)-1] != "express" {
		t.Fatalf("several advisories are one update of the union: %v / %+v", resp.Msg, installer.plan.Argv)
	}
	if len(asked) != 1 {
		t.Fatalf("the evidence is read once however many advisories are named, got %d queries", len(asked))
	}
	if got := advisoryList(" GHSA-a,,GHSA-b  GHSA-a "); strings.Join(got, "|") != "GHSA-a|GHSA-b" {
		t.Fatalf("advisoryList = %v", got)
	}
}

func TestUpdateByAllAdvisoriesSelectsEveryExposedPackageOnTheSurface(t *testing.T) {
	h, _, installer := lockedFixture(t)
	var asked []*governancev1.ListSecurityGovernanceGapsRequest
	withEvidence(h, &asked,
		exposure("body-parser", "GHSA-a", "demo", "ui/pnpm-lock.yaml"),
		exposure("express", "GHSA-b", "demo", "scenarios/demo/ui/pnpm-lock.yaml"),
		exposure("cookie", "GHSA-c", "demo", "api/pnpm-lock.yaml"),                  // another surface
		exposure("qs", "GHSA-d", "other", "ui/pnpm-lock.yaml"),                      // another scenario
		exposure("send", "GHSA-e", "demo", "docs/internal/evidence/pnpm-lock.yaml"), // a retained copy
	)

	resp, err := h.UpdateLockedDependency(context.Background(), connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: "demo", Surface: "ui", Advisory: "all", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if !resp.Msg.GetUpdated() || strings.Join(installer.plan.UpdatedPackages, ",") != "body-parser,express" {
		t.Fatalf("all selects every package Security Health reports on this surface: %v / %+v", resp.Msg, installer.plan)
	}
	if !strings.Contains(resp.Msg.GetNotes()[0], "advisory all selects body-parser, express") {
		t.Fatalf("notes = %v", resp.Msg.GetNotes())
	}
}

func fix(pkg, id, introduced string, fixed ...string) *governancev1.SecurityVulnerabilityEvidence {
	item := exposure(pkg, id, "demo", "ui/pnpm-lock.yaml")
	if introduced != "" {
		item.AffectedRanges = []*governancev1.SecurityVersionRange{{Range: ">=" + introduced, Introduced: introduced}}
	}
	for _, f := range fixed {
		item.FixedRanges = append(item.FixedRanges, &governancev1.SecurityVersionRange{Version: f})
	}
	return item
}

func TestExposureNotesNameWhatAnUpdateLeftInsideAnAdvisory(t *testing.T) {
	fixes := advisoryFixes([]*governancev1.SecurityVulnerabilityEvidence{
		fix("body-parser", "GHSA-bp", "", "1.20.9", "2.4.0"),
		fix("vite", "GHSA-v1", "", "6.4.4", "7.3.7", "8.3.3"),
		fix("vite", "GHSA-v2", "", "6.4.4", "8.0.0-rc.1"), // a prerelease is never a fix to name
		fix("vite", "GHSA-vnew", "6.2.0", "6.4.4"),        // introduced later than the locked 5.x
		fix("vite", "GHSA-v3", "", "6.4.5"),
		fix("vite", "GHSA-v4", "", "6.4.6"),
		fix("katex", "GHSA-k", "0.11.0", "0.18.2"),
		fix("nofix", "GHSA-n", "1.0.0"),
		exposure("blank", "GHSA-blank", "demo", "ui/pnpm-lock.yaml"), // no range at all: never judged
	})
	locked := map[string][]string{
		"body-parser": {"1.20.9"},                   // meets the fix of its own line
		"vite":        {"5.4.21", "6.4.3", "9.0.0"}, // 5.x has no fix line, 6.4.3 is below the 6.x fixes, 9.x is above every fix line
		"katex":       {"0.16.47"},                  // the fix is outside the 0.16 range
		"nofix":       {"1.0.0"},
		"blank":       {"1.0.0"},
		"clean":       {"1.0.0"},
	}

	got := exposureNotes(locked, fixes)

	if len(got) != 4 {
		t.Fatalf("exposure notes: %v", got)
	}
	for i, prefix := range []string{
		"exposed: katex@0.16.47 to GHSA-k (no fix on this line; fixed in 0.18.2)",
		"exposed: nofix@1.0.0 to GHSA-n (no fixed version published)",
		"exposed: vite@5.4.21 to GHSA-v1, GHSA-v2, GHSA-v3 +1 more (no fix on this line; fixed in 6.4.4, 6.4.5, 6.4.6, 7.3.7, 8.3.3)",
		"exposed: vite@6.4.3 to GHSA-v1, GHSA-v2, GHSA-v3 +2 more (fix on this line: 6.4.6)",
	} {
		if got[i] != prefix {
			t.Errorf("note %d = %q\nwant   %q", i, got[i], prefix)
		}
	}
}

func TestAdmissionIsAMetadataJobOnlyWhenApplyWritesInputsAlone(t *testing.T) {
	for _, test := range []struct {
		name       string
		inputsOnly bool // the planner says apply downloads nothing and writes no package store
		peak       int64
		wantClass  string
		wantFloor  int64
	}{
		{"a lock-only or link-skipped JavaScript apply", true, 4 << 20, capacity.DiskJobClassMetadata, 0},
		{"a Go install whose modules are cached or first-party", true, 20 << 20, capacity.DiskJobClassMetadata, 0},
		{"an apply that links node_modules", false, 300 << 20, "", installgateway.DiskJobFloorBytes},
		{"a small apply that still fetches", false, 4 << 20, "", installgateway.DiskJobFloorBytes},
		{"an inputs-only plan whose size is somehow large", true, capacity.DiskJobMetadataMaxPeakBytes + 1, "", installgateway.DiskJobFloorBytes},
	} {
		t.Run(test.name, func(t *testing.T) {
			broker := &testDiskJobBroker{}
			planner := func(context.Context, installgateway.Resolution) (installgateway.DiskAdmissionPlan, error) {
				return installgateway.DiskAdmissionPlan{
					Paths:   []installgateway.DiskPathRequirement{{Path: os.TempDir(), Role: "test-target", PeakBytes: test.peak}},
					Manager: "test", WritesInputsOnly: test.inputsOnly,
				}, nil
			}
			plan, err := planInstallDisk(context.Background(), broker, planner, installgateway.Resolution{})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Release()
			if _, err := plan.reserve(context.Background(), broker); err != nil {
				t.Fatal(err)
			}
			if got := broker.requests[0]; got.Class != test.wantClass || got.FloorBytes != test.wantFloor {
				t.Fatalf("reserved class %q floor %d, want %q and %d", got.Class, got.FloorBytes, test.wantClass, test.wantFloor)
			}
			if metadata := strings.Contains(plan.summary(), "metadata job"); metadata != (test.wantClass != "") {
				t.Fatalf("the preview must say a metadata job exactly when one will be reserved: %s", plan.summary())
			}
		})
	}
}
