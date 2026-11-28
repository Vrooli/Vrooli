package installgateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	agentinstall "github.com/vrooli/vrooli/packages/harness-lifecycle"
)

const (
	DiskJobPlanningCeilingBytes int64 = 25 << 30
	DiskJobFloorBytes           int64 = 120 << 30
)

// DiskPathRequirement is a destination or transient directory that can grow
// while a governed package operation runs. The capacity owner canonicalizes
// paths and merges aliases by filesystem identity before admitting the job.
type DiskPathRequirement struct {
	Path      string
	Role      string
	PeakBytes int64
}

// DiskAdmissionPlan describes the paths and measured peak for a supported
// install shape. Package manager caches and registry selection are pinned by
// the executor so local configuration cannot silently redirect writes beyond
// the lease.
type DiskAdmissionPlan struct {
	Paths             []DiskPathRequirement
	CachePath         string
	BuildCachePath    string
	Manager           string
	Unsupported       string
	ProbeBytes        int64
	PythonExecutable  string
	PythonWheelURL    string
	PythonWheelSHA256 string
	PythonWheelBytes  int64
	PythonTarget      string
	PythonStage       string
	GoArchiveBytes    int64
	GoExpandedBytes   int64
	GoModuleCount     int64
	// WritesInputsOnly is set when apply downloads nothing and writes no package
	// store: it changes manifests, lockfiles and module-cache bookkeeping only (a
	// JavaScript change whose link is skipped, a Go install whose modules are
	// all cached or used in place). The capacity owner admits such a plan as a
	// bounded metadata job instead of holding it to the install-sized floor.
	WritesInputsOnly bool
	// JSStage is the resolved, unpublished pnpm change that admission sized.
	// Apply publishes exactly this stage; every caller must Release the plan.
	JSStage *JSStage
	// PythonStage is the bounded wheel-lock resolution that apply publishes.
	PythonLockStage *PythonLockStage
	JS              JSMeasurement
	// Notes are reviewer-facing facts discovered while planning.
	Notes []string
}

// Release frees resources a plan holds (staged JavaScript or Python resolution).
// It is safe to call more than once and on a plan that holds nothing.
func (p DiskAdmissionPlan) Release() { p.JSStage.Release(); p.PythonLockStage.Release() }

type installEnvironmentContextKey struct{}

func withInstallCacheEnvironment(ctx context.Context, manager, cachePath, buildCachePath string) context.Context {
	return context.WithValue(ctx, installEnvironmentContextKey{}, managerCacheEnvironment(manager, cachePath, buildCachePath))
}

func installEnvironmentFromContext(ctx context.Context) []string {
	env, _ := ctx.Value(installEnvironmentContextKey{}).([]string)
	return env
}

func managerCacheEnvironment(manager, cachePath, buildCachePath string) []string {
	if strings.TrimSpace(cachePath) == "" {
		return nil
	}
	var env []string
	switch manager {
	case "npm":
		// npm refuses to load one file as both the user and the global config, so
		// the global slot gets its own empty file.
		env = append(env, "npm_config_cache="+cachePath, "npm_config_registry=https://registry.npmjs.org/", "npm_config_userconfig="+os.DevNull, "npm_config_globalconfig="+emptyNpmGlobalConfig())
	case "pnpm":
		env = append(env, "npm_config_store_dir="+cachePath, "npm_config_registry=https://registry.npmjs.org/", "npm_config_userconfig="+os.DevNull, "npm_config_globalconfig="+os.DevNull)
	case "yarn":
		env = append(env, "YARN_CACHE_FOLDER="+cachePath, "npm_config_registry=https://registry.npmjs.org/", "npm_config_userconfig="+os.DevNull, "npm_config_globalconfig="+os.DevNull)
	case "bun":
		env = append(env, "BUN_INSTALL_CACHE_DIR="+cachePath, "npm_config_registry=https://registry.npmjs.org/", "npm_config_userconfig="+os.DevNull, "npm_config_globalconfig="+os.DevNull)
	case "go":
		env = append(env,
			"GOMODCACHE="+cachePath,
			"GOPROXY=https://proxy.golang.org",
			"GOSUMDB=sum.golang.org",
			"GONOPROXY=",
			"GONOSUMDB=",
			"GOPRIVATE=",
			"GOINSECURE=",
			"GOWORK=off",
			"GOTOOLCHAIN=local",
			"GOFLAGS=",
			"GOCACHEPROG=",
		)
	case "pip":
		env = append(env, "PIP_CACHE_DIR="+cachePath)
	case "uv", "uv-lock":
		env = append(env, "UV_CACHE_DIR="+cachePath)
	case "poetry":
		env = append(env, "POETRY_CACHE_DIR="+cachePath)
	case "cargo":
		env = append(env, "CARGO_HOME="+cachePath)
		if buildCachePath != "" {
			env = append(env, "CARGO_TARGET_DIR="+buildCachePath)
		}
	}
	if manager == "go" && buildCachePath != "" {
		env = append(env, "GOCACHE="+buildCachePath)
	}
	return env
}

