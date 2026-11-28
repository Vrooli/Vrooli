package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
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

	mu            sync.Mutex
	lastGoodTeams map[string]map[string]string
}

// CheckResult proves the typed transmitter contract itself instead of using
// process health as a proxy for the team-instrument feature.
func (p *promptManagerInstrumentProvider) CheckResult(ctx context.Context) capreg.CheckResult {
	teams, err := p.fetchDeclarations(ctx)
	if err != nil {
		reason := "feature_transmitter_unavailable"
		if strings.Contains(err.Error(), "Prompt Manager address is unavailable") {
			reason = "upstream_unavailable"
		}
		return capreg.CheckResult{
			Status:          capreg.StatusUnavailable,
			Message:         err.Error(),
			ReasonCode:      reason,
			ActionKind:      capreg.ActionKindOwnerGuidance,
			ActionLabel:     "Review Prompt Manager",
			OperatorCommand: "vrooli scenario status prompt-manager --json",
			FeatureStatus:   map[string]string{"team_instrument": "unknown"},
		}
	}
	p.rememberDeclarations(teams)
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
	current, err := p.fetchDeclarations(ctx)
	if err == nil {
		p.rememberDeclarations(current)
		return current
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	stale := cloneDeclarations(p.lastGoodTeams)
	for _, declaration := range stale {
		// Keep the owner-authored team and archetype visible, but do not present
		// a status from before the outage as if it were current.
		declaration["status"] = "unknown"
	}
	return stale
}

func (p *promptManagerInstrumentProvider) fetchDeclarations(ctx context.Context) (map[string]map[string]string, error) {
	var response *connect.Response[memberflowv1.JsonResponse]
	err := retryPromptManager(ctx, p.resolve, func(base string) error {
		var callErr error
		response, callErr = memberflowconnect.NewMemberflowServiceClient(p.http, base).GetInstruments(ctx, connect.NewRequest(&memberflowv1.EmptyRequest{}))
		return callErr
	})
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg.GetData() == nil {
		return nil, errors.New("Prompt Manager team-instrument response has no data")
	}
	data, ok := response.Msg.GetData().AsInterface().(map[string]interface{})
	if !ok {
		return nil, errors.New("Prompt Manager team-instrument response is malformed")
	}
	rows, ok := data["teams"].([]interface{})
	if !ok {
		return nil, errors.New("Prompt Manager team-instrument response has no teams list")
	}
	declarations := make(map[string]map[string]string, len(rows))
	for _, raw := range rows {
		row, ok := raw.(map[string]interface{})
		if !ok {
			return nil, errors.New("Prompt Manager team-instrument response has a malformed team")
		}
		teamID, _ := row["teamId"].(string)
		instrument, ok := row["instrument"].(map[string]interface{})
		if strings.TrimSpace(teamID) == "" || !ok {
			return nil, errors.New("Prompt Manager team-instrument response has an incomplete team")
		}
		status, statusOK := instrument["status"].(string)
		archetype, archetypeOK := instrument["archetype"].(string)
		if !statusOK || !archetypeOK {
			return nil, errors.New("Prompt Manager team-instrument response has an incomplete instrument")
		}
		declarations[teamID] = map[string]string{"status": status, "archetype": archetype}
	}
	return declarations, nil
}

func (p *promptManagerInstrumentProvider) rememberDeclarations(declarations map[string]map[string]string) {
	p.mu.Lock()
	p.lastGoodTeams = cloneDeclarations(declarations)
	p.mu.Unlock()
}

func cloneDeclarations(source map[string]map[string]string) map[string]map[string]string {
	copy := make(map[string]map[string]string, len(source))
	for team, declaration := range source {
		fields := make(map[string]string, len(declaration))
		for key, value := range declaration {
			fields[key] = value
		}
		copy[team] = fields
	}
	return copy
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
