package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	resourceenv "resource-cloudflare-ai-gateway/cli/internal/env"
)

var ErrNamedConfigNotFound = errors.New("named gateway config not found")

// GatewayConfig tracks repo-owned local state about the remote Cloudflare AI
// Gateway instance and any operator-selected defaults.
type GatewayConfig struct {
	GatewayID string   `json:"gateway_id,omitempty"`
	Active    bool     `json:"active"`
	Providers []string `json:"providers,omitempty"`
	CacheTTL  int      `json:"cache_ttl,omitempty"`
	RateLimit int      `json:"rate_limit,omitempty"`
}

// GatewayState captures lightweight operational status derived from previous
// gateway operations.
type GatewayState struct {
	Status    string     `json:"status"`
	LastCheck *time.Time `json:"last_check,omitempty"`
}

// RateLimitingConfig describes the default Cloudflare rate-limiting policy.
type RateLimitingConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute"`
}

// CachingConfig describes the default Cloudflare caching policy.
type CachingConfig struct {
	Enabled bool `json:"enabled"`
	TTL     int  `json:"ttl"`
}

// GatewayCreateRequest mirrors the create/update request body the old shell
// implementation emitted for Cloudflare AI Gateway.
type GatewayCreateRequest struct {
	Name         string             `json:"name"`
	Slug         string             `json:"slug"`
	RateLimiting RateLimitingConfig `json:"rate_limiting"`
	Caching      CachingConfig      `json:"caching"`
}

// Store owns local config/state/content files for the resource.
type Store struct {
	Runtime resourceenv.Runtime
}

// NewStore builds a Store using the provided runtime paths.
func NewStore(runtime resourceenv.Runtime) Store {
	return Store{Runtime: runtime}
}

// EnsureInitialized creates the resource data directories and default config/state files.
func (s Store) EnsureInitialized() error {
	if err := s.Runtime.EnsureDirectories(); err != nil {
		return err
	}
	if err := ensureJSONFile(s.Runtime.ConfigFile, GatewayConfig{}); err != nil {
		return err
	}
	return ensureJSONFile(s.Runtime.StateFile, GatewayState{Status: "inactive"})
}

// LoadGatewayConfig reads the local gateway config file.
func (s Store) LoadGatewayConfig() (GatewayConfig, error) {
	var cfg GatewayConfig
	if err := readJSONFile(s.Runtime.ConfigFile, &cfg); err != nil {
		return GatewayConfig{}, err
	}
	cfg.Providers = uniqueSorted(cfg.Providers)
	return cfg, nil
}

// SaveGatewayConfig writes the local gateway config file atomically.
func (s Store) SaveGatewayConfig(cfg GatewayConfig) error {
	cfg.Providers = uniqueSorted(cfg.Providers)
	return writeJSONFile(s.Runtime.ConfigFile, cfg)
}

// LoadGatewayState reads the local state file.
func (s Store) LoadGatewayState() (GatewayState, error) {
	var state GatewayState
	if err := readJSONFile(s.Runtime.StateFile, &state); err != nil {
		return GatewayState{}, err
	}
	if strings.TrimSpace(state.Status) == "" {
		state.Status = "inactive"
	}
	return state, nil
}

// SaveGatewayState writes the local state file atomically.
func (s Store) SaveGatewayState(state GatewayState) error {
	if strings.TrimSpace(state.Status) == "" {
		state.Status = "inactive"
	}
	return writeJSONFile(s.Runtime.StateFile, state)
}

// MarkStatus updates the persisted operational status and refresh timestamp.
func (s Store) MarkStatus(status string, checkedAt time.Time) error {
	return s.SaveGatewayState(GatewayState{
		Status:    strings.TrimSpace(status),
		LastCheck: &checkedAt,
	})
}

// DefaultGatewayCreateRequest returns the baseline Cloudflare AI Gateway
// payload used by this resource.
func DefaultGatewayCreateRequest() GatewayCreateRequest {
	return GatewayCreateRequest{
		Name: "vrooli-ai-gateway",
		Slug: "vrooli",
		RateLimiting: RateLimitingConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
		},
		Caching: CachingConfig{
			Enabled: true,
			TTL:     3600,
		},
	}
}

// GatewayAPIBaseURL derives the Cloudflare AI Gateway API root for an account.
func GatewayAPIBaseURL(endpointRoot, accountID string) string {
	root := strings.TrimRight(strings.TrimSpace(endpointRoot), "/")
	accountID = strings.Trim(strings.TrimSpace(accountID), "/")
	if root == "" || accountID == "" {
		return ""
	}
	return root + "/" + accountID + "/ai-gateway"
}

// NamedConfigPath resolves a repo-external config file path for a named gateway
// configuration payload.
func (s Store) NamedConfigPath(name string) (string, error) {
	sanitized := sanitizeName(name)
	if sanitized == "" {
		return "", fmt.Errorf("config name is required")
	}
	return filepath.Join(s.Runtime.ConfigsDir, sanitized+".json"), nil
}

// SaveNamedConfig persists a named config payload.
func (s Store) SaveNamedConfig(name string, payload json.RawMessage) error {
	path, err := s.NamedConfigPath(name)
	if err != nil {
		return err
	}
	return writeRawJSONFile(path, payload)
}

// LoadNamedConfig retrieves a named config payload.
func (s Store) LoadNamedConfig(name string) (json.RawMessage, error) {
	path, err := s.NamedConfigPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNamedConfigNotFound
		}
		return nil, err
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("%s contains invalid JSON", path)
	}
	return json.RawMessage(data), nil
}

// DeleteNamedConfig removes a named config payload.
func (s Store) DeleteNamedConfig(name string) error {
	path, err := s.NamedConfigPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNamedConfigNotFound
		}
		return err
	}
	return nil
}

// ListNamedConfigs enumerates available named config payloads.
func (s Store) ListNamedConfigs() ([]string, error) {
	entries, err := os.ReadDir(s.Runtime.ConfigsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		names = append(names, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	sort.Strings(names)
	return names, nil
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return strings.TrimSpace(name)
}

func ensureJSONFile(path string, value any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeJSONFile(path, value)
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomically(path, data)
}

func writeRawJSONFile(path string, payload json.RawMessage) error {
	if !json.Valid(payload) {
		return fmt.Errorf("invalid JSON payload")
	}
	data := append([]byte(nil), payload...)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return writeFileAtomically(path, data)
}

func writeFileAtomically(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
