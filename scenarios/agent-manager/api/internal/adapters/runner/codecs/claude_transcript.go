// Responsibility: retain claude declarations within their original package.
package codecs

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"time"
)

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
func (c *Claude) NewTranscriptParser() runner.TranscriptParser {
	return &claudeTranscriptParser{state: &claudeState{}, pricing: c.pricingService}
}

func (p *claudeTranscriptParser) SetTranscriptModel(model string) {
	p.state.model = strings.TrimSpace(model)
	if p.state.model == "" {
		p.state.model = "unknown"
	}
}

func (p *claudeTranscriptParser) SetTranscriptBilling(billing domain.BillingSnapshot) {
	p.state.billing = billing
}

// SetStateBilling satisfies [runner.BillingStateSetter] for the live decode
// path so a subscription run never emits a metered charge.
func (s *claudeState) SetStateBilling(billing domain.BillingSnapshot) { s.billing = billing }

type claudeTranscriptParser struct {
	state *claudeState
	// pricing computes the interactive charge from the reconciliation-authority
	// usage when the on-disk transcript has no cost line. Optional.
	pricing PricingService
	// onDisk latches once a camelCase `sessionId` field is seen, marking
	// this replay as claude's on-disk interactive transcript rather than
	// the --print stdout stream. In on-disk mode the parser synthesizes
	// the terminal marker from assistant stop_reason=end_turn (there is no
	// `result` line on disk, design §3 / R2). The stdout dialect never
	// carries sessionId, so this never trips during pipe-mode replay.
	onDisk bool
	// openTurnUsage is the latest on-disk assistant response usage since the
	// last turn boundary; requestPending records that a user line (a tool
	// result or a prompt) followed it, so a new request may have been sent.
	openTurnUsage  []*domain.UsageEventData
	requestPending bool
}

func (p *claudeTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := parseClaudeStreamEvents(p.state, runID, line)
	result := runner.TranscriptParseResult{
		Events:    events,
		Timestamp: transcriptLineTimestamp(line),
		Err:       err,
	}

	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &streamEvent); err == nil {
		result.SessionID = streamEvent.SessionID
		if streamEvent.SessionIDAlt != "" {
			result.SessionID = streamEvent.SessionIDAlt
			p.onDisk = true
		}
		if streamEvent.Type == "cost-state" && p.onDisk {
			// A retained cost snapshot is accounting, not an interactive turn
			// completion signal. Only explicit, internally consistent no-API
			// evidence establishes zero; absent fields and unknown models do not.
			var snapshot struct {
				Cost              *float64                   `json:"totalCostUSD"`
				API               *int64                     `json:"totalAPIDuration"`
				APIWithoutRetries *int64                     `json:"totalAPIDurationWithoutRetries"`
				Models            map[string]json.RawMessage `json:"modelUsage"`
				Unknown           *bool                      `json:"hasUnknownModelCost"`
			}
			if json.Unmarshal([]byte(line), &snapshot) == nil && snapshot.Cost != nil && *snapshot.Cost == 0 && snapshot.API != nil && *snapshot.API == 0 && snapshot.APIWithoutRetries != nil && *snapshot.APIWithoutRetries == 0 && snapshot.Models != nil && len(snapshot.Models) == 0 && snapshot.Unknown != nil && !*snapshot.Unknown && p.state.turn == 0 {
				charge := nativeChargeEvent(runID, domain.RunnerTypeClaudeCode, p.state.model, p.state.billing, 0).Data.(*domain.ChargeEventData)
				result.Events = append(result.Events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: result.Timestamp, Data: &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, RunnerType: string(domain.RunnerTypeClaudeCode), ReconciliationAuthority: true, Charge: charge}})
			}
		}
		if streamEvent.Type == "ai-title" && strings.TrimSpace(streamEvent.AiTitle) != "" {
			result.Label = strings.TrimSpace(streamEvent.AiTitle)
			result.LabelSource = domain.RunLabelSourceHarness
		}
		// On-disk interactive dialect: no `result` line ever arrives, so
		// the terminal marker is the final assistant turn whose
		// stop_reason is end_turn. stop_reason=tool_use means the turn is
		// mid-work and is NOT terminal. Phase 4 layers an idle debounce on
		// top of this marker (interactive sessions stay open awaiting input).
		if p.onDisk && strings.EqualFold(streamEvent.Type, "assistant") &&
			streamEvent.Message != nil && streamEvent.Message.StopReason == "end_turn" {
			var receipts []*domain.UsageEventData
			for _, event := range result.Events {
				usage, ok := event.Data.(*domain.UsageEventData)
				if !ok {
					continue
				}
				// The final on-disk assistant usage is the best available
				// terminal snapshot for interactive Claude. It remains a
				// token receipt even when pricing is unavailable.
				usage.ReconciliationAuthority = true
				receipts = append(receipts, usage)
			}
			result.Events = append(result.Events, p.receiptCharges(runID, receipts)...)
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0, TerminalReason: "turn_boundary"}
		}
		if strings.EqualFold(streamEvent.Type, "result") {
			terminal := &runner.TranscriptTerminal{
				Success:  !streamEvent.IsError,
				ExitCode: 0,
			}
			if streamEvent.IsError {
				terminal.Success = false
				terminal.ExitCode = 1
				terminal.ErrorMessage = strings.TrimSpace(decodeClaudeResultString(streamEvent.Result))
				// Use the same rate-limit detector as the live stream path
				// (DecodeStreamLine → parseClaudeResultEvent →
				// detectClaudeRateLimit). Previously the transcript-replay
				// code had a narrower string check that diverged from the
				// live path; unifying them ensures a recovered run sees
				// the same exit code as the live run.
				if rl := detectClaudeRateLimit(terminal.ErrorMessage); rl.Detected {
					terminal.ExitCode = 429
				}
				if terminal.ErrorMessage == "" {
					terminal.ErrorMessage = "runner reported terminal error"
				}
			}
			result.Terminal = terminal
		}
		if marker, ok := claudeGoalStatus(line); ok {
			result.Goal = &marker
		}
		p.trackOpenTurn(streamEvent.Type, result)
	}
	return result
}

