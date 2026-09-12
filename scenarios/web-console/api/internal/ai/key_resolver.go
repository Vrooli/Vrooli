package ai

import (
	"context"
	"os"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type KeySource string

const (
	KeySourceCredentialAuthority KeySource = "credential_authority"
	KeySourceEnvironment         KeySource = "environment"
	KeySourceNone                KeySource = "none"
)

type KeyResolver interface {
	Resolve(context.Context) (string, KeySource, error)
}

type CredentialKeyResolver struct {
	Client credentialclient.Client
}

func NewCredentialKeyResolver(client credentialclient.Client) *CredentialKeyResolver {
	return &CredentialKeyResolver{Client: client}
}

func (r *CredentialKeyResolver) Resolve(ctx context.Context) (string, KeySource, error) {
	if r != nil && r.Client != nil {
		key, err := r.Client.Resolve(ctx, "vrooli/openrouter", "api-key")
		if err == nil && strings.TrimSpace(key) != "" {
			return key, KeySourceCredentialAuthority, nil
		}
	}
	if key := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); key != "" {
		return key, KeySourceEnvironment, nil
	}
	return "", KeySourceNone, nil
}

type StaticKeyResolver struct {
	Key    string
	Source KeySource
}

func (r StaticKeyResolver) Resolve(context.Context) (string, KeySource, error) {
	if strings.TrimSpace(r.Key) == "" {
		return "", KeySourceNone, nil
	}
	return r.Key, r.Source, nil
}
