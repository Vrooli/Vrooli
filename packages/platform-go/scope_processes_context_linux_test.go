//go:build linux

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScopeProcessesContextVerifiesCollectedScope(t *testing.T) {
	for _, load := range []string{"not-found", "loaded"} {
		t.Run(load, func(t *testing.T) {
			ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-collected.scope", nil)
			ref.Name = "vrooli-test-collected"
			if err := os.RemoveAll(filepath.Join(cgroupMount, ref.Path)); err != nil {
				t.Fatal(err)
			}
			old := queryScopeState
			t.Cleanup(func() { queryScopeState = old })
			called := false
			queryScopeState = func(context.Context, string) (string, string, error) { called = true; return load, "inactive", nil }
			pids, err := ScopeProcessesContext(context.Background(), ref)
			if err != nil || len(pids) != 0 || !called {
				t.Fatalf("gone scope rejected %v %v queried=%t", pids, err, called)
			}
		})
	}
}

func TestScopeProcessesContextRefusesInconsistentOrUnknownGoneState(t *testing.T) {
	for _, kind := range []string{"membership-only", "active-unit", "query-error", "wrong-name"} {
		t.Run(kind, func(t *testing.T) {
			ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-collected.scope", nil)
			ref.Name = "vrooli-test-collected"
			if kind != "membership-only" {
				_ = os.RemoveAll(filepath.Join(cgroupMount, ref.Path))
			}
			if kind == "wrong-name" {
				ref.Name = "other"
			}
			old := queryScopeState
			t.Cleanup(func() { queryScopeState = old })
			queryScopeState = func(context.Context, string) (string, string, error) {
				if kind == "query-error" {
					return "", "", errors.New("bus unavailable")
				}
				if kind == "active-unit" {
					return "loaded", "active", nil
				}
				return "not-found", "inactive", nil
			}
			if _, err := ScopeProcessesContext(context.Background(), ref); err == nil {
				t.Fatal("unverified empty state accepted")
			}
		})
	}
}
