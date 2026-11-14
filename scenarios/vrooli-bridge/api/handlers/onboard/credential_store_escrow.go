package onboard

import (
	"context"
	"fmt"

	internalgrant "vrooli-bridge/internal/credentialgrant"
	internalonboard "vrooli-bridge/internal/onboard"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// NewCredentialStoreEscrow keeps node credential-store passphrases in the
// control-plane credential authority — the custody boundary Bridge already
// uses for durable named secrets (credential grants), never its DB. Each
// machine has one address under internalonboard.CredentialStoreEscrowNamespace.
func NewCredentialStoreEscrow() internalonboard.CredentialStoreEscrow {
	return authorityEscrow{}
}

type authorityEscrow struct{}

func (authorityEscrow) identity(key string) (credentialauthority.Identity, error) {
	return credentialauthority.ParseIdentity(internalonboard.CredentialStoreLogicalID(key))
}

func (e authorityEscrow) load(_ context.Context, key, field string) ([]byte, bool, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, false, fmt.Errorf("control-plane credential authority unavailable: %w", err)
	}
	identity, err := e.identity(key)
	if err != nil {
		return nil, false, err
	}
	status := authority.Status(identity, field)
	if !status.Configured {
		if state := string(status.ProviderState); state != "" && state != "available" {
			return nil, false, fmt.Errorf("control-plane credential store is %s: %s", state, status.ProviderDetail)
		}
		return nil, false, nil
	}
	value, err := authority.Require(identity, internalonboard.CredentialStoreEscrowField)
	if err != nil {
		return nil, false, fmt.Errorf("read escrowed node passphrase: %w", err)
	}
	// The authority API is string-typed, so this copy cannot be zeroed; it is
	// process-memory-only, like the SSH KeyCopier residue in SECURITY.md.
	return []byte(value), true, nil
}

func (e authorityEscrow) Load(ctx context.Context, key string) ([]byte, bool, error) {
	return e.load(ctx, key, internalonboard.CredentialStoreEscrowField)
}

func (e authorityEscrow) Save(ctx context.Context, key string, secret []byte) error {
	return e.save(ctx, key, internalonboard.CredentialStoreEscrowField, secret)
}

// LoadPending, SavePending, and ClearPending hold a rotation's new passphrase
// at a second field of the same address. No grant names that field, so it is
// never pushed to a node.
func (e authorityEscrow) LoadPending(ctx context.Context, key string) ([]byte, bool, error) {
	return e.load(ctx, key, internalonboard.CredentialStoreEscrowPendingField)
}

func (e authorityEscrow) SavePending(ctx context.Context, key string, secret []byte) error {
	return e.save(ctx, key, internalonboard.CredentialStoreEscrowPendingField, secret)
}

func (e authorityEscrow) ClearPending(_ context.Context, key string) error {
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("control-plane credential authority unavailable: %w", err)
	}
	identity, err := e.identity(key)
	if err != nil {
		return err
	}
	if err := authority.Delete(identity, internalonboard.CredentialStoreEscrowPendingField); err != nil {
		return fmt.Errorf("clear pending node passphrase: %w", err)
	}
	return nil
}

func (e authorityEscrow) save(_ context.Context, key, field string, secret []byte) error {
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("control-plane credential authority unavailable: %w", err)
	}
	identity, err := e.identity(key)
	if err != nil {
		return err
	}
	if err := authority.Put(identity, field, string(secret)); err != nil {
		return fmt.Errorf("escrow node passphrase: %w", err)
	}
	return nil
}

// NodeStoreGrants is the slice of the credential-grant service the ensurer
// needs; internal/credentialgrant.Service satisfies it.
type NodeStoreGrants interface {
	List(ctx context.Context, nodeID string) ([]internalgrant.Grant, error)
	Create(ctx context.Context, in internalgrant.CreateInput) (internalgrant.Grant, error)
}

// NewNodeStoreGrantEnsurer grants a node its own store passphrase as an
// ephemeral infrastructure credential and asks the grant handler to deliver
// it. Ephemeral is deliberate: infrastructure grants cannot be durable, and an
// ephemeral grant is re-pushed sealed on every reconnect — which is what lets
// the node agent reopen the store after a reboot. deliver is the grant
// handler's SyncNode; the value resolves from the same escrow address.
func NewNodeStoreGrantEnsurer(grants NodeStoreGrants, deliver func(ctx context.Context, nodeID string) error) internalonboard.NodeStoreGrantEnsurer {
	return storeGrantEnsurer{grants: grants, deliver: deliver}
}

type storeGrantEnsurer struct {
	grants  NodeStoreGrants
	deliver func(ctx context.Context, nodeID string) error
}

func (e storeGrantEnsurer) EnsureNodeStoreGrant(ctx context.Context, nodeID, logicalID, field string) error {
	if e.grants == nil {
		return fmt.Errorf("credential grants are not configured")
	}
	existing, err := e.grants.List(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("list node grants: %w", err)
	}
	held := false
	for _, grant := range existing {
		if grant.RevokedAt.IsZero() && grant.LogicalID == logicalID && grant.Field == field &&
			grant.Class == internalgrant.ClassInfrastructure && grant.Retention == internalgrant.RetentionEphemeral {
			held = true
			break
		}
	}
	if !held {
		if _, err := e.grants.Create(ctx, internalgrant.CreateInput{
			NodeID: nodeID, LogicalID: logicalID, Field: field,
			Class: internalgrant.ClassInfrastructure, Retention: internalgrant.RetentionEphemeral,
		}); err != nil {
			return fmt.Errorf("grant node its store passphrase: %w", err)
		}
	}
	if e.deliver != nil {
		if err := e.deliver(ctx, nodeID); err != nil {
			return fmt.Errorf("deliver node store grant: %w", err)
		}
	}
	return nil
}
