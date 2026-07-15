package onboard_test

import (
	"context"
	"os/user"
	"testing"

	onboardhandler "vrooli-bridge/handlers/onboard"
	"vrooli-bridge/internal/auth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// TestGetLocalNodeSuggestion_ReturnsLoopbackAndOSUser proves the prefill helper
// reports the loopback host and the control-plane process's real OS user (the
// value the browser cannot resolve on its own), gated on owner auth.
func TestGetLocalNodeSuggestion_ReturnsLoopbackAndOSUser(t *testing.T) {
	handler := onboardhandler.NewConnectHandler(onboardhandler.Deps{})
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	resp, err := handler.GetLocalNodeSuggestion(ownerCtx, connect.NewRequest(&onboardv1.GetLocalNodeSuggestionRequest{}))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", resp.Msg.GetHost())

	// The suggested user must be the process's actual OS user — never a guess.
	if u, uerr := user.Current(); uerr == nil && u.Username != "" {
		require.True(t, resp.Msg.GetAvailable(), "a resolvable OS user must mark the suggestion available")
		require.Equal(t, u.Username, resp.Msg.GetUser())
	}
}

// TestGetLocalNodeSuggestion_RequiresOwner proves the helper is owner-gated like
// every other onboard verb: no identity in context ⇒ unauthenticated.
func TestGetLocalNodeSuggestion_RequiresOwner(t *testing.T) {
	handler := onboardhandler.NewConnectHandler(onboardhandler.Deps{})

	_, err := handler.GetLocalNodeSuggestion(context.Background(), connect.NewRequest(&onboardv1.GetLocalNodeSuggestionRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
