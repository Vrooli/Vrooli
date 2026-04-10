package storage

import (
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
