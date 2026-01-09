// Package signing provides CLI commands for managing code signing configurations.
package signing

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

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

// Run dispatches to the appropriate signing subcommand.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return c.Help(nil)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "show":
		return c.Show(rest)
	case "set":
		return c.Set(rest)
	case "remove":
		return c.Remove(rest)
	case "validate":
		return c.Validate(rest)
	case "prerequisites":
		return c.Prerequisites(rest)
	case "discover":
		return c.Discover(rest)
	case "help", "-h", "--help":
		return c.Help(rest)
	default:
		return fmt.Errorf("unknown signing subcommand: %s\nRun 'deployment-manager signing help' for usage", sub)
	}
}

// Help displays signing command help.
func (c *Commands) Help(args []string) error {
	help := `Code Signing Configuration Commands

Usage:
  deployment-manager signing <command> [options]

Commands:
  show <profile>              Show signing configuration for a profile
  set <profile> --platform    Configure signing for a platform
  remove <profile>            Remove signing configuration
  validate <profile>          Validate signing prerequisites
  prerequisites               Check available signing tools
  discover --platform         Discover available certificates/identities
  help                        Show this help message

Examples:
  # Show current signing config
  deployment-manager signing show my-profile

  # Configure Windows signing
  deployment-manager signing set my-profile --platform windows \
    --cert ./cert.pfx \
    --password-env WIN_CERT_PASSWORD \
    --timestamp http://timestamp.digicert.com

  # Configure macOS signing with API key
  deployment-manager signing set my-profile --platform macos \
    --identity "Developer ID Application: My Company (TEAMID)" \
    --team-id TEAMID \
    --hardened-runtime \
    --notarize \
    --api-key-id KEYID \
    --api-key-file ./AuthKey.p8 \
    --api-issuer ISSUER-UUID

  # Validate signing prerequisites
  deployment-manager signing validate my-profile

  # Check available signing tools
  deployment-manager signing prerequisites

  # Discover available certificates (platform-specific)
  deployment-manager signing discover --platform windows
  deployment-manager signing discover --platform macos
  deployment-manager signing discover --platform linux
`
	fmt.Println(help)
	return nil
}

