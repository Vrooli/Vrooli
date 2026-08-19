package instrument

import (
	"context"
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

var _ CredentialResolver = (*CredentialClientResolver)(nil)

type UnavailableResolver struct{ Cause error }

func (r UnavailableResolver) Resolve(context.Context, string, string) (string, error) {
	if r.Cause == nil {
		r.Cause = errors.New("credential authority unavailable")
	}
	return "", r.Cause
}

var _ CredentialResolver = UnavailableResolver{}
