package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/vrooli/vrooli/internal/credentialauthority"
)

func acquireCloudflareConnectorLaunchLock(ctx context.Context, manifest ResourceManifest) (func(), error) {
	if manifest.ManagedService != nil {
		for _, declaration := range manifest.ManagedService.CredentialFiles {
			if declaration.LogicalID == "vrooli/tunnel-manager" && declaration.Field == "cloudflare-connector-token" {
				return credentialauthority.AcquireCloudflareConnectorLock(ctx)
			}
		}
	}
	return func() {}, nil
}

// Hold the same lock across the stop and replacement launch so a pending or
// concurrent rotation cannot turn a serving restart into an avoidable outage.
func prepareCloudflareConnectorRestart(ctx context.Context, manifest ResourceManifest) (func(), error) {
	release, err := acquireCloudflareConnectorLaunchLock(ctx, manifest)
	if err != nil {
		return nil, fmt.Errorf("acquire cloudflare connector restart lock: %w", err)
	}
	if err := guardCloudflareConnectorRotation(manifest); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

// A rotation intent is durable before the remote mutation and remains until
// the connector credential has been reconciled. Never expose its value.
func guardCloudflareConnectorRotation(manifest ResourceManifest) error {
	if manifest.ManagedService == nil {
		return nil
	}
	for _, declaration := range manifest.ManagedService.CredentialFiles {
		if declaration.LogicalID != "vrooli/tunnel-manager" || declaration.Field != "cloudflare-connector-token" {
			continue
		}
		authority, err := credentialauthority.DefaultAuthority()
		if err != nil {
			return fmt.Errorf("cloudflare connector rotation authority unavailable: %w", err)
		}
		identity, err := credentialauthority.ParseIdentity(declaration.LogicalID)
		if err != nil {
			return err
		}
		_, err = authority.Resolve(identity, "cloudflare-connector-rotation-intent")
		if err == nil {
			return fmt.Errorf("cloudflare connector rotation is pending; reconcile credentials before launching")
		}
		if !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return fmt.Errorf("cloudflare connector rotation authority unavailable: %w", err)
		}
	}
	return nil
}

// An update snapshot can predate a completed rotation, even after the intent
// has been cleared. Do not launch a connector from that stale snapshot.
func verifyCloudflareConnectorSnapshot(logicalID, field string, value []byte) error {
	if logicalID != "vrooli/tunnel-manager" || field != "cloudflare-connector-token" {
		return nil
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return fmt.Errorf("cloudflare connector authority unavailable: %w", err)
	}
	identity, err := credentialauthority.ParseIdentity(logicalID)
	if err != nil {
		return err
	}
	current, err := authority.Resolve(identity, field)
	if err != nil {
		return fmt.Errorf("cloudflare connector authority unavailable: %w", err)
	}
	if !bytes.Equal(value, []byte(current)) {
		return fmt.Errorf("cloudflare connector update snapshot is stale; prepare a new update")
	}
	return nil
}
