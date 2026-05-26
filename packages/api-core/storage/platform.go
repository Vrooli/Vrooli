package storage

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	envStorageRoot = "VROOLI_STORAGE_ROOT"
	envConfigRoot  = "VROOLI_CONFIG_ROOT"
	envDataRoot    = "VROOLI_DATA_ROOT"
	envCacheRoot   = "VROOLI_CACHE_ROOT"
	envLogsRoot    = "VROOLI_LOGS_ROOT"
	envStateRoot   = "VROOLI_STATE_ROOT"
)

// runtimeHomeEntryPath resolves a well-known runtime-home entry path through the
// repo-contract authority. It is a package var so tests can inject contract-load
// failures (mirrors the seam style in api-core/secrets); production always uses
// the real authority.
var runtimeHomeEntryPath = repocontract.RuntimeHomeEntryPath

type classRoots struct {
	config string
	data   string
	cache  string
	logs   string
	state  string
}

type env struct {
	get         func(string) string
	userHomeDir func() (string, error)
}

func newEnv(cfg ResolverConfig) env {
	get := cfg.EnvGet
	if get == nil {
		get = os.Getenv
	}
	home := cfg.UserHomeDir
	if home == nil {
		home = os.UserHomeDir
	}
	return env{get: get, userHomeDir: home}
}

func resolveClassRoots(profile Profile, e env, rootOverride string) (classRoots, error) {
	if rootOverride != "" {
		rootOverride = strings.TrimSpace(rootOverride)
		if !filepath.IsAbs(rootOverride) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: "root override must be absolute", Details: rootOverride}
		}
		return classRoots{
			config: filepath.Join(rootOverride, string(ClassConfig)),
			data:   filepath.Join(rootOverride, string(ClassData)),
			cache:  filepath.Join(rootOverride, string(ClassCache)),
			logs:   filepath.Join(rootOverride, string(ClassLogs)),
			state:  filepath.Join(rootOverride, string(ClassState)),
		}, nil
	}

	if global := strings.TrimSpace(e.get(envStorageRoot)); global != "" {
		if !filepath.IsAbs(global) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envStorageRoot + " must be absolute", Details: global}
		}
		return classRoots{
			config: filepath.Join(global, string(ClassConfig)),
			data:   filepath.Join(global, string(ClassData)),
			cache:  filepath.Join(global, string(ClassCache)),
			logs:   filepath.Join(global, string(ClassLogs)),
			state:  filepath.Join(global, string(ClassState)),
		}, nil
	}

	roots, err := defaultClassRoots(profile, e)
	if err != nil {
		return classRoots{}, err
	}

	if v := strings.TrimSpace(e.get(envConfigRoot)); v != "" {
		if !filepath.IsAbs(v) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envConfigRoot + " must be absolute", Details: v}
		}
		roots.config = v
	}
	if v := strings.TrimSpace(e.get(envDataRoot)); v != "" {
		if !filepath.IsAbs(v) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envDataRoot + " must be absolute", Details: v}
		}
		roots.data = v
	}
	if v := strings.TrimSpace(e.get(envCacheRoot)); v != "" {
		if !filepath.IsAbs(v) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envCacheRoot + " must be absolute", Details: v}
		}
		roots.cache = v
	}
	if v := strings.TrimSpace(e.get(envLogsRoot)); v != "" {
		if !filepath.IsAbs(v) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envLogsRoot + " must be absolute", Details: v}
		}
		roots.logs = v
	}
	if v := strings.TrimSpace(e.get(envStateRoot)); v != "" {
		if !filepath.IsAbs(v) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envStateRoot + " must be absolute", Details: v}
		}
		roots.state = v
	}

	return roots, nil
}

func defaultClassRoots(profile Profile, e env) (classRoots, error) {
	switch profile {
	case "", ProfileAuto, ProfileDesktop, ProfileMobile:
		return defaultUserClassRoots(e)
	case ProfileVPS:
		return classRoots{
			config: "/etc",
			data:   "/var/lib",
			cache:  "/var/cache",
			logs:   "/var/log",
			state:  "/var/lib/vrooli-state",
		}, nil
	default:
		return classRoots{}, &Error{Kind: ErrInvalidInput, Message: "unknown storage profile", Details: string(profile)}
	}
}

// defaultUserClassRoots resolves the user-profile class roots under the operator
// runtime home (~/.vrooli/{config,data,cache,logs,state}) via the repo-contract
// runtime_home authority. The roots are OS-agnostic by design: the runtime home
// is operator-home-shaped, so there is no XDG/darwin/windows branching here. A
// contract that cannot be loaded is a hard error (no silent fallback), matching
// docs/repo-contract.md's "no fallback" stance.
func defaultUserClassRoots(e env) (classRoots, error) {
	home, err := e.userHomeDir()
	if err != nil {
		return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve user home dir", Err: err}
	}

	roots := classRoots{}
	for _, m := range []struct {
		key string
		dst *string
	}{
		{repocontract.HomeKeyConfig, &roots.config},
		{repocontract.HomeKeyData, &roots.data},
		{repocontract.HomeKeyCache, &roots.cache},
		{repocontract.HomeKeyLogs, &roots.logs},
		{repocontract.HomeKeyState, &roots.state},
	} {
		path, err := runtimeHomeEntryPath(home, m.key)
		if err != nil {
			return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve runtime-home class root", Details: m.key, Err: err}
		}
		*m.dst = path
	}
	return roots, nil
}
