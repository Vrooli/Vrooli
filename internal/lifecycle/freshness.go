package lifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/scenario"
)

// artifactFreshness binds a single build artifact to the content-fingerprint
// contract used to decide whether it is stale. The recorded manifest lives at
// ManifestPath; KeyInputs carries the non-file build inputs (toolchain version,
// NODE_ENV) folded into the digest so an identical source tree built under a
// different toolchain/env still reads as stale.
type artifactFreshness struct {
	Target       string
	ArtifactPath string
	ManifestPath string
	Spec         cliutil.FreshnessSpec
	KeyInputs    map[string]string
	CheckType    string
}

// FreshnessCheckResult is one artifact's freshness verdict within a
// FreshnessReport.
type FreshnessCheckResult struct {
	CheckType string // "binaries" | "ui-bundle"
	Target    string // artifact target path (as declared in the check)
	Stale     bool
	Cause     string // machine cause, e.g. "content changed" / "source newer" / "missing artifact"
	File      string // offending input (rel path) or target, when known
}

// FreshnessDependencyPolicy reports the resolved freshness_policy for one
// declared scenario dependency edge — the answer to "if this dep is stale, will
// it be restarted?". The value is the normalized per-edge policy; runtime
// arbitration may further reduce disruption when other live consumers exist.
type FreshnessDependencyPolicy struct {
	Name   string
	Policy string
}

// FreshnessReport is the structured answer to "is this scenario's build fresh,
// and if not, why?" — the data behind `vrooli scenario freshness`.
type FreshnessReport struct {
	Scenario     string
	Stale        bool
	Checks       []FreshnessCheckResult
	Dependencies []FreshnessDependencyPolicy
}

// FreshnessReport evaluates every freshness check (binaries/ui-bundle) of a
// scenario and returns the per-artifact verdicts, plus the resolved
// freshness_policy of each declared scenario dependency. It mirrors the engine
// the lifecycle uses to decide rebuild/restart, so `--explain` is a faithful
// account of why a dependency would (or would not) be disrupted. Like the start
// path it opportunistically stamps a manifest for an already-fresh artifact that
// has none yet (one-time, self-healing adoption).
// FreshnessReportByName resolves a scenario by name (optionally from customPath)
// and returns its freshness report. It is the entry point used by the
// `vrooli scenario freshness` CLI command.
func (r *Runner) FreshnessReportByName(name, customPath string) (FreshnessReport, error) {
	item, err := r.loadScenario(name, customPath)
	if err != nil {
		return FreshnessReport{}, err
	}
	return r.FreshnessReport(item)
}

func (r *Runner) FreshnessReport(item scenario.Scenario) (FreshnessReport, error) {
	report := FreshnessReport{Scenario: item.Slug}
	deps := defaultHostProbeDeps()

	componentNames := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		componentNames = append(componentNames, name)
	}
	sort.Strings(componentNames)
	for _, name := range componentNames {
		component := item.Manifest.Components[name]
		if strings.TrimSpace(component.Build.Reuse) != "" {
			continue
		}
		artifacts, err := componentFreshnessArtifacts(item.Path, r.Root, component, deps)
		if err != nil {
			return FreshnessReport{}, fmt.Errorf("component %s freshness: %w", name, err)
		}
		for _, artifact := range artifacts {
			verdict, err := r.evaluateArtifactFreshness(artifact, deps)
			if err != nil {
				return FreshnessReport{}, err
			}
			report.appendVerdict(component.Build.Kind, verdict)
		}
	}

	for name, dep := range item.Manifest.Dependencies.Scenarios {
		report.Dependencies = append(report.Dependencies, FreshnessDependencyPolicy{
			Name:   name,
			Policy: dep.NormalizedFreshnessPolicy(),
		})
	}
	sort.Slice(report.Dependencies, func(i, j int) bool { return report.Dependencies[i].Name < report.Dependencies[j].Name })

	return report, nil
}