// emptyNpmGlobalConfig returns the path of a shared, empty npmrc used to
// neutralize npm's global configuration.
func emptyNpmGlobalConfig() string {
	path := filepath.Join(os.TempDir(), "sda-empty-npmrc")
	if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
		return path
	}
	_ = os.WriteFile(path, nil, 0o644)
	return path
}

func overlayInstallEnvironment(base, overlay []string) []string {
	if len(overlay) == 0 {
		return base
	}
	keys := make(map[string]struct{}, len(overlay))
	for _, item := range overlay {
		key, _, _ := strings.Cut(item, "=")
		keys[strings.ToLower(key)] = struct{}{}
	}
	out := make([]string, 0, len(base)+len(overlay))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, replace := keys[strings.ToLower(key)]; !replace {
			out = append(out, item)
		}
	}
	return append(out, overlay...)
}

// PlanDiskAdmission maps every resolved install profile to write locations.
// Unsupported means that an owner-specific acquisition path is not yet
// exposed as a deterministic path; callers must fail closed instead of
// running that profile with a partial reservation.
func PlanDiskAdmission(ctx context.Context, r Resolution) (DiskAdmissionPlan, error) {
	plan, err := planDiskAdmission(ctx, r)
	if err != nil {
		plan.Release()
		return DiskAdmissionPlan{}, err
	}
	if plan.Unsupported != "" {
		// A refused plan is never applied, so its staged resolution is waste.
		plan.Release()
		plan.JSStage = nil
		plan.PythonLockStage = nil
	}
	return plan, nil
}

