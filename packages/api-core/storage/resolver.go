package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

const defaultAppID = "vrooli"

// Resolver resolves storage paths with profile-aware defaults and override seams.
type Resolver struct {
	appID   string
	profile Profile
	env     env
}

// NewResolver builds a storage path resolver.
//
// Defaults:
//   - AppID: "vrooli"
//   - Profile: ProfileAuto
//
// Validation:
//   - AppID must not contain path separators.
func NewResolver(cfg ResolverConfig) (*Resolver, error) {
	appID := strings.TrimSpace(cfg.AppID)
	if appID == "" {
		appID = defaultAppID
	}
	if strings.Contains(appID, string(filepath.Separator)) || strings.Contains(appID, "/") || strings.Contains(appID, "\\") {
		return nil, &Error{Kind: ErrInvalidInput, Message: "app id must not contain path separators", Details: appID}
	}

	profile := cfg.Profile
	if profile == "" {
		profile = ProfileAuto
	}

	return &Resolver{
		appID:   appID,
		profile: profile,
		env:     newEnv(cfg),
	}, nil
}

// Resolve resolves all storage class directories for one scenario.
//
// Returned paths are absolute and class-scoped to "<class-root>/<app>/<scenario>".
// ScenarioID is validated before resolution.
func (r *Resolver) Resolve(opts Options) (Paths, error) {
	scenarioID := cleanScenarioID(opts.ScenarioID)
	if !isValidScenarioID(scenarioID) {
		return Paths{}, &Error{Kind: ErrInvalidInput, Message: "invalid scenario id", Details: opts.ScenarioID}
	}

	roots, err := resolveClassRoots(r.profile, r.env, opts.RootOverride)
	if err != nil {
		return Paths{}, err
	}

	p := Paths{
		ConfigDir: filepath.Join(roots.config, r.appID, scenarioID),
		DataDir:   filepath.Join(roots.data, r.appID, scenarioID),
		CacheDir:  filepath.Join(roots.cache, r.appID, scenarioID),
		LogsDir:   filepath.Join(roots.logs, r.appID, scenarioID),
		StateDir:  filepath.Join(roots.state, r.appID, scenarioID),
	}

	return p, nil
}

// Path resolves a class root and safely appends rel.
//
// Safety rules:
//   - rel must be relative (not absolute)
//   - rel must not escape the class root via ".."
func (r *Resolver) Path(opts Options, class Class, rel string) (string, error) {
	paths, err := r.Resolve(opts)
	if err != nil {
		return "", err
	}
	base, err := paths.ForClass(class)
	if err != nil {
		return "", err
	}
	return cleanJoin(base, rel)
}

// ResolveOwner resolves a storage declaration through this resolver. A class
// declaration is the canonical form: the owner identity and class select the
// root, and Subpath is only a relative location below that root. Explicit
// path/byOS declarations remain readable during migration and are resolved by
// the same resolver so callers never choose a second path authority.
func (r *Resolver) ResolveOwner(repoRoot string, owner OwnerManifest, entry StorageEntry, requested Platform, seams PlatformSeams) (string, error) {
	platform := NormalizePlatform(string(requested))
	if platform == "" {
		return "", fmt.Errorf("unsupported storage platform %q", requested)
	}
	if !platformIncluded(EffectivePlatforms(owner, entry), platform) {
		return "", &NotApplicable{Entry: entry.Name, Platform: platform}
	}
	if entry.Class != "" && isClassDeclaration(entry) {
		if owner.Kind == OwnerScenario {
			if runtimeData := r.runtimeScenarioData(owner.ID); runtimeData != "" && entry.Class == ClassData {
				return cleanJoin(runtimeData, entry.Subpath)
			}
			return r.Path(Options{ScenarioID: owner.ID}, entry.Class, entry.Subpath)
		}
		return r.resolveNonScenarioClass(owner, entry, platform, seams)
	}
	return r.resolveLegacyOwnerPath(repoRoot, owner, entry, platform, seams)
}

func (r *Resolver) runtimeScenarioData(ownerID string) string {
	if r == nil || r.env.get == nil {
		return ""
	}
	current := strings.TrimSpace(r.env.get("SCENARIO_NAME"))
	if current == "" {
		current = strings.TrimSpace(r.env.get("VROOLI_SCENARIO"))
	}
	if current != "" && current != ownerID {
		return ""
	}
	return strings.TrimSpace(r.env.get("SCENARIO_DATA_DIR"))
}