func (report *FreshnessReport) appendVerdict(checkType string, verdict artifactVerdict) {
	if verdict.Stale {
		report.Stale = true
	}
	report.Checks = append(report.Checks, FreshnessCheckResult{
		CheckType: checkType,
		Target:    verdict.Target,
		Stale:     verdict.Stale,
		Cause:     verdict.Cause,
		File:      verdict.File,
	})
}

// stampScenarioFreshness records a fresh manifest for every freshness-checked
// artifact of a scenario. Called after a successful setup build so the next
// freshness check is manifest-authoritative (and the artifact-digest gate has a
// baseline). Best-effort: a stamp failure is logged, never fatal to start.
func (r *Runner) stampScenarioFreshness(item scenario.Scenario) {
	deps := defaultHostProbeDeps()
	for name, component := range item.Manifest.Components {
		if strings.TrimSpace(component.Build.Reuse) != "" {
			continue
		}
		artifacts, err := componentFreshnessArtifacts(item.Path, r.Root, component, deps)
		if err != nil {
			r.logDebug("Freshness stamp skipped (component spec error)", logx.AttrScenario, item.Slug, "component", name, "error", err.Error())
			continue
		}
		for _, artifact := range artifacts {
			if _, statErr := deps.stat(artifact.ArtifactPath); statErr != nil {
				continue
			}
			if err := r.stampArtifactFreshness(artifact); err != nil {
				r.logDebug("Freshness stamp failed", logx.AttrScenario, item.Slug, "artifact", artifact.ArtifactPath, "error", err.Error())
			}
		}
	}
}

// binariesFreshnessArtifacts builds the freshness contract for a "binaries"
// setup check. Inputs are expressed relative to the repo root so the module
// directory and any local replace dirs share one collision-free namespace; the
// repo-root replace (`=> ../../..`) is deliberately excluded so an edit to an
// unrelated scenario never marks this artifact stale. _test.go files and the
// recorded manifest itself are excluded from the input set.
func binariesFreshnessArtifacts(appRoot, repoRoot string, check scenario.ConditionCheck, deps hostProbeDeps) ([]artifactFreshness, error) {
	root := repoRoot
	if strings.TrimSpace(root) == "" {
		root = appRoot
	}

	out := make([]artifactFreshness, 0, len(check.Targets))
	for _, target := range check.Targets {
		artifact := resolveCheckPath(appRoot, target)
		binaryDir := filepath.Dir(artifact)

		inputs, err := binariesFreshnessInputs(binaryDir, root, deps)
		if err != nil {
			return nil, err
		}

		// The canonical builder is the sole build-command authority. Freshness
		// keys therefore follow the same declared command that setup executes.
		keyInputs := goBuildKeyInputs(deps, strings.Join(goModuleBuildArgv(), " "))

		out = append(out, artifactFreshness{
			Target:       target,
			ArtifactPath: artifact,
			ManifestPath: cliutil.FreshnessManifestPath(artifact),
			CheckType:    "binaries",
			KeyInputs:    keyInputs,
			Spec: cliutil.FreshnessSpec{
				SourceRoot:      binaryDir,
				ContextRoot:     root,
				Inputs:          inputs,
				SkipFiles:       []string{filepath.Base(artifact)},
				SkipSuffixes:    []string{"_test.go", cliutil.FreshnessManifestSuffix},
				CaseInsensitive: deps.volumeCaseInsensitive(artifact),
			},
		})
	}
	return out, nil
}

