package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// composeImagePinDebt lists resources whose compose files still reference
// floating image tags. This is grandfathered migration debt, not an accepted
// state: remove an entry here in the same change that pins the image
// (docs/resources/deployment-contract.md, "Pinned Runtime Principle").
var composeImagePinDebt = []string{}

// gpuCapacityDebt lists resources that declare a gpu block without a capacity
// block, making their VRAM invisible to the capacity broker. Remove an entry
// when the resource gains a capacity claim (or a recorded decision that the
// broker must not manage it).
var gpuCapacityDebt = []string{
	"ollama", // manages its own model residency; broker integration undecided
}

var composeImageLinePattern = regexp.MustCompile(`^\s*image:\s*(.+?)\s*$`)

// TestRealComposeImagesArePinned enforces the Pinned Runtime Principle over
// compose-service resources: every image pulled from a registry must be an
// immutable reference. Locally built images (a compose file with a build
// directive) are exempt — their content is pinned by the repo's Dockerfile.
func TestRealComposeImagesArePinned(t *testing.T) {
	repoRoot := findRepoRoot(t)
	for _, m := range loadRealManifests(t, repoRoot) {
		if m.manifest.Driver != "compose-service" {
			continue
		}
		if slices.Contains(composeImagePinDebt, m.manifest.Name) {
			continue
		}
		for _, composePath := range composeFilesFor(repoRoot, m) {
			content, err := os.ReadFile(composePath)
			if err != nil {
				t.Fatalf("%s: read %s: %v", m.manifest.Name, composePath, err)
			}
			text := string(content)
			if strings.Contains(text, "build:") {
				continue
			}
			for _, line := range strings.Split(text, "\n") {
				match := composeImageLinePattern.FindStringSubmatch(line)
				if match == nil {
					continue
				}
				image := unwrapComposeDefault(strings.Trim(match[1], `"'`))
				if err := manifest.ValidatePinnedImageRef(image); err != nil {
					t.Errorf("%s: %s: %v", m.manifest.Name, filepath.Base(composePath), err)
				}
			}
		}
	}
}

// TestRealGPUResourcesDeclareCapacity ensures a resource that declares a gpu
// block also declares a capacity block, so the capacity broker can see its
// VRAM claim alongside co-tenant GPU resources.
func TestRealGPUResourcesDeclareCapacity(t *testing.T) {
	repoRoot := findRepoRoot(t)
	for _, m := range loadRealManifests(t, repoRoot) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(m.raw, &raw); err != nil {
			t.Fatalf("%s: parse raw manifest: %v", m.manifest.Name, err)
		}
		_, hasGPU := raw["gpu"]
		_, hasCapacity := raw["capacity"]
		if hasGPU && !hasCapacity && !slices.Contains(gpuCapacityDebt, m.manifest.Name) {
			t.Errorf("%s: declares gpu without a capacity block; add a capacity claim or record it in gpuCapacityDebt with a rationale", m.manifest.Name)
		}
	}
}

type realManifest struct {
	dir      string
	raw      []byte
	manifest manifest.ResourceManifest
}

func loadRealManifests(t *testing.T, repoRoot string) []realManifest {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(repoRoot, "resources", "*", "resource.json"))
	if err != nil {
		t.Fatalf("glob resource manifests: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no resource manifests found")
	}
	manifests := make([]realManifest, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		m, err := manifest.Load(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		manifests = append(manifests, realManifest{dir: filepath.Dir(path), raw: raw, manifest: m})
	}
	return manifests
}

func composeFilesFor(repoRoot string, m realManifest) []string {
	files := make([]string, 0, 2)
	if m.manifest.ComposeFile != "" {
		files = append(files, filepath.Join(m.dir, m.manifest.ComposeFile))
	}
	if m.manifest.GPU != nil && m.manifest.GPU.ComposeOverlay != "" {
		files = append(files, filepath.Join(m.dir, m.manifest.GPU.ComposeOverlay))
	}
	return files
}

// unwrapComposeDefault reduces a compose env interpolation like
// "${WHISPER_IMAGE:-repo/image:tag}" to its default value; plain references
// pass through unchanged.
func unwrapComposeDefault(image string) string {
	if strings.HasPrefix(image, "${") && strings.HasSuffix(image, "}") {
		inner := image[2 : len(image)-1]
		if idx := strings.Index(inner, ":-"); idx >= 0 {
			return inner[idx+2:]
		}
	}
	return image
}
