package installgateway

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Sizing a staged pnpm install: pnpm already resolved the closure into the
// staged lockfile, so admission only has to size the packages that lock adds
// and node_modules does not already hold. Registry packages are sized from
// their published metadata, a tarball URL dependency is downloaded once into
// memory so its bytes are checked against the lock's integrity, and a local
// `file:` package is measured on disk.

type pnpmLockResolution struct {
	Integrity string `yaml:"integrity"`
	Tarball   string `yaml:"tarball"`
	Directory string `yaml:"directory"`
	Type      string `yaml:"type"`
	Repo      string `yaml:"repo"`
}

type pnpmLockPackage struct {
	Resolution pnpmLockResolution `yaml:"resolution"`
	OS         []string           `yaml:"os"`
	CPU        []string           `yaml:"cpu"`
}

type pnpmLockfile struct {
	Packages map[string]pnpmLockPackage `yaml:"packages"`
}

func readPnpmLockPackages(data []byte) (map[string]pnpmLockPackage, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var lock pnpmLockfile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("decode pnpm lockfile: %w", err)
	}
	return lock.Packages, nil
}

// splitPnpmPackageKey splits a lockfile package key into its name and
// specifier. Scoped names begin with "@", so the separator is the first "@"
// after the first character.
func splitPnpmPackageKey(key string) (name, spec string, ok bool) {
	if len(key) < 3 {
		return "", "", false
	}
	idx := strings.IndexByte(key[1:], '@')
	if idx < 0 {
		return "", "", false
	}
	idx++
	name, spec = key[:idx], key[idx+1:]
	return name, spec, name != "" && spec != ""
}

func measurePnpmStage(ctx context.Context, s *JSStage, r Resolution) error {
	nextLock := s.next["pnpm-lock.yaml"]
	if nextLock == nil {
		nextLock = s.base["pnpm-lock.yaml"]
	}
	packages, err := readPnpmLockPackages(nextLock)
	if err != nil {
		return err
	}
	previous, err := readPnpmLockPackages(s.base["pnpm-lock.yaml"])
	if err != nil {
		return err
	}
	modulesRoot := filepath.Join(s.surfaceRoot, "node_modules")
	linked, inSync := pnpmInstalledState(modulesRoot, s.base["pnpm-lock.yaml"])

	requestedSpec := requestedSpec(r)
	var sizer registrySizer
	type job struct {
		key, name, spec string
		res             pnpmLockResolution
	}
	var jobs []job
	var localBytes int64
	keys := make([]string, 0, len(packages))
	for key := range packages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if entry := packages[key]; !platformAllows(entry.OS, entry.CPU) {
			continue // Locked for another platform; pnpm never installs it here.
		}
		res := packages[key].Resolution
		name, spec, ok := splitPnpmPackageKey(key)
		if !ok {
			return fmt.Errorf("pnpm lock package key %q is not name@specifier", key)
		}
		_, wasLocked := previous[key]
		if inSync && wasLocked {
			continue
		}
		switch {
		case res.Directory != "" || strings.HasPrefix(spec, "file:"):
			if r.ExternalWorkspace {
				return fmt.Errorf("local directory dependency %s cannot be admitted for an external workspace", key)
			}
			if wasLocked {
				continue
			}
			size, err := measureLocalPnpmPackage(r.RepositoryRoot, r.SurfaceRoot, name, spec)
			if err != nil {
				return err
			}
			if localBytes, err = addBoundedBytes(localBytes, size, DiskJobPlanningCeilingBytes); err != nil {
				return fmt.Errorf("local package closure exceeds the bounded size: %w", err)
			}
			s.Measured.PackageCount++
		case strings.HasPrefix(spec, "https://"):
			if spec != requestedSpec {
				return fmt.Errorf("the resolved lock adds tarball URL dependency %s that was not the requested package; install only registry packages or the exact URL that was approved", key)
			}
			jobs = append(jobs, job{key, name, spec, res})
		case res.Integrity != "" && exactNpmVersion.MatchString(strings.SplitN(spec, "(", 2)[0]):
			if !inSync && pnpmVirtualStoreHas(linked, name, spec) {
				continue
			}
			jobs = append(jobs, job{key, name, strings.SplitN(spec, "(", 2)[0], res})
		default:
			return fmt.Errorf("the resolved lock adds %s with a resolution (%s) that cannot be sized for disk admission; install a published registry release instead", key, describePnpmResolution(res))
		}
	}
	if len(jobs) > 0 {
		if sizer, err = newRegistrySizer(); err != nil {
			return err
		}
	}
	sizes := make([]artifactSize, len(jobs))
	if err := runSizingJobs(ctx, len(jobs), func(ctx context.Context, i int) error {
		j := jobs[i]
		var size artifactSize
		var err error
		if strings.HasPrefix(j.spec, "https://") {
			if pinned, ok := s.tarballSizes[j.res.Tarball]; ok {
				size = pinned // downloaded once already, while pinning its integrity
			} else {
				size, err = sizer.Tarball(ctx, j.spec, j.res.Integrity)
				if err == nil {
					s.noteTarball(j.name, j.spec, j.res.Integrity)
				}
			}
		} else {
			size, err = sizer.Registry(ctx, j.name, j.spec, j.res.Integrity)
		}
		if err != nil {
			return fmt.Errorf("size %s: %w", j.key, err)
		}
		sizes[i] = size
		return nil
	}); err != nil {
		return err
	}
	total := localBytes
	for _, size := range sizes {
		artifact, err := addBoundedBytes(size.Unpacked, size.Archive, DiskJobPlanningCeilingBytes)
		if err != nil {
			return fmt.Errorf("package artifact exceeds the bounded size: %w", err)
		}
		if total, err = addBoundedBytes(total, artifact, DiskJobPlanningCeilingBytes); err != nil {
			return fmt.Errorf("the packages this install adds exceed the bounded size (%d bytes per job): %w", DiskJobPlanningCeilingBytes, err)
		}
	}
	return s.finishMeasurement(s.Measured.PackageCount+len(jobs), total)
}

