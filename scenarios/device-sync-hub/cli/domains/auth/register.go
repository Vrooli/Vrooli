// Package auth is the CLI's owner sign-in / registration surface. Device Sync
// Hub does not own identity — the owner is a scenario-authenticator account —
// but the CLI never talks to scenario-authenticator directly. Every
// inter-scenario hop is API-to-API: these commands call device-sync-hub's OWN
// IdentityService (the same same-origin facade the UI uses), and the hub
// forwards to scenario-authenticator (resolved by name via api-core/discovery).
// One identity front door, no port-hunting, no AUTH_SERVICE_URL.
//
// The returned owner JWT is persisted into the cli-core config token, so every
// owner-authed devices command (`devices setup|list|pair|approve|rename|revoke`)
// then rides it automatically via cli-core's Authorization: Bearer seam.
//
// First-run bootstrap from the CLI: `auth register` (or `auth login`) →
// `devices setup` claims the hub and trusts this machine.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity/identity_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// GroupName is the manifest group name this package owns.
const GroupName = "auth"

// Register builds the `auth` subcommand group. The two API-backed commands
// (`login`, `register`) are declared in cli/manifest.json and wired here via
// the bindings map — they call the hub's own IdentityService (the hub forwards
// to scenario-authenticator). The two token-only commands (`logout`, `whoami`)
// operate purely on the locally-stored CLI token (no API binding, no proto
// method), so they are appended after LoadFromManifest, exactly as the
// transfer domain appends its REST byte edges.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings := map[string]func(cliapp.RunContext) error{
		"IdentityService.Login":    func(ctx cliapp.RunContext) error { return runLogin(core, argsFromCtx(ctx, "email", "password")) },
		"IdentityService.Register": func(ctx cliapp.RunContext) error { return runRegister(core, argsFromCtx(ctx, "email", "password", "username")) },
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("auth: load from manifest: %w", err)
	}
	group.NeedsAPI = true

	group.Subcommands = append(group.Subcommands,
		cliapp.Command{Name: "logout", Description: "Clear the stored owner token", Run: func(args []string) error { return runLogout(core, args) }},
		cliapp.Command{Name: "whoami", Description: "Show the signed-in owner identity (from the stored token)", Run: func(args []string) error { return runWhoami(core, args) }},
	)
	return group, nil
}

// argsFromCtx reprojects the named valued flags parsed by LoadFromManifest back
// into the `--name value` slice form the run* helpers (and their tests) expect.
// Empty flags are skipped; the run* helpers enforce required-ness and trimming.
// The --json pseudo-flag is forwarded so JSON output still works end to end.
func argsFromCtx(ctx cliapp.RunContext, names ...string) []string {
	args := make([]string, 0, len(names)*2+1)
	for _, name := range names {
		if v := ctx.Flag(name); v != "" {
			args = append(args, "--"+name, v)
		}
	}
	if ctx.JSON() {
		args = append(args, "--json")
	}
	return args
}

func identityClient(core *cliapp.ScenarioApp) identityconnect.IdentityServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return identityconnect.NewIdentityServiceClient(httpClient, baseURL)
}

func runLogin(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	email := fs.String("email", "", "Owner account email (required)")
	password := fs.String("password", "", "Owner account password (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" || *password == "" {
		return fmt.Errorf("--email and --password are required")
	}

	resp, err := identityClient(core).Login(context.Background(), connect.NewRequest(&identityv1.LoginRequest{
		Email:    strings.TrimSpace(*email),
		Password: *password,
	}))
	if err != nil {
		return cliapp.WrapAPIError("sign in", err, nil)
	}
	return storeTokenAndReport(core, *jsonOut, resp.Msg.GetToken(), resp.Msg.GetEmail(), resp.Msg.GetUserId(), "Signed in as the hub owner.")
}

func runRegister(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth register", flag.ContinueOnError)
	email := fs.String("email", "", "Owner account email (required)")
	password := fs.String("password", "", "Owner account password (required)")
	username := fs.String("username", "", "Optional display name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" || *password == "" {
		return fmt.Errorf("--email and --password are required")
	}

	resp, err := identityClient(core).Register(context.Background(), connect.NewRequest(&identityv1.RegisterRequest{
		Email:    strings.TrimSpace(*email),
		Password: *password,
		Username: strings.TrimSpace(*username),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create owner account", err, nil)
	}
	return storeTokenAndReport(core, *jsonOut, resp.Msg.GetToken(), resp.Msg.GetEmail(), resp.Msg.GetUserId(), "Created the hub owner account and signed in.")
}

// storeTokenAndReport persists the issued JWT and renders the next-step guidance
// shared by login and register.
func storeTokenAndReport(core *cliapp.ScenarioApp, jsonOut bool, token, email, userID, headline string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("the hub returned no owner token")
	}
	core.Config.Token = token
	if err := core.SaveConfig(); err != nil {
		return fmt.Errorf("persist owner token: %w", err)
	}
	return render(jsonOut, cliapp.MutationReport{
		Result: []string{
			headline,
			fmt.Sprintf("owner=%s", firstNonEmpty(email, userID)),
		},
		Changes: []string{"Stored owner token in CLI config"},
		NextCommand: []string{
			"`device-sync-hub devices setup --name <this-device>` — claim the hub and trust this device",
			"`device-sync-hub auth whoami` — confirm the signed-in owner",
		},
	})
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

// ownerTokenClaims is the subset of the owner JWT the CLI surfaces for whoami.
type ownerTokenClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Exp    int64    `json:"exp"`
}

// runWhoami shows the stored token's claims. It decodes the JWT payload locally
// (the token is the caller's own); the hub verifies the signature on every real
// owner-authed call, so this is a convenience readout, not an authorization
// check. Avoids any direct authenticator round-trip.
func runWhoami(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("auth whoami", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	token := strings.TrimSpace(core.Config.Token)
	if token == "" {
		return fmt.Errorf("not signed in — run `device-sync-hub auth login --email <email> --password <password>`")
	}
	claims, err := decodeClaims(token)
	if err != nil {
		return fmt.Errorf("the stored owner token is not a readable JWT — run `device-sync-hub auth login` again: %w", err)
	}

	results := []string{fmt.Sprintf("Owner ID: %s", claims.UserID)}
	if claims.Email != "" {
		results = append(results, fmt.Sprintf("Email: %s", claims.Email))
	}
	if len(claims.Roles) > 0 {
		results = append(results, fmt.Sprintf("Roles: %s", strings.Join(claims.Roles, ", ")))
	}
	if claims.Exp > 0 {
		exp := time.Unix(claims.Exp, 0)
		status := "expires"
		if time.Now().After(exp) {
			status = "EXPIRED at"
		}
		results = append(results, fmt.Sprintf("Session %s: %s", status, exp.Format(time.RFC3339)))
	}
	return render(*jsonOut, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Signed in as %s", firstNonEmpty(claims.Email, claims.UserID))},
		ResultsHeading: "Owner",
		Results:        results,
	})
}

// decodeClaims base64url-decodes a JWT's payload segment. It does NOT verify the
// signature (display only) — server-side verification is what gates access.
func decodeClaims(token string) (ownerTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ownerTokenClaims{}, fmt.Errorf("not a JWT")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ownerTokenClaims{}, fmt.Errorf("decode payload: %w", err)
	}
	var c ownerTokenClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return ownerTokenClaims{}, fmt.Errorf("parse claims: %w", err)
	}
	if c.UserID == "" {
		return ownerTokenClaims{}, fmt.Errorf("token carries no user_id")
	}
	return c, nil
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
