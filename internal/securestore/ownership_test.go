package securestore

import (
	"context"
	"os"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

func TestRepairCredentialStoreOwnershipUsesTypedBrokerForUnreadableStore(t *testing.T) {
	if os.Getuid() == 0 || os.Getgid() == 0 {
		t.Skip("the ownership-repair test requires a non-root test user")
	}
	path := t.TempDir() + "/secrets.enc.json"
	if err := os.WriteFile(path, []byte("sealed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}

	previousPath := credentialStorePath
	previousRepair := credentialStoreOwnershipRepairDo
	t.Cleanup(func() {
		credentialStorePath = previousPath
		credentialStoreOwnershipRepairDo = previousRepair
	})
	credentialStorePath = func() (string, error) { return path, nil }
	credentialStoreOwnershipRepairDo = func(_ context.Context, request privilegebroker.Request) (privilegebroker.Result, error) {
		if request.Action != privilegebroker.ActionRuntimeHomeOwnershipRepair || request.RuntimeHome == nil || request.RuntimeHome.Class != repocontract.HomeKeySecretsEnc {
			t.Fatalf("unexpected broker request: %+v", request)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return privilegebroker.Result{}, err
		}
		return privilegebroker.Result{
			Version: privilegebroker.ProtocolVersion,
			Action:  privilegebroker.ActionRuntimeHomeOwnershipRepair,
			Status:  "changed",
			Changed: true,
		}, nil
	}

	if err := RepairCredentialStoreOwnership(); err != nil {
		t.Fatalf("RepairCredentialStoreOwnership: %v", err)
	}
	if mode := mustFileMode(t, path); mode != 0o600 {
		t.Fatalf("repaired mode = %#o, want 0600", mode)
	}
}

func mustFileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
