package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	credentials "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials/credentials_v1connect"
)

// Register exposes the typed agent/runtime broker contract. The service owns
// grant evaluation, capability custody, origin pinning, and redaction; the CLI
// only maps declared arguments to generated requests and renders responses.
func Register(core *cliapp.ScenarioApp, manifest []byte) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := credentialsconnect.NewCredentialBrokerServiceClient(httpClient, baseURL)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "broker", map[string]cliapp.PrimitiveHandler{
		"CredentialBrokerService.GetAccessRequest":       getAccessRequestPrimitive(client),
		"CredentialBrokerService.WaitAccessRequest":      waitAccessRequestPrimitive(client),
		"CredentialBrokerService.RequestAccess":          requestAccessPrimitive(client),
		"CredentialBrokerService.CreateBrokerSession":    createSessionPrimitive(client),
		"CredentialBrokerService.ExecuteBrokerOperation": executePrimitive(client),
		"CredentialBrokerService.RevokeBrokerSession":    revokePrimitive(client),
	})
	if err != nil {
		panic(fmt.Sprintf("broker: load from manifest: %v", err))
	}
	return group
}

func waitAccessRequestPrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*credentials.WaitAccessRequestResponse, error) {
			timeout, err := durationFlag(ctx, "timeout-seconds")
			if err != nil {
				return nil, err
			}
			response, err := client.WaitAccessRequest(context.Background(), connect.NewRequest(&credentials.WaitAccessRequestRequest{
				RequestId: ctx.Flag("request-id"), TimeoutSeconds: timeout,
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("wait for broker access request", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.WaitAccessRequestResponse) cliapp.ListReport {
			request := response.GetRequest()
			if request == nil {
				return cliapp.ListReport{Summary: []string{"Access request wait timed out without a decision."}}
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Access request %s is %s.", request.GetId(), request.GetStatus())}}
		},
	)
}

func getAccessRequestPrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*credentials.AccessRequest, error) {
			response, err := client.GetAccessRequest(context.Background(), connect.NewRequest(&credentials.GetAccessRequestRequest{RequestId: ctx.Flag("request-id")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get broker access request", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.AccessRequest) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Access request %s is %s.", response.GetId(), response.GetStatus())}}
		},
	)
}

func requestAccessPrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*credentials.AccessRequest, error) {
			duration, err := durationFlag(ctx, "duration-seconds")
			if err != nil {
				return nil, err
			}
			response, err := client.RequestAccess(context.Background(), connect.NewRequest(&credentials.RequestAccessRequest{
				GrantId: ctx.Flag("grant-id"), ItemId: ctx.Flag("item-id"), Operation: ctx.Flag("operation"), DurationSeconds: duration,
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("request broker access", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.AccessRequest) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Access request %s is %s.", response.GetId(), response.GetStatus())}, Changes: []string{"No credential value is returned."}}
		},
	)
}

func createSessionPrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*credentials.BrokerSession, error) {
			duration, err := durationFlag(ctx, "duration-seconds")
			if err != nil {
				return nil, err
			}
			response, err := client.CreateBrokerSession(context.Background(), connect.NewRequest(&credentials.CreateBrokerSessionRequest{
				GrantId: ctx.Flag("grant-id"), ItemId: ctx.Flag("item-id"), TargetOrigin: ctx.Flag("target-origin"), AllowInternal: ctx.BoolFlag("allow-internal"), DurationSeconds: duration,
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("create broker session", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.BrokerSession) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Broker session %s created for %s.", response.GetSessionId(), response.GetTargetOrigin())}, Changes: []string{"Use --json when a machine needs the returned capability; human output omits it."}}
		},
	)
}

func executePrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*credentials.BrokerOperationResult, error) {
			headers, err := parseHeaders(ctx.Flag("headers"))
			if err != nil {
				return nil, err
			}
			response, err := client.ExecuteBrokerOperation(context.Background(), connect.NewRequest(&credentials.ExecuteBrokerOperationRequest{
				SessionId: ctx.Flag("session-id"), SessionToken: ctx.Flag("session-token"), Method: ctx.Flag("method"), Path: ctx.Flag("path"), Headers: headers, Body: ctx.Flag("body"),
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("execute broker operation", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.BrokerOperationResult) cliapp.MutationReport {
			result := fmt.Sprintf("Broker operation completed with status %d.", response.GetStatus())
			if response.GetBodyTruncated() {
				result += " The response body was truncated."
			}
			return cliapp.MutationReport{Result: []string{result}, Changes: []string{"The broker projection controls exposed fields."}}
		},
	)
}

func revokePrimitive(client credentialsconnect.CredentialBrokerServiceClient) cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*credentials.RevokeBrokerSessionResponse, error) {
			response, err := client.RevokeBrokerSession(context.Background(), connect.NewRequest(&credentials.RevokeBrokerSessionRequest{SessionId: ctx.Flag("session-id")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("revoke broker session", err, nil)
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *credentials.RevokeBrokerSessionResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Broker session %s is %s.", response.GetSessionId(), response.GetStatus())}}
		},
	)
}

func durationFlag(ctx cliapp.OperationContext, name string) (int32, error) {
	value := strings.TrimSpace(ctx.Flag(name))
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return int32(parsed), nil
}

func parseHeaders(value string) (map[string]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(value), &headers); err != nil {
		return nil, fmt.Errorf("--headers must be a JSON object: %w", err)
	}
	return headers, nil
}
