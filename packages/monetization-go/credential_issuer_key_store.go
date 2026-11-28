package monetization

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/api-core/consumeridentity"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

const issuerKeyIdentity = "vrooli/lpbs-issuer-trust"
const issuerKeyFieldName = "jwks-keysets"

// IssuerCredentialClient is the narrow credential-authority surface needed to
// keep issuer keys outside the editable application-data lease cache.
type IssuerCredentialClient interface {
	Resolve(context.Context, string, string) (string, error)
	Provision(context.Context, credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error)
	Status(context.Context, string, string) (credentialclient.CredentialStatus, error)
}

// CredentialIssuerKeyStore keeps public JWKS in the OS-backed credential
// authority. A modified lease JSON record cannot replace its own trust root.
type CredentialIssuerKeyStore struct {
	Client IssuerCredentialClient
}

func NewCredentialIssuerKeyStore(client IssuerCredentialClient) *CredentialIssuerKeyStore {
	return &CredentialIssuerKeyStore{Client: client}
}

func (s *CredentialIssuerKeyStore) Load(ctx context.Context, authority string) ([]byte, error) {
	if s == nil || s.Client == nil {
		return nil, entitlementclient.ErrLeaseTrustUnavailable
	}
	key, err := issuerKeyID(authority)
	if err != nil {
		return nil, err
	}
	status, err := s.Client.Status(ctx, issuerKeyIdentity, issuerKeyFieldName)
	if err != nil || status.ProviderState == "unavailable" || status.ProviderState == "absent" {
		return nil, fmt.Errorf("issuer key credential unavailable: %w", entitlementclient.ErrLeaseTrustUnavailable)
	}
	if !status.Configured {
		return nil, entitlementclient.ErrLeaseTrustUnavailable
	}
	value, err := s.Client.Resolve(ctx, issuerKeyIdentity, issuerKeyFieldName)
	if err != nil {
		return nil, fmt.Errorf("resolve issuer key credential: %w", err)
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &entries); err != nil {
		return nil, fmt.Errorf("decode issuer key sets: %w", err)
	}
	var raw []byte
	if err := json.Unmarshal(entries[key], &raw); err != nil || len(raw) == 0 {
		return nil, entitlementclient.ErrLeaseTrustUnavailable
	}
	return raw, nil
}

func (s *CredentialIssuerKeyStore) Save(ctx context.Context, authority string, jwks []byte) error {
	if s == nil || s.Client == nil {
		return entitlementclient.ErrLeaseTrustUnavailable
	}
	if _, err := consumeridentity.ParseJWKS(jwks); err != nil {
		return fmt.Errorf("refuse invalid issuer key set: %w", err)
	}
	key, err := issuerKeyID(authority)
	if err != nil {
		return err
	}
	status, err := s.Client.Status(ctx, issuerKeyIdentity, issuerKeyFieldName)
	if err != nil || status.ProviderState == "unavailable" || status.ProviderState == "absent" {
		return fmt.Errorf("issuer key credential unavailable: %w", entitlementclient.ErrLeaseTrustUnavailable)
	}
	entries := make(map[string]json.RawMessage)
	if status.Configured {
		current, resolveErr := s.Client.Resolve(ctx, issuerKeyIdentity, issuerKeyFieldName)
		if resolveErr != nil {
			return fmt.Errorf("resolve prior issuer key set: %w", resolveErr)
		}
		if err := json.Unmarshal([]byte(current), &entries); err != nil {
			return fmt.Errorf("decode prior issuer key sets: %w", err)
		}
	}
	var current []byte
	if err := json.Unmarshal(entries[key], &current); err == nil && bytes.Equal(current, jwks) {
		return nil
	}
	encodedKeySet, err := json.Marshal(jwks)
	if err != nil {
		return err
	}
	entries[key] = encodedKeySet
	encoded, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode issuer key sets: %w", err)
	}
	if _, err := s.Client.Provision(ctx, credentialclient.ProvisionRequest{Identity: issuerKeyIdentity, Field: issuerKeyFieldName, Value: string(encoded)}); err != nil {
		return fmt.Errorf("store issuer key set: %w", err)
	}
	return nil
}

func issuerKeyID(authority string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(authority))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("issuer authority must be an absolute URL")
	}
	origin := strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
	digest := sha256.Sum256([]byte(origin))
	return hex.EncodeToString(digest[:16]), nil
}
