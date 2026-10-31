package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	envStorageRoot = "VROOLI_RESOURCE_STORAGE_ROOT"
	envConfigRoot  = "VROOLI_RESOURCE_CONFIG_ROOT"
	envDataRoot    = "VROOLI_RESOURCE_DATA_ROOT"
	envCacheRoot   = "VROOLI_RESOURCE_CACHE_ROOT"
	envLogsRoot    = "VROOLI_RESOURCE_LOGS_ROOT"
	envStateRoot   = "VROOLI_RESOURCE_STATE_ROOT"
)

type classRoots struct {
	config string
	data   string
	cache  string
	logs   string
	state  string
}

type env struct {
	get           func(string) string
	os            string
	userHomeDir   func() (string, error)
	userConfigDir func() (string, error)
	userCacheDir  func() (string, error)
}

func newEnv(cfg ResolverConfig) env {
	get := cfg.EnvGet
	if get == nil {
		get = os.Getenv
	}
	osName := cfg.RuntimeOS
	if osName == "" {
		osName = runtime.GOOS
	}
	home := cfg.UserHomeDir
	if home == nil {
		home = os.UserHomeDir
	}
	configDir := cfg.UserConfigDir
	if configDir == nil {
		configDir = os.UserConfigDir
	}
	cacheDir := cfg.UserCacheDir
	if cacheDir == nil {
		cacheDir = os.UserCacheDir
	}
	return env{get: get, os: osName, userHomeDir: home, userConfigDir: configDir, userCacheDir: cacheDir}
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
			state:  "/var/lib/vrooli-resource-state",
		}, nil
	default:
		return classRoots{}, &Error{Kind: ErrInvalidInput, Message: "unknown storage profile", Details: string(profile)}
	}
}

func defaultUserClassRoots(e env) (classRoots, error) {
	configDir, err := e.userConfigDir()
	if err != nil {
		return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve user config dir", Err: err}
	}
	cacheDir, err := e.userCacheDir()
	if err != nil {
		return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve user cache dir", Err: err}
	}
	homeDir, err := e.userHomeDir()
	if err != nil {
		return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve user home dir", Err: err}
	}

	switch e.os {
	case string(hostreqspec.PlatformWindows):
		localAppData := strings.TrimSpace(e.get("LOCALAPPDATA"))
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		return classRoots{
			config: configDir,
			data:   configDir,
			cache:  cacheDir,
			logs:   filepath.Join(localAppData, "Logs"),
			state:  filepath.Join(localAppData, "State"),
		}, nil
	case string(hostreqspec.PlatformDarwin):
		library := filepath.Join(homeDir, "Library")
		return classRoots{
			config: filepath.Join(library, "Application Support"),
			data:   filepath.Join(library, "Application Support"),
			cache:  filepath.Join(library, "Caches"),
			logs:   filepath.Join(library, "Logs"),
			state:  filepath.Join(library, "Application Support", "State"),
		}, nil
	default:
		dataHome := strings.TrimSpace(e.get("XDG_DATA_HOME"))
		if dataHome == "" {
			dataHome = filepath.Join(homeDir, ".local", "share")
		}
		stateHome := strings.TrimSpace(e.get("XDG_STATE_HOME"))
		if stateHome == "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		return classRoots{
			config: configDir,
			data:   dataHome,
			cache:  cacheDir,
			logs:   filepath.Join(stateHome, "logs"),
			state:  stateHome,
		}, nil
	}
}
