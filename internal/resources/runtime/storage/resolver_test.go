package storage

import (
	"path/filepath"
	"testing"
)

func TestResolveUsesResourceScopedPaths(t *testing.T) {
	r, err := NewResolver(ResolverConfig{
		RuntimeOS: "linux",
		EnvGet:    func(string) string { return "" },
		UserHomeDir: func() (string, error) {
			return "/home/tester", nil
		},
		UserConfigDir: func() (string, error) {
			return "/home/tester/.config", nil
		},
		UserCacheDir: func() (string, error) {
			return "/home/tester/.cache", nil
		},
	})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	paths, err := r.Resolve(Options{ResourceID: "postgres"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if paths.DataDir != filepath.Join("/home/tester/.local/share", "vrooli", "resources", "postgres") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
	if paths.ConfigDir != filepath.Join("/home/tester/.config", "vrooli", "resources", "postgres") {
		t.Fatalf("ConfigDir = %q", paths.ConfigDir)
	}
}

func TestPathRejectsEscapes(t *testing.T) {
	r, err := NewResolver(ResolverConfig{
		RuntimeOS: "linux",
		EnvGet:    func(string) string { return "" },
		UserHomeDir: func() (string, error) {
			return "/home/tester", nil
		},
		UserConfigDir: func() (string, error) {
			return "/home/tester/.config", nil
		},
		UserCacheDir: func() (string, error) {
			return "/home/tester/.cache", nil
		},
	})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if _, err := r.Path(Options{ResourceID: "postgres"}, ClassData, "../escape"); err == nil {
		t.Fatal("expected escape rejection")
	}
}

func TestResolveHonorsRootOverride(t *testing.T) {
	r, err := NewResolver(ResolverConfig{})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	paths, err := r.Resolve(Options{ResourceID: "redis", RootOverride: "/tmp/vrooli-res"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if paths.LogsDir != filepath.Join("/tmp/vrooli-res", "logs", "vrooli", "resources", "redis") {
		t.Fatalf("LogsDir = %q", paths.LogsDir)
	}
}
