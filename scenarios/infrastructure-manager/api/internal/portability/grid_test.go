package portability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/packages/hostreq"
)

// repoRoot walks up from the test's working directory to the repository root,
// identified by the capability vocabulary the grid is computed against. It is
// resolved rather than hard-coded so the tests keep working if the scenario
// moves in the tree.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli", "capability-vocabulary.json")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no repository root with .vrooli/capability-vocabulary.json was found above the test directory")
		}
		dir = parent
	}
}

func TestAttachObservedQualificationsSeparatesDeclarationFromHostEvidence(t *testing.T) {
	grid := Grid{Capabilities: []Entry{{
		Capability: "credential-storage",
		Platforms: []PlatformEntry{
			{
				HostOS: deployability.HostOSLinux, Status: deployability.CapabilityImplemented, Qualification: deployability.QualificationQualified,
				Declarers: []deployability.CapabilityDeclarer{{Name: "login_keyring_unlock", Role: "control", Resolved: true}},
			},
			{
				HostOS: deployability.HostOSWindows, Status: deployability.CapabilityImplemented, Qualification: deployability.QualificationQualified,
				Declarers: []deployability.CapabilityDeclarer{{Name: "login_keyring_unlock", Role: "control"}},
			},
		},
	}}}
	observed := []hostreq.ObservedSafeguard{{Name: "login_keyring_unlock", ExecutionState: "pending", ObservedAt: time.Now().UTC()}}
	grid = AttachObservedQualifications(grid, observed, deployability.HostOSLinux)
	local := grid.Capabilities[0].Platforms[0]
	if local.Status != deployability.CapabilityImplemented || local.Qualification != deployability.QualificationQualified {
		t.Fatalf("host evidence changed declaration resolution: %+v", local)
	}
	if local.ObservedQualification != deployability.QualificationUnqualified || len(local.ObservedDeclarers) != 1 {
		t.Fatalf("pending safeguard was not qualified as unresolved evidence: %+v", local)
	}
	remote := grid.Capabilities[0].Platforms[1]
	if !strings.HasPrefix(remote.ObservedQualificationReason, "host_not_sampled") {
		t.Fatalf("remote platform did not retain explicit unread reason: %+v", remote)
	}
}

