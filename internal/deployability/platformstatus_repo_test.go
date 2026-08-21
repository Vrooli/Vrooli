package deployability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The repo-wide platform-status gate.
//
// Every other test in this package proves the resolver handles the vocabulary
// correctly given a declaration. This one proves the LIVE REPOSITORY only ever
// authors declarations inside that vocabulary — so a malformed `tool.json`,
// `safeguard.json`, or scenario `service.json` fails `go test ./internal/...`
// rather than waiting for a human to run `vrooli capability ledger` by hand.
//
// It walks the same three manifest families the ledger reads, and it walks them
// from the repository root rather than from a fixture, deliberately: a gate that
// runs against fixtures cannot catch the file somebody added yesterday.
//
// When this fails, do NOT relax the vocabulary to accommodate the token. Either
// the manifest is wrong, or the token is a genuinely new authored status and
// belongs in `platformstatus.go` with a qualification rung and a stated reason.

// platformStatusSite is one authored `status` token and where it came from, so a
// failure names the file a human has to open.
type platformStatusSite struct {
	path       string
	capability string
	hostOS     string
	status     string
}

func TestLiveRepositoryAuthorsOnlyVocabularyPlatformStatuses(t *testing.T) {
	repoRoot := repositoryRoot(t)
	sites, err := collectPlatformStatusSites(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) == 0 {
		// A gate that silently finds nothing is worse than no gate: it reports
		// green forever the moment its walk breaks.
		t.Fatal("platform status walk found no declarations; the walk is broken, not the repository")
	}

	var offenders []string
	for _, site := range sites {
		if _, err := ParsePlatformStatus(site.status); err != nil {
			offenders = append(offenders, fmt.Sprintf(
				"%s: capability %q on %s declares status %q — %v",
				site.path, site.capability, site.hostOS, site.status, err,
			))
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf(
			"%d platform declaration(s) author a status outside the vocabulary %v:\n  %s",
			len(offenders), PlatformStatuses(), strings.Join(offenders, "\n  "),
		)
	}
	t.Logf("checked %d platform status declarations across the live repository", len(sites))
}

// TestLiveRepositoryExercisesTheWholeVocabulary is a coverage check on the
// vocabulary itself, not on the repository. A constant that nothing in the repo
// ever authors is either dead or a token somebody removed without removing its
// handling — both worth knowing about. It reports rather than fails, because a
// status legitimately falls out of use before it is retired.
func TestLiveRepositoryExercisesTheWholeVocabulary(t *testing.T) {
	repoRoot := repositoryRoot(t)
	sites, err := collectPlatformStatusSites(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[PlatformStatus]int{}
	for _, site := range sites {
		if status, err := ParsePlatformStatus(site.status); err == nil {
			seen[status]++
		}
	}
	for _, status := range PlatformStatuses() {
		t.Logf("%-16s authored %d time(s)", status, seen[status])
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

// collectPlatformStatusSites walks the three manifest families that may author a
// platform status: host tools, host safeguards, and scenario service manifests.
func collectPlatformStatusSites(repoRoot string) ([]platformStatusSite, error) {
	var sites []platformStatusSite

	toolSites, err := collectCapabilityBlockSites(repoRoot, "internal/tools", "tool.json")
	if err != nil {
		return nil, err
	}
	sites = append(sites, toolSites...)

	safeguardSites, err := collectCapabilityBlockSites(repoRoot, "internal/safeguards", "safeguard.json")
	if err != nil {
		return nil, err
	}
	sites = append(sites, safeguardSites...)

	scenarioSites, err := collectScenarioPlatformCapabilitySites(repoRoot)
	if err != nil {
		return nil, err
	}
	return append(sites, scenarioSites...), nil
}

// capabilityBlockManifest is the subset of a tool or safeguard manifest that
// carries platform declarations.
//
// Note the field name: a tool manifest's `platforms` is a LIST OF OS NAMES with
// no status attached, and the per-OS status lives in `platform_declarations`.
// This mirrors how the `portability` domain in the infrastructure-manager
// scenario reads the same files, and the two must not drift — a gate that reads
// a different field from the consumer it guards is worse than no gate.
type capabilityBlockManifest struct {
	Name                 string                         `json:"name"`
	Capability           string                         `json:"capability"`
	PlatformDeclarations map[string]platformStatusEntry `json:"platform_declarations"`
}

type platformStatusEntry struct {
	Status    string `json:"status"`
	Mechanism string `json:"mechanism"`
}

func collectCapabilityBlockSites(repoRoot, relDir, manifestName string) ([]platformStatusSite, error) {
	root := filepath.Join(repoRoot, relDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	var sites []platformStatusSite
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), manifestName)
		payload, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var manifest capabilityBlockManifest
		if err := json.Unmarshal(payload, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		rel := filepath.Join(relDir, entry.Name(), manifestName)
		capability := strings.TrimSpace(manifest.Capability)
		if capability == "" {
			capability = entry.Name()
		}
		for hostOS, declaration := range manifest.PlatformDeclarations {
			if strings.TrimSpace(declaration.Status) == "" {
				continue
			}
			sites = append(sites, platformStatusSite{
				path: rel, capability: capability, hostOS: hostOS, status: declaration.Status,
			})
		}
	}
	return sites, nil
}

func collectScenarioPlatformCapabilitySites(repoRoot string) ([]platformStatusSite, error) {
	root := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	var sites []platformStatusSite
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), ".vrooli", "service.json")
		payload, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var service struct {
			Service struct {
				Name                 string                                    `json:"name"`
				PlatformCapabilities map[string]map[string]platformStatusEntry `json:"platform_capabilities"`
			} `json:"service"`
		}
		if err := json.Unmarshal(payload, &service); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		rel := filepath.Join("scenarios", entry.Name(), ".vrooli", "service.json")
		for capability, declarations := range service.Service.PlatformCapabilities {
			for hostOS, declaration := range declarations {
				if strings.TrimSpace(declaration.Status) == "" {
					continue
				}
				sites = append(sites, platformStatusSite{
					path: rel, capability: capability, hostOS: hostOS, status: declaration.Status,
				})
			}
		}
	}
	return sites, nil
}

