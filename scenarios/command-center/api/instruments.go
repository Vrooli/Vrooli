package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	memberflowv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"
)

// InstrumentProvider is the typed transmitter boundary for Prompt Manager's
// team instrument declarations. Command Center does not read Prompt Manager's
// store files directly; unavailable data remains an honest empty projection.
type InstrumentProvider interface {
	Declarations(context.Context) map[string]map[string]string
}

type ObjectiveProvider interface {
	ObjectivesAvailable(context.Context) bool
}

type promptManagerInstrumentProvider struct {
	resolve func() string
	http    *http.Client
}

// CheckResult proves the typed transmitter contract itself instead of using
// process health as a proxy for the team-instrument feature.
func (p *promptManagerInstrumentProvider) CheckResult(ctx context.Context) capreg.CheckResult {
	base := strings.TrimRight(p.resolve(), "/")
	if base == "" {
		return capreg.CheckResult{
			Status:          capreg.StatusUnavailable,
			Message:         "Prompt Manager address is unavailable",
			ReasonCode:      "upstream_unavailable",
			ActionKind:      capreg.ActionKindOwnerGuidance,
			ActionLabel:     "Review Prompt Manager",
			OperatorCommand: "vrooli scenario status prompt-manager --json",
			FeatureStatus:   map[string]string{"team_instrument": "unknown"},
		}
	}
	var response *connect.Response[memberflowv1.JsonResponse]
	err := retryPromptManager(ctx, p.resolve, func(base string) error {
		var callErr error
		response, callErr = memberflowconnect.NewMemberflowServiceClient(p.http, base).GetInstruments(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
		return callErr
	})
	if err != nil || response.Msg.GetData() == nil {
		message := "Prompt Manager team-instrument transmitter is unavailable"
		if err != nil {
			message = err.Error()
		}
		return capreg.CheckResult{
			Status:          capreg.StatusUnavailable,
			Message:         message,
			ReasonCode:      "feature_transmitter_unavailable",
			ActionKind:      capreg.ActionKindOwnerGuidance,
			ActionLabel:     "Review Prompt Manager",
			OperatorCommand: "vrooli scenario status prompt-manager --json",
			FeatureStatus:   map[string]string{"team_instrument": "unknown"},
		}
	}
	return capreg.CheckResult{
		Status:        capreg.StatusAvailable,
		Message:       "typed team-instrument transmitter reachable",
		ReasonCode:    "typed_transmitter_reachable",
		FeatureStatus: map[string]string{"team_instrument": "compatible"},
		FeatureReason: map[string]string{"team_instrument": "Prompt Manager returned its typed instrument projection"},
	}
}

func (p *promptManagerInstrumentProvider) Check(ctx context.Context) (capreg.Status, string) {
	result := p.CheckResult(ctx)
	return result.Status, result.Message
}

func newPromptManagerInstrumentProvider(resolve func() string) *promptManagerInstrumentProvider {
	return &promptManagerInstrumentProvider{resolve: resolve, http: &http.Client{Timeout: 5 * time.Second}}
}

func (p *promptManagerInstrumentProvider) Declarations(ctx context.Context) map[string]map[string]string {
	out := map[string]map[string]string{}
	base := strings.TrimRight(p.resolve(), "/")
	if base == "" {
		return out
	}
	var response *connect.Response[memberflowv1.JsonResponse]
	err := retryPromptManager(ctx, p.resolve, func(base string) error {
		var callErr error
		response, callErr = memberflowconnect.NewMemberflowServiceClient(p.http, base).GetInstruments(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
		return callErr
	})
	if err != nil || response.Msg.GetData() == nil {
		return out
	}
	data, ok := response.Msg.GetData().AsInterface().(map[string]interface{})
	if !ok {
		return out
	}
	rows, ok := data["teams"].([]interface{})
	if !ok {
		return out
	}
	for _, raw := range rows {
		row, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		teamID, _ := row["teamId"].(string)
		instrument, _ := row["instrument"].(map[string]interface{})
		if strings.TrimSpace(teamID) == "" || instrument == nil {
			continue
		}
		status, _ := instrument["status"].(string)
		archetype, _ := instrument["archetype"].(string)
		out[teamID] = map[string]string{"status": status, "archetype": archetype}
	}
	return out
}

func (p *promptManagerInstrumentProvider) ObjectivesAvailable(ctx context.Context) bool {
	base := strings.TrimRight(p.resolve(), "/")
	if base == "" {
		return false
	}
	var response *connect.Response[memberflowv1.JsonResponse]
	err := retryPromptManager(ctx, p.resolve, func(base string) error {
		var callErr error
		response, callErr = memberflowconnect.NewMemberflowServiceClient(p.http, base).GetObjectives(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
		return callErr
	})
	return err == nil && response.Msg.GetData() != nil
}

// retryPromptManager retries only transport-like Connect failures. Each
// attempt resolves the address again so a lifecycle-managed port change is
// visible without restarting Command Center.
func retryPromptManager(ctx context.Context, resolve func() string, call func(string) error) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		base := strings.TrimRight(resolve(), "/")
		if base == "" {
			return connect.NewError(connect.CodeUnavailable, errors.New("Prompt Manager address is unavailable"))
		}
		lastErr = call(base)
		if lastErr == nil || !retryablePromptManagerError(ctx, lastErr) || attempt == 1 {
			return lastErr
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
	return lastErr
}

func retryablePromptManagerError(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeResourceExhausted:
		return true
	default:
		return false
	}
}
