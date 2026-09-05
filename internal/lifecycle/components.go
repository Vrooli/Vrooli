package lifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/buildflags"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	goModuleMemoryReservation = 394 << 20
	pnpmMemoryReservation     = 752 << 20
	componentPlaceholderEnd   = 2
)

// BuilderSpec is the declarative build contract shared by Tier 1 freshness,
// setup derivation, and later Tier 2 projection. Command slices are argv
// templates; the registry never invokes them by itself.
type BuilderSpec struct {
	Kind            string
	Inputs          []string
	SkipSuffixes    []string
	DigestKeys      []string
	KeyResolver     string
	ClosureResolver string
	DefaultOutput   string
	Environment     map[string]string
	Install         []string
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
		SkipSuffixes:           []string{"_test.go"},
		KeyResolver:            "go_env",
		DigestKeys:             []string{"toolchain", "goos", "goarch", "cgo_enabled", "goamd64", "goarm", "goflags"},
		ClosureResolver:        "go_list",
		DefaultOutput:          goModuleDefaultOutput,
		Environment:            map[string]string{"GOWORK": "off"},
		Install:                []string{"go", "mod", "download"},
		InstallInputs:          []string{"go.mod", "go.sum"},
		MemoryReservationBytes: goModuleMemoryReservation,
		StagedOutput:           true,
		Build:                  goModuleBuildArgv(),
	},
	"pnpm_vite": {
		Kind:                     "pnpm_vite",
		Inputs:                   []string{"{src}/**", "package.json", "vite.config.*", "tsconfig.json", "index.html"},
		KeyResolver:              "node",
		DigestKeys:               []string{"NODE_ENV", "node_major", "build_mode"},
		DefaultOutput:            "{dir}/dist/index.html",
		Install:                  []string{"pnpm", "install", "--ignore-workspace"},
		InstallInputs:            []string{"package.json", "pnpm-lock.yaml"},
		InstallStateFile:         "node_modules/.modules.yaml",
		MemoryReservationBytes:   pnpmMemoryReservation,
		StagedOutput:             true,
		StageDirectory:           true,
		StageArgs:                []string{"--outDir", "{stage_output_dir}"},
		Build:                    []string{"pnpm", "run", "build"},
		ProfileBuild:             []string{"pnpm", "run", "build:profile"},
		FollowsWorkspaceFileDeps: true,
	},
	"node_bundle": {
		Kind:                     "node_bundle",
		Inputs:                   []string{"{src}/**", "package.json", "tsconfig.json"},
		SkipSuffixes:             []string{"_test.ts", "_test.tsx"},
		DigestKeys:               []string{"NODE_ENV", "node_major", "build_mode"},
		KeyResolver:              "node",
		DefaultOutput:            "{dir}/dist/index.js",
		Install:                  []string{"pnpm", "install", "--ignore-workspace"},
		InstallInputs:            []string{"package.json", "pnpm-lock.yaml"},
		InstallStateFile:         "node_modules/.modules.yaml",
		MemoryReservationBytes:   pnpmMemoryReservation,
		Build:                    []string{"pnpm", "run", "build"},
		ProfileBuild:             []string{"pnpm", "run", "build:profile"},
		FollowsWorkspaceFileDeps: true,
	},
	"python_uv": {
		Kind:                   "python_uv",
		Inputs:                 []string{"{src}/**", "pyproject.toml", "uv.lock"},
		SkipSuffixes:           []string{"_test.py"},
		DigestKeys:             []string{"python_version", "uv_version"},
		KeyResolver:            "python",
		DefaultOutput:          "{dir}/.venv/pyvenv.cfg",
		Install:                []string{"uv", "sync", "--frozen"},
		InstallInputs:          []string{"pyproject.toml", "uv.lock"},
		InstallStateFile:       ".venv/pyvenv.cfg",
		Build:                  []string{"uv", "sync", "--frozen"},
		MemoryReservationBytes: goModuleMemoryReservation,
	},
	"cargo": {
		Kind:     "cargo",
		Reserved: true,
	},
}

func goModuleBuildArgv() []string {
	return []string{"go", "build", "-trimpath", "-o", "{output}", "{entry}"}
}

func goModuleBuildArgvForRoot(root string) ([]string, error) {
	argv := goModuleBuildArgv()
	policy, err := buildflags.Load(root)
	if err != nil {
		if os.IsNotExist(err) {
			return argv, nil
		}
		return nil, err
	}
	flags := policy.For("scenario")
	insertAt := len(argv)
	for index, value := range argv {
		if value == "-o" {
			insertAt = index
			break
		}
	}
	for _, flag := range flags {
		found := false
		for _, value := range argv {
			if value == flag {
				found = true
				break
			}
		}
		if !found {
			argv = append(argv, "")
			copy(argv[insertAt+1:], argv[insertAt:])
			argv[insertAt] = flag
			insertAt++
		}
	}
	return argv, nil
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
		spec.SkipSuffixes = append([]string(nil), spec.SkipSuffixes...)
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
	slices.Sort(kinds)
	return kinds
}

// ResolveComponentArgv expands Tier 1 component path placeholders without
// invoking a shell. {{bin.<component>}} resolves through build.output (or the
// builder default); {{ext}} is .exe on Windows.
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
	if strings.EqualFold(strings.TrimSpace(goos), string(hostreqspec.PlatformWindows)) {
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
		end := start + endOffset + componentPlaceholderEnd
		name := strings.TrimSpace(value[start+len("{{bin.") : start+endOffset])
		artifact, err := componentArtifact(name, scenarioRoot, scenarioName, components, goos)
		if err != nil {
			return "", err
		}
		value = value[:start] + artifact + value[end:]
	}
}

func componentArtifact(name, scenarioRoot, scenarioName string, components map[string]scenario.Component, goos string) (string, error) {
	component, ok := components[name]
	if !ok {
		return "", fmt.Errorf("component %q does not exist", name)
	}
	if reused := strings.TrimSpace(component.Build.Reuse); reused != "" {
		return "", fmt.Errorf("component %q uses deprecated build.reuse=%q; declare the artifact once and reference it by path", name, reused)
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
	if strings.EqualFold(strings.TrimSpace(goos), string(hostreqspec.PlatformWindows)) {
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
	if !ok || spec.Reserved {
		return nil, fmt.Errorf("build kind %q has no freshness implementation", component.Build.Kind)
	}
	return builderFreshnessContext(context.Background(), appRoot, repoRoot, filepath.Base(appRoot), "component", component, spec, deps)
}

func componentFreshnessArtifactsContextWithName(ctx context.Context, appRoot, repoRoot, scenarioName, componentName string, component scenario.Component, deps hostProbeDeps) ([]artifactFreshness, error) {
	spec, ok := builderRegistry[strings.TrimSpace(component.Build.Kind)]
	if !ok || spec.Reserved {
		return nil, fmt.Errorf("build kind %q has no freshness implementation", component.Build.Kind)
	}
	return builderFreshnessContext(ctx, appRoot, repoRoot, scenarioName, componentName, component, spec, deps)
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
