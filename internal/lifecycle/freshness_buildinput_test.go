package lifecycle

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// buildInputDeps returns a hostProbeDeps whose build-environment seams return the
// supplied canned values. goListJSON is left nil so the artifact resolver falls
// back to the static walk (the file set we want to pin), and lookPath reports go
// present so hostGoToolchain is exercised through its own cache.
func buildInputDeps(goEnvVals map[string]string, nodeMajor string) hostProbeDeps {
	deps := defaultHostProbeDeps()
	deps.goListJSON = nil
	deps.goEnv = func(keys ...string) map[string]string {
		out := map[string]string{}
		for _, k := range keys {
			if v, ok := goEnvVals[k]; ok && v != "" {
				out[k] = v
			}
		}
		return out
	}
	deps.nodeVersion = func() string { return nodeMajor }
	return deps
}

func fullGoEnv() map[string]string {
	return map[string]string{
		"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "1", "GOAMD64": "v1",
	}
}

// TestFileSetInvariance is the guardrail: the file input set must be byte-identical
// regardless of the keyed build-environment values, so the feature cannot re-create
// the false-positive cascade (which came from the file set, never KeyInputs).
func TestFileSetInvariance(t *testing.T) {
	appRoot := "/repo/scenarios/demo"
	repoRoot := "/repo"
	check := scenario.ConditionCheck{Type: "binaries", Targets: []string{"api/demo-api"}}

	variants := []hostProbeDeps{
		buildInputDeps(map[string]string{}, ""), // omit-on-unknown (toolchain-only-ish)
		buildInputDeps(fullGoEnv(), "20"),       // full host-native set
		buildInputDeps(map[string]string{"GOOS": "darwin", "GOARCH": "arm64", "CGO_ENABLED": "0"}, "18"),      // cross-compile
		buildInputDeps(map[string]string{"GOOS": "linux", "GOARCH": "amd64", "GOFLAGS": "-mod=vendor"}, "22"), // extra determinant
	}
	steps := []scenario.PhaseStep{{Run: "cd api && go build -tags foo -o demo-api ."}}

	var baseline []string
	for i, deps := range variants {
		arts, err := binariesFreshnessArtifacts(appRoot, repoRoot, check, steps, deps)
		if err != nil {
			t.Fatalf("variant %d: %v", i, err)
		}
		if len(arts) != 1 {
			t.Fatalf("variant %d: expected 1 artifact, got %d", i, len(arts))
		}
		got := append([]string(nil), arts[0].Spec.Inputs...)
		sort.Strings(got)
		if i == 0 {
			baseline = got
			continue
		}
		if !reflect.DeepEqual(got, baseline) {
			t.Fatalf("file input set changed across key-input variation:\n base=%v\n var%d=%v", baseline, i, got)
		}
	}
}

// stampThenEvaluate stamps a manifest in tmp dir for a tiny source tree under the
// given key inputs, then evaluates an unchanged source tree under a (possibly
// different) key input set. The source bytes are identical between stamp and
// check; only KeyInputs vary.
func stampThenEvaluate(t *testing.T, stampKeys, checkKeys map[string]string) cliutil.FreshnessVerdict {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := cliutil.FreshnessSpec{SourceRoot: dir, ContextRoot: dir}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "binaries", stampKeys, 1)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	verdict, err := cliutil.EvaluateFreshness(spec, manifest, checkKeys)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return verdict
}

// TestPerKeyFalseNegativeClosure: identical source + a changed build determinant
// ⇒ stale, naming the exact key. This is the false-negative the feature closes.
func TestPerKeyFalseNegativeClosure(t *testing.T) {
	base := map[string]string{
		"goos": "linux", "goarch": "amd64", "cgo_enabled": "1", "goflags": "",
		"goexperiment": "", "goamd64": "v1", "build_tags": "", "ldflags": "", "trimpath": "",
	}
	changes := map[string]string{
		"goos": "darwin", "goarch": "arm64", "cgo_enabled": "0", "goflags": "-mod=vendor",
		"goexperiment": "rangefunc", "goamd64": "v3", "build_tags": "foo", "ldflags": "-s -w", "trimpath": "true",
	}
	for key, newVal := range changes {
		t.Run(key, func(t *testing.T) {
			stamp := map[string]string{}
			check := map[string]string{}
			for k, v := range base {
				if v != "" {
					stamp[k] = v
					check[k] = v
				}
			}
			// Make the determinant present on both sides but differ.
			stamp[key] = base[key]
			if stamp[key] == "" {
				delete(stamp, key)
			}
			check[key] = newVal
			// When base value was empty, the stamp omits the key; an added key must
			// still flip stale (compareKeyInputs treats "" != newVal).
			verdict := stampThenEvaluate(t, stamp, check)
			if !verdict.Stale {
				t.Fatalf("expected stale on %s change", key)
			}
			if verdict.Reason != "build input changed" {
				t.Fatalf("reason = %q, want %q", verdict.Reason, "build input changed")
			}
			if verdict.File != key {
				t.Fatalf("file = %q, want %q", verdict.File, key)
			}
		})
	}
}

