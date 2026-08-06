package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"resource-adguard-home/cli/internal/adguard"
	resourceenv "resource-adguard-home/cli/internal/env"
	"resource-adguard-home/cli/internal/health"
)

const (
	appName    = "adguard-home"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "AdGuard Home resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	commands := append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
		Title: "Diagnostics",
		Commands: []cliapp.Command{
			{
				Name:        "api-health",
				Description: "Probe AdGuard Home control API health, auth, protection, upstream, and privacy posture",
				Run: func(args []string) error {
					return runAPIHealth(args, os.Stdout)
				},
			},
			{
				Name:        "bootstrap",
				Description: "Complete first-run setup and store admin credentials through the credential authority",
				Run: func(args []string) error {
					return runBootstrap(args, os.Stdout)
				},
			},
		},
	})
	app.SetCommandsWithSubgroups(commands, []cliapp.SubcommandGroup{
		{
			Name:        "config",
			Description: "Inspect and preview AdGuard Home configuration changes",
			Subcommands: []cliapp.Command{
				{
					Name:        "preview",
					Description: "Preview upstream DNS changes without mutating AdGuard Home",
					Run: func(args []string) error {
						return runConfigPreview(args, os.Stdout)
					},
				},
			},
		},
		{
			Name:        "clients",
			Description: "Inspect AdGuard Home configured and discovered clients",
			Subcommands: []cliapp.Command{
				{
					Name:        "list",
					Description: "List AdGuard Home clients without query-level DNS log data",
					Run: func(args []string) error {
						return runClientsList(args, os.Stdout)
					},
				},
			},
		},
		{
			Name:        "querylog",
			Description: "Inspect AdGuard Home query-log privacy posture",
			Subcommands: []cliapp.Command{
				{
					Name:        "privacy",
					Description: "Report query-log configuration without reading query entries",
					Run: func(args []string) error {
						return runQueryLogPrivacy(args, os.Stdout)
					},
				},
			},
		},
	})
	return app, nil
}

