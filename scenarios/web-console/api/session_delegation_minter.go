package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// sessionDelegationResource is the resource the delegation is audience-scoped
// to. It is deliberately EMPTY: scenario-authenticator resolves an empty
// resource to the realm DEFAULT audience (scenario-authenticator:default),
// which is exactly what agent-manager's owner-identity verifier accepts out of
// the box (it defaults to scenario-authenticator:default when
// VROOLI_AUTH_SCENARIO_AUDIENCE is unset). A named resource would be refused
// unless that resource audience is registered in the deployment.
const sessionDelegationResource = ""

// mintSessionDelegation authenticates to scenario-authenticator with the
// operator's own verified credential and mints a short-lived, session-scoped
// human-owner delegation, storing it for the env builder. It is BEST-EFFORT:
// every failure path (unresolved authenticator, RPC error, empty token) logs
// and returns without affecting the already-created session. Web Console never
// calls agent-manager here — the delegation is verified by agent-manager later,
// off the session-launch path, so a down agent-manager never blocks a launch.
func mintSessionDelegation(ctx context.Context, sessionID, rawCredential string) bool {
	if sessionID == "" || rawCredential == "" {
		return false
	}
	// Bound the mint so a slow/absent authenticator cannot delay session launch.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	if err != nil || base == "" {
		log.Printf("session-delegation: scenario-authenticator unresolved; session %s attaches unattributed: %v", sessionID, err)
		return false
	}
	client := accountsconnect.NewAccountsServiceClient(&http.Client{Timeout: 5 * time.Second}, base)
	resp, err := client.MintSessionDelegation(ctx, connect.NewRequest(&accountsv1.MintSessionDelegationRequest{
		AccessToken: bearerTokenValue(rawCredential),
		WcSessionId: sessionID,
		Resource:    sessionDelegationResource,
	}))
	if err != nil {
		// A refusal (non-human caller, unverifiable credential) is expected and
		// benign: the session simply attaches unattributed. Do not surface it.
		// A refusal (non-human caller, unverifiable credential) is expected and
		// benign: the session simply attaches unattributed.
		log.Printf("session-delegation: mint declined for session %s (attaches unattributed)", sessionID)
		return false
	}
	token := ""
	var expires time.Time
	if resp != nil && resp.Msg != nil {
		token = resp.Msg.GetDelegationToken()
		if ts := resp.Msg.GetExpiresAt(); ts != nil {
			expires = ts.AsTime()
		}
	}
	if token == "" || expires.IsZero() || !time.Now().Before(expires) {
		return false
	}
	activeSessionDelegations.Put(sessionID, token, expires)
	return true
}

// bearerTokenValue strips a leading "Bearer " so the authenticator receives the
// bare token. A Cloudflare Access assertion carries no scheme and passes through.
func bearerTokenValue(raw string) string {
	const prefix = "Bearer "
	if len(raw) > len(prefix) && (raw[:len(prefix)] == prefix || raw[:len(prefix)] == "bearer ") {
		return raw[len(prefix):]
	}
	return raw
}
