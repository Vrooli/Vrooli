// Package hostpaths resolves the filesystem roots the built-in file cleanup
// providers walk.
//
// It lives outside internal/ deliberately. internal/ is seam-only by contract
// (see internal/cleanup/no_real_cleanup_test.go): nothing under it may touch the
// real host. Root resolution reads real environment variables and the real user
// home, so it belongs at the edge alongside handlers/ and hostfs/.
//
// # Why these roots are not storage.Resolver paths
//
// api-core/storage is the authority for paths Vrooli itself owns — the
// config/data/cache/logs/state classes under the operator runtime home. Every
// root here is the opposite: a location owned by somebody else that Vrooli only
// cleans up after. The OS temp directory, the XDG trash, Go's build cache and
// Playwright's browser cache are all foreign namespaces with their own layout
// rules, and storage deliberately models no "temp" class at all. Resolving them
// through storage would invent Vrooli-shaped paths that the owning tools never
// write to, so the providers would walk empty directories forever — which is a
// slower version of the bug this package exists to fix.
//
// Cross-platform resolution therefore leans on the standard library, which
// already encodes the per-OS rules:
//
//	os.TempDir()      TMPDIR (unix) / TMP,TEMP (windows), with per-OS defaults
//	os.UserCacheDir() $XDG_CACHE_HOME|~/.cache, ~/Library/Caches, %LocalAppData%
//
// Only the trash root genuinely diverges per OS with no stdlib equivalent, so
// that one — and only that one — is build-tagged. See trash_*.go.
package hostpaths

import (
	"os"
	"path/filepath"
	"strings"
)

// Roots bundles the resolved roots for each built-in file provider.
//
// Every field may be empty. An empty root list is a normal, correct outcome —
// a host with no Playwright install has no Playwright cache — and leaves the
// corresponding provider reporting nothing rather than failing.
type Roots struct {
	Tmp             []string
	Trash           []string
	GoBuildCache    []string
	PlaywrightCache []string
}

// Resolve determines the cleanup roots for the current host and user.
//
// It never returns an error. Root resolution is best-effort by design: a host
// that cannot report a home directory should still get temp-directory cleanup
// rather than failing to construct the registry and taking the whole scenario
// down. Unresolvable roots are simply absent.
func Resolve() Roots {
	return Roots{
		Tmp:             existing(os.TempDir()),
		Trash:           existing(trashRoots()...),
		GoBuildCache:    existing(goBuildCacheRoot()),
		PlaywrightCache: existing(playwrightCacheRoot()),
	}
}

// goBuildCacheRoot resolves Go's build cache.
//
// GOCACHE is consulted first because that is exactly the precedence the go
// command itself uses, so an operator who moved their cache gets it cleaned
// rather than silently skipped. The fallback mirrors Go's own default of
// <UserCacheDir>/go-build, which os.UserCacheDir already resolves correctly on
// linux, darwin, and windows.
//
// GOCACHE=off is a valid setting meaning "no cache"; it is not a path, and
// treating it as one would produce a nonsense root.
func goBuildCacheRoot() string {
	if cache := strings.TrimSpace(os.Getenv("GOCACHE")); cache != "" {
		if strings.EqualFold(cache, "off") {
			return ""
		}
		return cache
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "go-build")
}

// playwrightCacheRoot resolves Playwright's downloaded-browser cache.
//
// PLAYWRIGHT_BROWSERS_PATH is Playwright's own override and takes precedence.
// The value "0" is special-cased by Playwright to mean "install beside the
// package" rather than naming a directory, so it is not a root.
//
// The default is <UserCacheDir>/ms-playwright, which happens to be correct on
// all three platforms once os.UserCacheDir has applied the per-OS rule —
// ~/.cache on linux, ~/Library/Caches on darwin, %LocalAppData% on windows.
func playwrightCacheRoot() string {
	if path := strings.TrimSpace(os.Getenv("PLAYWRIGHT_BROWSERS_PATH")); path != "" {
		if path == "0" {
			return ""
		}
		return path
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ms-playwright")
}

// existing filters to absolute paths that are present and are directories.
//
// Filtering here rather than in the providers keeps a missing directory from
// being reported as a walk failure later: a host without a trash directory is
// ordinary, not broken. Relative paths are dropped because every downstream
// containment check (FileProvider.withinConfiguredRoot) compares cleaned
// absolute paths, and a relative root could never match one.
func existing(candidates ...string) []string {
	out := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		path := filepath.Clean(candidate)
		if !filepath.IsAbs(path) || seen[path] {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
