package lifecycle

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// BuilderSpec is the declarative build contract shared by Tier 1 freshness,
// setup derivation, and later Tier 2 projection. Command slices are argv
// templates; the registry never invokes them by itself.
type BuilderSpec struct {
	Kind          string
	Inputs        []string
	DigestKeys    []string
	DefaultOutput string
	Environment   map[string]string
	Install       []string
	// InstallInputs are build-dir-relative files/globs whose content digest
	// controls whether Install must run. Empty means always install.
	InstallInputs []string
	// InstallStateFile is a builder-owned marker proving the install completed.
	// Empty means the lifecycle uses its sidecar digest marker only.
	InstallStateFile string
	// MemoryReservationBytes is the conservative peak reservation used by
	// dependency fan-out admission. Zero uses the global fallback budget.
	MemoryReservationBytes int64
	// StagedOutput requests an atomic replacement of the declared artifact.
	// StageArgs are appended to BuildArgv with {stage_output_dir} expanded.
	StagedOutput bool
	StageArgs    []string
	// StageDirectory means the declared output is a directory artifact (for
	// example a Vite dist tree) even when the manifest names a file inside it.
	StageDirectory bool
	Build          []string
	// ProfileBuild is the perf-build channel argv, selected when
	// VROOLI_BUILD_MODE=profile. Empty means the builder has no separate
	// channel and Build is used for every mode. Keeping the channel as its
	// own argv is what lets the selection stay a Go decision instead of a
	// shell conditional inside a package script — see BuildArgv.
	ProfileBuild             []string
	FollowsWorkspaceFileDeps bool
	Reserved                 bool
	freshness                func(string, string, scenario.Component, hostProbeDeps) ([]artifactFreshness, error)
}

// BuildModeEnvVar names the perf-build channel selector. performance-health's
// capture path sets it on `vrooli scenario restart` so a profile bundle is
// produced through the standard lifecycle.
const BuildModeEnvVar = "VROOLI_BUILD_MODE"

// buildModeProfile is the only recognised non-default channel. An unrecognised
// value falls back to the default build rather than selecting a package script
// that is not guaranteed to exist.
const buildModeProfile = "profile"

const goModuleDefaultOutput = "{dir}/{scenario}-api{{ext}}"

var builderRegistry = map[string]BuilderSpec{
	"go_module": {
		Kind:                   "go_module",
		Inputs:                 []string{"**/*.go", "go.mod", "go.sum"},
		DigestKeys:             []string{"go_toolchain", "GOOS", "GOARCH", "CGO_ENABLED", "GOAMD64", "GOARM64", "GOFLAGS", "build_tags"},
		DefaultOutput:          goModuleDefaultOutput,
		Environment:            map[string]string{"GOWORK": "off"},
		Install:                []string{"go", "mod", "download"},
		InstallInputs:          []string{"go.mod", "go.sum"},
		MemoryReservationBytes: 394 << 20,
		StagedOutput:           true,
		Build:                  goModuleBuildArgv(),
		freshness:              goModuleComponentFreshness,
	},
	"pnpm_vite": {
		Kind:                     "pnpm_vite",
		Inputs:                   []string{"{src}/**", "package.json", "vite.config.*", "tsconfig.json", "index.html"},
		DigestKeys:               []string{"NODE_ENV", "node_major", "pnpm_version", "build_mode"},
		DefaultOutput:            "{dir}/dist/index.html",
		Install:                  []string{"pnpm", "install", "--ignore-workspace"},
		InstallInputs:            []string{"package.json", "pnpm-lock.yaml"},
		InstallStateFile:         "node_modules/.modules.yaml",
		MemoryReservationBytes:   752 << 20,
		StagedOutput:             true,
		StageDirectory:           true,
		StageArgs:                []string{"--outDir", "{stage_output_dir}"},
		Build:                    []string{"pnpm", "run", "build"},
		ProfileBuild:             []string{"pnpm", "run", "build:profile"},
		FollowsWorkspaceFileDeps: true,
		freshness:                pnpmViteComponentFreshness,
	},
	"node_bundle": {
		Kind:                     "node_bundle",
		Inputs:                   []string{"{src}/**", "package.json", "tsconfig.json"},
		DigestKeys:               []string{"NODE_ENV", "node_major", "pnpm_version", "build_mode"},
		Install:                  []string{"pnpm", "install", "--ignore-workspace"},
		InstallInputs:            []string{"package.json", "pnpm-lock.yaml"},
		InstallStateFile:         "node_modules/.modules.yaml",
		MemoryReservationBytes:   752 << 20,
		Build:                    []string{"pnpm", "run", "build"},
		ProfileBuild:             []string{"pnpm", "run", "build:profile"},
		FollowsWorkspaceFileDeps: true,
		freshness:                pnpmViteComponentFreshness,
	},
	// reuse remains in the registry vocabulary for schema compatibility with
	// older manifests. Reuse components never execute a builder; componentArtifact
	// resolves their target through Build.Reuse before consulting this entry.
	"reuse": {
		Kind: "reuse",
	},
	"python_uv": {
		Kind:     "python_uv",
		Reserved: true,
	},
	"cargo": {
		Kind:     "cargo",
		Reserved: true,
	},
}