// Show displays the signing configuration for a profile.
func (c *Commands) Show(args []string) error {
	fs := flag.NewFlagSet("signing show", flag.ContinueOnError)
	platform := fs.String("platform", "", "show only specific platform (windows|macos|linux)")
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	if len(remaining) < 1 {
		return errors.New("usage: signing show <profile> [--platform <platform>]")
	}
	profileID := remaining[0]

	body, err := c.api.Get(fmt.Sprintf("/api/v1/profiles/%s/signing", profileID), nil)
	if err != nil {
		return fmt.Errorf("failed to get signing config: %w", err)
	}

	var config SigningConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("parse signing config: %w", err)
	}

	// Filter by platform if specified
	if *platform != "" {
		switch strings.ToLower(*platform) {
		case "windows":
			if config.Windows == nil {
				fmt.Println("No Windows signing configuration")
				return nil
			}
			body, _ = json.MarshalIndent(config.Windows, "", "  ")
		case "macos":
			if config.MacOS == nil {
				fmt.Println("No macOS signing configuration")
				return nil
			}
			body, _ = json.MarshalIndent(config.MacOS, "", "  ")
		case "linux":
			if config.Linux == nil {
				fmt.Println("No Linux signing configuration")
				return nil
			}
			body, _ = json.MarshalIndent(config.Linux, "", "  ")
		default:
			return fmt.Errorf("invalid platform: %s (valid: windows, macos, linux)", *platform)
		}
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "table" {
		c.printSigningTable(&config)
		return nil
	}

	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

// printSigningTable prints signing config in table format.
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

// Set configures signing for a profile.
func (c *Commands) Set(args []string) error {
	fs := flag.NewFlagSet("signing set", flag.ContinueOnError)

	// Platform selection
	platform := fs.String("platform", "", "platform to configure (windows|macos|linux) [required]")

	// Windows options
	cert := fs.String("cert", "", "path to certificate file (.pfx/.p12)")
	passwordEnv := fs.String("password-env", "", "env var containing certificate password")
	thumbprint := fs.String("thumbprint", "", "certificate thumbprint (for store)")
	timestamp := fs.String("timestamp", "http://timestamp.digicert.com", "timestamp server URL")
	algorithm := fs.String("algorithm", "sha256", "signing algorithm (sha256|sha384|sha512)")
	dualSign := fs.Bool("dual-sign", false, "enable SHA1+SHA256 dual signing")

	// macOS options
	identity := fs.String("identity", "", "signing identity (e.g., 'Developer ID Application: Name (TEAMID)')")
	teamID := fs.String("team-id", "", "Apple Developer Team ID")
	hardenedRuntime := fs.Bool("hardened-runtime", true, "enable hardened runtime")
	notarize := fs.Bool("notarize", true, "enable Apple notarization")
	entitlements := fs.String("entitlements", "", "path to entitlements.plist")
	appleIDEnv := fs.String("apple-id-env", "", "env var containing Apple ID email")
	applePasswordEnv := fs.String("apple-password-env", "", "env var containing app-specific password")
	apiKeyID := fs.String("api-key-id", "", "Apple API Key ID")
	apiKeyFile := fs.String("api-key-file", "", "path to .p8 API key file")
	apiIssuer := fs.String("api-issuer", "", "Apple API Issuer ID")

	// Linux options
	gpgKeyID := fs.String("gpg-key", "", "GPG key ID or fingerprint")
	gpgPassphraseEnv := fs.String("gpg-passphrase-env", "", "env var containing GPG passphrase")
	gpgHomedir := fs.String("gpg-homedir", "", "GPG home directory override")

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	if len(remaining) < 1 {
		return errors.New("usage: signing set <profile> --platform <platform> [options]")
	}
	profileID := remaining[0]

	if *platform == "" {
		return errors.New("--platform is required (windows|macos|linux)")
	}

	var payload interface{}

	switch strings.ToLower(*platform) {
	case "windows":
		source := "file"
		if *thumbprint != "" {
			source = "store"
		}
		payload = WindowsSigningConfig{
			CertificateSource:      source,
			CertificateFile:        *cert,
			CertificatePasswordEnv: *passwordEnv,
			CertificateThumbprint:  *thumbprint,
			TimestampServer:        *timestamp,
			SignAlgorithm:          *algorithm,
			DualSign:               *dualSign,
		}
	case "macos":
		if *identity == "" || *teamID == "" {
			return errors.New("macOS signing requires --identity and --team-id")
		}
		payload = MacOSSigningConfig{
			Identity:            *identity,
			TeamID:              *teamID,
			HardenedRuntime:     *hardenedRuntime,
			Notarize:            *notarize,
			EntitlementsFile:    *entitlements,
			AppleIDEnv:          *appleIDEnv,
			AppleIDPasswordEnv:  *applePasswordEnv,
			AppleAPIKeyID:       *apiKeyID,
			AppleAPIKeyFile:     *apiKeyFile,
			AppleAPIIssuerID:    *apiIssuer,
		}
	case "linux":
		payload = LinuxSigningConfig{
			GPGKeyID:         *gpgKeyID,
			GPGPassphraseEnv: *gpgPassphraseEnv,
			GPGHomedir:       *gpgHomedir,
		}
	default:
		return fmt.Errorf("invalid platform: %s (valid: windows, macos, linux)", *platform)
	}

	body, err := c.api.Request("PATCH", fmt.Sprintf("/api/v1/profiles/%s/signing/%s", profileID, strings.ToLower(*platform)), nil, payload)
	if err != nil {
		return fmt.Errorf("failed to set signing config: %w", err)
	}

	cliutil.PrintJSON(body)
	return nil
}

// Remove removes signing configuration for a profile.
func (c *Commands) Remove(args []string) error {
	fs := flag.NewFlagSet("signing remove", flag.ContinueOnError)
	platform := fs.String("platform", "", "remove only specific platform (windows|macos|linux)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	if len(remaining) < 1 {
		return errors.New("usage: signing remove <profile> [--platform <platform>]")
	}
	profileID := remaining[0]

	var endpoint string
	if *platform != "" {
		endpoint = fmt.Sprintf("/api/v1/profiles/%s/signing/%s", profileID, strings.ToLower(*platform))
	} else {
		endpoint = fmt.Sprintf("/api/v1/profiles/%s/signing", profileID)
	}

	body, err := c.api.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to remove signing config: %w", err)
	}

	cliutil.PrintJSON(body)
	return nil
}

