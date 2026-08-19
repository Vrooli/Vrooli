package config

import (
	"context"
	"fmt"
	"slices"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	credentialSourceMissing   = "missing"
	credentialSourceAuthority = "credential-authority"
	credentialSourceMixed     = "mixed"
	cloudflareCredentialID    = "vrooli/tunnel-manager"
	credentialKeyAccountID    = "cloudflare.account_id"
	credentialKeyTunnelID     = "cloudflare.tunnel_id"
	credentialKeyAPIToken     = "cloudflare.api_token"
)

// CredentialStoreOptions injects the canonical credential authority. The
// scenario does not select a backend and does not open a plaintext file.
type CredentialStoreOptions struct {
	Authority CredentialAuthority
}

type CredentialAuthority interface {
	Resolve(credentialauthority.Identity, string) (string, error)
	Put(credentialauthority.Identity, string, string) error
	Delete(credentialauthority.Identity, string) error
	Status(credentialauthority.Identity, string) credentialauthority.Status
	Provider() string
}

type cloudflareCredentialStore struct{ authority CredentialAuthority }

type credentialFieldSpec struct {
	Name      string
	Field     string
	UpdateVal func(CredentialUpdate) string
	SetConfig func(*CFConfig, string)
}

var credentialFieldSpecs = []credentialFieldSpec{
	{Name: cloudflareAccountIDField, Field: "cloudflare-account-id", UpdateVal: func(u CredentialUpdate) string { return u.AccountID }, SetConfig: func(c *CFConfig, v string) { c.AccountID = v }},
	{Name: cloudflareTunnelIDField, Field: "cloudflare-tunnel-id", UpdateVal: func(u CredentialUpdate) string { return u.TunnelID }, SetConfig: func(c *CFConfig, v string) { c.TunnelID = v }},
	{Name: cloudflareAPITokenField, Field: "cloudflare-api-token", UpdateVal: func(u CredentialUpdate) string { return u.APIToken }, SetConfig: func(c *CFConfig, v string) { c.APIToken = v }},
	{Name: cloudflareConnectorTokenField, Field: "cloudflare-connector-token", UpdateVal: func(CredentialUpdate) string { return "" }, SetConfig: func(c *CFConfig, v string) { c.ConnectorToken = v }},
}

func NewCloudflareCredentialStore(opts CredentialStoreOptions) (CredentialStore, error) {
	authority := opts.Authority
	if authority == nil {
		var err error
		authority, err = credentialauthority.Default()
		if err != nil {
			return nil, err
		}
	}
	return &cloudflareCredentialStore{authority: authority}, nil
}

func (s *cloudflareCredentialStore) Status(ctx context.Context) (CredentialStatus, error) {
	return s.status(ctx)
}

func (s *cloudflareCredentialStore) Resolve(ctx context.Context) (CFConfig, error) {
	status, cfg, err := s.resolve(ctx)
	if err != nil {
		return CFConfig{}, err
	}
	cfg.Source = status.Source
	cfg.TokenRef = status.Ref
	cfg.Missing = append([]string(nil), status.MissingFields...)
	return cfg, nil
}

func (s *cloudflareCredentialStore) Save(ctx context.Context, values CredentialUpdate) (CredentialStatus, error) {
	identity, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		return CredentialStatus{}, err
	}
	for _, spec := range credentialFieldSpecs {
		value := strings.TrimSpace(spec.UpdateVal(values))
		if value == "" {
			continue
		}
		if err := s.authority.Put(identity, spec.Field, value); err != nil {
			return CredentialStatus{}, fmt.Errorf("save %s: %w", spec.Name, err)
		}
	}
	return s.status(ctx)
}

func (s *cloudflareCredentialStore) Delete(ctx context.Context, keys []string) (CredentialStatus, error) {
	identity, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		return CredentialStatus{}, err
	}
	if len(keys) == 0 {
		keys = []string{"all"}
	}
	for _, key := range keys {
		for _, field := range deleteStoreKeys(strings.TrimSpace(key)) {
			if err := s.authority.Delete(identity, field); err != nil {
				return CredentialStatus{}, fmt.Errorf("delete %s: %w", field, err)
			}
		}
	}
	return s.status(ctx)
}

func (s *cloudflareCredentialStore) status(ctx context.Context) (CredentialStatus, error) {
	status, _, err := s.resolve(ctx)
	return status, err
}

func (s *cloudflareCredentialStore) resolve(context.Context) (CredentialStatus, CFConfig, error) {
	var cfg CFConfig
	fields := make([]CredentialFieldStatus, 0, len(credentialFieldSpecs))
	for _, spec := range credentialFieldSpecs {
		value, field, err := s.resolveField(spec)
		if err != nil {
			return CredentialStatus{}, CFConfig{}, err
		}
		spec.SetConfig(&cfg, value)
		fields = append(fields, field)
	}
	status := buildCredentialStatus(fields)
	return status, cfg, nil
}

func (s *cloudflareCredentialStore) resolveField(spec credentialFieldSpec) (string, CredentialFieldStatus, error) {
	identity, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		return "", CredentialFieldStatus{}, err
	}
	value, err := s.authority.Resolve(identity, spec.Field)
	if err != nil {
		status := s.authority.Status(identity, spec.Field)
		if status.ProviderState != "available" {
			return "", CredentialFieldStatus{}, fmt.Errorf("credential authority unavailable: %s", status.ProviderDetail)
		}
		return "", CredentialFieldStatus{Name: spec.Name, Source: credentialSourceMissing, Writable: true}, nil
	}
	value = strings.TrimSpace(value)
	return value, CredentialFieldStatus{
		Name: spec.Name, Present: value != "", Source: credentialSourceAuthority,
		Ref: cloudflareCredentialID + ":" + spec.Field, Writable: true,
	}, nil
}

func buildCredentialStatus(fields []CredentialFieldStatus) CredentialStatus {
	status := CredentialStatus{Fields: append([]CredentialFieldStatus(nil), fields...), Ready: true}
	sourceSet := map[string]struct{}{}
	for _, field := range fields {
		if !field.Present {
			status.Ready = false
			status.MissingFields = append(status.MissingFields, field.Name)
			continue
		}
		sourceSet[field.Source] = struct{}{}
		if field.Name == cloudflareAPITokenField {
			status.Ref = field.Ref
		}
	}
	switch len(sourceSet) {
	case 0:
		status.Source = credentialSourceMissing
	case 1:
		for source := range sourceSet {
			status.Source = source
		}
	default:
		status.Source = credentialSourceMixed
	}
	return status
}

func deleteStoreKeys(key string) []string {
	switch strings.ToLower(key) {
	case "", "all":
		return []string{"cloudflare-account-id", "cloudflare-tunnel-id", "cloudflare-api-token", "cloudflare-connector-token"}
	case "account_id", "cloudflare_account_id", cloudflareAccountIDField:
		return []string{"cloudflare-account-id"}
	case "tunnel_id", "cloudflare_tunnel_id", cloudflareTunnelIDField:
		return []string{"cloudflare-tunnel-id"}
	case "api_token", "cloudflare_api_token", cloudflareAPITokenField:
		return []string{"cloudflare-api-token"}
	case "connector_token", "cloudflare_connector_token", cloudflareConnectorTokenField:
		return []string{"cloudflare-connector-token"}
	default:
		if slices.Contains([]string{"cloudflare-account-id", "cloudflare-tunnel-id", "cloudflare-api-token", "cloudflare-connector-token"}, key) {
			return []string{key}
		}
		return nil
	}
}