// TestIdenticalDeterminantsFresh: identical source + identical determinants ⇒ fresh.
func TestIdenticalDeterminantsFresh(t *testing.T) {
	keys := map[string]string{"goos": "linux", "goarch": "amd64", "cgo_enabled": "1", "toolchain": "go1.25.0"}
	verdict := stampThenEvaluate(t, keys, keys)
	if verdict.Stale {
		t.Fatalf("expected fresh, got stale: %s (%s)", verdict.Reason, verdict.File)
	}
}

// TestOmitOnUnknown: when goEnv resolves nothing (go absent), the env keys are
// absent — no spurious stale, and stamp/evaluate round-trips cleanly.
func TestOmitOnUnknown(t *testing.T) {
	deps := buildInputDeps(map[string]string{}, "")
	keys := goBuildKeyInputs(deps, "")
	for _, banned := range goEnvDeterminants {
		if _, ok := keys[banned]; ok {
			t.Fatalf("determinant %q must be omitted when goEnv is empty", banned)
		}
	}
	// Stamp and evaluate with the omitted set on both sides ⇒ fresh.
	verdict := stampThenEvaluate(t, keys, keys)
	if verdict.Stale {
		t.Fatalf("omit-on-unknown should be fresh, got stale: %s", verdict.Reason)
	}
}

// TestTagNormalization: tag ordering noise must not flap the digest.
func TestTagNormalization(t *testing.T) {
	a := parseBuildCommandFlags("go build -tags b,a -o x .")
	b := parseBuildCommandFlags("go build -tags 'a b' -o x .")
	if a["build_tags"] != b["build_tags"] {
		t.Fatalf("tag normalization mismatch: %q vs %q", a["build_tags"], b["build_tags"])
	}
	if a["build_tags"] != "a,b" {
		t.Fatalf("normalized tags = %q, want a,b", a["build_tags"])
	}
	// Absent GOFLAGS vs present-empty must be treated identically (key omitted).
	full := buildInputDeps(map[string]string{"GOOS": "linux", "GOFLAGS": ""}, "")
	keys := goBuildKeyInputs(full, "")
	if _, ok := keys["goflags"]; ok {
		t.Fatalf("present-empty GOFLAGS must be omitted, got %q", keys["goflags"])
	}
}

// TestParseBuildCommandFlags covers the recognized shapes and the omit-on-ambiguity
// default.
func TestParseBuildCommandFlags(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want map[string]string
	}{
		{"no flags", "go build -o demo-api .", map[string]string{}},
		{"inline tags", "go build -tags=foo,bar .", map[string]string{"build_tags": "bar,foo"}},
		{"spaced ldflags quoted", `go build -ldflags "-s -w" .`, map[string]string{"ldflags": "-s -w"}},
		{"inline ldflags", `go build -ldflags=-X=main.v=1 .`, map[string]string{"ldflags": "-X=main.v=1"}},
		{"trimpath bare", "go build -trimpath -o x .", map[string]string{"trimpath": "true"}},
		{"combined", `go build -trimpath -tags a -ldflags "-w" -o x .`, map[string]string{"trimpath": "true", "build_tags": "a", "ldflags": "-w"}},
		{"empty cmd", "", map[string]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBuildCommandFlags(tc.cmd)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parse(%q) = %v, want %v", tc.cmd, got, tc.want)
			}
		})
	}
}