// Validate validates signing prerequisites for a profile.
func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("signing validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	if len(remaining) < 1 {
		return errors.New("usage: signing validate <profile>")
	}
	profileID := remaining[0]

	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/profiles/%s/signing/validate", profileID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to validate signing: %w", err)
	}

	var result struct {
		Valid    bool                            `json:"valid"`
		Message  string                          `json:"message,omitempty"`
		Errors   []map[string]string             `json:"errors,omitempty"`
		Warnings []map[string]string             `json:"warnings,omitempty"`
		Platforms map[string]map[string]interface{} `json:"platforms,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		cmdutil.PrintByFormat(cmdutil.ResolveFormat(*format), body)
		return nil
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) != "table" {
		cmdutil.PrintByFormat(formatVal, body)
		return nil
	}

	// Table format
	if result.Valid {
		fmt.Println("✓ Signing configuration is valid")
		if result.Message != "" {
			fmt.Println("  " + result.Message)
		}
	} else {
		fmt.Println("✗ Signing validation failed")
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  • [%s] %s: %s\n", e["platform"], e["code"], e["message"])
			if rem := e["remediation"]; rem != "" {
				fmt.Printf("    Remediation: %s\n", rem)
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  • [%s] %s: %s\n", w["platform"], w["code"], w["message"])
		}
	}

	return nil
}

// Prerequisites checks available signing tools.
func (c *Commands) Prerequisites(args []string) error {
	fs := flag.NewFlagSet("signing prerequisites", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	_, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	body, err := c.api.Get("/api/v1/signing/prerequisites", nil)
	if err != nil {
		return fmt.Errorf("failed to check prerequisites: %w", err)
	}

	var result struct {
		Tools []struct {
			Platform    string `json:"platform"`
			Tool        string `json:"tool"`
			Installed   bool   `json:"installed"`
			Path        string `json:"path,omitempty"`
			Version     string `json:"version,omitempty"`
			Error       string `json:"error,omitempty"`
			Remediation string `json:"remediation,omitempty"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		cmdutil.PrintByFormat(cmdutil.ResolveFormat(*format), body)
		return nil
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) != "table" {
		cmdutil.PrintByFormat(formatVal, body)
		return nil
	}

	// Table format
	fmt.Println("Signing Tool Prerequisites:\n")

	rows := [][]string{}
	for _, t := range result.Tools {
		status := "✓"
		if !t.Installed {
			status = "✗"
		}
		version := t.Version
		if version == "" {
			version = "-"
		}
		path := t.Path
		if path == "" {
			if t.Error != "" {
				path = t.Error
			} else {
				path = "-"
			}
		}
		rows = append(rows, []string{status, t.Platform, t.Tool, version, path})
	}

	cmdutil.PrintTable([]string{"Status", "Platform", "Tool", "Version", "Path/Error"}, rows)
	return nil
}

// DiscoveredCertificate matches the API response structure.
type DiscoveredCertificate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Subject      string `json:"subject,omitempty"`
	Issuer       string `json:"issuer,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	DaysToExpiry int    `json:"days_to_expiry"`
	IsExpired    bool   `json:"is_expired"`
	IsCodeSign   bool   `json:"is_code_sign"`
	Type         string `json:"type,omitempty"`
	Platform     string `json:"platform"`
	UsageHint    string `json:"usage_hint,omitempty"`
}

// Discover discovers available signing certificates/identities for a platform.
func (c *Commands) Discover(args []string) error {
	fs := flag.NewFlagSet("signing discover", flag.ContinueOnError)
	platform := fs.String("platform", "", "platform to discover (windows|macos|linux) [required]")
	format := fs.String("format", "", "output format (json|table)")
	_, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	if *platform == "" {
		return errors.New("--platform is required (windows|macos|linux)")
	}

	platformLower := strings.ToLower(*platform)
	if platformLower != "windows" && platformLower != "macos" && platformLower != "linux" {
		return fmt.Errorf("invalid platform: %s (valid: windows, macos, linux)", *platform)
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/signing/discover/%s", platformLower), nil)
	if err != nil {
		return fmt.Errorf("failed to discover certificates: %w", err)
	}

	var result struct {
		Platform     string                  `json:"platform"`
		Certificates []DiscoveredCertificate `json:"certificates"`
		Errors       []string                `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		cmdutil.PrintByFormat(cmdutil.ResolveFormat(*format), body)
		return nil
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) != "table" {
		cmdutil.PrintByFormat(formatVal, body)
		return nil
	}

	// Table format
	platformName := strings.Title(platformLower)
	if platformLower == "macos" {
		platformName = "macOS"
	}

	fmt.Printf("%s Signing Certificates/Identities Found:\n\n", platformName)

	if len(result.Certificates) == 0 {
		fmt.Println("  No certificates found.")
		if len(result.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range result.Errors {
				fmt.Printf("  • %s\n", e)
			}
		}
		return nil
	}

	for i, cert := range result.Certificates {
		status := "✓"
		if cert.IsExpired {
			status = "✗ EXPIRED"
		} else if cert.DaysToExpiry >= 0 && cert.DaysToExpiry <= 30 {
			status = "⚠ Expiring soon"
		}

		fmt.Printf("  %d) %s\n", i+1, cert.Name)
		fmt.Printf("     ID: %s\n", cert.ID)
		if cert.Type != "" {
			fmt.Printf("     Type: %s\n", cert.Type)
		}
		if cert.ExpiresAt != "" && cert.ExpiresAt != "never" {
			fmt.Printf("     Expires: %s (%d days) %s\n", cert.ExpiresAt, cert.DaysToExpiry, status)
		} else if cert.ExpiresAt == "never" {
			fmt.Printf("     Expires: Never\n")
		}
		if cert.IsCodeSign {
			fmt.Printf("     Code Signing: ✓\n")
		}
		if cert.UsageHint != "" {
			fmt.Printf("     Usage: %s\n", cert.UsageHint)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range result.Errors {
			fmt.Printf("  • %s\n", e)
		}
	}

	return nil
}
