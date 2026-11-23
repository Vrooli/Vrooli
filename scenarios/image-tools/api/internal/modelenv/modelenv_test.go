package modelenv

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func envMap(env []string) map[string]string {
	out := map[string]string{}
	for _, item := range env {
		k, v, _ := strings.Cut(item, "=")
		out[k] = v
	}
	return out
}

func TestTableIsValid(t *testing.T) {
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

// Contract: every known cache variable in the table is pinned to a directory
// strictly inside <root>/cache for inference AND acquisition processes, whatever
// the operator's shell exported, and nothing a library might consult is left to
// its home-directory default.
func TestEveryKnownCacheVariableIsPinnedInsideTheManagedRoot(t *testing.T) {
	root := t.TempDir()
	table := MustLoad()
	hostile := []string{"PATH=/usr/bin"}
	for _, d := range table.Dirs {
		hostile = append(hostile, d.Var+"=/home/operator/.cache/elsewhere")
	}
	cacheRoot := filepath.Join(root, CacheSubdir)
	for _, s := range table.Scrub {
		hostile = append(hostile, s.Var+"=/home/operator/.cache/legacy")
	}
	for _, mode := range []Mode{ModeInference, ModeAcquisition} {
		got := envMap(overlayAt(table, hostile, mode, root))
		for _, s := range table.Scrub {
			if _, present := got[s.Var]; present {
				t.Fatalf("%s: legacy variable %s must be removed, it can override the pins", mode, s.Var)
			}
		}
		for _, d := range table.Dirs {
			value, ok := got[d.Var]
			if !ok {
				t.Fatalf("%s: %s is not pinned", mode, d.Var)
			}
			rel, err := filepath.Rel(cacheRoot, value)
			if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
				t.Fatalf("%s: %s=%s escapes %s", mode, d.Var, value, cacheRoot)
			}
		}
		if got["PATH"] != "/usr/bin" {
			t.Fatalf("%s: unrelated variables must pass through untouched", mode)
		}
	}
}

func TestInferenceIsOfflineAndAcquisitionIsTheOnlyNetworkMode(t *testing.T) {
	root := t.TempDir()
	table := MustLoad()
	inference := envMap(overlayAt(table, []string{"HF_HUB_OFFLINE=0"}, ModeInference, root))
	for _, name := range []string{"HF_HUB_OFFLINE", "TRANSFORMERS_OFFLINE", "HF_DATASETS_OFFLINE"} {
		if inference[name] != "1" {
			t.Fatalf("inference must run %s=1, got %q", name, inference[name])
		}
	}
	if inference["IMAGE_TOOLS_SIDECAR_NETWORK"] != "blocked" {
		t.Fatalf("inference must block the network: %q", inference["IMAGE_TOOLS_SIDECAR_NETWORK"])
	}
	acquisition := envMap(overlayAt(table, []string{"HF_HUB_OFFLINE=1"}, ModeAcquisition, root))
	if acquisition["HF_HUB_OFFLINE"] != "0" || acquisition["IMAGE_TOOLS_SIDECAR_NETWORK"] != "governed-acquisition" {
		t.Fatalf("acquisition must be able to fetch: %v", acquisition)
	}
}

// Contract: every Python package in the governed lock that is known to fetch
// weights at run time is covered by a row, so adding such a library without a
// pin fails here. A package on the watchlist but absent from every row's
// `packages` list is the defect.
func TestEveryDownloadingPackageInTheLockIsCovered(t *testing.T) {
	lock, err := os.ReadFile(filepath.Join("..", "pydeps", "requirements.lock"))
	if err != nil {
		t.Fatal(err)
	}
	table := MustLoad()
	covered := map[string]bool{}
	for _, d := range table.Dirs {
		for _, p := range d.Packages {
			covered[normalize(p)] = true
		}
	}
	pin := regexp.MustCompile(`(?m)^([A-Za-z0-9_.\-]+)==`)
	installed := map[string]bool{}
	for _, m := range pin.FindAllStringSubmatch(string(lock), -1) {
		installed[normalize(m[1])] = true
	}
	for _, w := range table.Watchlist {
		if installed[normalize(w)] && !covered[normalize(w)] {
			t.Errorf("package %s fetches weights at run time but no row in pins.json lists it: add a dir row (or extend an existing row's packages) so its cache is pinned", w)
		}
	}
	// The libraries image-tools already runs must be covered today.
	for _, p := range []string{"huggingface-hub", "diffusers", "transformers", "torch", "torchvision", "lpips", "kornia", "onnx"} {
		if installed[normalize(p)] && !covered[normalize(p)] {
			t.Errorf("installed library %s has no pin row", p)
		}
	}
}

func normalize(name string) string {
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(name))
}

func TestApplyCreatesPinnedDirectoriesUnderTheManagedRoot(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(SwapRoot(root))
	env := Apply([]string{"HOME=/home/operator"}, ModeInference)
	got := envMap(env)
	for name, dir := range MustLoad().Pins(root) {
		if got[name] != dir {
			t.Fatalf("%s=%q, want %q", name, got[name], dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("pinned directory %s was not created: %v", dir, err)
		}
	}
	if got["HOME"] != "/home/operator" {
		t.Fatal("HOME is not part of the pin table and must pass through")
	}
}

func TestUnconfiguredRootNeverFallsBackToAHomeDirectory(t *testing.T) {
	t.Cleanup(SwapRoot(""))
	if strings.Contains(Root(), os.Getenv("HOME")+string(filepath.Separator)+".cache") || !strings.HasPrefix(Root(), os.TempDir()) {
		t.Fatalf("an unconfigured root must stay private under the temp dir, got %s", Root())
	}
}

func TestDefaultLocationsExpandForTheStrayScan(t *testing.T) {
	locations := MustLoad().DefaultLocations("/home/x")
	want := map[string]bool{"/home/x/.cache/huggingface": false, "/home/x/.insightface": false, "/home/x/.u2net": false, "/home/x/.cache/torch": false}
	var otherOwned int
	for _, l := range locations {
		if _, ok := want[l.Path]; ok {
			want[l.Path] = true
		}
		if l.Other {
			otherOwned++
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("default location %s is not scanned", path)
		}
	}
	if otherOwned == 0 {
		t.Error("other applications' model directories must be listed as report-only")
	}
}
