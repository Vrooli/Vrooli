package admin

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

	"connectrpc.com/connect"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
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
		{Name: "admin-stripe-verify-price", Method: "GET", Path: "/admin/stripe/verify-price", Description: "Verify Stripe price"},
		{Name: "admin-reset-demo-data", Method: "POST", Path: "/admin/reset-demo-data", Description: "Reset demo data"},
	})...)
	commands = append(commands, stripeSettingsCommands(deps)...)
	return cliapp.CommandGroup{Title: "Admin Core", Commands: commands}
}

func stripeSettingsClient(deps support.Dependencies) (lpbsconnect.StripeSettingsServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewStripeSettingsServiceClient(httpClient, baseURL), nil
}

func stripeSettingsCommands(deps support.Dependencies) []cliapp.Command {
	get := cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*lpbsv1.GetStripeSettingsResponse, error) {
			client, err := stripeSettingsClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.GetStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.GetStripeSettingsRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get Stripe settings", err, nil)
			}
			return response.Msg, nil
		},
		func(cliapp.OperationContext, *lpbsv1.GetStripeSettingsResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{"Stripe settings (credentials redacted)."}, ResultsHeading: "Settings"}
		},
	)
	update := cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*lpbsv1.UpdateStripeSettingsResponse, error) {
			payload, err := support.ParseBody(ctx.Flag("body"))
			if err != nil {
				return nil, err
			}
			request := &lpbsv1.UpdateStripeSettingsRequest{}
			if err := protojson.Unmarshal(payload, request); err != nil {
				return nil, fmt.Errorf("decode Stripe settings: %w", err)
			}
			client, err := stripeSettingsClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.UpdateStripeSettings(context.Background(), connect.NewRequest(request))
			if err != nil {
				return nil, cliapp.WrapAPIError("update Stripe settings", err, nil)
			}
			return response.Msg, nil
		},
		func(cliapp.OperationContext, *lpbsv1.UpdateStripeSettingsResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Stripe settings updated."}, Changes: []string{"Runtime configuration refreshed."}}
		},
	)
	reveal := cliapp.ProtoOperational(
		func(ctx cliapp.OperationContext) (*lpbsv1.RevealStripeSecretResponse, error) {
			field := ctx.Flag("field")
			if field == "" && ctx.FlagProvided("query") {
				query, err := support.ParseQueries([]string{ctx.Flag("query")})
				if err != nil {
					return nil, err
				}
				field = query.Get("field")
			}
			if field == "" {
				return nil, fmt.Errorf("--field or --query field=<field> is required")
			}
			client, err := stripeSettingsClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.RevealStripeSecret(context.Background(), connect.NewRequest(&lpbsv1.RevealStripeSecretRequest{Field: field}))
			if err != nil {
				return nil, cliapp.WrapAPIError("reveal Stripe secret", err, nil)
			}
			return response.Msg, nil
		},
		func(ctx cliapp.OperationContext, response *lpbsv1.RevealStripeSecretResponse) cliapp.OperationalReport {
			return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Revealed %s.", response.GetField())}, NextSteps: []string{"Treat the revealed value as sensitive."}}
		},
	)
	return []cliapp.Command{
		(cliapp.Command{Name: "admin-stripe-settings", NeedsAPI: true, Description: "Get redacted Stripe settings", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(get),
		(cliapp.Command{Name: "admin-stripe-settings-update", NeedsAPI: true, Description: "Update Stripe settings", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "JSON body payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(update),
		(cliapp.Command{Name: "admin-stripe-secret", NeedsAPI: true, Description: "Reveal one Stripe setting", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "field", Description: "secret_key, webhook_secret, publishable_key, or anomaly_webhook_url"}, {Name: "query", Description: "Legacy query form: field=<field>"}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveOperational}}).WithPrimitive(reveal),
	}
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
