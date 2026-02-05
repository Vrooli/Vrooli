package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"scenario-to-desktop-api/generation"
	apperrors "scenario-to-desktop-api/shared/errors"
	"scenario-to-desktop-api/shared/path"
)

const (
	VersionUpdateModeSet  = "set"
	VersionUpdateModeBump = "bump"

	VersionSourceBoth    = "both"
	VersionSourceService = "service"
	VersionSourceUI      = "ui"

	VersionBumpPatch  = "patch"
	VersionBumpMinor  = "minor"
	VersionBumpMajor  = "major"
	VersionBumpMedium = "medium"
	VersionBumpAuto   = "auto"
)

var semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

type semver struct {
	major int
	minor int
	patch int
	pre   string
	build string
}

func parseSemver(input string) (semver, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return semver{}, fmt.Errorf("version is empty")
	}
	matches := semverPattern.FindStringSubmatch(trimmed)
	if len(matches) == 0 {
		return semver{}, fmt.Errorf("version %q is not valid semver (expected x.y.z)", input)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return semver{}, fmt.Errorf("invalid major version in %q", input)
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return semver{}, fmt.Errorf("invalid minor version in %q", input)
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return semver{}, fmt.Errorf("invalid patch version in %q", input)
	}
	return semver{
		major: major,
		minor: minor,
		patch: patch,
		pre:   matches[4],
		build: matches[5],
	}, nil
}

func (v semver) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.pre != "" {
		base += "-" + v.pre
	}
	if v.build != "" {
		base += "+" + v.build
	}
	return base
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	if a.pre == "" && b.pre != "" {
		return 1
	}
	if a.pre != "" && b.pre == "" {
		return -1
	}
	return strings.Compare(a.pre, b.pre)
}

func bumpSemver(current semver, bump string) (semver, error) {
	next := current
	next.pre = ""
	next.build = ""

	switch bump {
	case VersionBumpMajor:
		next.major++
		next.minor = 0
		next.patch = 0
	case VersionBumpMinor, VersionBumpMedium:
		next.minor++
		next.patch = 0
	case VersionBumpPatch, VersionBumpAuto:
		next.patch++
	default:
		return semver{}, fmt.Errorf("unknown bump %q (expected patch, minor, medium, major, auto)", bump)
	}

	return next, nil
}

type versionUpdateFS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
}

type osVersionFS struct{}

func (osVersionFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osVersionFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osVersionFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

type versionRollbackEntry struct {
	path string
	data []byte
	mode os.FileMode
}

type versionRollback struct {
	fs      versionUpdateFS
	entries []versionRollbackEntry
	once    sync.Once
}

func (r *versionRollback) Restore() error {
	if r == nil {
		return nil
	}

	var result error
	r.once.Do(func() {
		var errs []error
		for _, entry := range r.entries {
			if err := r.fs.WriteFile(entry.path, entry.data, entry.mode); err != nil {
				errs = append(errs, fmt.Errorf("restore %s: %w", entry.path, err))
			}
		}
		if len(errs) > 0 {
			result = errors.Join(errs...)
		}
	})

	return result
}

type versionUpdater struct {
	vrooliRoot string
	fs         versionUpdateFS
}

func newVersionUpdater(vrooliRoot string) *versionUpdater {
	if vrooliRoot == "" {
		vrooliRoot = path.DetectVrooliRoot()
	}
	return &versionUpdater{
		vrooliRoot: vrooliRoot,
		fs:         osVersionFS{},
	}
}

func (u *versionUpdater) Apply(scenarioName string, update *VersionUpdateRequest) (string, *versionRollback, *apperrors.DomainError) {
	if update == nil {
		return "", nil, nil
	}
	if strings.TrimSpace(scenarioName) == "" {
		return "", nil, apperrors.ErrBadRequest("scenario name is required for version updates")
	}

	mode := strings.ToLower(strings.TrimSpace(update.Mode))
	if mode == "" {
		return "", nil, apperrors.ErrBadRequest("version_update.mode is required")
	}

	source := strings.ToLower(strings.TrimSpace(update.Source))
	if source == "" {
		source = VersionSourceBoth
	}
	if source != VersionSourceBoth && source != VersionSourceService && source != VersionSourceUI {
		return "", nil, apperrors.ErrBadRequest("version_update.source must be one of: both, service, ui")
	}

	scenarioPath := filepath.Join(u.vrooliRoot, "scenarios", scenarioName)
	if _, err := u.fs.Stat(scenarioPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil, apperrors.ErrScenarioNotFound(scenarioName)
		}
		return "", nil, apperrors.ErrInternalf("failed to resolve scenario path: %v", err)
	}

	analyzer := generation.NewAnalyzer(u.vrooliRoot)
	metadata, err := analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		return "", nil, apperrors.ErrScenarioAnalysisFailed(err, scenarioName)
	}
	currentVersion := metadata.Version

	var nextVersion string
	switch mode {
	case VersionUpdateModeSet:
		if strings.TrimSpace(update.Bump) != "" {
			return "", nil, apperrors.ErrBadRequest("version_update.bump is only valid when mode is bump")
		}
		nextVersion = strings.TrimSpace(update.Version)
		if nextVersion == "" {
			return "", nil, apperrors.ErrBadRequest("version_update.version is required when mode is set")
		}
	case VersionUpdateModeBump:
		if strings.TrimSpace(update.Version) != "" {
			return "", nil, apperrors.ErrBadRequest("version_update.version is only valid when mode is set")
		}
		bump := strings.ToLower(strings.TrimSpace(update.Bump))
		if bump == "" {
			return "", nil, apperrors.ErrBadRequest("version_update.bump is required when mode is bump")
		}
		currentSemver, parseErr := parseSemver(currentVersion)
		if parseErr != nil {
			return "", nil, apperrors.ErrBadRequest(fmt.Sprintf("current version %q is not valid semver; use mode=set first", currentVersion))
		}
		bumped, bumpErr := bumpSemver(currentSemver, bump)
		if bumpErr != nil {
			return "", nil, apperrors.ErrBadRequest(bumpErr.Error())
		}
		nextVersion = bumped.String()
	default:
		return "", nil, apperrors.ErrBadRequest("version_update.mode must be set or bump")
	}

	nextSemver, parseErr := parseSemver(nextVersion)
	if parseErr != nil {
		return "", nil, apperrors.ErrBadRequest(parseErr.Error())
	}

	if !update.AllowDowngrade {
		if currentSemver, err := parseSemver(currentVersion); err == nil {
			if compareSemver(nextSemver, currentSemver) < 0 {
				return "", nil, apperrors.ErrBadRequest(fmt.Sprintf(
					"version downgrade from %s to %s requires allow_downgrade",
					currentSemver.String(),
					nextSemver.String(),
				))
			}
		}
	}

	var rollback *versionRollback
	if update.Persist {
		var derr *apperrors.DomainError
		rollback, derr = u.persistVersion(scenarioPath, nextVersion, source)
		if derr != nil {
			return "", nil, derr
		}
	}

	return nextSemver.String(), rollback, nil
}