// TestPlatformStatusGateCatchesAMalformedManifest proves the gate actually
// fails rather than merely passing on a clean repository.
//
// It builds a throwaway tree with the same layout the walk expects and asserts
// the offending token is found, so the gate's own correctness does not depend
// on somebody first breaking a real manifest. A gate nobody has seen fail is a
// gate nobody should trust.
func TestPlatformStatusGateCatchesAMalformedManifest(t *testing.T) {
	root := t.TempDir()
	toolDir := filepath.Join(root, "internal", "tools", "fixture-tool")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
	  "name": "fixture-tool",
	  "capability": "fixture-capability",
	  "platforms": ["linux", "macos"],
	  "platform_declarations": {
	    "linux": {"status": "supported", "mechanism": "procfs"},
	    "macos": {"status": "probably-fine", "mechanism": "guesswork"}
	  }
	}`
	if err := os.WriteFile(filepath.Join(toolDir, "tool.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	sites, err := collectPlatformStatusSites(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 2 {
		t.Fatalf("expected both declarations to be collected, got %d: %#v", len(sites), sites)
	}

	var offenders []platformStatusSite
	for _, site := range sites {
		if _, err := ParsePlatformStatus(site.status); err != nil {
			offenders = append(offenders, site)
		}
	}
	if len(offenders) != 1 {
		t.Fatalf("expected exactly one offending declaration, got %d: %#v", len(offenders), offenders)
	}
	offender := offenders[0]
	if offender.status != "probably-fine" {
		t.Errorf("wrong offending token: %q", offender.status)
	}
	// The failure must name the file a human has to open, and the capability
	// and host OS that identify the line within it.
	if !strings.Contains(offender.path, filepath.Join("internal", "tools", "fixture-tool", "tool.json")) {
		t.Errorf("offender does not name its manifest path: %q", offender.path)
	}
	if offender.capability != "fixture-capability" || offender.hostOS != "macos" {
		t.Errorf("offender does not identify the declaration: %#v", offender)
	}
}
