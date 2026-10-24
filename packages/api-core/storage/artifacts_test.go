package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactPathMatchesResolverPath(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	opts := Options{ScenarioID: "test-genie", RootOverride: t.TempDir()}
	ref := ArtifactRef{Owner: "test-genie", Domain: "runs", Class: ClassData, Segments: []string{"run-1", "phase-results", "unit.json"}}

	got, err := resolver.ArtifactPath(opts, ref)
	if err != nil {
		t.Fatal(err)
	}
	want, err := resolver.Path(opts, ClassData, filepath.Join("runs", "run-1", "phase-results", "unit.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ArtifactPath() = %q, Resolver.Path() = %q", got, want)
	}
}

func TestArtifactPathSeparatesLiveAndShadow(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	root := t.TempDir()
	ref := ArtifactRef{Owner: "test-genie", Domain: "runs", Class: ClassData, Segments: []string{"run-1"}}
	live, err := resolver.ArtifactPath(Options{ScenarioID: "test-genie", RootOverride: root}, ref)
	if err != nil {
		t.Fatal(err)
	}
	shadow, err := resolver.ArtifactPath(Options{ScenarioID: "test-genie_shadow", RootOverride: root}, ref)
	if err != nil {
		t.Fatal(err)
	}
	if live == shadow {
		t.Fatalf("live and shadow resolved to the same artifact path %q", live)
	}
	if want := filepath.Join(root, "data", "vrooli", "test-genie_shadow", "runs", "run-1"); shadow != want {
		t.Fatalf("shadow path = %q, want %q", shadow, want)
	}
}

func TestArtifactPathRejectsUnsafeSegmentsWithTypedError(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, ResolverConfig{EnvGet: mapEnv(nil)})
	for _, segment := range []string{"..", "run..old", ".hidden", "nested/file", `nested\file`, ""} {
		_, err := resolver.ArtifactPath(Options{RootOverride: t.TempDir()}, ArtifactRef{
			Owner: "test-genie", Domain: "runs", Class: ClassData, Segments: []string{segment},
		})
		var storageErr *Error
		if !errors.As(err, &storageErr) || storageErr.Kind != ErrInvalidInput {
			t.Errorf("segment %q error = %v, want typed invalid-input error", segment, err)
		}
	}
}

func TestArtifactPathUsesPortableClassRoots(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		home string
	}{
		{name: "linux", home: "/home/test"},
		{name: "macos", home: "/Users/test"},
		{name: "windows", home: "/C/Users/test"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resolver := mustResolver(t, ResolverConfig{
				AppID: "vrooli", EnvGet: mapEnv(nil), UserHomeDir: func() (string, error) { return tc.home, nil },
			})
			got, err := resolver.ArtifactPath(Options{}, ArtifactRef{
				Owner: "test-genie", Domain: "phase-cache", Class: ClassCache, Segments: []string{"entry.json"},
			})
			if err != nil {
				t.Fatal(err)
			}
			paths, err := resolver.Resolve(Options{ScenarioID: "test-genie"})
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(paths.CacheDir, "phase-cache", "entry.json")
			if got != want {
				t.Fatalf("ArtifactPath() = %q, want portable class path %q", got, want)
			}
		})
	}
}

func TestEnsureArtifactDirCreatesResolvedDirectory(t *testing.T) {
	t.Parallel()

	resolver := mustResolver(t, ResolverConfig{EnvGet: mapEnv(nil)})
	got, err := resolver.EnsureArtifactDir(Options{RootOverride: t.TempDir()}, ArtifactRef{
		Owner: "test-genie", Domain: "runs", Class: ClassData, Segments: []string{"run-1", "artifacts"},
	}, 0o750)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("resolved artifact directory %q is not a directory", got)
	}
}
