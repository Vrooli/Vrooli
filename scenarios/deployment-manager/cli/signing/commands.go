package signing

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

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
		return errors.New("signing validation is owned by scenario-to-desktop; use its signing CLI or API")
	case "prerequisites":
		return errors.New("signing prerequisite checks are owned by scenario-to-desktop; use its signing CLI or API")
	case "discover":
		return errors.New("certificate discovery is owned by scenario-to-desktop; use its signing CLI or API")
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

`
	fmt.Println(help)
	return nil
}

// Show displays the signing configuration for a profile.
func (c *Commands) Show(args []string) error {
	fs := flag.NewFlagSet("signing show", flag.ContinueOnError)
	platform := fs.String("platform", "", "show only specific platform (windows|macos|linux)")
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

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
		report := signingShowReport(&config)
		if *platform != "" {
			report.Summary = append(report.Summary, fmt.Sprintf("Platform filter: %s", *platform))
		}
		return cliapp.RenderListReport(os.Stdout, report)
	}

	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

// printSigningTable prints signing config in table format.
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

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
			Identity:           *identity,
			TeamID:             *teamID,
			HardenedRuntime:    *hardenedRuntime,
			Notarize:           *notarize,
			EntitlementsFile:   *entitlements,
			AppleIDEnv:         *appleIDEnv,
			AppleIDPasswordEnv: *applePasswordEnv,
			AppleAPIKeyID:      *apiKeyID,
			AppleAPIKeyFile:    *apiKeyFile,
			AppleAPIIssuerID:   *apiIssuer,
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

	if len(remaining) < 1 {
		return errors.New("usage: signing validate <profile>")
	}
	profileID := remaining[0]

	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/profiles/%s/signing/validate", profileID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to validate signing: %w", err)
	}

	var result struct {
		Valid     bool                              `json:"valid"`
		Message   string                            `json:"message,omitempty"`
		Errors    []map[string]string               `json:"errors,omitempty"`
		Warnings  []map[string]string               `json:"warnings,omitempty"`
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

	report := cliapp.OperationalReport{}
	if result.Valid {
		report.Status = append(report.Status, "Signing validation: valid")
	} else {
		report.Status = append(report.Status, "Signing validation: failed")
	}
	if result.Message != "" {
		report.Status = append(report.Status, fmt.Sprintf("Message: %s", result.Message))
	}
	if len(result.Errors) > 0 {
		group := cliapp.TriageGroup{Heading: "Errors"}
		for _, e := range result.Errors {
			item := fmt.Sprintf("[%s] %s: %s", e["platform"], e["code"], e["message"])
			if rem := e["remediation"]; rem != "" {
				item += fmt.Sprintf(" | Remediation: %s", rem)
			}
			group.Items = append(group.Items, item)
		}
		report.Triage = append(report.Triage, group)
	}

	if len(result.Warnings) > 0 {
		group := cliapp.TriageGroup{Heading: "Warnings"}
		for _, w := range result.Warnings {
			group.Items = append(group.Items, fmt.Sprintf("[%s] %s: %s", w["platform"], w["code"], w["message"]))
		}
		report.Triage = append(report.Triage, group)
	}
	if result.Valid {
		report.NextSteps = []string{"deployment-manager signing show <profile-id>"}
	} else {
		report.NextSteps = []string{
			"deployment-manager signing prerequisites",
			"deployment-manager signing show <profile-id>",
		}
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// Prerequisites checks available signing tools.
func (c *Commands) Prerequisites(args []string) error {
	fs := flag.NewFlagSet("signing prerequisites", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Signing tools checked: %d", len(result.Tools))},
		ResultsHeading: "Prerequisites",
		RetrievalHints: []string{"deployment-manager signing discover --platform <windows|macos|linux>"},
	}
	for _, t := range result.Tools {
		status := "installed"
		if !t.Installed {
			status = "missing"
		}
		version := t.Version
		if version == "" {
			version = "-"
		}
		location := t.Path
		if location == "" {
			location = fallbackString(t.Error, "-")
		}
		report.Results = append(report.Results, fmt.Sprintf("%s %s %s version=%s source=%s", t.Platform, t.Tool, status, version, location))
	}
	return cliapp.RenderListReport(os.Stdout, report)
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

	platformName := strings.ToUpper(platformLower[:1]) + platformLower[1:]
	if platformLower == "macos" {
		platformName = "macOS"
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Platform: %s", platformName),
			fmt.Sprintf("Certificates found: %d", len(result.Certificates)),
		},
		ResultsHeading: "Certificates",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager signing set <profile> --platform %s ...", platformLower),
		},
	}
	if len(result.Certificates) == 0 {
		report.Results = []string{"(none found)"}
		if len(result.Errors) > 0 {
			report.RetrievalHints = append(report.RetrievalHints, result.Errors...)
		}
		return cliapp.RenderListReport(os.Stdout, report)
	}

	for i, cert := range result.Certificates {
		status := "valid"
		if cert.IsExpired {
			status = "expired"
		} else if cert.DaysToExpiry >= 0 && cert.DaysToExpiry <= 30 {
			status = "expiring-soon"
		}
		line := fmt.Sprintf("%d. %s id=%s status=%s", i+1, cert.Name, cert.ID, status)
		if cert.Type != "" {
			line += fmt.Sprintf(" type=%s", cert.Type)
		}
		if cert.ExpiresAt != "" && cert.ExpiresAt != "never" {
			line += fmt.Sprintf(" expires=%s (%d days)", cert.ExpiresAt, cert.DaysToExpiry)
		} else if cert.ExpiresAt == "never" {
			line += " expires=never"
		}
		if cert.IsCodeSign {
			line += " code-signing=yes"
		}
		if cert.UsageHint != "" {
			line += fmt.Sprintf(" usage=%s", cert.UsageHint)
		}
		report.Results = append(report.Results, line)
	}

	if len(result.Errors) > 0 {
		report.RetrievalHints = append(report.RetrievalHints, result.Errors...)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
