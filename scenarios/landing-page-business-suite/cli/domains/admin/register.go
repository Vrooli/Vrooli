package admin

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := []cliapp.Command{
		{Name: "admin-login", NeedsAPI: true, Description: "Admin login (stores session)", Run: func(args []string) error { return runLogin(deps, args) }},
		{Name: "admin-logout", NeedsAPI: true, Description: "Admin logout (clears session)", Run: func(args []string) error { return runLogout(deps, args) }},
		{Name: "admin-session", NeedsAPI: true, Description: "Admin session status", Run: func(args []string) error { return runSession(deps, args) }},
	}
	commands = append(commands, deps.EndpointCommands([]support.EndpointDef{
		{Name: "admin-profile", Method: "GET", Path: "/admin/profile", Description: "Admin profile"},
		{Name: "admin-profile-update", Method: "PUT", Path: "/admin/profile", Description: "Update admin profile"},
		{Name: "admin-stripe-settings", Method: "GET", Path: "/admin/settings/stripe", Description: "Get Stripe settings"},
		{Name: "admin-stripe-settings-update", Method: "PUT", Path: "/admin/settings/stripe", Description: "Update Stripe settings"},
		{Name: "admin-stripe-secret", Method: "GET", Path: "/admin/settings/stripe/reveal", Description: "Reveal Stripe secret"},
		{Name: "admin-stripe-verify-price", Method: "GET", Path: "/admin/stripe/verify-price", Description: "Verify Stripe price"},
		{Name: "admin-reset-demo-data", Method: "POST", Path: "/admin/reset-demo-data", Description: "Reset demo data"},
	})...)
	return cliapp.CommandGroup{Title: "Admin Core", Commands: commands}
}

func runLogin(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-login", flag.ContinueOnError)
	email := fs.String("email", "", "Admin email")
	password := fs.String("password", "", "Admin password or @file")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}

	emailValue := strings.TrimSpace(*email)
	if emailValue == "" {
		return fmt.Errorf("usage: admin-login --email <email> --password <password> [--json]")
	}
	passwordValue, err := support.ResolveSecretArg(*password)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if strings.TrimSpace(passwordValue) == "" {
		return fmt.Errorf("usage: admin-login --email <email> --password <password> [--json]")
	}

	base := strings.TrimRight(strings.TrimSpace(deps.ScenarioApp().APIClient.BaseURL()), "/")
	if base == "" {
		return fmt.Errorf("api base URL is empty; configure an API base first")
	}

	payload, err := json.Marshal(map[string]string{
		"email":    emailValue,
		"password": passwordValue,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	endpoint, err := deps.ResolveURL("/admin/login", false, nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: deps.ScenarioApp().HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	var loginResp support.AdminLoginResponse
	if err := json.Unmarshal(data, &loginResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !loginResp.Authenticated {
		return fmt.Errorf("admin login failed")
	}

	cookie := support.FindCookie(resp.Cookies(), "admin_session")
	if cookie == nil || strings.TrimSpace(cookie.Value) == "" {
		return fmt.Errorf("admin login did not return a session cookie")
	}

	cfg := support.AdminSessionConfig{
		APIBase:   base,
		Session:   cookie.Value,
		Email:     loginResp.Email,
		ExpiresAt: support.DeriveCookieExpiry(cookie),
	}
	if strings.TrimSpace(cfg.Email) == "" {
		cfg.Email = emailValue
	}
	if err := deps.SaveAdminSession(cfg); err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(data)
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Admin session stored for %s", cfg.Email)},
		Changes: func() []string {
			if cfg.ExpiresAt == nil {
				return []string{"Session persisted for the current API base"}
			}
			return []string{
				"Session persisted for the current API base",
				fmt.Sprintf("Session expires at %s", cfg.ExpiresAt.Format(time.RFC3339)),
			}
		}(),
		NextCommand: []string{"landing-page-business-suite admin-session"},
	})
}

func runLogout(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-logout", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-logout [--json]")
	}

	session, err := deps.LoadAdminSession()
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.Session) == "" {
		return fmt.Errorf("no admin session configured. Run admin-login first")
	}

	resp, err := deps.RequestAdmin("POST", "/admin/logout", nil, nil)
	if err != nil {
		if apiErr, ok := err.(*cliutil.APIError); ok {
			if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
				_ = deps.ClearAdminSession()
				if *jsonOut {
					cliutil.PrintJSON([]byte(`{"success":true}`))
					return nil
				}
				return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
					Result:      []string{"Admin session cleared"},
					Changes:     []string{"Stale local session cookie removed"},
					NextCommand: []string{"landing-page-business-suite admin-login --email <email> --password @/path/to/password.txt"},
				})
			}
		}
		return err
	}

	if err := deps.ClearAdminSession(); err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Admin session cleared"},
		Changes:     []string{"Local admin session cookie removed for the current API base"},
		NextCommand: []string{"landing-page-business-suite admin-login --email <email> --password @/path/to/password.txt"},
	})
}

func runSession(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-session", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-session [--json]")
	}

	resp, err := deps.RequestAdmin("GET", "/admin/session", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}