// finishMeasurement records sized packages, the allowance for writing
// published inputs twice, staged local patches, and bounded resolver overhead.
func (s *JSStage) finishMeasurement(count int, packageBytes int64) error {
	var inputs int64
	for _, data := range s.next {
		var err error
		inputs, err = addBoundedBytes(inputs, int64(len(data)), DiskJobPlanningCeilingBytes)
		if err != nil {
			return err
		}
	}
	writtenInputs, err := addBoundedBytes(inputs, inputs, DiskJobPlanningCeilingBytes)
	if err != nil {
		return err
	}
	stagedFiles, err := addBoundedBytes(writtenInputs, s.patchBytes, DiskJobPlanningCeilingBytes)
	if err != nil {
		return err
	}
	inputs, err = addBoundedBytes(stagedFiles, jsInputAllowanceBytes, DiskJobPlanningCeilingBytes)
	if err != nil {
		return err
	}
	probe, err := addBoundedBytes(packageBytes, inputs, DiskJobPlanningCeilingBytes)
	if err != nil {
		return err
	}
	s.Measured = JSMeasurement{PackageCount: count, PackageBytes: packageBytes, InputBytes: inputs, ProbeBytes: probe}
	return nil
}

func (s *JSStage) noteTarball(name, rawURL, integrity string) {
	note := fmt.Sprintf("tarball dependency %s: %s, lock integrity %s (verified against the downloaded bytes)", name, rawURL, integrity)
	for _, existing := range s.Notes {
		if existing == note {
			return
		}
	}
	s.Notes = append(s.Notes, note)
}

func describePnpmResolution(res pnpmLockResolution) string {
	switch {
	case res.Type != "":
		return "type " + res.Type
	case res.Repo != "":
		return "git repository"
	case res.Tarball != "":
		return "custom tarball"
	default:
		return "no integrity"
	}
}

// pnpmInstalledState reads node_modules once. It returns the package ids that
// the virtual store already holds and whether node_modules was installed from
// exactly the lock the stage started with, which is when only the lock's new
// entries can grow the disk.
func pnpmInstalledState(modulesRoot string, baseLock []byte) (map[string]struct{}, bool) {
	linked := map[string]struct{}{}
	entries, err := os.ReadDir(filepath.Join(modulesRoot, ".pnpm"))
	if err == nil {
		for _, entry := range entries {
			if id, ok := pnpmVirtualStoreID(entry.Name()); ok {
				linked[id] = struct{}{}
			}
		}
	}
	installed, readErr := readRegularBounded(filepath.Join(modulesRoot, ".pnpm", "lock.yaml"), maxPnpmLockBytes)
	return linked, readErr == nil && baseLock != nil && bytes.Equal(installed, baseLock)
}

// pnpmVirtualStoreID reduces a node_modules/.pnpm entry such as
// "@babel+core@7.29.0_peer@1" to "@babel/core@7.29.0".
func pnpmVirtualStoreID(entry string) (string, bool) {
	if len(entry) < 3 {
		return "", false
	}
	at := strings.IndexByte(entry[1:], '@')
	if at < 0 {
		return "", false
	}
	at++
	version := entry[at+1:]
	if cut := strings.IndexAny(version, "_("); cut >= 0 {
		version = version[:cut]
	}
	return strings.ReplaceAll(entry[:at], "+", "/") + "@" + version, version != ""
}

func pnpmVirtualStoreHas(linked map[string]struct{}, name, version string) bool {
	_, ok := linked[name+"@"+version]
	return ok
}
