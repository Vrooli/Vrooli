package credentials

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/domain"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// ClientRecovery is the lost-host recovery client over the remote credential
// authority: `vrooli credentials recovery verify|restore` on the replacement
// host through packages/credentialclient-go. The passphrase travels on the
// verb's standard input; the bundle is already on the replacement host.
type ClientRecovery struct {
	Client credentialclient.Client
	Target string
}

// Verify inspects the bundle and returns the descriptors it covers. A locked
// or unavailable store on the replacement host is credential_store_locked.
func (r ClientRecovery) Verify(ctx context.Context, bundleRef, passphrase string) ([]domain.CredentialDescriptor, error) {
	if err := r.storeReady(ctx); err != nil {
		return nil, err
	}
	resp, err := r.Client.RecoveryVerify(ctx, credentialclient.RecoveryVerifyRequest{InputPath: bundleRef, Passphrase: passphrase})
	if err != nil {
		return nil, fmt.Errorf("verify recovery bundle: %w", err)
	}
	out := make([]domain.CredentialDescriptor, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		descriptor, err := domain.ParseCredentialDescriptor(entry)
		if err != nil {
			return nil, err
		}
		out = append(out, descriptor)
	}
	return out, nil
}

// Restore writes the bundle into the replacement host's authority.
func (r ClientRecovery) Restore(ctx context.Context, bundleRef, passphrase string) error {
	if err := r.storeReady(ctx); err != nil {
		return err
	}
	if err := r.Client.RecoveryRestore(ctx, credentialclient.RecoveryRestoreRequest{InputPath: bundleRef, Passphrase: passphrase}); err != nil {
		return fmt.Errorf("restore recovery bundle: %w", err)
	}
	return nil
}

func (r ClientRecovery) storeReady(ctx context.Context) error {
	status, err := r.Client.StoreStatus(ctx)
	if err != nil {
		return fmt.Errorf("read replacement host store status: %w", err)
	}
	if status.Initialized && !status.Unlocked {
		return StoreLocked(strings.TrimSpace(r.Target))
	}
	return nil
}
