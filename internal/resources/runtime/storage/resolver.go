package storage

import (
	"path/filepath"
	"strings"
)

const defaultAppID = "vrooli"

type Resolver struct {
	appID   string
	profile Profile
	env     env
}

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

func (r *Resolver) Resolve(opts Options) (Paths, error) {
	resourceID := cleanResourceID(opts.ResourceID)
	if !isValidResourceID(resourceID) {
		return Paths{}, &Error{Kind: ErrInvalidInput, Message: "invalid resource id", Details: opts.ResourceID}
	}

	roots, err := resolveClassRoots(r.profile, r.env, opts.RootOverride)
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		ConfigDir: filepath.Join(roots.config, r.appID, "resources", resourceID),
		DataDir:   filepath.Join(roots.data, r.appID, "resources", resourceID),
		CacheDir:  filepath.Join(roots.cache, r.appID, "resources", resourceID),
		LogsDir:   filepath.Join(roots.logs, r.appID, "resources", resourceID),
		StateDir:  filepath.Join(roots.state, r.appID, "resources", resourceID),
	}, nil
}

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

