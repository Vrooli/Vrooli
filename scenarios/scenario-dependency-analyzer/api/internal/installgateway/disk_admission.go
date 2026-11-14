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
	GoSourceBytes     int64
	GoArchiveBytes    int64
	GoExpandedBytes   int64
	GoModuleCount     int64
}

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
		env = append(env, "npm_config_cache="+cachePath, "npm_config_registry=https://registry.npmjs.org/", "npm_config_userconfig="+os.DevNull, "npm_config_globalconfig="+os.DevNull)
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

	cache, err := managerCachePath(r.PackageManager, home)
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	plan := DiskAdmissionPlan{CachePath: cache, Manager: r.PackageManager}
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
		plan.GoSourceBytes = probe.SourceBytes
		plan.GoArchiveBytes = probe.ArchiveBytes
		plan.GoExpandedBytes = probe.ExpandedBytes
		plan.GoModuleCount = probe.ModuleCount
		probeBytes = probe.SourceBytes + probe.ArchiveBytes + probe.ExpandedBytes + probe.MetadataBytes
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
		prefix, e := agentinstall.NPMInstallPrefix(userHome, name, r.HostToolTargetVersion)
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
		} else if err := add(r.SurfaceRoot, "package-environment-and-staging", envPeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
		if err := add(cache, "package-manager-cache", cachePeak); err != nil {
			return DiskAdmissionPlan{}, err
		}
		if err := add(os.TempDir(), "package-download-and-unpack-temp", tempPeak); err != nil {
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

// probePackageGrowth performs a bounded, non-mutating npm registry metadata
// request. It admits only exact, dependency-free packages with a published
// unpacked size. Other ecosystems and dependency closures need their owner to
// provide an equivalent deterministic probe before they can be applied.
func probePackageGrowth(ctx context.Context, r Resolution) (int64, error) {
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
	if r.HostTool || r.HostToolRefresh || r.HostSource || r.TemplateVersion != "" || r.PackageManager != "npm" {
		return 0, fmt.Errorf("%s install has no bounded package size probe", r.PackageManager)
	}
	if err := validateEmptyNpmInstallSurface(r); err != nil {
		return 0, err
	}
	version := strings.TrimSpace(r.HostToolTargetVersion)
	for i := len(r.Argv) - 1; i >= 0; i-- {
		arg := strings.TrimSpace(r.Argv[i])
		if strings.HasPrefix(arg, r.PackageName+"@") {
			version = strings.TrimPrefix(arg, r.PackageName+"@")
			break
		}
	}
	if !exactNpmVersion.MatchString(version) {
		return 0, fmt.Errorf("install requires an exact package version for a bounded metadata probe")
	}
	if err := validateNpmRegistryFiles(r.SurfaceRoot); err != nil {
		return 0, err
	}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		lowerKey := strings.ToLower(key)
		if !ok || !strings.HasPrefix(lowerKey, "npm_config_") || !strings.Contains(lowerKey, "registry") {
			continue
		}
		configured, parseErr := url.Parse(strings.TrimSpace(value))
		if parseErr != nil || configured.Scheme != "https" || configured.Host != "registry.npmjs.org" || configured.User != nil {
			return 0, fmt.Errorf("custom npm registry environment setting cannot be covered by the approved bounded size probe")
		}
	}
	registry := strings.TrimSpace(os.Getenv("npm_config_registry"))
	if registry == "" {
		registry = strings.TrimSpace(os.Getenv("NPM_CONFIG_REGISTRY"))
	}
	if registry == "" {
		registry = "https://registry.npmjs.org"
	}
	parsed, err := url.Parse(registry)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "registry.npmjs.org" || parsed.User != nil {
		return 0, fmt.Errorf("package registry is not the approved public HTTPS registry; refusing an unbounded probe")
	}
	endpoint := strings.TrimRight(registry, "/") + "/" + url.PathEscape(r.PackageName) + "/" + url.PathEscape(version)
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/vnd.npm.install-v1+json")
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("bounded package metadata probe failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bounded package metadata probe returned HTTP %d", response.StatusCode)
	}
	const maxProbeBytes = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProbeBytes+1))
	if err != nil || len(body) > maxProbeBytes {
		return 0, fmt.Errorf("package metadata probe exceeded its 1 MiB response limit")
	}
	var metadata struct {
		Name, Version                                        string
		Dependencies, OptionalDependencies, PeerDependencies map[string]string
		Dist                                                 struct {
			UnpackedSize int64  `json:"unpackedSize"`
			Tarball      string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return 0, fmt.Errorf("decode bounded package metadata: %w", err)
	}
	if metadata.Name != r.PackageName || metadata.Version != version || metadata.Dist.UnpackedSize <= 0 || metadata.Dist.Tarball == "" {
		return 0, fmt.Errorf("package metadata did not include a matching version, tarball, and positive unpacked size")
	}
	if len(metadata.Dependencies)+len(metadata.OptionalDependencies)+len(metadata.PeerDependencies) != 0 {
		return 0, fmt.Errorf("package metadata has transitive or peer dependencies; bounded closure sizing is unavailable")
	}
	if metadata.Dist.UnpackedSize > DiskJobPlanningCeilingBytes/3 {
		return 0, fmt.Errorf("package unpacked size exceeds the bounded per-job planning ceiling")
	}
	// Reserve two unpacked copies for install environment/staging and one
	// unpacked copy plus compressed tarball for the package cache. The archive
	// itself is never downloaded by this probe.
	archiveURL, err := url.Parse(metadata.Dist.Tarball)
	if err != nil || archiveURL.Scheme != "https" || archiveURL.Host != "registry.npmjs.org" {
		return 0, fmt.Errorf("package metadata tarball is outside the approved HTTPS registry")
	}
	headCtx, headCancel := context.WithTimeout(ctx, 5*time.Second)
	defer headCancel()
	headRequest, err := http.NewRequestWithContext(headCtx, http.MethodHead, archiveURL.String(), nil)
	if err != nil {
		return 0, err
	}
	headResponse, err := client.Do(headRequest)
	if err != nil {
		return 0, fmt.Errorf("bounded package archive size probe failed: %w", err)
	}
	defer headResponse.Body.Close()
	archiveBytes := headResponse.ContentLength
	if headResponse.StatusCode < 200 || headResponse.StatusCode >= 300 || archiveBytes <= 0 || archiveBytes > DiskJobPlanningCeilingBytes/3 {
		return 0, fmt.Errorf("package tarball size is unavailable or exceeds the bounded probe limit")
	}
	return metadata.Dist.UnpackedSize + archiveBytes, nil
}

