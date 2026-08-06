package repoctx_test

import (
	"testing"

	"resource-kopia/cli/internal/credentials/mocks"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"
)

func TestFilesystemResolutionUsesAuthorityWithoutVault(t *testing.T) {
	store := mocks.NewFakeStore()
	store.SeedPassphrase("nightly", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(t.TempDir() + "/registry.json")
	if err := reg.Upsert(registry.Entry{
		Name:       "nightly",
		Backend:    registry.BackendFilesystem,
		ConfigFile: "/tmp/nightly.config",
		Path:       "/tmp/nightly",
	}); err != nil {
		t.Fatal(err)
	}

	target, err := (repoctx.Resolver{Registry: reg, Credentials: store}).Resolve(t.Context(), "nightly")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if target.Env[repoctx.EnvPassword] == "" {
		t.Fatal("filesystem resolution did not inject the authority passphrase")
	}
	if _, ok := target.Env[repoctx.EnvAWSAccessKey]; ok {
		t.Fatal("filesystem resolution injected S3 credentials")
	}
}
