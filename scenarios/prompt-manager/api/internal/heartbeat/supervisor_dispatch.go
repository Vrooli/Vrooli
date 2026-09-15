package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	amconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/encoding/protojson"
)

type SupervisorRunClient interface {
	CreateSupervisorRun(context.Context, *api.CreateSupervisorRunRequest, string) (*Run, error)
}

func resolveSupervisorDispatchCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", errors.New("supervisor credential authority unavailable; inspect vrooli credentials status")
	}
	authority.Recheck()
	token, err := authority.Require(credentialauthority.Identity("vrooli/prompt-manager/effort-supervision"), "dispatcher")
	if err != nil {
		return "", errors.New("supervisor dispatch credential unavailable; operator must issue-dispatch through AM or repair the canonical credential authority")
	}
	return token, nil
}

// CreateSupervisorRun is distinct from the human-owner CreateRunDelegated path.
// The response/error and default client never retain the bearer credential.
func (c *AgentManagerClient) CreateSupervisorRun(ctx context.Context, req *api.CreateSupervisorRunRequest, token string) (*Run, error) {
	if req == nil || token == "" || req.AuthorizationId == "" || req.IdempotencyKey == "" {
		return nil, errors.New("supervisor dispatch requires explicit authorization, credential and wake identity")
	}
	base, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, errors.New("supervisor dispatch owner unavailable")
	}
	client := *c.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	request := connect.NewRequest(req)
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := amconnect.NewAgentManagerServiceClient(&client, base, connect.WithProtoJSON()).CreateSupervisorRun(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("supervisor dispatch refused or uncertain (%s); reconcile the original wake before another dispatch", connect.CodeOf(err))
	}
	encoded, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(response.Msg)
	if err != nil {
		return nil, errors.New("supervisor run response invalid")
	}
	var result CreateRunResponse
	if err = json.Unmarshal(encoded, &result); err != nil || result.Run == nil || result.Run.ID == "" {
		return nil, errors.New("supervisor run identity unavailable; reconcile the original wake")
	}
	return result.Run, nil
}