func planDiskAdmission(ctx context.Context, r Resolution) (DiskAdmissionPlan, error) {
	if strings.TrimSpace(r.SurfaceRoot) == "" {
		return DiskAdmissionPlan{}, fmt.Errorf("disk admission requires a resolved surface root")
	}
	if r.Profile.LifecycleMode == "managed-service-artifact" && r.PackageName != "" {
		return planManagedServiceArtifactDisk(ctx, r)
	}
	if r.Profile.LifecycleMode == MT7927SourceLifecycle {
		return planMT7927SourceDisk(r)
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return DiskAdmissionPlan{}, fmt.Errorf("resolve package install home for disk admission: %w", err)
	}
	if r.PackageManager == "pip-local-wheel" {
		return planYtDlpLocalWheel(ctx, r)
	}
	if r.HostTool && r.PackageManager == "npm" && r.Profile.LifecycleMode == "scripts-disabled-by-default" {
		return planNPMFrozenHostToolDisk(ctx, r)
	}

	plan := DiskAdmissionPlan{Manager: r.PackageManager}
	if r.PackageManager == "uv-lock" {
		stage, stageErr := stagePythonLocks(ctx, r)
		if stageErr != nil {
			plan.Unsupported = stageErr.Error()
			return plan, nil
		}
		plan.PythonLockStage = stage
		plan.WritesInputsOnly = true
		plan.ProbeBytes = pythonLockStagePeakBytes
		plan.Paths = []DiskPathRequirement{{Path: r.SurfaceRoot, Role: "python-lock-resolution-and-publication-stage", PeakBytes: pythonLockStagePeakBytes}}
		return plan, nil
	}
	var cache string
	staged := stagedJSInstall(r)
	if staged {
		// The manager resolves into a staging copy first; the same resolution
		// supplies the size admission needs, and for pnpm the store the install
		// will link against.
		stage, stageErr := stageJSResolution(ctx, r)
		if stageErr != nil {
			plan.Unsupported = stageErr.Error()
			return plan, nil
		}
		plan.JSStage = stage
		plan.WritesInputsOnly = stage.LinkSkipped != ""
		plan.Notes = append(plan.Notes, stage.Notes...)
		if r.PackageManager == "pnpm" {
			cache = stage.store.Base
		} else if cache, err = managerCachePath(r.PackageManager, home); err != nil {
			return DiskAdmissionPlan{}, err
		}
	} else {
		var cacheErr error
		if cache, cacheErr = managerCachePath(r.PackageManager, home); cacheErr != nil {
			return DiskAdmissionPlan{}, cacheErr
		}
	}
	plan.CachePath = cache
	if r.PackageManager == "go" {
		plan.BuildCachePath = managerBuildCachePath(r.PackageManager, home)
	} else if r.PackageManager == "cargo" {
		plan.BuildCachePath = filepath.Join(r.SurfaceRoot, "target")
	}
	add := func(path, role string, peak int64) error {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("disk admission path for %s is not absolute: %q", role, path)
		}
		plan.Paths = append(plan.Paths, DiskPathRequirement{Path: filepath.Clean(path), Role: role, PeakBytes: peak})
		return nil
	}
	var probeBytes int64
	if r.PackageManager == "go" {
		probe, err := probeGoModuleGrowthDetails(ctx, r)
		if err != nil {
			plan.Unsupported = err.Error()
			return plan, nil
		}
		plan.GoArchiveBytes = probe.ArchiveBytes
		plan.GoExpandedBytes = probe.ExpandedBytes
		plan.GoModuleCount = probe.ModuleCount
		plan.WritesInputsOnly = probe.ArchiveBytes == 0 && probe.ExpandedBytes == 0
		probeBytes = probe.ArchiveBytes + probe.ExpandedBytes + probe.MetadataBytes
	} else if staged {
		plan.JS = plan.JSStage.Measured
		probeBytes = plan.JS.ProbeBytes
	} else {
		var probeErr error
		probeBytes, probeErr = probePackageGrowth(ctx, r)
		if probeErr != nil {
			plan.Unsupported = probeErr.Error()
			return plan, nil
		}
	}
	plan.ProbeBytes = probeBytes
	if probeBytes <= 0 || probeBytes > DiskJobPlanningCeilingBytes {
		plan.Unsupported = fmt.Sprintf("bounded package size probe returned %d bytes outside the allowed range", probeBytes)
		return plan, nil
	}
	envPeak := probeBytes * 2
	cachePeak := probeBytes
	tempPeak := probeBytes
	if envPeak+cachePeak+tempPeak > DiskJobPlanningCeilingBytes {
		plan.Unsupported = fmt.Sprintf("measured install peak %d exceeds the %d-byte per-job planning ceiling", envPeak+cachePeak+tempPeak, DiskJobPlanningCeilingBytes)
		return plan, nil
	}
	buildPeak := int64(1 << 20)
	if r.PackageManager == "go" {
		buildPeak = goBuildCacheFloorBytes
		if probeBytes > goBuildCacheFloorBytes/2 {
			buildPeak = probeBytes * 2
		}
		if plan.WritesInputsOnly {
			// go get loads packages and rewrites go.mod and go.sum; with nothing to
			// fetch it compiles nothing, so the build cache sees only bookkeeping.
			buildPeak = goInputsOnlyBuildAllowance
		}
	}
	if envPeak+cachePeak+tempPeak+buildPeak > DiskJobPlanningCeilingBytes {
		plan.Unsupported = fmt.Sprintf("bounded install and build-cache peak %d exceeds the %d-byte per-job planning ceiling", envPeak+cachePeak+tempPeak+buildPeak, DiskJobPlanningCeilingBytes)
		return plan, nil
	}

	switch r.Profile.LifecycleMode {
	case "system-host-tool":
		plan.Unsupported = "system package provisioning has host-package-manager downloads and caches that the dependency gateway cannot enumerate"
	case "managed-service-artifact":
		plan.Unsupported = "managed-service staging path must be provided by the resource artifact owner"
	case "pinned-release", "pinned-source-tree":
		plan.Unsupported = "pinned binary acquisition has no complete bounded archive, unpack, and download-cache size contract"
	case "resource-agent":
		userHome, e := hostreqkit.InvokingUserHomeDir()
		if e != nil {
			return DiskAdmissionPlan{}, fmt.Errorf("resolve resource-agent home: %w", e)
		}
		name := filepath.Base(r.SurfaceRoot)
		prefix, e := agentinstall.NPMInstallPrefixForChannel(userHome, name, r.Channel, r.HostToolTargetVersion)
		if e != nil {
			return DiskAdmissionPlan{}, fmt.Errorf("resolve resource-agent prefix: %w", e)
		}
		if err := add(filepath.Dir(prefix), "resource-agent-candidate-and-install", envPeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
		if err := add(cache, "npm-package-cache", cachePeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
	case "native-governed", "scripts-disabled-by-default", "":
		if r.Profile.LifecycleMode == "pinned-source-tree" {
			break
		}
		if r.HostToolRefresh {
			if err := add(os.TempDir(), "private-host-tool-refresh-candidate", buildPeak); err != nil {
				return DiskAdmissionPlan{}, err
			}
		}
		if r.TemplateVersion != "" {
			if err := add(filepath.Join(r.RepositoryRoot, "scenarios"), "template-install-staging-and-cache", envPeak); err != nil {
				return DiskAdmissionPlan{}, err
			}
		} else {
			role := "package-environment-and-staging"
			if r.PackageManager == "pnpm" || r.PackageManager == "npm" {
				role = r.PackageManager + "-package-environment-and-lockfile-staging"
			}
			if err := add(r.SurfaceRoot, role, envPeak); err != nil {
				return DiskAdmissionPlan{}, err
			}
		}
		cacheRole := "package-manager-cache"
		if r.PackageManager == "pnpm" {
			cacheRole = "pnpm-content-addressable-store"
		}
		if err := add(cache, cacheRole, cachePeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
		tempRole := "package-download-and-unpack-temp"
		if r.PackageManager == "pnpm" || r.PackageManager == "npm" {
			tempRole = r.PackageManager + "-download-and-unpack-temp"
		}
		if err := add(os.TempDir(), tempRole, tempPeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
		if r.PackageManager == "go" {
			if err := add(plan.BuildCachePath, "go-build-cache", buildPeak); err != nil {
				return DiskAdmissionPlan{}, err
			}
		}
	default:
		plan.Unsupported = "unknown install lifecycle profile " + r.Profile.LifecycleMode
	}
	if len(plan.Paths) == 0 && plan.Unsupported == "" {
		plan.Unsupported = "install profile has no complete disk path map"
	}
	return plan, nil
}

type ytDlpWheelMetadata struct {
	URL    string
	SHA256 string
	Bytes  int64
}

var ytDlpWheelMetadataProbe = probeYtDlpWheel

func planYtDlpLocalWheel(ctx context.Context, r Resolution) (DiskAdmissionPlan, error) {
	plan := DiskAdmissionPlan{Manager: r.PackageManager}
	if r.PackageName != "yt-dlp" || !r.ExactVersion || !strings.Contains(strings.Join(r.Argv, " "), "yt-dlp==2026.10.26") {
		plan.Unsupported = "surface-local pip wheel route only admits the exact approved yt-dlp version 2026.10.26"
		return plan, nil
	}
	rootInfo, err := os.Lstat(r.SurfaceRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		plan.Unsupported = "surface-local Python install root is unavailable or linked"
		return plan, nil
	}
	probe, python, err := ytDlpWheelMetadataProbe(ctx)
	if err != nil {
		plan.Unsupported = err.Error()
		return plan, nil
	}
	plan.PythonExecutable, plan.PythonWheelURL = python, probe.URL
	plan.PythonWheelSHA256, plan.PythonWheelBytes = probe.SHA256, probe.Bytes
	plan.ProbeBytes = probe.Bytes
	plan.PythonTarget = filepath.Join(r.SurfaceRoot, ".sda-python-packages", "yt-dlp", "2026.10.26")
	plan.PythonStage = filepath.Join(r.SurfaceRoot, ".sda-python-staging", "yt-dlp", "2026.10.26")
	for _, candidate := range []string{plan.PythonTarget, plan.PythonStage} {
		if err := validateNoSymlinkPathComponents(r.SurfaceRoot, candidate); err != nil {
			plan.Unsupported = fmt.Sprintf("local yt-dlp path is linked or unsafe: %v", err)
			return plan, nil
		}
	}
	for _, candidate := range []string{plan.PythonTarget, plan.PythonStage} {
		if _, err := os.Lstat(candidate); err == nil {
			plan.Unsupported = "a prior local yt-dlp target or staging directory exists and requires inspection before replacement"
			return plan, nil
		} else if !os.IsNotExist(err) {
			plan.Unsupported = fmt.Sprintf("inspect local yt-dlp path: %v", err)
			return plan, nil
		}
	}
	const targetPeak = int64(512 << 20)
	const stageOverhead = int64(64 << 20)
	stagePeak := probe.Bytes + ytDlpUnpackedLimit + stageOverhead
	if stagePeak <= 0 || targetPeak+stagePeak > DiskJobPlanningCeilingBytes {
		plan.Unsupported = "bounded yt-dlp wheel and unpack peak exceeds the 25 GiB per-job ceiling"
		return plan, nil
	}
	plan.Paths = []DiskPathRequirement{
		{Path: plan.PythonTarget, Role: "yt-dlp-surface-python-target-unpack", PeakBytes: targetPeak},
		{Path: plan.PythonStage, Role: "yt-dlp-wheel-download-and-temporary-files", PeakBytes: stagePeak},
	}
	return plan, nil
}

func validateNoSymlinkPathComponents(root, candidate string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes its surface root")
	}
	current := rootAbs
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink component %s", current)
		}
		if current != candidateAbs && !info.IsDir() {
			return fmt.Errorf("non-directory path component %s", current)
		}
	}
	return nil
}

