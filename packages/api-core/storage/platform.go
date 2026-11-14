package storage

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

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

// Resolve one batch through the runtime-home owner, including a compiled
// native pin. Source discovery cannot substitute for pinned path authority.
var loadRuntimeHomeEntries = repocontract.RuntimeHomeEntries

// Single-entry resolution is also used by non-scenario owners. Those callers
// do not need the batched class-root load above.
var runtimeHomeEntryPath = repocontract.RuntimeHomeEntryPath

type classRoots struct {
	config   string
	data     string
	cache    string
	logs     string
	state    string
	testRuns string
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
			config:   filepath.Join(rootOverride, string(ClassConfig)),
			data:     filepath.Join(rootOverride, string(ClassData)),
			cache:    filepath.Join(rootOverride, string(ClassCache)),
			logs:     filepath.Join(rootOverride, string(ClassLogs)),
			state:    filepath.Join(rootOverride, string(ClassState)),
			testRuns: filepath.Join(rootOverride, string(ClassTestRuns)),
		}, nil
	}

	if global := strings.TrimSpace(e.get(envStorageRoot)); global != "" {
		if !filepath.IsAbs(global) {
			return classRoots{}, &Error{Kind: ErrInvalidInput, Message: envStorageRoot + " must be absolute", Details: global}
		}
		return classRoots{
			config:   filepath.Join(global, string(ClassConfig)),
			data:     filepath.Join(global, string(ClassData)),
			cache:    filepath.Join(global, string(ClassCache)),
			logs:     filepath.Join(global, string(ClassLogs)),
			state:    filepath.Join(global, string(ClassState)),
			testRuns: filepath.Join(global, string(ClassTestRuns)),
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
	if err := refuseLiveRootsUnderTest(profile, e); err != nil {
		return classRoots{}, err
	}

	return roots, nil
}

// refuseLiveRootsUnderTest is the test-isolation backstop for every caller of
// the resolver. Under `go test`, resolving the operator's own runtime home
// (the account home, with any class root left at its default) is refused, so a
// test cannot read or write live per-install storage whatever file it
// targets. A test isolates with VROOLI_STORAGE_ROOT (storagetest), a
// redirected HOME, an explicit Options.RootOverride, or every per-class root.
func refuseLiveRootsUnderTest(profile Profile, e env) error {
	if !testing.Testing() {
		return nil
	}
	switch profile {
	case "", ProfileAuto, ProfileDesktop, ProfileMobile:
	default:
		return nil
	}
	allRedirected := true
	for _, key := range []string{envConfigRoot, envDataRoot, envCacheRoot, envLogsRoot, envStateRoot} {
		if strings.TrimSpace(e.get(key)) == "" {
			allRedirected = false
			break
		}
	}
	if allRedirected {
		return nil
	}
	home, err := e.userHomeDir()
	if err != nil {
		return nil
	}
	account, err := user.Current()
	if err != nil || strings.TrimSpace(account.HomeDir) == "" || filepath.Clean(home) != filepath.Clean(account.HomeDir) {
		return nil
	}
	return &Error{
		Kind:    ErrResolve,
		Message: "a test resolved the operator's live storage",
		Details: "isolate with storagetest.RunWithIsolatedStorage (TestMain) or storagetest.IsolateStorage, a redirected HOME, or Options.RootOverride",
	}
}

func defaultClassRoots(profile Profile, e env) (classRoots, error) {
	switch profile {
	case "", ProfileAuto, ProfileDesktop, ProfileMobile:
		return defaultUserClassRoots(e)
	case ProfileVPS:
		return classRoots{
			config:   "/etc",
			data:     "/var/lib",
			cache:    "/var/cache",
			logs:     "/var/log",
			state:    "/var/lib/vrooli-state",
			testRuns: "/var/lib/vrooli-test-runs",
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
	entries, err := loadRuntimeHomeEntries(home)
	if err != nil {
		return classRoots{}, &Error{Kind: ErrResolve, Message: "resolve runtime-home class roots", Err: err}
	}
	byKey := make(map[string]repocontract.HomeEntry, len(entries))
	for _, entry := range entries {
		byKey[entry.Key] = entry
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
		{repocontract.HomeKeyTestRuns, &roots.testRuns},
	} {
		entry, ok := byKey[m.key]
		if !ok {
			return classRoots{}, &Error{Kind: ErrResolve, Message: "runtime-home class root is missing", Details: m.key}
		}
		*m.dst = entry.AbsPath
	}
	return roots, nil
}
