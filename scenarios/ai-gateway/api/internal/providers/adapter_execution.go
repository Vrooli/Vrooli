package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func (a Adapter) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if a.Door != nil {
		return a.Door.Run(ctx, req)
	}
	if a.HTTP != nil {
		return a.HTTP.Execute(ctx, req)
	}
	if a.Provider == ProviderMetered {
		return a.executeMetered(ctx, req)
	}
	command, err := a.executionCommand(req)
	if err != nil {
		return ExecutionResult{}, err
	}
	result, err := a.runner().Run(ctx, command)
	if err != nil {
		return ExecutionResult{ExitCode: result.ExitCode}, mapCommandError(a.Provider, err)
	}
	output := strings.TrimSpace(result.Stdout)
	var spendAuthorization *inferencev1.SpendAuthorization
	if a.Provider == ProviderOpenRouter && req.SpendLimit != nil {
		output, spendAuthorization, err = cleanOpenRouterResponse(output)
		if err != nil {
			return ExecutionResult{}, err
		}
	}
	return ExecutionResult{OutputText: output, ExitCode: result.ExitCode, SpendAuthorization: spendAuthorization, Timing: reportedProviderTiming(output)}, nil
}

// reportedProviderTiming reads the timing a resource command reports beside
// its answer (`admission_wait_ns`, `provider_duration_ns`). Output that is not
// a JSON object, or that reports nothing, yields zero timing.
func reportedProviderTiming(output string) ProviderTiming {
	if !strings.HasPrefix(output, "{") {
		return ProviderTiming{}
	}
	var reported struct {
		AdmissionWaitNS    int64 `json:"admission_wait_ns"`
		ProviderDurationNS int64 `json:"provider_duration_ns"`
	}
	if err := json.Unmarshal([]byte(output), &reported); err != nil {
		return ProviderTiming{}
	}
	return ProviderTiming{AdmissionWaitNS: max(reported.AdmissionWaitNS, 0), RuntimeNS: max(reported.ProviderDurationNS, 0)}
}

func (a Adapter) executeMetered(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if a.Metered == nil {
		return ExecutionResult{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
	}
	images := make([]MeteredImage, 0, len(req.Attachments))
	for _, attachment := range req.Attachments {
		if attachment == nil || attachment.GetInlineBytes() == nil {
			return ExecutionResult{}, errors.New("metered route requires inline image references")
		}
		images = append(images, MeteredImage{Data: base64.StdEncoding.EncodeToString(attachment.GetInlineBytes()), MediaType: attachment.GetMediaType(), SHA256: digestBytes(attachment.GetInlineBytes())})
	}
	var browserSession *MeteredBrowserSession
	if req.BrowserSession != nil {
		browserSession = &MeteredBrowserSession{ReservationID: req.BrowserSession.GetReservationId(), RunID: req.BrowserSession.GetRunId(), SessionID: req.BrowserSession.GetSessionId(), GrantReceipt: req.BrowserSession.GetGrantReceipt()}
	}
	result, err := a.Metered.Run(ctx, MeteredRequest{
		Role: req.Role, Messages: []MeteredMessage{{Role: "user", Content: req.InputText}},
		ConstraintsJSON: req.SchemaJSON, MaxTokens: int(req.MaxOutputTokens),
		Profile: req.Profile, SpendLimit: req.SpendLimit, Images: images,
		RootOperationID: req.RootOperationID,
		Metadata:        map[string]string{"gateway_root_operation_id": req.RootOperationID},
		BrowserSession:  browserSession,
	})
	if err != nil {
		return ExecutionResult{}, mapCommandError(a.Provider, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("encode metered inference result: %w", err)
	}
	return ExecutionResult{OutputText: string(encoded), SpendAuthorization: result.SpendAuthorization, BillingOperationID: result.BillingOperationID}, nil
}

func cleanOpenRouterResponse(output string) (string, *inferencev1.SpendAuthorization, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		return "", nil, fmt.Errorf("decode OpenRouter capped response: %w", err)
	}
	authorizationJSON := envelope["vrooli_spend_authorization"]
	if len(authorizationJSON) == 0 {
		return "", nil, errors.New("OpenRouter owner omitted spend authorization evidence")
	}
	spendAuthorization := &inferencev1.SpendAuthorization{}
	if err := json.Unmarshal(authorizationJSON, spendAuthorization); err != nil {
		return "", nil, fmt.Errorf("decode OpenRouter spend authorization: %w", err)
	}
	delete(envelope, "vrooli_spend_authorization")
	clean, err := json.Marshal(envelope)
	if err != nil {
		return "", nil, err
	}
	return string(clean), spendAuthorization, nil
}