func goModuleBuildArgv() []string {
	return []string{"go", "build", "-o", "{output}", "{entry}"}
}

// NormalizeBuildMode folds a raw VROOLI_BUILD_MODE value to the recognised
// channel name, or "" for the default channel. Unrecognised values normalize to
// "" rather than erroring: the variable is operator-set, and a typo should
// produce the ordinary build, never a lookup for a package script that no
// scenario is required to declare.
func NormalizeBuildMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), buildModeProfile) {
		return buildModeProfile
	}
	return ""
}

// BuildArgv returns the build argv for the requested channel. The selection is
// a Go decision on a declared argv, never a shell conditional inside the
// package script — that is what keeps the perf-build channel reachable on
// Windows, where package scripts run through cmd.exe.
//
// A builder with no ProfileBuild, or a mode with no matching channel, falls
// back to Build.
func (s BuilderSpec) BuildArgv(mode string) []string {
	if NormalizeBuildMode(mode) == buildModeProfile && len(s.ProfileBuild) > 0 {
		return append([]string(nil), s.ProfileBuild...)
	}
	return append([]string(nil), s.Build...)
}

// BuildModeForEnv resolves the requested build channel. A lifecycle override
// wins over the process environment so a caller can pin the channel explicitly;
// otherwise the value inherited from the operator's shell (or from
// performance-health's `vrooli scenario restart`) applies.
func BuildModeForEnv(env map[string]string, getenv func(string) string) string {
	if raw, ok := env[BuildModeEnvVar]; ok {
		return NormalizeBuildMode(raw)
	}
	if getenv == nil {
		return ""
	}
	return NormalizeBuildMode(getenv(BuildModeEnvVar))
}

// BuilderRegistry returns a defensive copy so callers cannot mutate the
// process-wide contract.
func BuilderRegistry() map[string]BuilderSpec {
	out := make(map[string]BuilderSpec, len(builderRegistry))
	for key, spec := range builderRegistry {
		spec.Inputs = append([]string(nil), spec.Inputs...)
		spec.DigestKeys = append([]string(nil), spec.DigestKeys...)
		spec.Install = append([]string(nil), spec.Install...)
		spec.InstallInputs = append([]string(nil), spec.InstallInputs...)
		spec.Build = append([]string(nil), spec.Build...)
		spec.ProfileBuild = append([]string(nil), spec.ProfileBuild...)
		spec.StageArgs = append([]string(nil), spec.StageArgs...)
		spec.Environment = cloneStringMap(spec.Environment)
		out[key] = spec
	}
	return out
}

