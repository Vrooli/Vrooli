package sessions

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions/sessions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sessionsconnect.SessionsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sessionsconnect.NewSessionsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListSessions(context.Background(), connect.NewRequest(&sessionsv1.ListSessionsRequest{
		AccessToken: ctx.Flag("access-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list sessions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sessions response")
	}
	results := make([]string, 0, len(resp.Msg.Sessions))
	for _, s := range resp.Msg.Sessions {
		results = append(results, formatSession(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d active session(s).", len(resp.Msg.Sessions))},
		ResultsHeading: "Sessions",
		Results:        results,
		RetrievalHints: []string{
			"`sessions revoke <session-id>` — revoke one session",
			"`sessions revoke-all --access-token <token>` — log out everywhere",
		},
	})
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	resp, err := h.client.RevokeSession(context.Background(), connect.NewRequest(&sessionsv1.RevokeSessionRequest{
		SessionId: id,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("revoke session %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Revoked session %s (idempotent).", id)},
	})
}

func (h *handlers) revokeAll(ctx cliapp.RunContext) error {
	resp, err := h.client.RevokeAllSessions(context.Background(), connect.NewRequest(&sessionsv1.RevokeAllSessionsRequest{
		AccessToken: ctx.Flag("access-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("revoke all sessions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Revoked %d session(s).", resp.Msg.RevokedCount)},
	})
}

func formatSession(s *sessionsv1.Session) string {
	if s == nil {
		return "(nil)"
	}
	created := ""
	if s.CreatedAt != nil {
		created = s.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — ip=%s ua=%q [created=%s]", s.Id, s.IpAddress, s.UserAgent, created)
}
