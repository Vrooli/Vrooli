package access

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access"
)

func TestAgentCannotMutatePersonaACL(t *testing.T) {
	// [REQ:PSN-P0-004] ACL mutations are control-plane operations, not agent-facing RPCs.
	handler := NewConnectHandler(nil)

	create := connect.NewRequest(&accessv1.CreateGrantRequest{PersonaId: "persona-1", HumanSubject: "human-1", Level: accessv1.GrantLevel_GRANT_LEVEL_PROPOSE})
	create.Header().Set(cliutil.HeaderAgentIdentityToken, "verified-agent-token")
	if _, err := handler.CreateGrant(context.Background(), create); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("CreateGrant() error = %v, code = %v", err, connect.CodeOf(err))
	}

	remove := connect.NewRequest(&accessv1.RemoveGrantRequest{GrantId: "grant-1"})
	remove.Header().Set(cliutil.HeaderAgentIdentityToken, "verified-agent-token")
	if _, err := handler.RemoveGrant(context.Background(), remove); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("RemoveGrant() error = %v, code = %v", err, connect.CodeOf(err))
	}
}
