package auth

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client accountsconnect.AccountsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: accountsconnect.NewAccountsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) register(ctx cliapp.RunContext) error {
	resp, err := h.client.Register(context.Background(), connect.NewRequest(&accountsv1.RegisterRequest{
		Email:    ctx.Flag("email"),
		Password: ctx.Flag("password"),
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
	resp, err := h.client.Login(context.Background(), connect.NewRequest(&accountsv1.LoginRequest{
		Email:    ctx.Flag("email"),
		Password: ctx.Flag("password"),
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
