package storagetest

import (
	"os"
	"testing"
)

func TestIsolateStorageSetsAndRestoresRoot(t *testing.T) {
	t.Setenv(RootEnv, "/outer")
	var inner string
	t.Run("inner", func(tt *testing.T) {
		inner = IsolateStorage(tt)
		if got := os.Getenv(RootEnv); got != inner {
			tt.Fatalf("%s = %q, want %q", RootEnv, got, inner)
		}
		if again := IsolateStorage(tt); again != inner {
			tt.Fatalf("second call returned %q, want %q", again, inner)
		}
	})
	if got := os.Getenv(RootEnv); got != "/outer" {
		t.Fatalf("%s after cleanup = %q, want /outer", RootEnv, got)
	}
}

func TestPinGoToolchainLocationsKeepsHomeDerivedDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	for _, key := range []string{"GOPATH", "GOCACHE", "GOENV"} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
	pinGoToolchainLocations()
	if got := os.Getenv("GOPATH"); got != home+"/go" {
		t.Fatalf("GOPATH = %q, want %q", got, home+"/go")
	}
	if got := os.Getenv("GOCACHE"); got != home+"/.cache/go-build" {
		t.Fatalf("GOCACHE = %q", got)
	}
	t.Setenv("GOPATH", "/explicit")
	pinGoToolchainLocations()
	if got := os.Getenv("GOPATH"); got != "/explicit" {
		t.Fatalf("explicit GOPATH overwritten: %q", got)
	}
}