// binariesFreshnessInputs resolves the freshness input set for one binary target,
// expressed relative to the repo root. It prefers the precise `go list -deps`
// import closure (only the repo-local packages the binary actually compiles, plus
// the local modules' go.mod/go.sum) and falls back to the static walk — the
// binary's own directory plus its sub-package replace dirs, excluding the
// repo-root replace — when go list is unavailable or the scenario is non-Go.
//
// The adapter is strictly more precise than the fallback in both directions: it
// drops unrelated dirs the binary never imports (no false positives) AND it
// includes genuinely-imported repo-root-replace packages (e.g. api-core/cli-core)
// that the fallback excludes wholesale, closing a false negative.
func binariesFreshnessInputs(binaryDir, repoRoot string, deps hostProbeDeps) ([]string, error) {
	if inputs, ok := goListFreshnessInputs(binaryDir, repoRoot, deps); ok {
		return inputs, nil
	}

	inputs := []string{relUnder(repoRoot, binaryDir)}
	replacePaths, err := localReplacePathsWithDeps(filepath.Join(binaryDir, "go.mod"), deps)
	if err != nil {
		return nil, err
	}
	for _, replacePath := range replacePaths {
		resolved := filepath.Clean(filepath.Join(binaryDir, replacePath))
		if filepath.Clean(repoRoot) == resolved {
			continue // repo-root replace: excluded here; the go list adapter handles it precisely
		}
		inputs = append(inputs, relUnder(repoRoot, resolved))
	}
	return inputs, nil
}

// goListPackage is the subset of `go list -json` output the freshness adapter
// consumes: the package's own directory and the local module it belongs to.
type goListPackage struct {
	Dir      string `json:"Dir"`
	Standard bool   `json:"Standard"`
	Module   *struct {
		Dir   string `json:"Dir"`
		GoMod string `json:"GoMod"`
	} `json:"Module"`
}

// goListFreshnessInputs derives the precise input set from the binary's import
// closure via the goListJSON seam. It returns (inputs, true) only when the seam
// is wired, the toolchain is present, the command succeeds, and at least one
// repo-local package is found; otherwise (nil, false) so the caller falls back to
// the static walk. Inputs are repo-relative: each imported repo-local package
// directory (the engine walks it, hashing *.go and skipping _test.go) plus the
// go.mod/go.sum of every repo-local module in the closure (so a dependency bump
// marks the binary stale). Standard-library and module-cache packages live outside
// the repo root and are dropped by the under-repo filter.
func goListFreshnessInputs(binaryDir, repoRoot string, deps hostProbeDeps) ([]string, bool) {
	if deps.goListJSON == nil || strings.TrimSpace(repoRoot) == "" {
		return nil, false
	}
	if _, err := deps.lookPath("go"); err != nil {
		return nil, false
	}
	raw, err := deps.goListJSON(binaryDir)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil, false
	}

	cleanRepo := filepath.Clean(repoRoot)
	seen := map[string]struct{}{}
	var inputs []string
	add := func(abs string) {
		if strings.TrimSpace(abs) == "" {
			return
		}
		clean := filepath.Clean(abs)
		if !pathUnderRoot(cleanRepo, clean) {
			return
		}
		rel := relUnder(cleanRepo, clean)
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		inputs = append(inputs, rel)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	for dec.More() {
		var pkg goListPackage
		if err := dec.Decode(&pkg); err != nil {
			return nil, false // malformed stream: distrust the whole result, fall back
		}
		if pkg.Standard {
			continue
		}
		add(pkg.Dir)
		if pkg.Module != nil && strings.TrimSpace(pkg.Module.GoMod) != "" {
			add(pkg.Module.GoMod)
			add(filepath.Join(filepath.Dir(pkg.Module.GoMod), "go.sum"))
		}
	}

	if len(inputs) == 0 {
		return nil, false
	}
	sort.Strings(inputs)
	return inputs, true
}

// pathUnderRoot reports whether target is root itself or lives beneath it.
func pathUnderRoot(root, target string) bool {
	if target == root {
		return true
	}
	return strings.HasPrefix(target, root+string(filepath.Separator))
}

