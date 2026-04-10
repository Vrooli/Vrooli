// Package signing provides CLI commands for managing code signing configurations.
package signing

import (
	"fmt"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

// SigningConfig matches the API response structure.
type SigningConfig struct {
	Enabled bool                  `json:"enabled"`
	Windows *WindowsSigningConfig `json:"windows,omitempty"`
	MacOS   *MacOSSigningConfig   `json:"macos,omitempty"`
	Linux   *LinuxSigningConfig   `json:"linux,omitempty"`
}

// WindowsSigningConfig contains Windows Authenticode settings.
type WindowsSigningConfig struct {
	CertificateSource      string `json:"certificate_source"`
	CertificateFile        string `json:"certificate_file,omitempty"`
	CertificatePasswordEnv string `json:"certificate_password_env,omitempty"`
	CertificateThumbprint  string `json:"certificate_thumbprint,omitempty"`
	TimestampServer        string `json:"timestamp_server,omitempty"`
	SignAlgorithm          string `json:"sign_algorithm,omitempty"`
	DualSign               bool   `json:"dual_sign,omitempty"`
}

// MacOSSigningConfig contains Apple code signing settings.
type MacOSSigningConfig struct {
	Identity            string `json:"identity"`
	TeamID              string `json:"team_id"`
	HardenedRuntime     bool   `json:"hardened_runtime"`
	Notarize            bool   `json:"notarize"`
	EntitlementsFile    string `json:"entitlements_file,omitempty"`
	ProvisioningProfile string `json:"provisioning_profile,omitempty"`
	AppleIDEnv          string `json:"apple_id_env,omitempty"`
	AppleIDPasswordEnv  string `json:"apple_id_password_env,omitempty"`
	AppleAPIKeyID       string `json:"apple_api_key_id,omitempty"`
	AppleAPIKeyFile     string `json:"apple_api_key_file,omitempty"`
	AppleAPIIssuerID    string `json:"apple_api_issuer_id,omitempty"`
}

// LinuxSigningConfig contains Linux GPG signing settings.
type LinuxSigningConfig struct {
	GPGKeyID         string `json:"gpg_key_id,omitempty"`
	GPGPassphraseEnv string `json:"gpg_passphrase_env,omitempty"`
	GPGHomedir       string `json:"gpg_homedir,omitempty"`
}

// Commands provides signing CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new signing Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) printSigningTable(config *SigningConfig) {
	fmt.Printf("Signing Enabled: %v\n\n", config.Enabled)

	if config.Windows != nil {
		fmt.Println("Windows Configuration:")
		rows := [][]string{
			{"Certificate Source", config.Windows.CertificateSource},
			{"Certificate File", config.Windows.CertificateFile},
			{"Password Env", config.Windows.CertificatePasswordEnv},
			{"Timestamp Server", config.Windows.TimestampServer},
			{"Algorithm", config.Windows.SignAlgorithm},
		}
		cmdutil.PrintTable([]string{"Setting", "Value"}, rows)
		fmt.Println()
	}

	if config.MacOS != nil {
		fmt.Println("macOS Configuration:")
		rows := [][]string{
			{"Identity", config.MacOS.Identity},
			{"Team ID", config.MacOS.TeamID},
			{"Hardened Runtime", fmt.Sprintf("%v", config.MacOS.HardenedRuntime)},
			{"Notarize", fmt.Sprintf("%v", config.MacOS.Notarize)},
		}
		cmdutil.PrintTable([]string{"Setting", "Value"}, rows)
		fmt.Println()
	}

	if config.Linux != nil {
		fmt.Println("Linux Configuration:")
		rows := [][]string{
			{"GPG Key ID", config.Linux.GPGKeyID},
			{"Passphrase Env", config.Linux.GPGPassphraseEnv},
		}
		cmdutil.PrintTable([]string{"Setting", "Value"}, rows)
	}

	if config.Windows == nil && config.MacOS == nil && config.Linux == nil {
		fmt.Println("No platform signing configurations")
	}
}
