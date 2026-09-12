package securestore

import (
	"context"
	"fmt"
	"os"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

// credentialStoreOwnershipRepairDo is a seam for tests. Production code
// always uses the setup-installed local broker; it never invokes a shell or
// asks the operator to repair the file outside Vrooli.
var credentialStoreOwnershipRepairDo = func(ctx context.Context, request privilegebroker.Request) (privilegebroker.Result, error) {
	return privilegebroker.NewClient().Do(ctx, request)
}

// RepairCredentialStoreOwnership makes the encrypted store readable by the
// invoking Vrooli user. The broker receives only the repository-contract class
// name and the expected identity; it resolves the exact secrets.enc.json path
// itself. A readable store is left untouched.
func RepairCredentialStoreOwnership() error {
	path, err := credentialStorePath()
	if err != nil {
		return fmt.Errorf("resolve credential store path for ownership repair: %w", err)
	}
	return repairCredentialStoreOwnership(path)
}

func repairCredentialStoreOwnership(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect credential store ownership: %w", err)
	}

	if readable, err := os.Open(path); err == nil {
		_ = readable.Close()
		return nil
	}

	// This handles the supported root-via-sudo case without involving the
	// daemon. Normal-user execution falls through to the broker, because only
	// the broker has authority to repair a root-owned file.
	_ = config.ChownToInvokingUser(path)
	if readable, err := os.Open(path); err == nil {
		_ = readable.Close()
		return nil
	}

	uid, gid := config.RepairIdentity()
	if uid == 0 || gid == 0 {
		return fmt.Errorf("credential store is not readable and its invoking-user identity is unavailable")
	}
	request := privilegebroker.Request{
		Version:   privilegebroker.ProtocolVersion,
		RequestID: fmt.Sprintf("credential-store-ownership-%d", time.Now().UnixNano()),
		Action:    privilegebroker.ActionRuntimeHomeOwnershipRepair,
		RuntimeHome: &privilegebroker.RuntimeHomeSubject{
			Class:       repocontract.HomeKeySecretsEnc,
			ExpectedUID: uid,
			ExpectedGID: gid,
		},
	}
	result, err := credentialStoreOwnershipRepairDo(context.Background(), request)
	if err != nil {
		return fmt.Errorf("repair credential store ownership through Vrooli privilege broker: %w", err)
	}
	if result.Status == "failed" {
		if result.Code == "" {
			return fmt.Errorf("Vrooli privilege broker could not repair credential store ownership")
		}
		return fmt.Errorf("Vrooli privilege broker could not repair credential store ownership: %s", result.Code)
	}
	readable, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("credential store remains unreadable after Vrooli ownership repair: %w", err)
	}
	_ = readable.Close()
	return nil
}
