// Package storagetest isolates contract-routed storage for tests. Under
// `go test` the storage resolver refuses the operator's live runtime home, so
// a package whose code resolves storage on its own isolates it here.
//
// This package is test-only; production code must not import it.
package storagetest

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// RootEnv is the resolver's global class-root override. When set, every
// contract-routed class resolves beneath it and per-class overrides are
// ignored.
const RootEnv = "VROOLI_STORAGE_ROOT"

// RunWithIsolatedStorage runs a package's tests with all contract-routed
// storage under a private temporary root, then removes it. Child processes
// the tests start inherit the root. Use it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
func RunWithIsolatedStorage(m *testing.M) int {
	root, err := os.MkdirTemp("", "vrooli-test-storage-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create isolated test storage root: %v\n", err)
		return 1
	}
	defer os.RemoveAll(root)
	previous, had := os.LookupEnv(RootEnv)
	if err := os.Setenv(RootEnv, root); err != nil {
		fmt.Fprintf(os.Stderr, "isolate test storage: %v\n", err)
		return 1
	}
	defer func() {
		if had {
			_ = os.Setenv(RootEnv, previous)
		} else {
			_ = os.Unsetenv(RootEnv)
		}
	}()
	return m.Run()
}

var isolatedRoots sync.Map // testing.TB -> string

// IsolateStorage gives one test its own storage root for the rest of the test
// and returns it, so state one test adopts cannot leak into the next. Repeated
// calls within a test return the same root.
func IsolateStorage(tb testing.TB) string {
	tb.Helper()
	if root, ok := isolatedRoots.Load(tb); ok {
		return root.(string)
	}
	root := tb.TempDir()
	tb.Setenv(RootEnv, root)
	isolatedRoots.Store(tb, root)
	tb.Cleanup(func() { isolatedRoots.Delete(tb) })
	return root
}

// RunWithPrivateHome runs a package's tests with HOME pointed at a private
// temporary directory. Use it instead of RunWithIsolatedStorage when tests
// resolve storage through an injected environment (ResolverConfig.EnvGet,
// SQLiteConfig.EnvGet): an injected environment does not see
// VROOLI_STORAGE_ROOT, but the default runtime home still follows HOME. It also
// suits tests that redirect HOME themselves and expect default-layout paths
// beneath it, which a global storage root would override. The Go toolchain's
// own locations are pinned to their current values first, so tests that run
// `go` keep the warm build cache and module cache.
func RunWithPrivateHome(m *testing.M) int {
	pinGoToolchainLocations()
	home, err := os.MkdirTemp("", "vrooli-test-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create private test home: %v\n", err)
		return 1
	}
	defer os.RemoveAll(home)
	previous, had := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		fmt.Fprintf(os.Stderr, "set private test home: %v\n", err)
		return 1
	}
	defer func() {
		if had {
			_ = os.Setenv("HOME", previous)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()
	return m.Run()
}

// pinGoToolchainLocations fixes GOPATH, GOCACHE and GOENV, whose defaults
// derive from HOME, at the values they have before HOME moves. A variable the
// caller already set is left alone.
func pinGoToolchainLocations() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	pin := func(key, value string) {
		if _, set := os.LookupEnv(key); !set && value != "" {
			_ = os.Setenv(key, value)
		}
	}
	pin("GOPATH", filepath.Join(home, "go"))
	if cache, err := os.UserCacheDir(); err == nil {
		pin("GOCACHE", filepath.Join(cache, "go-build"))
	}
	if config, err := os.UserConfigDir(); err == nil {
		pin("GOENV", filepath.Join(config, "go", "env"))
	}
}