// receiptCharges prices reconciliation-authority usage. The on-disk dialect has
// no cost line: when a pricing lookup is wired, compute the charge from the
// receipt usage; otherwise the charge stays unknown and the allowance must
// remain reserved. The first event buildCostEvents returns duplicates the
// usage, so only its charge events are kept.
func (p *claudeTranscriptParser) receiptCharges(runID uuid.UUID, receipts []*domain.UsageEventData) []*domain.RunEvent {
	if p.pricing == nil {
		return nil
	}
	var charges []*domain.RunEvent
	for _, usage := range receipts {
		tokens := usageTokens{
			InputTokens:         usage.InputTokens,
			OutputTokens:        usage.OutputTokens,
			CacheReadTokens:     usage.CacheReadTokens,
			CacheCreationTokens: usage.CacheCreationTokens,
		}
		built := buildCostEvents(runID, domain.RunnerTypeClaudeCode, p.pricing, p.state.model, tokens, p.state.billing)
		if len(built) > 1 {
			charges = append(charges, built[1:]...)
		}
	}
	return charges
}

// trackOpenTurn remembers the latest on-disk assistant usage since the last
// turn boundary and whether a user line followed it. Claude writes a response's
// usage only once the response completes, and writes the tool result or prompt
// before it sends the next request, so these two facts decide whether a request
// could have been outstanding when the transcript ends.
func (p *claudeTranscriptParser) trackOpenTurn(lineType string, result runner.TranscriptParseResult) {
	if !p.onDisk {
		return
	}
	switch {
	case result.Terminal != nil:
		p.openTurnUsage, p.requestPending = nil, false
	case strings.EqualFold(lineType, "assistant"):
		var usage []*domain.UsageEventData
		for _, event := range result.Events {
			if data, ok := event.Data.(*domain.UsageEventData); ok {
				usage = append(usage, data)
			}
		}
		if len(usage) > 0 {
			p.openTurnUsage, p.requestPending = usage, false
		}
	case strings.EqualFold(lineType, "user"):
		p.requestPending = true
	}
}

// FinalizeInterruptedTurn satisfies [runner.TranscriptInterruptedTurnFinalizer].
// A transcript that ends on a completed assistant response, with no tool result
// or prompt after it, had no request outstanding: that response's usage is the
// turn's final snapshot and is promoted exactly as end_turn would promote it.
func (p *claudeTranscriptParser) FinalizeInterruptedTurn(runID uuid.UUID, at time.Time) ([]*domain.RunEvent, bool) {
	if !p.onDisk || p.requestPending || len(p.openTurnUsage) == 0 {
		return nil, false
	}
	receipts := make([]*domain.UsageEventData, 0, len(p.openTurnUsage))
	events := make([]*domain.RunEvent, 0, len(p.openTurnUsage))
	for _, usage := range p.openTurnUsage {
		receipt := *usage
		receipt.ReconciliationAuthority = true
		receipt.ReconciliationSource = domain.ReconciliationSourceInterruptedTurn
		receipts = append(receipts, &receipt)
		events = append(events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: at, Data: &receipt})
	}
	for _, charge := range p.receiptCharges(runID, receipts) {
		charge.Timestamp = at
		events = append(events, charge)
	}
	return events, true
}

// =============================================================================
// Stream parser
// =============================================================================