// sharedPackageFreshnessKeyInputs adds the digest of each governed package's
// declared lifecycle outputs to the consuming UI's freshness contract. A
// file: dependency is copied by pnpm, so the package source directory alone is
// not a sufficient identity for the installed consumer.
func sharedPackageFreshnessKeyInputs(repoRoot, uiDir string) map[string]string {
	inputs := make(map[string]string)
	dependencies, err := sharedPackageDependencies(repoRoot, filepath.Join(uiDir, "package.json"))
	if err != nil {
		return inputs
	}
	for _, dependency := range dependencies {
		commands := append(append([]packagegov.CommandSpec{}, dependency.Generation...), dependency.Build...)
		for index, command := range commands {
			digest, err := sharedPackageOutputDigest(dependency.Root, command.Outputs)
			if err != nil {
				digest = "error"
			}
			name := command.Name
			if strings.TrimSpace(name) == "" {
				name = fmt.Sprintf("lifecycle-%d", index)
			}
			inputs["shared_package:"+dependency.Name+":"+name] = digest
		}
	}
	return inputs
}

func uiFileDependencyFreshnessInputs(repoRoot, sourceDir, dependencyName, dependencyRoot string, deps hostProbeDeps) []string {
	if dependencyName == "@vrooli/proto-types" {
		if inputs, ok := protoTypesFreshnessInputs(repoRoot, sourceDir, dependencyRoot, deps); ok {
			return inputs
		}
	}
	return []string{relUnder(repoRoot, dependencyRoot)}
}

var tsImportSourceRE = regexp.MustCompile(`\bfrom\s+["']([^"']+)["']|import\s*\(\s*["']([^"']+)["']\s*\)`)

func protoTypesFreshnessInputs(repoRoot, sourceDir, dependencyRoot string, deps hostProbeDeps) ([]string, bool) {
	seen := map[string]struct{}{}
	queue := []string{}
	add := func(path string) {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		queue = append(queue, path)
	}

	ok := true
	if err := deps.walkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == ".vite" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTypeScriptSource(path) {
			return nil
		}
		data, readErr := deps.readFile(path)
		if readErr != nil {
			return readErr
		}
		for _, source := range importSources(string(data)) {
			if !strings.HasPrefix(source, "@vrooli/proto-types/") {
				continue
			}
			resolved, resolvedOK := resolveProtoTypesImport(dependencyRoot, path, source, deps)
			if !resolvedOK {
				ok = false
				return errStopWalk
			}
			add(resolved)
		}
		return nil
	}); err != nil && !errors.Is(err, errStopWalk) {
		return nil, false
	}
	if !ok || len(queue) == 0 {
		return nil, false
	}

	for idx := 0; idx < len(queue); idx++ {
		current := queue[idx]
		data, err := deps.readFile(current)
		if err != nil {
			return nil, false
		}
		for _, source := range importSources(string(data)) {
			if !strings.HasPrefix(source, ".") && !strings.HasPrefix(source, "@vrooli/proto-types/") {
				continue
			}
			resolved, resolvedOK := resolveProtoTypesImport(dependencyRoot, current, source, deps)
			if !resolvedOK {
				return nil, false
			}
			add(resolved)
		}
	}

	inputs := make([]string, 0, len(seen))
	for path := range seen {
		inputs = append(inputs, relUnder(repoRoot, path))
	}
	sort.Strings(inputs)
	return inputs, true
}

func importSources(source string) []string {
	matches := tsImportSourceRE.FindAllStringSubmatch(source, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		for _, candidate := range match[1:] {
			if candidate != "" {
				out = append(out, candidate)
				break
			}
		}
	}
	return out
}

func resolveProtoTypesImport(dependencyRoot, importerPath, source string, deps hostProbeDeps) (string, bool) {
	switch {
	case strings.HasPrefix(source, "@vrooli/proto-types/"):
		rel := strings.TrimPrefix(source, "@vrooli/proto-types/")
		return resolveGeneratedTypeScriptModule(filepath.Join(dependencyRoot, filepath.FromSlash(rel)), deps)
	case strings.HasPrefix(source, "."):
		return resolveGeneratedTypeScriptModule(filepath.Join(filepath.Dir(importerPath), filepath.FromSlash(source)), deps)
	default:
		return "", false
	}
}

