package desktophelper

import (
	"encoding/base64"
	"path/filepath"
)

// OwnerBootstrap is protected local configuration, not a public API payload.
// It requests provisioning without carrying any signing key or public-key pin.
type OwnerBootstrap struct {
	OperatorSubject  string `json:"operator_subject,omitempty"`
	DeviceID         string `json:"device_id"`
	HelperConfigPath string `json:"helper_config_path"`
	Helper           Config `json:"helper"`
}

func LoadOwnerBootstrap(path string) (OwnerBootstrap, error) {
	data, err := readPrivate(path, 64*1024)
	if err != nil {
		return OwnerBootstrap{}, err
	}
	var config OwnerBootstrap
	if decodeStrict(data, &config) != nil || config.DeviceID == "" || len(config.DeviceID) > 256 || !filepath.IsAbs(config.HelperConfigPath) || config.HelperConfigPath == path || config.Helper.PublicKey != "" {
		return OwnerBootstrap{}, ErrBootstrap
	}
	if filepath.Clean(config.HelperConfigPath) != config.HelperConfigPath || path == config.Helper.GrantStatusFile || path == config.Helper.AccessibilitySocket || config.HelperConfigPath == config.Helper.AccessibilitySocket || config.HelperConfigPath == config.Helper.GrantStatusFile || config.HelperConfigPath == config.Helper.XAuthorityFile {
		return OwnerBootstrap{}, ErrBootstrap
	}
	validation := config.Helper
	validation.PublicKey = base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err = validateConfig(validation); err != nil {
		return OwnerBootstrap{}, err
	}
	return config, nil
}
