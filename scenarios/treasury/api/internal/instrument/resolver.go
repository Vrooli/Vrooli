package instrument

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type CredentialClientResolver struct{ client credentialclient.Client }

func NewCredentialClientResolver(client credentialclient.Client) (*CredentialClientResolver, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: credential client is required", ErrInvalid)
	}
	return &CredentialClientResolver{client: client}, nil
}

func (r *CredentialClientResolver) Resolve(ctx context.Context, reference, field string) (string, error) {
	identity, err := credentialauthority.ParseIdentity(reference)
	if err != nil {
		return "", fmt.Errorf("invalid credential reference")
	}
	return r.client.Resolve(ctx, string(identity), field)
}

func (r *CredentialClientResolver) Store(ctx context.Context, reference, field, value string) error {
	identity, err := credentialauthority.ParseIdentity(reference)
	if err != nil {
		return fmt.Errorf("invalid credential reference")
	}
	if value == "" {
		return fmt.Errorf("%w: credential value is empty", ErrInvalid)
	}
	_, err = r.client.Provision(ctx, credentialclient.ProvisionRequest{Identity: string(identity), Field: field, Value: value})
	return err
}

func settlementRecoveryIdentity(id string) string {
	digest := sha256.Sum256([]byte(id))
	return "vrooli/treasury/x402-recovery/" + hex.EncodeToString(digest[:])
}

// StoreSettlementRecoveryInput keeps replay authorization inside Credential
// Authority and returns only its logical address for the settlement record.
func (r *CredentialClientResolver) StoreSettlementRecoveryInput(ctx context.Context, id, payload string) (string, error) {
	identity := settlementRecoveryIdentity(id)
	_, err := r.client.Provision(ctx, credentialclient.ProvisionRequest{Identity: identity, Field: "signed-payment", Value: payload})
	if err != nil {
		return "", err
	}
	return identity, nil
}

func (r *CredentialClientResolver) ResolveSettlementRecoveryInput(ctx context.Context, reference string) (string, error) {
	identity, err := credentialauthority.ParseIdentity(reference)
	if err != nil {
		return "", fmt.Errorf("invalid recovery credential reference")
	}
	return r.client.Resolve(ctx, string(identity), "signed-payment")
}

func (r *CredentialClientResolver) DeleteSettlementRecoveryInput(ctx context.Context, reference string) error {
	identity, err := credentialauthority.ParseIdentity(reference)
	if err != nil {
		return fmt.Errorf("invalid recovery credential reference")
	}
	return r.client.Delete(ctx, string(identity), "signed-payment")
}

func (r *CredentialClientResolver) Delete(ctx context.Context, reference, field string) error {
	identity, err := credentialauthority.ParseIdentity(reference)
	if err != nil {
		return fmt.Errorf("invalid credential reference")
	}
	return r.client.Delete(ctx, string(identity), field)
}

var (
	_ CredentialResolver = (*CredentialClientResolver)(nil)
	_ CredentialWriter   = (*CredentialClientResolver)(nil)
)

type UnavailableResolver struct{ Cause error }

func (r UnavailableResolver) Resolve(context.Context, string, string) (string, error) {
	if r.Cause == nil {
		r.Cause = errors.New("credential authority unavailable")
	}
	return "", r.Cause
}

var _ CredentialResolver = UnavailableResolver{}