// RegisteredBuilderKinds is the stable vocabulary consumed by validation and
// documentation surfaces.
func RegisteredBuilderKinds() []string {
	kinds := make([]string, 0, len(builderRegistry))
	for kind := range builderRegistry {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// ResolveComponentArgv expands Tier 1 component path placeholders without
// invoking a shell. {{bin.<component>}} resolves through build.output (or the
// builder default), including build.reuse chains; {{ext}} is .exe on Windows.
func ResolveComponentArgv(argv []string, scenarioRoot, scenarioName string, components map[string]scenario.Component) ([]string, error) {
	return resolveComponentArgvForOS(argv, scenarioRoot, scenarioName, components, runtime.GOOS)
}

func componentWorkingDir(root, cwd string) (string, error) {
	root = filepath.Clean(root)
	if strings.TrimSpace(cwd) == "" {
		return root, nil
	}
	target := filepath.Clean(filepath.Join(root, filepath.FromSlash(cwd)))
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("cwd %q escapes the scenario root", cwd)
	}
	return target, nil
}

func resolveComponentArgvForOS(argv []string, scenarioRoot, scenarioName string, components map[string]scenario.Component, goos string) ([]string, error) {
	out := make([]string, len(argv))
	for index, value := range argv {
		resolved, err := resolveComponentValue(value, scenarioRoot, scenarioName, components, goos)
		if err != nil {
			return nil, fmt.Errorf("argv[%d]: %w", index, err)
		}
		out[index] = resolved
	}
	return out, nil
}

func resolveComponentValue(value, scenarioRoot, scenarioName string, components map[string]scenario.Component, goos string) (string, error) {
	ext := ""
	if strings.EqualFold(strings.TrimSpace(goos), "windows") {
		ext = ".exe"
	}
	value = strings.ReplaceAll(value, "{{ext}}", ext)
	for {
		start := strings.Index(value, "{{bin.")
		if start < 0 {
			return value, nil
		}
		endOffset := strings.Index(value[start:], "}}")
		if endOffset < 0 {
			return "", fmt.Errorf("unterminated component binary placeholder in %q", value)
		}
		end := start + endOffset + 2
		name := strings.TrimSpace(value[start+len("{{bin.") : start+endOffset])
		artifact, err := componentArtifact(name, scenarioRoot, scenarioName, components, goos, map[string]bool{})
		if err != nil {
			return "", err
		}
		value = value[:start] + artifact + value[end:]
	}
}

func componentArtifact(name, scenarioRoot, scenarioName string, components map[string]scenario.Component, goos string, visiting map[string]bool) (string, error) {
	component, ok := components[name]
	if !ok {
		return "", fmt.Errorf("component %q does not exist", name)
	}
	if visiting[name] {
		return "", fmt.Errorf("component build.reuse cycle at %q", name)
	}
	visiting[name] = true
	defer delete(visiting, name)

	if reused := strings.TrimSpace(component.Build.Reuse); reused != "" {
		return componentArtifact(reused, scenarioRoot, scenarioName, components, goos, visiting)
	}
	spec, ok := builderRegistry[strings.TrimSpace(component.Build.Kind)]
	if !ok || spec.Reserved {
		return "", fmt.Errorf("component %q has no resolvable builder kind %q", name, component.Build.Kind)
	}
	return componentArtifactWithOutput(name, scenarioRoot, scenarioName, component, spec, component.Build.Output, goos)
}

func componentArtifactWithOutput(name, scenarioRoot, scenarioName string, component scenario.Component, spec BuilderSpec, declaredOutput, goos string) (string, error) {
	output := strings.TrimSpace(declaredOutput)
	if output == "" {
		output = spec.DefaultOutput
	}
	if output == "" {
		return "", fmt.Errorf("component %q build output is required", name)
	}
	ext := ""
	if strings.EqualFold(strings.TrimSpace(goos), "windows") {
		ext = ".exe"
	}
	output = strings.NewReplacer(
		"{dir}", strings.TrimSpace(component.Build.Dir),
		"{scenario}", strings.TrimSpace(scenarioName),
		"{component}", name,
		"{{ext}}", ext,
	).Replace(output)
	if filepath.IsAbs(output) {
		return filepath.Clean(output), nil
	}
	return filepath.Join(scenarioRoot, filepath.FromSlash(output)), nil
}

type componentBuildTarget struct {
	Entry  string
	Output string
}

// componentBuildTargets returns the primary component artifact followed by
// explicitly declared secondary artifacts. Keeping this expansion here makes
// run.argv resolution, setup builds, and freshness use the same artifact
// contract instead of each inferring outputs independently.
func componentBuildTargets(name, scenarioRoot, scenarioName string, component scenario.Component, spec BuilderSpec, goos string) ([]componentBuildTarget, error) {
	if strings.TrimSpace(spec.Kind) == "" || spec.Reserved {
		return nil, fmt.Errorf("component %q has no resolvable builder kind %q", name, component.Build.Kind)
	}
	primary, err := componentArtifactWithOutput(name, scenarioRoot, scenarioName, component, spec, component.Build.Output, goos)
	if err != nil {
		return nil, err
	}
	entry := strings.TrimSpace(component.Build.Entry)
	if entry == "" {
		entry = "."
	}
	targets := []componentBuildTarget{{Entry: entry, Output: primary}}
	for index, extra := range component.Build.Outputs {
		if strings.TrimSpace(extra.Output) == "" {
			return nil, fmt.Errorf("component %q build.outputs[%d].output is required", name, index)
		}
		extraOutput, err := componentArtifactWithOutput(name, scenarioRoot, scenarioName, component, spec, extra.Output, goos)
		if err != nil {
			return nil, fmt.Errorf("component %q build.outputs[%d]: %w", name, index, err)
		}
		extraEntry := strings.TrimSpace(extra.Entry)
		if extraEntry == "" {
			extraEntry = "."
		}
		targets = append(targets, componentBuildTarget{Entry: extraEntry, Output: extraOutput})
	}
	return targets, nil
}

func componentFreshnessArtifacts(appRoot, repoRoot string, component scenario.Component, deps hostProbeDeps) ([]artifactFreshness, error) {
	spec, ok := builderRegistry[strings.TrimSpace(component.Build.Kind)]
	if !ok || spec.Reserved || spec.freshness == nil {
		return nil, fmt.Errorf("build kind %q has no freshness implementation", component.Build.Kind)
	}
	return spec.freshness(appRoot, repoRoot, component, deps)
}

func goModuleComponentFreshness(appRoot, repoRoot string, component scenario.Component, deps hostProbeDeps) ([]artifactFreshness, error) {
	targets, err := componentBuildTargets("component", appRoot, filepath.Base(appRoot), component, BuilderSpec{
		Kind:          "go_module",
		DefaultOutput: goModuleDefaultOutput,
	}, runtimeGOOS())
	if err != nil {
		return nil, err
	}
	declared := make([]string, 0, len(targets))
	for _, target := range targets {
		rel, err := filepath.Rel(appRoot, target.Output)
		if err != nil {
			return nil, err
		}
		declared = append(declared, filepath.ToSlash(rel))
	}
	return binariesFreshnessArtifacts(appRoot, repoRoot, scenario.ConditionCheck{Type: "binaries", Targets: declared}, deps)
}

func pnpmViteComponentFreshness(appRoot, repoRoot string, component scenario.Component, deps hostProbeDeps) ([]artifactFreshness, error) {
	dir := strings.TrimSpace(component.Build.Dir)
	if dir == "" {
		return nil, fmt.Errorf("build.dir is required")
	}
	output := strings.TrimSpace(component.Build.Output)
	if output == "" {
		output = filepath.ToSlash(filepath.Join(dir, "dist", "index.html"))
	}
	root := repoRoot
	if strings.TrimSpace(root) == "" {
		root = appRoot
	}
	artifactPath := resolveCheckPath(appRoot, output)
	buildDir := resolveCheckPath(appRoot, dir)
	sourceDir := filepath.Join(buildDir, "src")
	inputs := []string{relUnder(root, sourceDir)}
	for _, file := range []string{"package.json", "vite.config.ts", "vite.config.js", "tsconfig.json", "index.html"} {
		inputs = append(inputs, relUnder(root, filepath.Join(buildDir, file)))
	}
	if specs, err := fileDependenciesWithDeps(filepath.Join(buildDir, "package.json"), deps); err == nil {
		for _, spec := range specs {
			resolved := resolveCheckPath(buildDir, strings.TrimPrefix(spec.Spec, "file:"))
			inputs = append(inputs, uiFileDependencyFreshnessInputs(root, sourceDir, spec.Name, resolved, deps)...)
		}
	}
	keyInputs := uiBuildKeyInputs(deps)
	for key, value := range sharedPackageFreshnessKeyInputs(repoRoot, buildDir) {
		keyInputs[key] = value
	}
	return []artifactFreshness{{
		Target:       output,
		ArtifactPath: artifactPath,
		ManifestPath: cliutil.FreshnessManifestPath(artifactPath),
		CheckType:    component.Build.Kind,
		KeyInputs:    keyInputs,
		Spec: cliutil.FreshnessSpec{
			SourceRoot:      buildDir,
			ContextRoot:     root,
			Inputs:          inputs,
			SkipSuffixes:    []string{cliutil.FreshnessManifestSuffix},
			CaseInsensitive: deps.volumeCaseInsensitive(artifactPath),
		},
	}}, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