func isClassDeclaration(entry StorageEntry) bool {
	return entry.Path.Value == "" && entry.Path.ByOS == nil
}

func (r *Resolver) resolveNonScenarioClass(owner OwnerManifest, entry StorageEntry, platform Platform, seams PlatformSeams) (string, error) {
	identity := SyntheticIdentity(platform)
	if platform == HostPlatform() {
		if home, err := hostHome(seams); err == nil && home != "" {
			identity.HomeDir = home
		}
	}
	resolvedSeams := mergeSeams(DefaultSeams(platform, identity), seams)
	var root string
	var err error
	switch entry.Class {
	case ClassConfig:
		root, err = resolvedSeams.UserConfigDir()
	case ClassData:
		root, err = userDataDir(platform, identity)
	case ClassCache:
		root, err = resolvedSeams.UserCacheDir()
	case ClassLogs:
		root, err = userLogsDir(platform, identity)
	case ClassState:
		root, err = resolvedSeams.UserStateDir()
	default:
		return "", &Error{Kind: ErrInvalidInput, Message: "unknown storage class", Details: string(entry.Class)}
	}
	if err != nil {
		return "", fmt.Errorf("resolve %s root: %w", entry.Class, err)
	}
	namespace := ownerNamespace(owner.Kind)
	base := filepath.Join(root, "vrooli", namespace, owner.ID)
	return cleanJoin(base, entry.Subpath)
}

func hostHome(seams PlatformSeams) (string, error) {
	if seams.UserHomeDir != nil {
		return seams.UserHomeDir()
	}
	identity, err := HostIdentity()
	return identity.HomeDir, err
}

func userDataDir(platform Platform, identity UserIdentity) (string, error) {
	switch NormalizePlatform(string(platform)) {
	case PlatformWindows:
		return filepath.FromSlash(strings.TrimRight(identity.HomeDir, `/\`) + `/AppData/Local`), nil
	case PlatformMacOS:
		return filepath.Join(identity.HomeDir, "Library", "Application Support"), nil
	default:
		return filepath.Join(identity.HomeDir, ".local", "share"), nil
	}
}

func userStateDir(platform Platform, identity UserIdentity, suffix string) (string, error) {
	base, err := userDataDir(platform, identity)
	if err != nil {
		return "", err
	}
	if platform == PlatformLinux {
		base = filepath.Join(identity.HomeDir, ".local", "state")
	}
	return filepath.Join(base, suffix), nil
}

func userLogsDir(platform Platform, identity UserIdentity) (string, error) {
	switch NormalizePlatform(string(platform)) {
	case PlatformWindows:
		return filepath.Join(identity.HomeDir, `AppData`, `Local`, `Logs`), nil
	case PlatformMacOS:
		return filepath.Join(identity.HomeDir, "Library", "Logs"), nil
	default:
		return filepath.Join(identity.HomeDir, ".local", "state", "logs"), nil
	}
}

func ownerNamespace(kind OwnerKind) string {
	switch kind {
	case OwnerResource:
		return "resources"
	case OwnerTool:
		return "tools"
	case OwnerSafeguard:
		return "safeguards"
	default:
		return string(kind)
	}
}

func (r *Resolver) resolveLegacyOwnerPath(repoRoot string, owner OwnerManifest, entry StorageEntry, platform Platform, seams PlatformSeams) (string, error) {
	value := strings.TrimSpace(entry.Path.Value)
	if entry.Path.ByOS != nil || isAbsoluteFor(platform, value) || containsPortableToken(value) {
		return ResolvePortablePath(entry.Name, entry.Path, platform, seams)
	}
	if value == "" {
		return "", fmt.Errorf("storage path is empty")
	}
	base := filepath.Dir(owner.ManifestPath)
	if owner.Kind == OwnerScenario {
		base = filepath.Dir(base)
	}
	if !filepath.IsAbs(base) && strings.TrimSpace(repoRoot) != "" {
		base = filepath.Join(repoRoot, base)
	}
	return filepath.Clean(filepath.Join(base, filepath.FromSlash(value))), nil
}
