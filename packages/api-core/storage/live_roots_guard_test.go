package storage

import (
	"errors"
	"os/user"
	"testing"
)

// Under go test the resolver must refuse the operator's own runtime home, so no
// test can read or write live per-install storage, whichever file it targets.
func TestResolverRefusesLiveStorageUnderTest(t *testing.T) {
	account, err := user.Current()
	if err != nil {
		t.Skipf("account home unavailable: %v", err)
	}
	t.Setenv("HOME", account.HomeDir)
	for _, key := range []string{envStorageRoot, envConfigRoot, envDataRoot, envCacheRoot, envLogsRoot, envStateRoot} {
		t.Setenv(key, "")
	}
	resolver, err := NewResolver(ResolverConfig{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(Options{ScenarioID: "demo"})
	var storageErr *Error
	if !errors.As(err, &storageErr) || storageErr.Kind != ErrResolve {
		t.Fatalf("Resolve() error = %v, want the live-storage refusal", err)
	}
	if _, err := resolver.Resolve(Options{ScenarioID: "demo", RootOverride: t.TempDir()}); err != nil {
		t.Fatalf("explicit RootOverride refused: %v", err)
	}
	t.Setenv(envStateRoot, t.TempDir())
	if _, err := resolver.Resolve(Options{ScenarioID: "demo"}); err == nil {
		t.Fatal("a single redirected class left the other classes live yet was accepted")
	}
	t.Setenv(envStorageRoot, t.TempDir())
	if _, err := resolver.Resolve(Options{ScenarioID: "demo"}); err != nil {
		t.Fatalf("VROOLI_STORAGE_ROOT refused: %v", err)
	}
}
