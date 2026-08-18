package securestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CopyConfig is non-secret configuration for the encrypted-store escrow. The
// sink is deliberately a location, never a credential or a passphrase.
type CopyConfig struct {
	Enabled                   bool          `json:"enabled"`
	Sink                      string        `json:"sink"`
	Interval                  time.Duration `json:"interval"`
	ObjectStoreCredentialID   string        `json:"object_store_credential_id,omitempty"`
	ObjectStoreRegion         string        `json:"object_store_region,omitempty"`
	ObjectStoreEndpoint       string        `json:"object_store_endpoint,omitempty"`
	ObjectStoreAccessKeyField string        `json:"object_store_access_key_field,omitempty"`
	ObjectStoreSecretKeyField string        `json:"object_store_secret_key_field,omitempty"`
	ObjectStoreSessionField   string        `json:"object_store_session_field,omitempty"`
}

const DefaultCopyInterval = 15 * time.Minute

func (c CopyConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Sink) == "" {
		return errors.New("credential-store copy sink is required when enabled")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("credential-store copy interval must be positive")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Sink)), "s3://") {
		if strings.TrimSpace(c.ObjectStoreCredentialID) == "" {
			return errors.New("object-store credential identity is required for an s3 sink")
		}
		if strings.TrimSpace(c.ObjectStoreRegion) == "" {
			return errors.New("object-store region is required for an s3 sink")
		}
	}
	return nil
}

func ReadCopyConfig(path string) (CopyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CopyConfig{}, nil
		}
		return CopyConfig{}, fmt.Errorf("read credential-store copy configuration: %w", err)
	}
	var config CopyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return CopyConfig{}, fmt.Errorf("decode credential-store copy configuration: %w", err)
	}
	if err := config.Validate(); err != nil {
		return CopyConfig{}, err
	}
	return config, nil
}

func WriteCopyConfig(path string, config CopyConfig) error {
	if config.Interval <= 0 {
		config.Interval = DefaultCopyInterval
	}
	if err := config.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential-store copy configuration: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), sealedDirPerm); err != nil {
		return fmt.Errorf("create credential-store copy configuration directory: %w", err)
	}
	if err := atomicCopy(path, data); err != nil {
		return fmt.Errorf("write credential-store copy configuration: %w", err)
	}
	return nil
}
