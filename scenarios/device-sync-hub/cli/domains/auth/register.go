// Package auth is the CLI's owner sign-in surface. Device Sync Hub does not own
// identity — the owner is a scenario-authenticator account — so these commands
// talk to scenario-authenticator directly (NOT device-sync-hub's own API) and
// persist the returned owner JWT into the cli-core config token. Every
// owner-authed devices command (`devices setup|list|pair|approve|rename|revoke`)
// then rides that token automatically via cli-core's Authorization: Bearer seam.
//
// This is the CLI half of the first-run bootstrap: `auth login` → `devices
// setup` claims the hub and trusts this machine, after which the owner can issue
// pairing codes for additional devices.
//
// Auth base resolution (in precedence order): the --auth-api-base flag, the
// $AUTH_SERVICE_URL env var (the same variable the API reads), then auto-detect
// of the sibling scenario-authenticator's API port via the typed vrooli CLI.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	vroolicli "github.com/vrooli/vrooli-cli-go"
)

const (
	// authScenario is the sibling scenario that owns identity.
	authScenario = "scenario-authenticator"
	// authPortName is its API lifecycle port role.
	authPortName = "API_PORT"
	// envAuthServiceURL mirrors the API's operator-configured authenticator URL.
	envAuthServiceURL = "AUTH_SERVICE_URL"
)

// authHTTPClient fails fast: a hung authenticator must never pin the CLI open.
var authHTTPClient = &http.Client{Timeout: 15 * time.Second}

// Register builds the `auth` subcommand group. These commands do not need
// device-sync-hub's own API (NeedsAPI: false) — they target the authenticator.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "auth",
		Description: "Sign in as the hub owner against scenario-authenticator",
		NeedsAPI:    false,
		Subcommands: []cliapp.Command{
			{Name: "login", Description: "Sign in as the owner and store the returned token", Run: func(args []string) error { return runLogin(core, args) }},
			{Name: "logout", Description: "Clear the stored owner token", Run: func(args []string) error { return runLogout(core, args) }},
			{Name: "whoami", Description: "Show the signed-in owner identity", Run: func(args []string) error { return runWhoami(core, args) }},
		},
	}
}

// authResponse mirrors scenario-authenticator's AuthResponse (api/models/user.go).
// The owner JWT is the `token` field (not `access_token`).
type authResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

// validateResponse mirrors scenario-authenticator's ValidationResponse.
type validateResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	ExpiresAt time.Time `json:"expires_at"`
}

