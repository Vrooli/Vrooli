// Package signing provides CLI commands for managing code signing configurations.
package signing

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
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

func signingShowReport(config *SigningConfig) cliapp.ListReport {
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Signing enabled: %v", config.Enabled),
		},
		ResultsHeading: "Platform Configuration",
		RetrievalHints: []string{
			"deployment-manager signing validate <profile>",
			"deployment-manager signing prerequisites",
		},
	}
	if config.Windows != nil {
		report.Results = append(report.Results,
			fmt.Sprintf("Windows certificate-source=%s", config.Windows.CertificateSource),
			fmt.Sprintf("Windows certificate-file=%s", emptySigningValue(config.Windows.CertificateFile)),
			fmt.Sprintf("Windows password-env=%s", emptySigningValue(config.Windows.CertificatePasswordEnv)),
			fmt.Sprintf("Windows timestamp-server=%s", emptySigningValue(config.Windows.TimestampServer)),
			fmt.Sprintf("Windows algorithm=%s", emptySigningValue(config.Windows.SignAlgorithm)),
		)
	}

	if config.MacOS != nil {
		report.Results = append(report.Results,
			fmt.Sprintf("macOS identity=%s", emptySigningValue(config.MacOS.Identity)),
			fmt.Sprintf("macOS team-id=%s", emptySigningValue(config.MacOS.TeamID)),
			fmt.Sprintf("macOS hardened-runtime=%v", config.MacOS.HardenedRuntime),
			fmt.Sprintf("macOS notarize=%v", config.MacOS.Notarize),
		)
	}

	if config.Linux != nil {
		report.Results = append(report.Results,
			fmt.Sprintf("Linux gpg-key-id=%s", emptySigningValue(config.Linux.GPGKeyID)),
			fmt.Sprintf("Linux passphrase-env=%s", emptySigningValue(config.Linux.GPGPassphraseEnv)),
		)
	}

	if config.Windows == nil && config.MacOS == nil && config.Linux == nil {
		report.Results = append(report.Results, "No platform signing configurations")
	}
	return report
}

func emptySigningValue(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