func liveReader(t *testing.T) *Reader {
	t.Helper()
	reader, err := NewReader(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func liveGrid(t *testing.T) Grid {
	t.Helper()
	grid, err := liveReader(t).Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return grid
}

func TestGridCoversEveryVocabularyCapabilityOnEveryHostOS(t *testing.T) {
	reader := liveReader(t)
	vocabulary, err := reader.Vocabulary()
	if err != nil {
		t.Fatal(err)
	}
	grid, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(grid.Capabilities) == 0 {
		t.Fatal("grid is empty")
	}
	if len(grid.Capabilities) != len(vocabulary.Capabilities) {
		t.Fatalf("grid has %d rows for %d vocabulary capabilities", len(grid.Capabilities), len(vocabulary.Capabilities))
	}
	for _, entry := range grid.Capabilities {
		if len(entry.Platforms) != len(operatingSystems)*2 {
			t.Fatalf("%s has %d platform entries, want %d", entry.Capability, len(entry.Platforms), len(operatingSystems)*2)
		}
		for _, hostOS := range operatingSystems {
			for _, architecture := range []string{"amd64", "arm64"} {
				if _, ok := entry.PlatformFor(hostOS, architecture); !ok {
					t.Fatalf("%s has no entry for %s/%s", entry.Capability, hostOS, architecture)
				}
			}
		}
	}
	if grid.ManifestRoot != repoRoot(t) {
		t.Fatalf("grid reports manifest root %q, want %q", grid.ManifestRoot, repoRoot(t))
	}
	if grid.ManifestsRead == 0 {
		t.Fatal("grid reports zero manifests read; the readout would be vacuous")
	}
	if grid.ComputedAt.IsZero() {
		t.Fatal("grid carries no computed_at")
	}
}

func TestGridClassifiesEveryCapabilitySituation(t *testing.T) {
	grid := liveGrid(t)
	seen := map[CapabilitySituation]bool{}
	for _, entry := range grid.Capabilities {
		seen[entry.Situation] = true
	}
	for _, situation := range Situations() {
		// Controls-only capabilities now resolve when every control is present;
		// a live repository with no incomplete control set need not manufacture a
		// controls_unported row. The same applies to scoped-out rows after the
		// Windows peers and explicit platform policies are wired.
		if situation == SituationControlsUnported || situation == SituationScopedOut || situation == SituationBuiltEverywhere {
			continue
		}
		if !seen[situation] {
			t.Errorf("grid did not classify any capability as %s", situation)
		}
	}
}

func TestWindowsDevelopmentTargetResolvesPeersAndControlPolicies(t *testing.T) {
	grid := liveGrid(t)
	for _, capability := range []string{"test-execution", "terminal-multiplexing"} {
		entry, ok := grid.Capability(capability)
		if !ok {
			t.Fatalf("missing capability %q", capability)
		}
		platform, ok := entry.PlatformFor(deployability.HostOSWindows, "amd64")
		if !ok {
			t.Fatalf("missing Windows/amd64 cell for %q", capability)
		}
		if !platform.HasImplementation || platform.Implementer == "" || platform.Mechanism == "" {
			t.Fatalf("%s Windows peer is not named and wired: %+v", capability, platform)
		}
	}
	for _, capability := range []string{"container-runtime", "credential-storage", "developer-utility", "gpu-driver-health", "service-resource-limits"} {
		entry, ok := grid.Capability(capability)
		if !ok {
			t.Fatalf("missing capability %q", capability)
		}
		platform, ok := entry.PlatformFor(deployability.HostOSWindows, "amd64")
		if !ok {
			t.Fatalf("missing Windows/amd64 cell for %q", capability)
		}
		if platform.Status == deployability.CapabilityControlsIncomplete || platform.Policy == "" {
			t.Fatalf("%s Windows control policy did not close the incomplete cell: %+v", capability, platform)
		}
	}
}

func TestEveryResolutionStatusHasASituation(t *testing.T) {
	statuses := []deployability.CapabilityResolutionStatus{
		deployability.CapabilityImplemented,
		deployability.CapabilityDegraded,
		deployability.CapabilityIneligible,
		deployability.CapabilityUnwired,
		deployability.CapabilityPeerless,
		deployability.CapabilityStatusInvalid,
		deployability.CapabilityControlsIncomplete,
	}
	for _, status := range statuses {
		if _, ok := situationByStatus[status]; !ok {
			t.Errorf("resolution status %q has no situation mapping", status)
		}
	}
	if _, _, err := classifySituation("injected", map[deployability.HostOS]deployability.CapabilityResolutionStatus{
		deployability.HostOSLinux: "injected_status",
	}, nil); err == nil {
		t.Fatal("unmapped resolution status was silently classified")
	}
	if _, _, err := classifySituation("empty", nil, nil); err == nil {
		t.Fatal("empty resolution status set was silently classified")
	}
}

func TestClassifySituationRequiresPolicyForNoEquivalent(t *testing.T) {
	situation, _, err := classifySituation("capability", map[deployability.HostOS]deployability.CapabilityResolutionStatus{
		deployability.HostOSLinux: deployability.CapabilityPeerless,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if situation != SituationControlsUnported {
		t.Fatalf("unasserted peerless capability classified as %s, want %s", situation, SituationControlsUnported)
	}
	situation, _, err = classifySituation("capability", map[deployability.HostOS]deployability.CapabilityResolutionStatus{
		deployability.HostOSLinux: deployability.CapabilityPeerless,
	}, map[string]map[string]string{"capability": {"linux": string(SituationNoEquivalentEver)}})
	if err != nil {
		t.Fatal(err)
	}
	if situation != SituationNoEquivalentEver {
		t.Fatalf("asserted no-equivalent policy classified as %s", situation)
	}
}

func TestGridNeverEmitsAnInvalidPlatformStatus(t *testing.T) {
	grid := liveGrid(t)
	for _, entry := range grid.Capabilities {
		for _, platform := range entry.Platforms {
			if platform.Status == deployability.CapabilityStatusInvalid {
				t.Errorf("%s/%s resolved to an invalid platform status: %s", entry.Capability, platform.HostOS, platform.Reason)
			}
			if platform.Qualification == "" {
				t.Errorf("%s/%s carries no honesty qualification", entry.Capability, platform.HostOS)
			}
			if platform.QualificationReason == "" {
				t.Errorf("%s/%s carries no qualification reason", entry.Capability, platform.HostOS)
			}
			if platform.HasImplementation != platform.Status.HasImplementation() {
				t.Errorf("%s/%s reports has_implementation=%v for status %s", entry.Capability, platform.HostOS, platform.HasImplementation, platform.Status)
			}
		}
	}
}

func TestGridRejectsUnprovenNonLinuxQualifiedClaims(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"probe"}, nil)
	writeToolFixtureWithoutEvidence(t, root, "probe-tool", "probe", map[string]string{"macos": string(deployability.StatusSupported)})
	reader, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Grid(time.Now()); err == nil || !strings.Contains(err.Error(), "macos") {
		t.Fatalf("unproven qualified macOS claim was accepted: %v", err)
	}
}

func TestGridDecaysAQualifiedCellWhenLatestHardwareEvidenceFails(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"probe"}, nil)
	writeToolFixture(t, root, "probe-tool", "probe", map[string]string{
		"linux":   string(deployability.StatusBuildVerified),
		"macos":   string(deployability.StatusSupported),
		"windows": string(deployability.StatusBuildVerified),
	})
	evidenceDir := filepath.Join(root, ".vrooli", "evidence", "native-platform")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, architecture := range []string{"amd64", "arm64"} {
		path := filepath.Join(evidenceDir, "probe-"+architecture+".json")
		data := []byte(`{"schema_version":1,"kind":"hardware-persistence","host_os":"darwin","architecture":"` + architecture + `","generated_at":"2026-08-25T15:00:00Z","passed":false,"source":"bridge-scheduled","run_id":"failed-run","host":"minimouse","surface":"lifecycle","artifact_uri":"artifact://failed-run","capabilities":["probe"]}`)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := grid.Capability("probe")
	if !ok {
		t.Fatal("probe capability missing")
	}
	for _, architecture := range []string{"amd64", "arm64"} {
		cell, ok := entry.PlatformFor(deployability.HostOSMacOS, architecture)
		if !ok {
			t.Fatalf("macOS/%s cell missing", architecture)
		}
		if cell.Qualification != deployability.QualificationBuildVerified {
			t.Fatalf("macOS/%s qualification = %s, want build_verified: %+v", architecture, cell.Qualification, cell)
		}
		if cell.Evidence != nil || !strings.Contains(cell.Reason, "failed-run") {
			t.Fatalf("macOS/%s did not expose decay reason and clear stale evidence: %+v", architecture, cell)
		}
	}
}

func TestHardwareEvidenceIsCapabilityScoped(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"probe", "other"}, nil)
	writeToolFixture(t, root, "probe-tool", "probe", map[string]string{
		"linux": string(deployability.StatusBuildVerified), "macos": string(deployability.StatusSupported), "windows": string(deployability.StatusBuildVerified),
	})
	evidenceDir := filepath.Join(root, ".vrooli", "evidence", "native-platform")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"schema_version":1,"kind":"hardware-persistence","host_os":"darwin","architecture":"amd64","generated_at":"2026-08-25T15:00:00Z","passed":true,"source":"bridge-scheduled","run_id":"other-run","host":"minimouse","surface":"lifecycle","artifact_uri":"artifact://other-run","capabilities":["other"]}`)
	if err := os.WriteFile(filepath.Join(evidenceDir, "other.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Grid(time.Now()); err == nil || !strings.Contains(err.Error(), "macos") {
		t.Fatal("a qualified capability without matching native evidence was accepted")
	}
}

func TestGridExposesBothArchitectureCells(t *testing.T) {
	grid := liveGrid(t)
	entry, ok := grid.Capability("source-control")
	if !ok {
		t.Fatal("source-control capability missing")
	}
	for _, hostOS := range operatingSystems {
		for _, architecture := range []string{"amd64", "arm64"} {
			if _, ok := entry.PlatformFor(hostOS, architecture); !ok {
				t.Fatalf("missing architecture cell %s/%s", hostOS, architecture)
			}
		}
	}
	if len(grid.Resources) == 0 {
		t.Fatal("resource profile architecture declarations were not read into the grid")
	}
	for _, claim := range grid.Resources {
		if claim.Mismatch {
			t.Fatalf("live resource architecture claim is contradictory: %+v", claim)
		}
	}
}

func TestGridAppliesArchitectureSpecificNoEquivalentPolicy(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"hardware-error-telemetry"}, map[string]map[string]string{
		"hardware-error-telemetry": {"linux/arm64": string(SituationNoEquivalentEver)},
	})
	writeToolFixture(t, root, "probe-tool", "hardware-error-telemetry", map[string]string{"linux": string(deployability.StatusBuildVerified)})
	grid, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	readout, err := grid.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	platform, ok := readout.Capabilities[0].PlatformFor(deployability.HostOSLinux, "arm64")
	if !ok {
		t.Fatal("linux/arm64 cell missing")
	}
	if platform.Status != deployability.CapabilityIneligible || platform.Policy != string(SituationNoEquivalentEver) {
		t.Fatalf("linux/arm64 policy was ignored: %+v", platform)
	}
}

// TestBuildVerifiedDeclarationsAreNotReportedUnwired drives the grid over the
// real repository manifests. system-monitor declares build-verified macOS and
// Windows collectors backed by real darwin and windows code; the grid must not
// call that "no implementation is declared".
func TestBuildVerifiedDeclarationsAreNotReportedUnwired(t *testing.T) {
	grid := liveGrid(t)
	buildVerified := 0
	for _, entry := range grid.Capabilities {
		for _, platform := range entry.Platforms {
			if platform.Qualification != deployability.QualificationBuildVerified {
				continue
			}
			buildVerified++
			if platform.Status == deployability.CapabilityUnwired {
				t.Errorf("%s/%s is build-verified but reported unwired", entry.Capability, platform.HostOS)
			}
			if !platform.Status.HasImplementation() {
				t.Errorf("%s/%s is build-verified but reports no implementation: %+v", entry.Capability, platform.HostOS, platform)
			}
			if platform.Implementer == "" {
				t.Errorf("%s/%s is build-verified but names no implementer", entry.Capability, platform.HostOS)
			}
		}
	}
	if buildVerified == 0 {
		t.Fatal("no build-verified declaration resolved; the repository census says there are twelve")
	}
}

// TestRepoManifestPlatformStatusContract is the repository gate: a tool.json,
// safeguard.json or service.json authoring a platform status outside the
// vocabulary fails this scenario's standard test run without anyone invoking
// a CLI verb.
func TestRepoManifestPlatformStatusContract(t *testing.T) {
	reader := liveReader(t)
	vocabulary, err := reader.Vocabulary()
	if err != nil {
		t.Fatal(err)
	}
	manifests, err := reader.CapabilityManifests()
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) == 0 {
		t.Fatal("no capability manifests were discovered; the gate would pass vacuously")
	}
	if err := deployability.ValidateManifestDeclarations(manifestDeclarations(manifests), vocabulary.Capabilities); err != nil {
		t.Fatalf("repository capability manifests violate the platform status contract: %v", err)
	}
}

func TestRepoManifestGateNamesTheOffendingFileAndToken(t *testing.T) {
	declarations := []deployability.ManifestDeclaration{{
		Path: "internal/tools/git/tool.json", Name: "git", Capability: "source-control", Role: "primary",
		Platforms: map[string]deployability.PlatformDeclaration{"macos": {Status: "available"}},
	}}
	err := deployability.ValidateManifestDeclarations(declarations, []string{"source-control"})
	if err == nil {
		t.Fatal("the gate accepted a platform status outside the vocabulary")
	}
	for _, fragment := range []string{"internal/tools/git/tool.json", "available", "macos"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("gate error %q does not name %q", err, fragment)
		}
	}
}

// TestEveryPlatformStatusResolvesThroughTheGrid drives one fixture manifest
// per vocabulary token through the whole grid and pins the (status,
// qualification) pair each token lands on.
//
// The pairs are asserted individually rather than merely for distinctness,
// because two tokens deliberately share one outcome: `experimental` and
// `unqualified` both mean "wired, unproven" and the platform status vocabulary
// maps both onto QualificationUnqualified. A distinctness assertion would fail
// on that intentional synonym while saying nothing about the token this test
// exists for — `build-verified`, which must resolve as an implementation like
// `supported` does, carry a strictly lower honesty rung than it, and never
// collapse into `unwired`.
func TestEveryPlatformStatusResolvesThroughTheGrid(t *testing.T) {
	type outcome struct {
		status        deployability.CapabilityResolutionStatus
		qualification deployability.Qualification
	}
	want := map[deployability.PlatformStatus]outcome{
		deployability.StatusSupported:      {deployability.CapabilityImplemented, deployability.QualificationQualified},
		deployability.StatusBuildVerified:  {deployability.CapabilityImplemented, deployability.QualificationBuildVerified},
		deployability.StatusExperimental:   {deployability.CapabilityImplemented, deployability.QualificationUnqualified},
		deployability.StatusUnqualified:    {deployability.CapabilityImplemented, deployability.QualificationUnqualified},
		deployability.StatusPartial:        {deployability.CapabilityDegraded, deployability.QualificationDegraded},
		deployability.StatusUnsupported:    {deployability.CapabilityIneligible, deployability.QualificationIneligible},
		deployability.StatusNotImplemented: {deployability.CapabilityIneligible, deployability.QualificationIneligible},
		deployability.StatusNotApplicable:  {deployability.CapabilityIneligible, deployability.QualificationIneligible},
	}
	if len(want) != len(deployability.PlatformStatuses()) {
		t.Fatalf("the expectation table covers %d tokens; the vocabulary has %d", len(want), len(deployability.PlatformStatuses()))
	}

	got := make(map[deployability.PlatformStatus]outcome, len(want))
	for _, status := range deployability.PlatformStatuses() {
		root := t.TempDir()
		writeVocabulary(t, root, []string{"probe"}, nil)
		writeToolFixture(t, root, "probe-tool", "probe", map[string]string{"linux": string(status)})
		reader, err := NewReader(root)
		if err != nil {
			t.Fatal(err)
		}
		grid, err := reader.Grid(time.Now())
		if err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		entry, ok := grid.Capability("probe")
		if !ok {
			t.Fatalf("%s: capability probe is missing from the grid", status)
		}
		platform, ok := entry.Platform(deployability.HostOSLinux)
		if !ok {
			t.Fatalf("%s: capability probe has no linux entry", status)
		}
		result := outcome{status: platform.Status, qualification: platform.Qualification}
		if result != want[status] {
			t.Errorf("platform status %s resolved %+v, want %+v", status, result, want[status])
		}
		got[status] = result
	}

	buildVerified := got[deployability.StatusBuildVerified]
	supported := got[deployability.StatusSupported]
	if !buildVerified.status.HasImplementation() {
		t.Errorf("build-verified resolved to %s, which reports no implementation", buildVerified.status)
	}
	if buildVerified.status == deployability.CapabilityUnwired {
		t.Error("build-verified collapsed into unwired")
	}
	if buildVerified == supported {
		t.Error("build-verified is indistinguishable from supported")
	}
	if buildVerified.qualification.AtLeast(supported.qualification) {
		t.Errorf("build-verified qualification %s is not below supported's %s", buildVerified.qualification, supported.qualification)
	}
}

// TestCapabilityWithNoImplementationStaysInTheGrid is the difference between
// "nobody has built this" and "nobody has named this". A vocabulary
// capability nothing implements must still appear, resolved peerless.
func TestCapabilityWithNoImplementationStaysInTheGrid(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"orphaned", "probe"}, nil)
	writeToolFixture(t, root, "probe-tool", "probe", map[string]string{"linux": "supported"})
	reader, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := grid.Capability("orphaned")
	if !ok {
		t.Fatal("a vocabulary capability with no implementation was dropped from the grid")
	}
	for _, hostOS := range operatingSystems {
		platform, ok := entry.Platform(hostOS)
		if !ok {
			t.Fatalf("orphaned has no entry for %s", hostOS)
		}
		if platform.Status != deployability.CapabilityPeerless {
			t.Errorf("orphaned/%s resolved %s, want peerless", hostOS, platform.Status)
		}
	}
	if entry.Situation == SituationBuiltEverywhere {
		t.Error("a capability nobody implements was classified as built everywhere")
	}
}

func TestGridChangesWhenManifestCapabilityChanges(t *testing.T) {
	root := t.TempDir()
	writeVocabulary(t, root, []string{"probe", "renamed"}, nil)
	writeToolFixture(t, root, "probe-tool", "probe", map[string]string{"linux": "supported"})
	reader, err := NewReader(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	probe, ok := before.Capability("probe")
	if !ok {
		t.Fatal("probe is missing from the grid")
	}
	if platform, _ := probe.Platform(deployability.HostOSLinux); !platform.Status.HasImplementation() {
		t.Fatalf("probe/linux resolved %s before the edit", platform.Status)
	}

	writeToolFixture(t, root, "probe-tool", "renamed", map[string]string{"linux": "supported"})
	after, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	probeAfter, ok := after.Capability("probe")
	if !ok {
		t.Fatal("probe is missing from the grid after the edit")
	}
	if platform, _ := probeAfter.Platform(deployability.HostOSLinux); platform.Status.HasImplementation() {
		t.Error("the grid still reports probe as implemented after the manifest moved to another capability")
	}
	renamed, ok := after.Capability("renamed")
	if !ok {
		t.Fatal("renamed is missing from the grid after the edit")
	}
	if platform, _ := renamed.Platform(deployability.HostOSLinux); !platform.Status.HasImplementation() {
		t.Error("the grid does not report renamed as implemented after the manifest moved to it")
	}
}

func TestUnreachableManifestRootIsAnErrorNotAnEmptyGrid(t *testing.T) {
	t.Run("empty root", func(t *testing.T) {
		if _, err := NewReader("   "); err == nil {
			t.Fatal("an empty manifest root was accepted")
		} else if !IsUnresolvedRoot(err) {
			t.Fatalf("empty root produced %T, want UnresolvedRootError", err)
		}
	})

	t.Run("root without a vocabulary", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "not-a-repository")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := NewReader(root)
		if err == nil {
			t.Fatal("a root with no capability vocabulary was accepted; the grid would have been empty and looked complete")
		}
		if !IsUnresolvedRoot(err) {
			t.Fatalf("missing vocabulary produced %T, want UnresolvedRootError", err)
		}
		for _, fragment := range []string{root, "capability-vocabulary.json"} {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("error %q does not name %q", err, fragment)
			}
		}
	})
}

func writeVocabulary(t *testing.T, root string, capabilities []string, policies map[string]map[string]string) {
	t.Helper()
	dir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Vocabulary{Capabilities: capabilities, PlatformPolicies: policies})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capability-vocabulary.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeToolFixture(t *testing.T, root, name, capability string, platforms map[string]string) {
	writeToolFixtureWithEvidence(t, root, name, capability, platforms, true)
}

func writeToolFixtureWithoutEvidence(t *testing.T, root, name, capability string, platforms map[string]string) {
	writeToolFixtureWithEvidence(t, root, name, capability, platforms, false)
}

func writeToolFixtureWithEvidence(t *testing.T, root, name, capability string, platforms map[string]string, includeEvidence bool) {
	t.Helper()
	dir := filepath.Join(root, "internal", "tools", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	declarations := make(map[string]PlatformDeclaration, len(platforms))
	for hostOS, status := range platforms {
		declaration := PlatformDeclaration{Status: status}
		if includeEvidence && status == string(deployability.StatusSupported) {
			declaration.Evidence = json.RawMessage(`{"run_id":"fixture-run","host":"fixture-host","os":"linux","arch":"amd64","date":"2026-08-25","surface":"grid-test","artifact_uri":"artifact://fixture-run"}`)
		}
		declarations[hostOS] = declaration
	}
	data, err := json.Marshal(Manifest{
		Name: name, Capability: capability, Role: "primary",
		PlatformDeclarations: declarations,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
