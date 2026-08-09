package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core     *cliapp.ScenarioApp
	client   accountsconnect.AccountsServiceClient
	password passwordSource
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:     core,
		client:   accountsconnect.NewAccountsServiceClient(httpClient, baseURL),
		password: newPasswordSource(),
	}
}

func (h *handlers) register(ctx cliapp.RunContext) error {
	password, err := h.password.one(ctx.BoolFlag("password-stdin"))
	if err != nil {
		return err
	}
	defer clear(password)
	resp, err := h.client.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{
		Email:    ctx.Flag("email"),
		Password: string(password),
		Username: ctx.Flag("username"),
		Realm:    ctx.Flag("realm"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("register account", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Account == nil {
		return fmt.Errorf("server returned no account")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Registered %s (%s).", resp.Msg.Account.Email, resp.Msg.Account.Id)},
		Changes:     tokenLines(resp.Msg.Tokens),
		NextCommand: []string{"`auth validate --access-token <token>` — verify the issued token"},
	})
}

func (h *handlers) login(ctx cliapp.RunContext) error {
	if resp, err := exchangeLocal(context.Background()); err == nil {
		return cliapp.RenderProtoMutation(ctx, resp, cliapp.MutationReport{
			Result:      []string{"Signed in through the local machine binding."},
			Changes:     tokenLines(resp.Tokens),
			NextCommand: []string{"`sessions list --access-token <token>` — show active sessions"},
		})
	}
	password, err := h.password.one(ctx.BoolFlag("password-stdin"))
	if err != nil {
		return err
	}
	defer clear(password)
	resp, err := h.client.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{
		Email:    ctx.Flag("email"),
		Password: string(password),
		Realm:    ctx.Flag("realm"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("login", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Account == nil {
		return fmt.Errorf("server returned no account")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Signed in as %s.", resp.Msg.Account.Email)},
		Changes:     tokenLines(resp.Msg.Tokens),
		NextCommand: []string{"`sessions list --access-token <token>` — show active sessions"},
	})
}

func (h *handlers) changePassword(ctx cliapp.RunContext) error {
	current, next, err := h.password.pair(ctx.BoolFlag("password-stdin"))
	if err != nil {
		return err
	}
	defer clear(current)
	defer clear(next)
	resp, err := h.client.ChangePassword(context.Background(), connect.NewRequest(&accountsv1.ChangePasswordRequest{
		AccessToken: ctx.Flag("access-token"), CurrentPassword: string(current), NewPassword: string(next),
	}))
	if err != nil {
		return cliapp.WrapAPIError("change password", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no password-change result")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Password changed; revoked %d live session(s).", resp.Msg.RevokedSessions)},
	})
}

func (h *handlers) refresh(ctx cliapp.RunContext) error {
	resp, err := h.client.Refresh(context.Background(), connect.NewRequest(&accountsv1.RefreshRequest{
		RefreshToken: ctx.Flag("refresh-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("refresh token", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no tokens")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Rotated refresh token."},
		Changes: tokenLines(resp.Msg.Tokens),
	})
}

func (h *handlers) logout(ctx cliapp.RunContext) error {
	resp, err := h.client.Logout(context.Background(), connect.NewRequest(&accountsv1.LogoutRequest{
		AccessToken: ctx.Flag("access-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("logout", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{"Signed out (token blacklisted, sessions revoked)."},
	})
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.Validate(context.Background(), connect.NewRequest(&accountsv1.ValidateRequest{
		AccessToken: ctx.Flag("access-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate token", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation result")
	}
	results := []string{fmt.Sprintf("valid=%t", resp.Msg.Valid)}
	if resp.Msg.Valid {
		results = append(results,
			fmt.Sprintf("user_id=%s", resp.Msg.UserId),
			fmt.Sprintf("email=%s", resp.Msg.Email),
			fmt.Sprintf("realm=%s", resp.Msg.Realm),
			fmt.Sprintf("roles=%v", resp.Msg.Roles),
		)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Token is %s.", validityWord(resp.Msg.Valid))},
		ResultsHeading: "Claims",
		Results:        results,
	})
}

func (h *handlers) grantScope(ctx cliapp.RunContext) error {
	resp, err := h.client.GrantScope(context.Background(), connect.NewRequest(&accountsv1.GrantScopeRequest{
		AccessToken: ctx.Flag("access-token"), PrincipalId: ctx.Flag("principal-id"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("grant scope", err, nil)
	}
	return renderScopes(ctx, "Scope granted.", resp.Msg.PrincipalId, resp.Msg.Scopes)
}

func (h *handlers) revokeScope(ctx cliapp.RunContext) error {
	resp, err := h.client.RevokeScope(context.Background(), connect.NewRequest(&accountsv1.RevokeScopeRequest{
		AccessToken: ctx.Flag("access-token"), PrincipalId: ctx.Flag("principal-id"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("revoke scope", err, nil)
	}
	return renderScopes(ctx, "Scope revoked.", resp.Msg.PrincipalId, resp.Msg.Scopes)
}

func (h *handlers) listScopes(ctx cliapp.RunContext) error {
	resp, err := h.client.ListScopes(context.Background(), connect.NewRequest(&accountsv1.ListScopesRequest{
		AccessToken: ctx.Flag("access-token"), PrincipalId: ctx.Flag("principal-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list scopes", err, nil)
	}
	return renderScopes(ctx, "Scopes listed.", resp.Msg.PrincipalId, resp.Msg.Scopes)
}

func (h *handlers) linkMachineAccount(ctx cliapp.RunContext) error {
	principal, err := currentLocalPrincipal()
	if err != nil {
		return fmt.Errorf("resolve local principal: %w", err)
	}
	machineID := strings.TrimSpace(ctx.Flag("machine-id"))
	if machineID == "" {
		machineID, err = os.Hostname()
		if err != nil || strings.TrimSpace(machineID) == "" {
			return fmt.Errorf("resolve machine id: %w", err)
		}
	}
	resp, err := h.client.LinkMachineAccount(context.Background(), connect.NewRequest(&accountsv1.LinkMachineAccountRequest{
		AccessToken: ctx.Flag("access-token"), MachineId: machineID, LocalPrincipal: principal, Realm: ctx.Flag("realm"), IsDefault: ctx.BoolFlag("default"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("link machine account", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no machine binding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Linked %s to %s (%s).", resp.Msg.MachineId, resp.Msg.AccountId, resp.Msg.LocalPrincipal)},
	})
}

func (h *handlers) issueBreakGlass(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("token-file"))
	if path == "" {
		path = strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_TOKEN_FILE"))
	}
	if path == "" {
		return fmt.Errorf("--token-file is required; the credential is never printed")
	}
	requested := splitScopes(ctx.Flag("scopes"))
	resp, err := h.client.IssueBreakGlass(context.Background(), connect.NewRequest(&accountsv1.IssueBreakGlassRequest{
		AccessToken: ctx.Flag("access-token"), Scopes: requested,
	}))
	if err != nil {
		return cliapp.WrapAPIError("issue break-glass credential", err, nil)
	}
	if resp == nil || resp.Msg == nil || strings.TrimSpace(resp.Msg.Credential) == "" || resp.Msg.ExpiresAt <= 0 {
		return fmt.Errorf("server returned no complete break-glass credential")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(resp.Msg.Credential+"\n"), 0o600); err != nil {
		return fmt.Errorf("write credential file: %w", err)
	}
	return cliapp.RenderProtoMutation(ctx, &accountsv1.IssueBreakGlassResponse{ExpiresAt: resp.Msg.ExpiresAt}, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Break-glass credential written owner-only to %s; expires at %s.", path, time.Unix(resp.Msg.ExpiresAt, 0).UTC().Format(time.RFC3339))},
	})
}

func splitScopes(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func renderScopes(ctx cliapp.RunContext, summary, principal string, scopes []string) error {
	return cliapp.RenderProtoList(ctx, &accountsv1.ScopeResponse{PrincipalId: principal, Scopes: scopes}, cliapp.ListReport{
		Summary: []string{summary}, ResultsHeading: "Scopes", Results: append([]string{fmt.Sprintf("principal_id=%s", principal)}, scopes...),
	})
}

func tokenLines(tp *accountsv1.TokenPair) []string {
	if tp == nil {
		return nil
	}
	lines := []string{
		fmt.Sprintf("access_token=%s", tp.AccessToken),
		fmt.Sprintf("refresh_token=%s", tp.RefreshToken),
	}
	if tp.AccessTokenExpiresAt != nil {
		lines = append(lines, fmt.Sprintf("access_token_expires_at=%s", tp.AccessTokenExpiresAt.AsTime().Format("2006-01-02T15:04:05Z07:00")))
	}
	return lines
}

func validityWord(valid bool) string {
	if valid {
		return "valid"
	}
	return "invalid"
}