func runAPIHealth(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("api-health", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the report as JSON")
	baseURL := fs.String("base-url", "", "Override AdGuard Home base URL (default: $ADGUARD_HOME_BASE_URL, $ADGUARD_HOME_URL, or http://localhost:3000)")
	username := fs.String("username", os.Getenv("ADGUARD_HOME_USERNAME"), "AdGuard Home username (defaults to $ADGUARD_HOME_USERNAME)")
	password := fs.String("password", os.Getenv("ADGUARD_HOME_PASSWORD"), "AdGuard Home password (defaults to $ADGUARD_HOME_PASSWORD)")
	credentialRef := fs.String("credential-ref", fallback(os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"), "vrooli/adguard-home"), "Credential-authority identity for admin credentials")
	timeout := fs.Duration("timeout", 10*time.Second, "Probe timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	creds, err := resolveDiagnosticCredentials(ctx, *credentialRef, *username, *password)
	if err != nil {
		return err
	}
	report, err := health.Probe(ctx, nil, resourceenv.ResolveBaseURL(*baseURL), health.Credentials{
		Username: creds.Username,
		Password: creds.Password,
	})
	if err != nil {
		return err
	}

	if *jsonOut {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}

	fmt.Fprintf(stdout, "AdGuard Home API health: %s\n", report.Status)
	fmt.Fprintf(stdout, "Base URL: %s\n", report.BaseURL)
	if report.Version != "" {
		fmt.Fprintf(stdout, "Version: %s\n", report.Version)
	}
	if report.ProtectionEnabled != nil {
		fmt.Fprintf(stdout, "Protection enabled: %t\n", *report.ProtectionEnabled)
	} else {
		fmt.Fprintln(stdout, "Protection enabled: unknown")
	}
	fmt.Fprintf(stdout, "Privacy posture: %s\n", report.PrivacyPosture)
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}

	if report.Status == health.StatusUnreachable || report.Status == health.StatusAuthFailed {
		return fmt.Errorf("AdGuard Home API health is %s", report.Status)
	}
	return nil
}

func runConfigPreview(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("config preview", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the preview as JSON")
	baseURL := fs.String("base-url", "", "Override AdGuard Home base URL")
	username := fs.String("username", os.Getenv("ADGUARD_HOME_USERNAME"), "AdGuard Home username")
	password := fs.String("password", os.Getenv("ADGUARD_HOME_PASSWORD"), "AdGuard Home password")
	credentialRef := fs.String("credential-ref", fallback(os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"), "vrooli/adguard-home"), "Credential-authority identity for admin credentials")
	timeout := fs.Duration("timeout", 10*time.Second, "Probe timeout")
	testUpstreams := fs.Bool("test-upstreams", false, "Ask AdGuard Home to test proposed upstream resolvers")
	var upstreams multiFlag
	fs.Var(&upstreams, "upstream", "Proposed upstream resolver; repeat for multiple values")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, ctx, cancel, err := newAdGuardClient(*baseURL, *username, *password, *credentialRef, *timeout)
	if err != nil {
		return err
	}
	defer cancel()

	preview, err := client.PreviewUpstreams(ctx, upstreams, *testUpstreams)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeIndentedJSON(stdout, preview)
	}

	fmt.Fprintf(stdout, "AdGuard Home upstream preview\n")
	fmt.Fprintf(stdout, "Current upstreams: %s\n", joinOrNone(preview.CurrentUpstreams))
	fmt.Fprintf(stdout, "Proposed upstreams: %s\n", joinOrNone(preview.ProposedUpstreams))
	fmt.Fprintf(stdout, "Changed: %t\n", preview.Changed)
	fmt.Fprintf(stdout, "Approval required: %t\n", preview.ApprovalRequired)
	for _, warning := range preview.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return nil
}

func runClientsList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("clients list", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the client report as JSON")
	baseURL := fs.String("base-url", "", "Override AdGuard Home base URL")
	username := fs.String("username", os.Getenv("ADGUARD_HOME_USERNAME"), "AdGuard Home username")
	password := fs.String("password", os.Getenv("ADGUARD_HOME_PASSWORD"), "AdGuard Home password")
	credentialRef := fs.String("credential-ref", fallback(os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"), "vrooli/adguard-home"), "Credential-authority identity for admin credentials")
	timeout := fs.Duration("timeout", 10*time.Second, "Probe timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, ctx, cancel, err := newAdGuardClient(*baseURL, *username, *password, *credentialRef, *timeout)
	if err != nil {
		return err
	}
	defer cancel()

	report, code, err := client.Clients(ctx)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("AdGuard Home clients endpoint returned HTTP %d", code)
	}
	if *jsonOut {
		return writeIndentedJSON(stdout, report)
	}

	fmt.Fprintf(stdout, "AdGuard Home clients: %d\n", report.Total)
	for _, client := range report.Configured {
		fmt.Fprintf(stdout, "configured: %s %s\n", fallback(client.Name, "(unnamed)"), strings.Join(client.IDs, ","))
	}
	for _, client := range report.Auto {
		fmt.Fprintf(stdout, "auto: %s %s\n", fallback(client.Name, "(unnamed)"), fallback(client.IP, "(no ip)"))
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return nil
}

func runQueryLogPrivacy(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("querylog privacy", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the privacy report as JSON")
	baseURL := fs.String("base-url", "", "Override AdGuard Home base URL")
	username := fs.String("username", os.Getenv("ADGUARD_HOME_USERNAME"), "AdGuard Home username")
	password := fs.String("password", os.Getenv("ADGUARD_HOME_PASSWORD"), "AdGuard Home password")
	credentialRef := fs.String("credential-ref", fallback(os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"), "vrooli/adguard-home"), "Credential-authority identity for admin credentials")
	timeout := fs.Duration("timeout", 10*time.Second, "Probe timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, ctx, cancel, err := newAdGuardClient(*baseURL, *username, *password, *credentialRef, *timeout)
	if err != nil {
		return err
	}
	defer cancel()

	config, endpoint, code, err := client.QueryLogConfig(ctx)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("AdGuard Home query-log endpoint returned HTTP %d", code)
	}
	report := struct {
		Endpoint       string                 `json:"endpoint"`
		QueryLog       adguard.QueryLogConfig `json:"query_log"`
		PrivacyPosture string                 `json:"privacy_posture"`
		Warnings       []string               `json:"warnings,omitempty"`
	}{
		Endpoint: endpoint,
		QueryLog: config,
	}
	if config.Enabled != nil && !*config.Enabled {
		report.PrivacyPosture = "minimal"
	} else if config.Enabled != nil && *config.Enabled {
		report.PrivacyPosture = "query_log_enabled"
		report.Warnings = append(report.Warnings, "Query log is enabled; do not expose query-level DNS history through Network Manager.")
	} else {
		report.PrivacyPosture = "unknown"
		report.Warnings = append(report.Warnings, "Query-log enabled state is not present in the AdGuard response.")
	}
	if *jsonOut {
		return writeIndentedJSON(stdout, report)
	}

	fmt.Fprintf(stdout, "AdGuard Home query-log privacy: %s\n", report.PrivacyPosture)
	fmt.Fprintf(stdout, "Endpoint: %s\n", report.Endpoint)
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return nil
}

func runBootstrap(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the bootstrap report as JSON")
	baseURL := fs.String("base-url", "", "Override AdGuard Home base URL")
	username := fs.String("username", fallback(os.Getenv("ADGUARD_HOME_USERNAME"), "admin"), "AdGuard Home admin username")
	password := fs.String("password", os.Getenv("ADGUARD_HOME_PASSWORD"), "AdGuard Home admin password; generated when empty")
	credentialRef := fs.String("credential-ref", fallback(os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"), "vrooli/adguard-home"), "Credential-authority identity for admin credentials")
	webIP := fs.String("web-ip", "0.0.0.0", "AdGuard Home web listen IP inside the container")
	webPort := fs.Int("web-port", 3000, "AdGuard Home web listen port inside the container")
	dnsIP := fs.String("dns-ip", "0.0.0.0", "AdGuard Home DNS listen IP inside the container")
	dnsPort := fs.Int("dns-port", 53, "AdGuard Home DNS listen port inside the container")
	timeout := fs.Duration("timeout", 30*time.Second, "Bootstrap timeout")
	overwriteSecret := fs.Bool("overwrite-secret", false, "Overwrite an existing stored password")
	skipConfigCheck := fs.Bool("skip-config-check", false, "Skip install/check_config preflight")
	disableQueryLog := fs.Bool("disable-query-log", true, "Disable AdGuard query logging after setup for privacy")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("initialize credential authority: %w", err)
	}
	creds, generated, err := ensureBootstrapCredentials(ctx, authority, *credentialRef, *username, *password, *overwriteSecret)
	if err != nil {
		return err
	}

	httpClient := &http.Client{
		Timeout: *timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client, err := adguard.NewClient(resourceenv.ResolveBaseURL(*baseURL), adguard.Credentials{}, adguard.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}
	cfg := adguard.InitialConfiguration{
		DNS:      adguard.AddressInfo{IP: strings.TrimSpace(*dnsIP), Port: *dnsPort},
		Web:      adguard.AddressInfo{IP: strings.TrimSpace(*webIP), Port: *webPort},
		Username: creds.Username,
		Password: creds.Password,
	}
	ref := normalizeCredentialRef(*credentialRef)
	report := bootstrapReport{
		Status:             "configured",
		BaseURL:            client.BaseURL(),
		CredentialRef:      ref,
		Username:           creds.Username,
		PasswordGenerated:  generated,
		CredentialsStored:  true,
		QueryLogHardening:  "not_requested",
		NetworkManagerHint: "network-manager resolver configure-adguard --base-url " + client.BaseURL() + " --credential-ref " + ref + " --json",
	}

	if !*skipConfigCheck {
		check, code, err := client.CheckInitialConfig(ctx, cfg)
		report.ConfigCheckHTTPStatus = code
		report.ConfigCheck = &check
		if err != nil {
			return err
		}
		if code < 200 || code >= 300 {
			report.Status = "config_check_failed"
			return writeBootstrapReport(stdout, *jsonOut, report, fmt.Errorf("AdGuard Home install config check returned HTTP %d", code))
		}
	}

	code, err := client.ConfigureInitial(ctx, cfg)
	report.ConfigureHTTPStatus = code
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		report.Status = "configure_failed"
		return writeBootstrapReport(stdout, *jsonOut, report, fmt.Errorf("AdGuard Home install configure returned HTTP %d", code))
	}

	if *disableQueryLog {
		authClient, err := adguard.NewClient(client.BaseURL(), adguard.Credentials{Username: creds.Username, Password: creds.Password}, adguard.WithHTTPClient(&http.Client{Timeout: *timeout}))
		if err != nil {
			return err
		}
		queryCode, queryErr := authClient.DisableQueryLog(ctx)
		report.QueryLogHTTPStatus = queryCode
		if queryErr != nil {
			report.QueryLogHardening = "failed"
			report.Warnings = append(report.Warnings, "Query-log hardening failed after setup.")
		} else if queryCode >= 200 && queryCode < 300 {
			report.QueryLogHardening = "disabled"
		} else {
			report.QueryLogHardening = "unsupported"
			report.Warnings = append(report.Warnings, fmt.Sprintf("Query-log hardening returned HTTP %d.", queryCode))
		}
	}

	return writeBootstrapReport(stdout, *jsonOut, report, nil)
}

func newAdGuardClient(baseURL, username, password, credentialRef string, timeout time.Duration) (*adguard.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	creds, err := resolveDiagnosticCredentials(ctx, credentialRef, username, password)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	client, err := adguard.NewClient(resourceenv.ResolveBaseURL(baseURL), adguard.Credentials{
		Username: creds.Username,
		Password: creds.Password,
	}, adguard.WithHTTPClient(&http.Client{Timeout: timeout}))
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return client, ctx, cancel, nil
}

func resolveDiagnosticCredentials(ctx context.Context, ref, username, password string) (adguard.Credentials, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if password != "" {
		if username == "" {
			username = "admin"
		}
		return adguard.Credentials{Username: username, Password: password}, nil
	}
	path := normalizeCredentialRef(ref)
	authority, err := credentialauthority.Default()
	if err != nil {
		return adguard.Credentials{}, fmt.Errorf("initialize credential authority: %w", err)
	}
	identity, err := credentialauthority.ParseIdentity(path)
	if err != nil {
		return adguard.Credentials{}, fmt.Errorf("invalid credential ref %q: %w", path, err)
	}
	if value, found, err := resolveAuthorityValue(authority, identity, "password"); err != nil {
		return adguard.Credentials{}, fmt.Errorf("read AdGuard password secret: %w", err)
	} else if found {
		password = value
	}
	if username == "" {
		if value, found, err := resolveAuthorityValue(authority, identity, "username"); err != nil {
			return adguard.Credentials{}, fmt.Errorf("read AdGuard username secret: %w", err)
		} else if found {
			username = value
		}
	}
	if username == "" {
		username = "admin"
	}
	if password == "" {
		return adguard.Credentials{}, fmt.Errorf("credential ref %s does not contain an AdGuard password", path)
	}
	return adguard.Credentials{Username: username, Password: password}, nil
}

func writeIndentedJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func fallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

type bootstrapReport struct {
	Status                string                       `json:"status"`
	BaseURL               string                       `json:"base_url"`
	CredentialRef         string                       `json:"credential_ref"`
	Username              string                       `json:"username"`
	PasswordGenerated     bool                         `json:"password_generated"`
	CredentialsStored     bool                         `json:"credentials_stored"`
	ConfigCheckHTTPStatus int                          `json:"config_check_http_status,omitempty"`
	ConfigCheck           *adguard.CheckConfigResponse `json:"config_check,omitempty"`
	ConfigureHTTPStatus   int                          `json:"configure_http_status,omitempty"`
	QueryLogHardening     string                       `json:"query_log_hardening"`
	QueryLogHTTPStatus    int                          `json:"query_log_http_status,omitempty"`
	Warnings              []string                     `json:"warnings,omitempty"`
	NetworkManagerHint    string                       `json:"network_manager_hint"`
}

func writeBootstrapReport(stdout io.Writer, jsonOut bool, report bootstrapReport, err error) error {
	if jsonOut {
		if encodeErr := writeIndentedJSON(stdout, report); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	fmt.Fprintf(stdout, "AdGuard Home bootstrap: %s\n", report.Status)
	fmt.Fprintf(stdout, "Base URL: %s\n", report.BaseURL)
	fmt.Fprintf(stdout, "Credential ref: %s\n", report.CredentialRef)
	fmt.Fprintf(stdout, "Username: %s\n", report.Username)
	fmt.Fprintf(stdout, "Password generated: %t\n", report.PasswordGenerated)
	fmt.Fprintf(stdout, "Query-log hardening: %s\n", report.QueryLogHardening)
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	fmt.Fprintf(stdout, "Next: %s\n", report.NetworkManagerHint)
	return err
}

type credentialStore interface {
	Resolve(credentialauthority.Identity, string) (string, error)
	Put(credentialauthority.Identity, string, string) error
}

func resolveAuthorityValue(store credentialStore, identity credentialauthority.Identity, field string) (string, bool, error) {
	value, err := store.Resolve(identity, field)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	value = strings.TrimSpace(value)
	return value, value != "", nil
}

func ensureBootstrapCredentials(ctx context.Context, store credentialStore, ref, username, password string, overwrite bool) (adguard.Credentials, bool, error) {
	_ = ctx
	path := normalizeCredentialRef(ref)
	identity, err := credentialauthority.ParseIdentity(path)
	if err != nil {
		return adguard.Credentials{}, false, fmt.Errorf("invalid credential ref %q: %w", path, err)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	password = strings.TrimSpace(password)

	if existingPassword, found, err := resolveAuthorityValue(store, identity, "password"); err != nil {
		return adguard.Credentials{}, false, fmt.Errorf("read existing AdGuard password secret: %w", err)
	} else if found && !overwrite {
		if password != "" && password != existingPassword {
			return adguard.Credentials{}, false, fmt.Errorf("credential ref %s already has a password; pass --overwrite-secret to replace it", path)
		}
		password = existingPassword
	}

	generated := false
	if password == "" {
		var err error
		password, err = generatePassword()
		if err != nil {
			return adguard.Credentials{}, false, err
		}
		generated = true
	}
	if err := store.Put(identity, "username", username); err != nil {
		return adguard.Credentials{}, generated, fmt.Errorf("store AdGuard username credential: %w", err)
	}
	if err := store.Put(identity, "password", password); err != nil {
		return adguard.Credentials{}, generated, fmt.Errorf("store AdGuard password credential: %w", err)
	}
	return adguard.Credentials{Username: username, Password: password}, generated, nil
}

func generatePassword() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate AdGuard admin password: %w", err)
	}
	password := base64.RawURLEncoding.EncodeToString(buf)
	if len(password) < 32 {
		return "", fmt.Errorf("generated AdGuard admin password too short")
	}
	return password, nil
}

func normalizeCredentialRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "vrooli/adguard-home"
	}
	return ref
}