func probeYtDlpWheel(ctx context.Context) (ytDlpWheelMetadata, string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, "https://pypi.org/pypi/yt-dlp/2026.10.26/json", nil)
	if err != nil {
		return ytDlpWheelMetadata{}, "", err
	}
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("bounded yt-dlp metadata probe failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("bounded yt-dlp metadata probe returned HTTP %d", response.StatusCode)
	}
	const maxPyPIBytes = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPyPIBytes+1))
	if err != nil || len(body) > maxPyPIBytes {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp metadata exceeded its 1 MiB response limit")
	}
	var document struct {
		Info struct {
			Name           string   `json:"name"`
			Version        string   `json:"version"`
			RequiresPython string   `json:"requires_python"`
			RequiresDist   []string `json:"requires_dist"`
		} `json:"info"`
		URLs []struct {
			Filename    string            `json:"filename"`
			Packagetype string            `json:"packagetype"`
			Python      string            `json:"python_version"`
			Size        int64             `json:"size"`
			Digests     map[string]string `json:"digests"`
			URL         string            `json:"url"`
			Yanked      bool              `json:"yanked"`
		} `json:"urls"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("decode yt-dlp metadata: %w", err)
	}
	if document.Info.Name != "yt-dlp" || document.Info.Version != "2026.10.26" {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("PyPI metadata did not match the approved yt-dlp version")
	}
	if len(document.Info.RequiresDist) != 0 {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp metadata declares dependencies; the local wheel route admits only a dependency-free artifact")
	}
	requiresPython := strings.TrimSpace(document.Info.RequiresPython)
	minPython := regexp.MustCompile(`^>=3\.(\d+)$`).FindStringSubmatch(requiresPython)
	if len(minPython) != 2 {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp Python requirement %q cannot be bounded by the local wheel route", requiresPython)
	}
	minMinor, err := strconv.Atoi(minPython[1])
	if err != nil || minMinor < 0 || minMinor > 30 {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp Python requirement is invalid")
	}
	var selected *ytDlpWheelMetadata
	for _, artifact := range document.URLs {
		if artifact.Packagetype != "bdist_wheel" || artifact.Python != "py3" || !strings.HasSuffix(artifact.Filename, "py3-none-any.whl") {
			continue
		}
		parsed, parseErr := url.Parse(artifact.URL)
		digest := strings.ToLower(artifact.Digests["sha256"])
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host != "files.pythonhosted.org" || parsed.User != nil || artifact.Yanked || artifact.Size <= 0 || artifact.Size > 64<<20 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(digest) {
			return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp wheel metadata is incomplete, yanked, or outside bounded limits")
		}
		if selected != nil {
			return ytDlpWheelMetadata{}, "", fmt.Errorf("yt-dlp metadata has ambiguous universal wheel artifacts")
		}
		selected = &ytDlpWheelMetadata{URL: artifact.URL, SHA256: digest, Bytes: artifact.Size}
	}
	if selected == nil {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("PyPI did not publish one bounded py3-none-any yt-dlp wheel")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("surface-local yt-dlp requires a local python3 interpreter: %w", err)
	}
	python, err = filepath.Abs(python)
	if err != nil {
		return ytDlpWheelMetadata{}, "", err
	}
	versionCtx, versionCancel := context.WithTimeout(ctx, 2*time.Second)
	defer versionCancel()
	versionOutput, err := exec.CommandContext(versionCtx, python, "--version").CombinedOutput()
	if err != nil {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("inspect local Python interpreter: %w", err)
	}
	versionMatch := regexp.MustCompile(`Python\s+3\.(\d+)`).FindStringSubmatch(string(versionOutput))
	if len(versionMatch) != 2 {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("local yt-dlp environment requires Python 3")
	}
	minor, _ := strconv.Atoi(versionMatch[1])
	if minor < minMinor {
		return ytDlpWheelMetadata{}, "", fmt.Errorf("local Python 3.%d does not meet yt-dlp's >=3.%d requirement", minor, minMinor)
	}
	return *selected, python, nil
}

// templateLockOnlyBytes is the allowance for a template declaration: the
// lock-only resolution writes registry metadata to the package manager's cache
// and a regenerated lockfile, but installs no packages.
const templateLockOnlyBytes int64 = 256 << 20

// stagedJSInstall reports whether a request is a direct pnpm or npm
// dependency change that is resolved in a staging copy and sized from its
// resolved lock. Template declarations and host tools take other routes.
func stagedJSInstall(r Resolution) bool {
	return (r.PackageManager == "pnpm" || r.PackageManager == "npm") && r.TemplateVersion == "" && !r.HostTool && !r.HostToolRefresh && !r.HostSource
}

// probePackageGrowth bounds the install shapes that are not staged. Everything
// it does not recognize is refused with the reason, never guessed.
func probePackageGrowth(ctx context.Context, r Resolution) (int64, error) {
	if r.Profile.LifecycleMode == "resource-agent" {
		return probeResourceAgentGrowth(ctx, r)
	}
	if r.Profile.LifecycleMode != "native-governed" && r.Profile.LifecycleMode != "scripts-disabled-by-default" {
		return 0, fmt.Errorf("install lifecycle %q has no bounded size probe", r.Profile.LifecycleMode)
	}
	if r.PackageManager == "pip" {
		return 0, fmt.Errorf("pip install does not expose a surface-scoped environment path; use a hashed resource lock workflow until that environment is owned")
	}
	if r.PackageManager == "go" {
		return probeGoModuleGrowth(ctx, r)
	}
	if r.PackageManager == "uv-lock" || r.PackageManager == "uv" || r.PackageManager == "poetry" {
		return 0, fmt.Errorf("Python resolver may rewrite or resolve a transitive lock closure; bounded full-closure size is unavailable")
	}
	if r.TemplateVersion != "" && (r.PackageManager == "pnpm" || r.PackageManager == "npm") && !r.HostTool && !r.HostToolRefresh && !r.HostSource {
		return templateLockOnlyBytes, nil
	}
	return 0, fmt.Errorf("%s install has no bounded package size probe", r.PackageManager)
}

func readRegularBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("expected regular file no larger than %d bytes", limit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d-byte read limit", limit)
	}
	return data, nil
}

func validateNpmRegistryFiles(surfaceRoot string) error {
	paths := []string{filepath.Join(surfaceRoot, ".npmrc")}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".npmrc"))
	}
	for _, key := range []string{"npm_config_userconfig", "NPM_CONFIG_USERCONFIG", "npm_config_globalconfig", "NPM_CONFIG_GLOBALCONFIG"} {
		if configured := strings.TrimSpace(os.Getenv(key)); configured != "" && configured != os.DevNull {
			paths = append(paths, configured)
		}
	}
	current := filepath.Clean(surfaceRoot)
	for {
		if current == "" || current == filepath.Dir(current) {
			break
		}
		paths = append(paths, filepath.Join(current, ".npmrc"))
		current = filepath.Dir(current)
	}
	seen := map[string]struct{}{}
	for _, file := range paths {
		file = filepath.Clean(file)
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		info, err := os.Lstat(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<10 {
			return fmt.Errorf("npm registry configuration is unreadable, linked, or exceeds 64 KiB: %s", file)
		}
		content, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("read npm registry configuration %s: %w", file, err)
		}
		scanner := bufio.NewScanner(content)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.ToLower(strings.TrimSpace(key))
			if key != "registry" && !strings.HasSuffix(key, ":registry") {
				continue
			}
			registry, parseErr := url.Parse(strings.TrimSpace(value))
			if parseErr != nil || registry.Scheme != "https" || registry.Host != "registry.npmjs.org" || registry.User != nil {
				_ = content.Close()
				return fmt.Errorf("custom npm registry configuration in %s cannot be covered by the approved bounded size probe", file)
			}
		}
		scanErr := scanner.Err()
		_ = content.Close()
		if scanErr != nil {
			return fmt.Errorf("read npm registry configuration %s: %w", file, scanErr)
		}
	}
	return nil
}

func managerCachePath(manager, home string) (string, error) {
	get := func(key, fallback string) string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if filepath.IsAbs(v) {
				return filepath.Clean(v)
			}
		}
		return fallback
	}
	xdgCache := get("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	dataRoot := get("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	switch manager {
	case "npm":
		return get("npm_config_cache", get("NPM_CONFIG_CACHE", filepath.Join(home, ".npm"))), nil
	case "pnpm":
		return get("npm_config_store_dir", filepath.Join(dataRoot, "pnpm", "store")), nil
	case "yarn":
		return get("YARN_CACHE_FOLDER", filepath.Join(xdgCache, "yarn")), nil
	case "bun":
		return get("BUN_INSTALL_CACHE_DIR", filepath.Join(home, ".bun", "install", "cache")), nil
	case "go":
		goPath := get("GOPATH", filepath.Join(home, "go"))
		return get("GOMODCACHE", filepath.Join(goPath, "pkg", "mod")), nil
	case "pip":
		return get("PIP_CACHE_DIR", filepath.Join(xdgCache, "pip")), nil
	case "uv", "uv-lock":
		return get("UV_CACHE_DIR", filepath.Join(xdgCache, "uv")), nil
	case "poetry":
		return get("POETRY_CACHE_DIR", filepath.Join(xdgCache, "pypoetry")), nil
	case "cargo":
		return get("CARGO_HOME", filepath.Join(home, ".cargo")), nil
	case "native-binaryfetch", "native-host-tool":
		return "", nil
	default:
		return "", fmt.Errorf("no bounded disk-cache mapping for package manager %q", manager)
	}
}

func managerBuildCachePath(manager, home string) string {
	if manager != "go" {
		return ""
	}
	if configured := strings.TrimSpace(os.Getenv("GOCACHE")); configured != "" && filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}
	return filepath.Join(home, ".cache", "go-build")
}
