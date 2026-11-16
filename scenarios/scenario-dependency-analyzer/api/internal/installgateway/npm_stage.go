package installgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vrooli/envkit-go"
)

const maxNpmLockBytes = 32 << 20

// npmManager is the npm side of a staged install. Resolution is
// `npm install --package-lock-only`; linking is `npm install` against the
// published lock, which installs what node_modules lacks without re-resolving.
var npmManager = &stagedManager{
	name:   "npm",
	inputs: []string{"package.json", "package-lock.json"},
	limits: map[string]int64{"package.json": maxPnpmManifestBytes, "package-lock.json": maxNpmLockBytes},
	resolveArgs: func(argv []string) []string {
		return append(append([]string{}, argv[1:]...), "--package-lock-only")
	},
	linkArgs: func([]string) []string {
		return []string{"install", "--ignore-scripts", "--no-audit", "--no-fund"}
	},
	environment: func(r Resolution, _ *JSStage, extra ...string) []string { return npmEnvironment(r, extra...) },
	cacheEnv:    "npm_config_cache",
	explain:     explainNpmResolveFailure,
	measure:     measureNpmStage,
}

func npmEnvironment(r Resolution, extra ...string) []string {
	cache := r.DiskCachePath
	if cache == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cache, _ = managerCachePath("npm", home)
		}
	}
	overlay := append(managerCacheEnvironment("npm", cache, ""), extra...)
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	return overlayInstallEnvironment(envkit.Toolchain(env, envkit.ToolchainOptions{}), overlay)
}

func explainNpmResolveFailure(r Resolution, runErr error, output string) error {
	tail := strings.TrimSpace(output)
	if len(tail) > 2000 {
		tail = tail[len(tail)-2000:]
	}
	hint := ""
	switch {
	case strings.Contains(output, "ETARGET"), strings.Contains(output, "notarget"):
		hint = " No published version matches the request; pick a version that exists."
	case strings.Contains(output, "E404"):
		hint = " The registry has no such package or version; check the spelling."
	case strings.Contains(output, "ERESOLVE"):
		hint = " The dependency tree has a peer conflict; choose a version compatible with the surface's existing packages."
	}
	return fmt.Errorf("npm could not resolve %s in a staging copy of %s (%v); the surface was not changed.%s\n%s", requestedSpecText(r), r.SurfaceRoot, runErr, hint, tail)
}

type npmLockEntry struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Resolved  string   `json:"resolved"`
	Integrity string   `json:"integrity"`
	Link      bool     `json:"link"`
	OS        []string `json:"os"`
	CPU       []string `json:"cpu"`
}

// measureNpmStage sizes the lock entries whose install directory in
// node_modules is missing or holds another version. npm's lock keys are the
// install paths, so presence is exact.
func measureNpmStage(ctx context.Context, s *JSStage, r Resolution) error {
	lock := s.next["package-lock.json"]
	if lock == nil {
		lock = s.base["package-lock.json"]
	}
	var document struct {
		Packages map[string]npmLockEntry `json:"packages"`
	}
	if err := json.Unmarshal(lock, &document); err != nil {
		return fmt.Errorf("decode package-lock.json: %w", err)
	}
	spec := requestedSpec(r)
	type job struct {
		key, name, version, resolved, integrity string
	}
	var jobs []job
	keys := make([]string, 0, len(document.Packages))
	for key := range document.Packages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := document.Packages[key]
		if key == "" || entry.Link || !platformAllows(entry.OS, entry.CPU) {
			continue
		}
		name := entry.Name
		if name == "" {
			if idx := strings.LastIndex(key, "node_modules/"); idx >= 0 {
				name = key[idx+len("node_modules/"):]
			}
		}
		if name == "" || entry.Version == "" {
			return fmt.Errorf("package-lock.json entry %q has no package name or version", key)
		}
		if npmInstalledVersion(filepath.Join(s.surfaceRoot, filepath.FromSlash(key))) == entry.Version {
			continue
		}
		jobs = append(jobs, job{key, name, entry.Version, entry.Resolved, entry.Integrity})
	}
	var sizer registrySizer
	if len(jobs) > 0 {
		var err error
		if sizer, err = newRegistrySizer(); err != nil {
			return err
		}
	}
	sizes := make([]artifactSize, len(jobs))
	if err := runSizingJobs(ctx, len(jobs), func(ctx context.Context, i int) error {
		j := jobs[i]
		var size artifactSize
		var err error
		switch {
		case strings.HasPrefix(j.resolved, "https://registry.npmjs.org/"):
			size, err = sizer.Registry(ctx, j.name, j.version, j.integrity)
		case j.resolved != "" && j.resolved == spec:
			size, err = sizer.Tarball(ctx, j.resolved, j.integrity)
			if err == nil {
				s.noteTarball(j.name, j.resolved, j.integrity)
			}
		default:
			err = fmt.Errorf("the resolved lock adds a package from %q, which cannot be sized for disk admission; install a published registry release or the exact approved tarball URL", j.resolved)
		}
		if err != nil {
			return fmt.Errorf("size %s@%s: %w", j.name, j.version, err)
		}
		sizes[i] = size
		return nil
	}); err != nil {
		return err
	}
	var total int64
	for _, size := range sizes {
		artifact, err := addBoundedBytes(size.Unpacked, size.Archive, DiskJobPlanningCeilingBytes)
		if err != nil {
			return fmt.Errorf("package artifact exceeds the bounded size: %w", err)
		}
		if total, err = addBoundedBytes(total, artifact, DiskJobPlanningCeilingBytes); err != nil {
			return fmt.Errorf("the packages this install adds exceed the bounded size (%d bytes per job): %w", DiskJobPlanningCeilingBytes, err)
		}
	}
	return s.finishMeasurement(len(jobs), total)
}

func npmInstalledVersion(dir string) string {
	data, err := readRegularBounded(filepath.Join(dir, "package.json"), maxPnpmManifestBytes)
	if err != nil {
		return ""
	}
	var manifest struct{ Version string }
	if json.Unmarshal(data, &manifest) != nil {
		return ""
	}
	return manifest.Version
}

// platformAllows reports whether a lock entry's os/cpu constraints admit this
// host. A package for another platform is in the lock but is never installed.
func platformAllows(osList, cpuList []string) bool {
	return listAllows(osList, runtime.GOOS, map[string]string{"windows": "win32"}) &&
		listAllows(cpuList, runtime.GOARCH, map[string]string{"amd64": "x64", "386": "ia32"})
}

func listAllows(list []string, current string, rename map[string]string) bool {
	if mapped, ok := rename[current]; ok {
		current = mapped
	}
	positive := false
	for _, item := range list {
		if item == "!"+current {
			return false
		}
		if !strings.HasPrefix(item, "!") {
			positive = true
			if item == current {
				return true
			}
		}
	}
	// A list of only negations ("!win32") allows every other platform.
	return !positive
}