func resolveGeneratedTypeScriptModule(base string, deps hostProbeDeps) (string, bool) {
	for _, candidate := range []string{base, base + ".ts", base + ".js", filepath.Join(base, "index.ts"), filepath.Join(base, "index.js")} {
		info, err := deps.stat(candidate)
		if err == nil && !info.IsDir() {
			return filepath.Clean(candidate), true
		}
	}
	return "", false
}

func isTypeScriptSource(path string) bool {
	switch filepath.Ext(path) {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

// artifactVerdict is the structured freshness result for one artifact: whether
// it is stale, the machine cause, the offending input (when known), and a
// preformatted human reason for logs/back-compat callers.
type artifactVerdict struct {
	Target      string
	Stale       bool
	Cause       string // e.g. "content changed", "source newer", "missing artifact"
	File        string // offending input rel-path, when known
	HumanReason string // the human-facing reason string (cause + file)
}

// evaluateArtifactFreshness returns the structured freshness verdict for an
// artifact. A missing/non-runnable artifact is always stale. When a recorded
// manifest is present it is authoritative (stat-cache compare). When absent
// (bootstrap / post-clean) the verdict falls back to the Phase-0 mtime heuristic
// and, when it reports fresh, the manifest is stamped opportunistically so
// already-fresh artifacts adopt the engine without a needless rebuild.
func (r *Runner) evaluateArtifactFreshness(art artifactFreshness, deps hostProbeDeps) (artifactVerdict, error) {
	info, err := deps.stat(art.ArtifactPath)
	if err != nil || !isRunnableArtifact(art.CheckType, art.ArtifactPath, info, deps.artifactRecognizer()) {
		return artifactVerdict{Target: art.Target, Stale: true, Cause: "missing artifact", File: art.Target, HumanReason: "Missing artifact: " + art.Target}, nil
	}

	manifest, ok, err := cliutil.ReadFreshnessManifest(art.ManifestPath)
	if err != nil {
		// A corrupt manifest must not wedge startup: fall back to bootstrap.
		r.logDebug("Freshness manifest unreadable; bootstrapping", "artifact", art.ArtifactPath, "error", err.Error())
		ok = false
	}
	if ok {
		verdict, err := cliutil.EvaluateFreshness(art.Spec, manifest, art.KeyInputs)
		if err != nil {
			return artifactVerdict{}, err
		}
		if verdict.Stale {
			return artifactVerdict{Target: art.Target, Stale: true, Cause: verdict.Reason, File: verdict.File, HumanReason: freshnessReason(art.Target, verdict)}, nil
		}
		return artifactVerdict{Target: art.Target}, nil
	}

	// Bootstrap: no manifest yet. Use the legacy mtime walk to decide, and on
	// "fresh" stamp the manifest so the next check is manifest-authoritative.
	stale, file, reason, err := bootstrapMtimeStale(art, deps)
	if err != nil {
		return artifactVerdict{}, err
	}
	if stale {
		return artifactVerdict{Target: art.Target, Stale: true, Cause: "source newer", File: file, HumanReason: reason}, nil
	}
	if stampErr := r.stampArtifactFreshness(art); stampErr != nil {
		r.logDebug("Opportunistic freshness stamp failed", "artifact", art.ArtifactPath, "error", stampErr.Error())
	}
	return artifactVerdict{Target: art.Target}, nil
}

// stampArtifactFreshness computes and writes the recorded manifest for an
// artifact. Called after a successful setup build and opportunistically on a
// bootstrap-fresh verdict.
func (r *Runner) stampArtifactFreshness(art artifactFreshness) error {
	manifest, err := cliutil.ComputeFreshnessManifest(art.Spec, art.CheckType, art.KeyInputs, r.runtimeDeps().now().UnixNano())
	if err != nil {
		return err
	}
	return cliutil.WriteFreshnessManifest(art.ManifestPath, manifest)
}

func freshnessReason(target string, verdict cliutil.FreshnessVerdict) string {
	if verdict.File != "" {
		return fmt.Sprintf("%s stale (%s): %s", target, verdict.Reason, verdict.File)
	}
	return fmt.Sprintf("%s stale (%s)", target, verdict.Reason)
}

// relUnder renders target as a slash path relative to base, falling back to the
// cleaned absolute target when no relative form exists (e.g. different volume).
func relUnder(base, target string) string {
	if rel, err := filepath.Rel(base, target); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(target))
}

// uiBuildKeyInputs builds the non-file keyed inputs for a UI bundle: NODE_ENV
// (Vite emits different output for dev vs prod), the host Node major version
// when resolvable (esbuild/Vite output can differ across Node majors), and the
// perf-build channel. The node_major key is omitted when node is absent
// (omit-on-unknown).
//
// build_mode is keyed because the profile channel emits a materially different
// bundle from byte-identical sources — react-dom aliased to the profiling
// build, identifier names kept through minification. Without it, a scenario
// already built in the default channel reads as fresh when the profile channel
// is requested, the rebuild is skipped, and the served bundle carries no
// instrumentation. performance-health then correctly reports the capture as
// unavailable, which looks like a broken browser rather than a skipped build.
func uiBuildKeyInputs(deps hostProbeDeps) map[string]string {
	keyInputs := map[string]string{"node_env": nodeEnvOrDefault(deps)}
	if deps.nodeVersion != nil {
		if major := strings.TrimSpace(deps.nodeVersion()); major != "" {
			keyInputs["node_major"] = major
		}
	}
	if mode := NormalizeBuildMode(deps.getenv(BuildModeEnvVar)); mode != "" {
		keyInputs["build_mode"] = mode
	}
	return keyInputs
}

func nodeEnvOrDefault(deps hostProbeDeps) string {
	if v := strings.TrimSpace(deps.getenv("NODE_ENV")); v != "" {
		return v
	}
	return "development"
}

var (
	goToolchainOnce  sync.Once
	goToolchainValue string
	goEnvOnce        sync.Once
	goEnvValue       map[string]string
	nodeVersionOnce  sync.Once
	nodeVersionValue string
)

// defaultGoEnv resolves the requested `go env` determinants to their effective
// values, cached once per process (the toolchain config does not change mid-run).
// It runs `go env -json` once, caches the full map, then returns only the
// requested keys that have a non-empty value. When go is unavailable the cache is
// an empty map, so every key is omitted (omit-on-unknown). This is the resolved
// build environment — `go env` reports the value the toolchain will actually use
// (defaults + overrides), which is what determines the compiled output, unlike a
// raw os.Getenv that is blank when a default would still apply.
func defaultGoEnv(keys ...string) map[string]string {
	goEnvOnce.Do(func() {
		goEnvValue = map[string]string{}
		goBin, err := exec.LookPath("go")
		if err != nil {
			return
		}
		out, err := exec.Command(goBin, "env", "-json").Output()
		if err != nil {
			return
		}
		var parsed map[string]string
		if err := json.Unmarshal(out, &parsed); err != nil {
			return
		}
		goEnvValue = parsed
	})
	resolved := map[string]string{}
	for _, k := range keys {
		if v := strings.TrimSpace(goEnvValue[k]); v != "" {
			resolved[k] = v
		}
	}
	return resolved
}

// defaultNodeVersion returns the host Node.js major version (e.g. "20"), cached
// once per process. Empty when node is absent or its output is unparseable
// (omit-on-unknown), in which case the node_major key is dropped.
func defaultNodeVersion() string {
	nodeVersionOnce.Do(func() {
		nodeBin, err := exec.LookPath("node")
		if err != nil {
			return
		}
		out, err := exec.Command(nodeBin, "--version").Output()
		if err != nil {
			return
		}
		nodeVersionValue = nodeMajor(string(out))
	})
	return nodeVersionValue
}

// nodeMajor extracts the major component from a `node --version` string such as
// "v20.11.0" → "20". Returns "" when no leading numeric major is present.
func nodeMajor(raw string) string {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '.'); i >= 0 {
		v = v[:i]
	}
	v = strings.TrimSpace(v)
	for _, r := range v {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return v
}

// goEnvDeterminants is the curated allowlist of output-determining `go env`
// vars folded into the binaries digest. Each entry changes the compiled bytes:
// GOOS/GOARCH (and sub-arch GOAMD64/GOARM) select the target; CGO_ENABLED changes
// linkage; GOFLAGS/GOEXPERIMENT alter codegen toolchain-wide. Volatile/irrelevant
// vars are deliberately absent — keying them would manufacture false positives.
var goEnvDeterminants = []string{
	"GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOEXPERIMENT", "GOAMD64", "GOARM",
}

// goBuildKeyInputs builds the non-file keyed inputs for a Go binary target: the
// resolved build environment (curated `go env` determinants + toolchain version)
// plus the build-command flags (-tags/-ldflags/-trimpath) parsed from the
// scenario's setup `go build` step. Every value is output-determining; an
// unresolvable determinant is omitted, never defaulted, so the change can only
// close false negatives, never widen the false-positive surface. Keys are the
// lower-snake contract names (goos, goarch, cgo_enabled, …).
func goBuildKeyInputs(deps hostProbeDeps, buildCmd string) map[string]string {
	keyInputs := map[string]string{}
	if deps.goEnv != nil {
		for k, v := range deps.goEnv(goEnvDeterminants...) {
			if val := strings.TrimSpace(v); val != "" {
				keyInputs[strings.ToLower(k)] = val
			}
		}
	}
	if tc := hostGoToolchain(deps); tc != "" {
		keyInputs["toolchain"] = tc
	}
	for k, v := range parseBuildCommandFlags(buildCmd) {
		keyInputs[k] = v
	}
	return keyInputs
}

// parseBuildCommandFlags extracts the output-determining flags from a `go build`
// command line: -tags, -ldflags, -trimpath (both `-flag value` and `-flag=value`
// forms, quote-aware). Tag lists are normalized (split on comma/space, sorted,
// comma-joined) so ordering noise never flaps the digest. A flag is keyed only
// when present with a non-empty value; on any parse ambiguity the flag is omitted
// (omit-on-unknown). Returns an empty map for a command with no recognized flags.
func parseBuildCommandFlags(buildCmd string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(buildCmd) == "" {
		return out
	}
	tokens := tokenizeCommand(buildCmd)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		value := func(inline string) (string, bool) {
			if inline != "" {
				return inline, true
			}
			if i+1 < len(tokens) {
				i++
				return tokens[i], true
			}
			return "", false
		}
		switch {
		case tok == "-trimpath" || tok == "--trimpath":
			out["trimpath"] = "true"
		case strings.HasPrefix(tok, "-trimpath="):
			if v := strings.TrimSpace(strings.TrimPrefix(tok, "-trimpath=")); v != "" {
				out["trimpath"] = v
			}
		case tok == "-tags" || tok == "--tags":
			if v, ok := value(""); ok {
				if nv := normalizeTagList(v); nv != "" {
					out["build_tags"] = nv
				}
			}
		case strings.HasPrefix(tok, "-tags="):
			if nv := normalizeTagList(strings.TrimPrefix(tok, "-tags=")); nv != "" {
				out["build_tags"] = nv
			}
		case tok == "-ldflags" || tok == "--ldflags":
			if v, ok := value(""); ok {
				if tv := strings.TrimSpace(v); tv != "" {
					out["ldflags"] = tv
				}
			}
		case strings.HasPrefix(tok, "-ldflags="):
			if tv := strings.TrimSpace(strings.TrimPrefix(tok, "-ldflags=")); tv != "" {
				out["ldflags"] = tv
			}
		}
	}
	return out
}