type versionUpdateTarget struct {
	path     string
	label    string
	updated  []byte
	original []byte
	mode     os.FileMode
	changed  bool
}

func (u *versionUpdater) persistVersion(scenarioPath, version, source string) (*versionRollback, *apperrors.DomainError) {
	var (
		serviceTarget versionUpdateTarget
		packageTarget versionUpdateTarget
		targets       []versionUpdateTarget
	)

	if source == VersionSourceBoth || source == VersionSourceService {
		servicePath := filepath.Join(scenarioPath, ".vrooli", "service.json")
		target, err := u.prepareJSONUpdate(servicePath, []string{"service", "version"}, version, "service.json")
		if err != nil {
			return nil, err
		}
		serviceTarget = target
		if serviceTarget.changed {
			targets = append(targets, serviceTarget)
		}
	}

	if source == VersionSourceBoth || source == VersionSourceUI {
		packagePath := filepath.Join(scenarioPath, "ui", "package.json")
		target, err := u.prepareJSONUpdate(packagePath, []string{"version"}, version, "ui/package.json")
		if err != nil {
			return nil, err
		}
		packageTarget = target
		if packageTarget.changed {
			targets = append(targets, packageTarget)
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	rollback := &versionRollback{
		fs: u.fs,
	}
	for _, target := range targets {
		rollback.entries = append(rollback.entries, versionRollbackEntry{
			path: target.path,
			data: target.original,
			mode: target.mode,
		})
	}

	for _, target := range targets {
		if err := u.fs.WriteFile(target.path, target.updated, target.mode); err != nil {
			_ = rollback.Restore()
			return nil, apperrors.ErrInternalf("failed to write %s: %v", target.label, err)
		}
	}

	return rollback, nil
}

func (u *versionUpdater) prepareJSONUpdate(path string, jsonPath []string, version string, label string) (versionUpdateTarget, *apperrors.DomainError) {
	info, err := u.fs.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return versionUpdateTarget{}, apperrors.ErrBadRequest(fmt.Sprintf("%s not found", label)).
				WithDetail("path", path)
		}
		return versionUpdateTarget{}, apperrors.ErrInternalf("failed to read %s: %v", label, err)
	}

	data, err := u.fs.ReadFile(path)
	if err != nil {
		return versionUpdateTarget{}, apperrors.ErrInternalf("failed to read %s: %v", label, err)
	}

	updated, changed, err := updateJSONVersionField(data, jsonPath, version)
	if err != nil {
		return versionUpdateTarget{}, apperrors.ErrBadRequest(fmt.Sprintf("failed to update %s: %v", label, err)).
			WithDetail("path", path)
	}

	return versionUpdateTarget{
		path:     path,
		label:    label,
		updated:  updated,
		original: data,
		mode:     info.Mode().Perm(),
		changed:  changed,
	}, nil
}

func updateJSONVersionField(data []byte, path []string, version string) ([]byte, bool, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false, fmt.Errorf("invalid JSON: %w", err)
	}

	target := payload
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		next, ok := target[key].(map[string]interface{})
		if !ok {
			return nil, false, fmt.Errorf("missing object %q", key)
		}
		target = next
	}

	key := path[len(path)-1]
	if existing, ok := target[key].(string); ok && existing == version {
		return nil, false, nil
	}
	target[key] = version

	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	updated = append(updated, '\n')
	return updated, true, nil
}
