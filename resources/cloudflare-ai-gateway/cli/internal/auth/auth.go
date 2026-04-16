package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	AccountIDEnv = "CLOUDFLARE_ACCOUNT_ID"
	APITokenEnv  = "CLOUDFLARE_API_TOKEN"

	defaultVaultCommand  = "resource-vault"
	defaultAccountIDPath = "resources/cloudflare/account_id"
	defaultAPITokenPath  = "resources/cloudflare/api_token"
)

var ErrCredentialsNotConfigured = errors.New("cloudflare credentials not configured")

// Credentials holds the Cloudflare account identity required for gateway API
// operations.
type Credentials struct {
	AccountID string
	APIToken  string
	Source    string
}

// Valid reports whether both required Cloudflare credential fields are present.
func (c Credentials) Valid() bool {
	return strings.TrimSpace(c.AccountID) != "" && strings.TrimSpace(c.APIToken) != ""
}

// AuthorizationHeader renders the bearer token header value for authenticated
// API requests.
func (c Credentials) AuthorizationHeader() string {
	if strings.TrimSpace(c.APIToken) == "" {
		return ""
	}
	return "Bearer " + c.APIToken
}

// RedactedToken returns a log-safe preview of the configured API token.
func (c Credentials) RedactedToken() string {
	token := strings.TrimSpace(c.APIToken)
	switch {
	case token == "":
		return ""
	case len(token) <= 8:
		return "********"
	default:
		return token[:4] + "..." + token[len(token)-4:]
	}
}

// Resolver owns the resource-specific credential lookup chain. It prefers the
// repo's Vault resource when available and falls back to environment variables.
type Resolver struct {
	LookupEnv    func(string) (string, bool)
	LookPathFunc func(string) (string, error)
	RunCommand   func(context.Context, string, ...string) (string, error)

	VaultCommand string
	AccountIDRef string
	APITokenRef  string
}

// NewResolver returns a Resolver with standard process and environment hooks.
func NewResolver() Resolver {
	return Resolver{
		LookupEnv:    os.LookupEnv,
		LookPathFunc: exec.LookPath,
		RunCommand:   runCommand,
		VaultCommand: defaultVaultCommand,
		AccountIDRef: defaultAccountIDPath,
		APITokenRef:  defaultAPITokenPath,
	}
}

// Resolve returns the first complete credential set found from Vault or the
// process environment.
func (r Resolver) Resolve(ctx context.Context) (Credentials, error) {
	if creds, ok := r.resolveFromVault(ctx); ok {
		return creds, nil
	}
	if creds, ok := r.resolveFromEnv(); ok {
		return creds, nil
	}
	return Credentials{}, ErrCredentialsNotConfigured
}

func (r Resolver) resolveFromEnv() (Credentials, bool) {
	lookup := r.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}

	accountID, _ := lookup(AccountIDEnv)
	apiToken, _ := lookup(APITokenEnv)
	creds := Credentials{
		AccountID: strings.TrimSpace(accountID),
		APIToken:  strings.TrimSpace(apiToken),
		Source:    "env",
	}
	return creds, creds.Valid()
}

func (r Resolver) resolveFromVault(ctx context.Context) (Credentials, bool) {
	command := strings.TrimSpace(r.VaultCommand)
	if command == "" {
		command = defaultVaultCommand
	}

	lookPath := r.LookPathFunc
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath(command); err != nil {
		return Credentials{}, false
	}

	run := r.RunCommand
	if run == nil {
		run = runCommand
	}

	accountRef := firstNonEmpty(r.AccountIDRef, defaultAccountIDPath)
	tokenRef := firstNonEmpty(r.APITokenRef, defaultAPITokenPath)

	accountID, err := run(ctx, command, "content", "get", "--path", accountRef, "--format", "raw")
	if err != nil {
		return Credentials{}, false
	}
	apiToken, err := run(ctx, command, "content", "get", "--path", tokenRef, "--format", "raw")
	if err != nil {
		return Credentials{}, false
	}

	creds := Credentials{
		AccountID: strings.TrimSpace(accountID),
		APIToken:  strings.TrimSpace(apiToken),
		Source:    "vault",
	}
	return creds, creds.Valid()
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
