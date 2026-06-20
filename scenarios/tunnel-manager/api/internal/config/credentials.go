package config

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/vrooli/api-core/secrets"
	repocontract "github.com/vrooli/repo-contract-go"
)

const scenarioID = "tunnel-manager"

const (
	credentialSourceMissing      = "missing"
	credentialSourceEnv          = "env:CLOUDFLARE_*" // #nosec G101 -- source label only, not a credential value.
	credentialSourceScenarioFile = "file:scenario"
	credentialSourceUserFile     = "file:user"
	credentialSourceMixed        = "mixed"

	credentialKeyAccountID = "cloudflare.account_id" // #nosec G101 -- secret-store key name only.
	credentialKeyTunnelID  = "cloudflare.tunnel_id"  // #nosec G101 -- secret-store key name only.
)

var credentialKeyAPIToken = credentialKey("api", "token") // #nosec G101 -- secret-store key name only.

type CredentialStoreOptions struct {
	EnvLookup   func(string) string
	HomeDir     string
	UserHomeDir func() (string, error)
}

type cloudflareCredentialStore struct {
	envLookup     func(string) string
	scenarioStore *secrets.Store
	userStore     *secrets.Store
}

type credentialFieldSpec struct {
	Name      string
	StoreKey  string
	UpdateVal func(CredentialUpdate) string
	SetConfig func(*CFConfig, string)
}

var credentialFieldSpecs = []credentialFieldSpec{
	{
		Name:      cloudflareAccountIDField,
		StoreKey:  credentialKeyAccountID,
		UpdateVal: func(u CredentialUpdate) string { return u.AccountID },
		SetConfig: func(c *CFConfig, v string) { c.AccountID = v },
	},
	{
		Name:      cloudflareTunnelIDField,
		StoreKey:  credentialKeyTunnelID,
		UpdateVal: func(u CredentialUpdate) string { return u.TunnelID },
		SetConfig: func(c *CFConfig, v string) { c.TunnelID = v },
	},
	{
		Name:      cloudflareAPITokenField,
		StoreKey:  credentialKeyAPIToken,
		UpdateVal: func(u CredentialUpdate) string { return u.APIToken },
		SetConfig: func(c *CFConfig, v string) { c.APIToken = v },
	},
}

func NewCloudflareCredentialStore(opts CredentialStoreOptions) (CredentialStore, error) {
	envLookup := opts.EnvLookup
	if envLookup == nil {
		envLookup = os.Getenv
	}
	homeDir, err := resolveCredentialHome(opts)
	if err != nil {
		return nil, err
	}
	scenarioPath, err := repocontract.UserScenarioPlaintextSecretsPath(homeDir, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario secrets path: %w", err)
	}
	scenarioStore, err := secrets.NewFileStore(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("open scenario credential store: %w", err)
	}
	userStore, err := secrets.NewUserStore(secrets.Config{
		HomeDir:   homeDir,
		EnvLookup: func(string) string { return "" },
	})
	if err != nil {
		return nil, fmt.Errorf("open user credential store: %w", err)
	}
	return &cloudflareCredentialStore{
		envLookup:     envLookup,
		scenarioStore: scenarioStore,
		userStore:     userStore,
	}, nil
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
	for _, spec := range credentialFieldSpecs {
		value := strings.TrimSpace(spec.UpdateVal(values))
		if value == "" {
			continue
		}
		if err := s.scenarioStore.SaveKey(spec.StoreKey, value); err != nil {
			return CredentialStatus{}, fmt.Errorf("save %s: %w", spec.Name, err)
		}
	}
	return s.status(ctx)
}

func (s *cloudflareCredentialStore) Delete(ctx context.Context, keys []string) (CredentialStatus, error) {
	if len(keys) == 0 {
		keys = []string{"all"}
	}
	for _, key := range keys {
		for _, storeKey := range deleteStoreKeys(strings.TrimSpace(key)) {
			if _, err := s.scenarioStore.DeleteKey(storeKey); err != nil {
				return CredentialStatus{}, fmt.Errorf("delete %s: %w", storeKey, err)
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
	if value := strings.TrimSpace(s.envLookup(spec.Name)); value != "" {
		return value, CredentialFieldStatus{
			Name:     spec.Name,
			Present:  true,
			Source:   credentialSourceEnv,
			Ref:      "env:" + spec.Name,
			Writable: false,
		}, nil
	}
	value, source, ref, err := resolveStoreField(s.scenarioStore, credentialSourceScenarioFile, spec.StoreKey)
	if err != nil {
		return "", CredentialFieldStatus{}, err
	}
	if value != "" {
		return value, CredentialFieldStatus{Name: spec.Name, Present: true, Source: source, Ref: ref, Writable: true}, nil
	}
	value, source, ref, err = resolveStoreField(s.userStore, credentialSourceUserFile, spec.StoreKey)
	if err != nil {
		return "", CredentialFieldStatus{}, err
	}
	if value != "" {
		return value, CredentialFieldStatus{Name: spec.Name, Present: true, Source: source, Ref: ref, Writable: true}, nil
	}
	return "", CredentialFieldStatus{Name: spec.Name, Source: credentialSourceMissing, Writable: true}, nil
}

func resolveStoreField(store *secrets.Store, source, key string) (string, string, string, error) {
	resolved, err := store.Resolve(key)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve %s: %w", key, err)
	}
	if strings.TrimSpace(resolved.Value) == "" {
		return "", "", "", nil
	}
	return strings.TrimSpace(resolved.Value), source, source + ":" + key, nil
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
		return []string{credentialKeyAccountID, credentialKeyTunnelID, credentialKeyAPIToken}
	case "account_id", "cloudflare_account_id", cloudflareAccountIDField:
		return []string{credentialKeyAccountID}
	case "tunnel_id", "cloudflare_tunnel_id", cloudflareTunnelIDField:
		return []string{credentialKeyTunnelID}
	case "api_token", "cloudflare_api_token", cloudflareAPITokenField:
		return []string{credentialKeyAPIToken}
	default:
		if slices.Contains([]string{credentialKeyAccountID, credentialKeyTunnelID, credentialKeyAPIToken}, key) {
			return []string{key}
		}
		return nil
	}
}

func resolveCredentialHome(opts CredentialStoreOptions) (string, error) {
	if strings.TrimSpace(opts.HomeDir) != "" {
		return strings.TrimSpace(opts.HomeDir), nil
	}
	userHomeDir := opts.UserHomeDir
	if userHomeDir == nil {
		userHomeDir = os.UserHomeDir
	}
	homeDir, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return homeDir, nil
}

func credentialKey(parts ...string) string {
	return "cloudflare." + strings.Join(parts, "_")
}