// validateEmptyNpmInstallSurface makes the metadata probe's direct package
// size a complete install closure: existing dependency trees and populated
// node_modules would let npm reify additional bytes that the probe did not
// measure, so those surfaces are refused until the owner can size their lock
// closure.
func validateEmptyNpmInstallSurface(r Resolution) error {
	surfaceInfo, err := os.Lstat(r.SurfaceRoot)
	if err != nil || !surfaceInfo.IsDir() || surfaceInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("npm surface root is unavailable or linked")
	}
	manifestData, err := readRegularBounded(r.ManifestPath, 1<<20)
	if err != nil {
		return fmt.Errorf("npm manifest is unavailable, linked, or exceeds the 1 MiB bounded admission limit: %w", err)
	}
	var manifest struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Optional        map[string]string `json:"optionalDependencies"`
		Peer            map[string]string `json:"peerDependencies"`
		Workspaces      json.RawMessage   `json:"workspaces"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("decode npm manifest for bounded install sizing: %w", err)
	}
	if len(manifest.Dependencies)+len(manifest.DevDependencies)+len(manifest.Optional)+len(manifest.Peer) != 0 {
		return fmt.Errorf("npm surface already declares dependencies; complete lock closure sizing is unavailable")
	}
	workspaces := strings.TrimSpace(string(manifest.Workspaces))
	if workspaces != "" && workspaces != "null" && workspaces != "[]" {
		return fmt.Errorf("npm workspace installs can modify multiple surfaces beyond the bounded disk plan")
	}
	lockPath := filepath.Join(r.SurfaceRoot, "package-lock.json")
	lockData, err := readRegularBounded(lockPath, 1<<20)
	if err != nil {
		return fmt.Errorf("npm package-lock is unavailable, linked, or exceeds the 1 MiB bounded admission limit: %w", err)
	}
	if _, err := os.Lstat(filepath.Join(r.SurfaceRoot, "npm-shrinkwrap.json")); err == nil {
		return fmt.Errorf("npm shrinkwrap takes precedence over the sized package-lock and is not admitted")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect npm shrinkwrap path: %w", err)
	}
	var lock struct {
		Packages     map[string]json.RawMessage `json:"packages"`
		Dependencies map[string]json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return fmt.Errorf("decode npm lockfile for bounded install sizing: %w", err)
	}
	for packagePath := range lock.Packages {
		if packagePath != "" {
			return fmt.Errorf("npm lockfile already contains packages; complete reification peak sizing is unavailable")
		}
	}
	if len(lock.Dependencies) != 0 {
		return fmt.Errorf("npm lockfile already contains packages; complete reification peak sizing is unavailable")
	}
	nodeModules := filepath.Join(r.SurfaceRoot, "node_modules")
	info, err := os.Lstat(nodeModules)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("npm node_modules path is not a plain directory")
	}
	entries, err := os.ReadDir(nodeModules)
	if err != nil || len(entries) != 0 {
		return fmt.Errorf("npm node_modules is populated; complete reification peak sizing is unavailable")
	}
	return nil
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