// TestMatchBuildCommand binds the right build step to the right target in a
// multi-target scenario; a step without a matching binary name yields "".
func TestMatchBuildCommand(t *testing.T) {
	steps := []scenario.PhaseStep{
		{Run: "cd api && go build -tags api -o demo-api ."},
		{Run: "cd cli && go build -tags cli -o demo-cli ."},
		{Run: "echo not-a-build"},
	}
	if got := matchBuildCommand(steps, "api/demo-api"); got != steps[0].Run {
		t.Fatalf("api target matched %q", got)
	}
	if got := matchBuildCommand(steps, "cli/demo-cli"); got != steps[1].Run {
		t.Fatalf("cli target matched %q", got)
	}
	if got := matchBuildCommand(steps, "web/demo-web"); got != "" {
		t.Fatalf("unmatched target should yield empty, got %q", got)
	}
}

// TestUIBuildKeyInputs: node_major keyed when present, omitted when absent.
func TestUIBuildKeyInputs(t *testing.T) {
	withNode := buildInputDeps(map[string]string{}, "20")
	keys := uiBuildKeyInputs(withNode)
	if keys["node_major"] != "20" {
		t.Fatalf("node_major = %q, want 20", keys["node_major"])
	}
	if keys["node_env"] == "" {
		t.Fatalf("node_env must always be present")
	}
	noNode := buildInputDeps(map[string]string{}, "")
	if _, ok := uiBuildKeyInputs(noNode)["node_major"]; ok {
		t.Fatalf("node_major must be omitted when node absent")
	}
}

// TestNodeMajor covers version-string parsing.
func TestNodeMajor(t *testing.T) {
	cases := map[string]string{
		"v20.11.0\n": "20", "v18.0.0": "18", "20.1.0": "20", "": "", "vX.Y": "", "  v22.3.1 ": "22",
	}
	for in, want := range cases {
		if got := nodeMajor(in); got != want {
			t.Fatalf("nodeMajor(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBinariesFreshnessBuildDeterminantNamesKeyInExplain proves the full
// lifecycle path (the surface behind `vrooli scenario freshness --explain`) names
// the changed build input as precisely as a changed file: identical source, a
// rewritten determinant in the recorded manifest ⇒ stale with the key in the
// reason. Mirrors the toolchain case but for a curated `go env` determinant.
func TestBinariesFreshnessBuildDeterminantNamesKeyInExplain(t *testing.T) {
	repoRoot, appPath, binPath, _ := freshnessTestScene(t)
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	check := item.Manifest.Lifecycle.Setup.Condition.Checks[0]

	if stale, _, err := r.binariesFreshness(item, check); err != nil || stale {
		t.Fatalf("bootstrap fresh expected: stale=%v err=%v", stale, err)
	}

	manifestPath := cliutil.FreshnessManifestPath(binPath)
	m, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil || !ok {
		t.Fatalf("read manifest: ok=%v err=%v", ok, err)
	}
	if m.KeyInputs == nil {
		m.KeyInputs = map[string]string{}
	}
	// Force a determinant disagreement: the live host resolves a real value via
	// `go env`; rewrite the recorded one so they differ.
	m.KeyInputs["cgo_enabled"] = "tampered"
	if err := cliutil.WriteFreshnessManifest(manifestPath, m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stale, reason, err := r.binariesFreshness(item, check)
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatalf("determinant change must mark stale, reason=%q", reason)
	}
	if !strings.Contains(reason, "build input changed") || !strings.Contains(reason, "cgo_enabled") {
		t.Fatalf("explain reason must name the changed build input, got %q", reason)
	}
}

// TestGoBuildKeyInputsLowercasing: go env keys are folded to the lower-snake
// contract names and build-command flags merge in.
func TestGoBuildKeyInputsLowercasing(t *testing.T) {
	deps := buildInputDeps(fullGoEnv(), "")
	keys := goBuildKeyInputs(deps, "go build -tags foo -o demo .")
	for _, want := range []string{"goos", "goarch", "cgo_enabled", "goamd64", "build_tags"} {
		if _, ok := keys[want]; !ok {
			t.Fatalf("expected key %q in %v", want, keys)
		}
	}
	for banned := range keys {
		if banned != "build_tags" && banned != "ldflags" && banned != "trimpath" && banned != "toolchain" &&
			banned != "node_env" && banned != "node_major" {
			if banned == "GOOS" || banned == "GOARCH" {
				t.Fatalf("key %q not lower-cased", banned)
			}
		}
	}
	if keys["goos"] != "linux" {
		t.Fatalf("goos = %q, want linux", keys["goos"])
	}
}