// normalizeTagList canonicalizes a Go build-tag list so "b,a" and "a b" hash
// identically: split on comma/whitespace, drop empties, sort, comma-join.
func normalizeTagList(raw string) string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(fields) == 0 {
		return ""
	}
	sort.Strings(fields)
	return strings.Join(fields, ",")
}

// tokenizeCommand splits a command line into tokens, honoring single and double
// quotes so a quoted flag value (e.g. -ldflags "-s -w") stays one token. It is
// deliberately minimal — no variable/glob expansion — because it only needs to
// recognize the -tags/-ldflags/-trimpath shape; anything it cannot cleanly split
// degrades to omitting the flag key.
func tokenizeCommand(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range cmd {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// hostGoToolchain returns the host Go toolchain version string (e.g.
// "go version go1.25.0 linux/amd64"), best-effort and cached for the process.
// Empty when the go tool is unavailable, in which case the key is simply omitted
// from the digest (no false staleness). This is the toolchain that builds
// scenarios, so an upgrade flips the key and forces a rebuild of byte-identical
// sources — closing the most common false-negative.
func hostGoToolchain(deps hostProbeDeps) string {
	goToolchainOnce.Do(func() {
		goBin, err := deps.lookPath("go")
		if err != nil {
			return
		}
		out, err := exec.Command(goBin, "version").Output()
		if err != nil {
			return
		}
		goToolchainValue = strings.TrimSpace(string(out))
	})
	return goToolchainValue
}

// isRunnableArtifact reports whether the stat'd path is a usable build artifact
// for the given check type. For "binaries" it must be a runnable compiled
// artifact per the OS artifact-recognition seam (exec bit on Unix, executable
// extension on Windows); other artifacts (e.g. a ui-bundle index.html) need only
// be a regular file. Decision logic carries no runtime.GOOS branch: the per-OS
// rule lives in the recognize seam, and an unavailable probe (Known:false)
// degrades to "assume runnable" so a probe gap never falsely condemns a built
// artifact as missing.
func isRunnableArtifact(checkType, path string, info os.FileInfo, recognize func(string, os.FileInfo) artifactEvidence) bool {
	if info == nil {
		return false
	}
	switch checkType {
	case "binaries":
		ev := recognize(path, info)
		if !ev.Known {
			return true // probe unavailable: assume runnable (degrade to reuse)
		}
		return ev.Runnable
	default:
		return !info.IsDir()
	}
}

// bootstrapMtimeStale is the legacy mtime heuristic used only when no recorded
// manifest exists yet. It walks the artifact's declared input dirs (already
// excluding the repo-root replace, _test.go, and the manifest file) for any
// file newer than the artifact. Once a manifest is stamped this path is never
// taken again for that artifact.
// bootstrapMtimeStale returns (stale, offendingFileRel, humanReason, err).
func bootstrapMtimeStale(art artifactFreshness, deps hostProbeDeps) (bool, string, string, error) {
	root := art.Spec.ContextRoot
	if strings.TrimSpace(root) == "" {
		root = art.Spec.SourceRoot
	}
	include := func(path string, _ fs.DirEntry) bool {
		rel := filepath.ToSlash(path)
		if strings.HasSuffix(rel, "_test.go") || strings.HasSuffix(rel, cliutil.FreshnessManifestSuffix) {
			return false
		}
		return true
	}
	for _, input := range art.Spec.Inputs {
		resolved := filepath.Join(root, filepath.FromSlash(input))
		if offender, found := firstFileNewerWithDeps(resolved, art.ArtifactPath, deps, include); found {
			file := relForReason(root, offender)
			return true, file, fmt.Sprintf("%s stale (source newer): %s", art.Target, file), nil
		}
	}
	return false, "", "", nil
}