func runLogin(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	email := fs.String("email", "", "Owner account email (required)")
	password := fs.String("password", "", "Owner account password (required)")
	authBase := fs.String("auth-api-base", "", "scenario-authenticator base URL (default: $AUTH_SERVICE_URL or auto-detected)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" || *password == "" {
		return fmt.Errorf("--email and --password are required")
	}

	base, err := resolveAuthBaseURL(context.Background(), *authBase)
	if err != nil {
		return err
	}

	var resp authResponse
	if err := postJSON(base+"/api/v1/auth/login", map[string]string{
		"email":    strings.TrimSpace(*email),
		"password": *password,
	}, "", &resp); err != nil {
		return err
	}
	if !resp.Success || strings.TrimSpace(resp.Token) == "" {
		return fmt.Errorf("login failed: %s", firstNonEmpty(resp.Message, "authenticator returned no token"))
	}

	core.Config.Token = resp.Token
	if err := core.SaveConfig(); err != nil {
		return fmt.Errorf("persist owner token: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Signed in as the hub owner.",
			fmt.Sprintf("owner=%s", firstNonEmpty(resp.User.Email, resp.User.ID)),
		},
		Changes: []string{"Stored owner token in CLI config"},
		NextCommand: []string{
			"`device-sync-hub devices setup --name <this-device>` — claim the hub and trust this device",
			"`device-sync-hub auth whoami` — confirm the signed-in owner",
		},
	}
	return render(*jsonOut, report)
}

func runLogout(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth logout", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	core.Config.Token = ""
	if err := core.SaveConfig(); err != nil {
		return fmt.Errorf("clear owner token: %w", err)
	}
	return render(*jsonOut, cliapp.MutationReport{
		Result:      []string{"Cleared the stored owner token."},
		Changes:     []string{"Removed token from CLI config"},
		NextCommand: []string{"`device-sync-hub auth login --email <email> --password <password>`"},
	})
}

func runWhoami(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth whoami", flag.ContinueOnError)
	authBase := fs.String("auth-api-base", "", "scenario-authenticator base URL (default: $AUTH_SERVICE_URL or auto-detected)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	token := strings.TrimSpace(core.Config.Token)
	if token == "" {
		return fmt.Errorf("not signed in — run `device-sync-hub auth login --email <email> --password <password>`")
	}

	base, err := resolveAuthBaseURL(context.Background(), *authBase)
	if err != nil {
		return err
	}

	var vr validateResponse
	if err := getJSON(base+"/api/v1/auth/validate", token, &vr); err != nil {
		return err
	}
	if !vr.Valid {
		return fmt.Errorf("the stored owner token is invalid or expired — run `device-sync-hub auth login` again")
	}

	results := []string{fmt.Sprintf("Owner ID: %s", vr.UserID)}
	if vr.Email != "" {
		results = append(results, fmt.Sprintf("Email: %s", vr.Email))
	}
	if len(vr.Roles) > 0 {
		results = append(results, fmt.Sprintf("Roles: %s", strings.Join(vr.Roles, ", ")))
	}
	if !vr.ExpiresAt.IsZero() {
		results = append(results, fmt.Sprintf("Session expires: %s", vr.ExpiresAt.Format(time.RFC3339)))
	}
	return render(*jsonOut, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Signed in as %s", firstNonEmpty(vr.Email, vr.UserID))},
		ResultsHeading: "Owner",
		Results:        results,
	})
}

// resolveAuthBaseURL finds scenario-authenticator's base URL by precedence:
// explicit override → $AUTH_SERVICE_URL → typed port auto-detection.
func resolveAuthBaseURL(ctx context.Context, override string) (string, error) {
	if v := strings.TrimSpace(override); v != "" {
		return strings.TrimRight(v, "/"), nil
	}
	if v := strings.TrimSpace(os.Getenv(envAuthServiceURL)); v != "" {
		return strings.TrimRight(v, "/"), nil
	}
	port, err := vroolicli.New().ScenarioPort(ctx, authScenario, authPortName)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s %s — set --auth-api-base or $AUTH_SERVICE_URL: %w", authScenario, authPortName, err)
	}
	if !port.GetSuccess() || port.GetPort() == 0 {
		detail := strings.TrimSpace(port.GetError())
		if detail == "" {
			detail = "is scenario-authenticator running?"
		}
		return "", fmt.Errorf("could not resolve %s %s (%s) — set --auth-api-base or $AUTH_SERVICE_URL", authScenario, authPortName, detail)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port.GetPort()), nil
}

// postJSON posts body as JSON to url and decodes the JSON response into out.
func postJSON(url string, body any, bearer string, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(buf)) //nolint:gosec // url is the operator/auto-resolved authenticator base, not user input
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return doJSON(req, out)
}

// getJSON GETs url (with an optional bearer token) and decodes JSON into out.
func getJSON(url, bearer string, out any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil) //nolint:gosec // url is the operator/auto-resolved authenticator base, not user input
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return doJSON(req, out)
}

func doJSON(req *http.Request, out any) error {
	resp, err := authHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("reach scenario-authenticator: %w (is it running?)", err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("decode authenticator response: %w", err)
		}
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authenticator rejected the credentials (HTTP %d)", resp.StatusCode)
	default:
		return fmt.Errorf("authenticator returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
}

func render(asJSON bool, report any) error {
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	switch r := report.(type) {
	case cliapp.MutationReport:
		return cliapp.RenderMutationReport(os.Stdout, r)
	case cliapp.ListReport:
		return cliapp.RenderListReport(os.Stdout, r)
	default:
		return fmt.Errorf("unrenderable report type %T", report)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
