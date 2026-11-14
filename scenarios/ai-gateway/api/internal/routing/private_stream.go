package routing

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

// SupplierEndpoint is deployment-owned connection data; request bodies cannot
// select endpoints or credentials. Policy records remain the eligibility owner.
type SupplierEndpoint struct {
	URL    string
	APIKey string
}
type HTTPStreamExecutor struct {
	DB              *sql.DB
	ResolveSupplier func(context.Context, string) (SupplierEndpoint, error)
	Suppliers       map[string]SupplierEndpoint
	Client          *http.Client
}

// ErrPrivateDispatchUncertain distinguishes a failure after a durable claim
// from a refusal that proves no supplier work occurred. Callers retain funding.
var ErrPrivateDispatchUncertain = errors.New("provider dispatch or receipt is uncertain")

type PrivateStreamRequest struct {
	OperationID string               `json:"operation_id"`
	Plan        PrivatePlan          `json:"plan"`
	Chat        json.RawMessage      `json:"chat,omitempty"`
	Feature     *PrivateFeatureInput `json:"feature,omitempty"`
}

func (s *PrivatePolicyService) SetStreamExecutor(executor *HTTPStreamExecutor) { s.stream = executor }

// Stream claims the operation durably before network dispatch. An interrupted
// dispatch is never automatically retried, including after process restart.
func (s *PrivatePolicyService) Stream(ctx context.Context, request PrivateStreamRequest, w http.ResponseWriter) (PrivateReceipt, error) {
	if s.stream == nil {
		return PrivateReceipt{}, errors.New("provider stream executor unavailable")
	}
	saved, err := s.plans.GetPlan(ctx, request.OperationID)
	if err != nil {
		return PrivateReceipt{}, err
	}
	if saved != request.Plan || saved.OperationID != request.OperationID {
		return PrivateReceipt{}, ErrPrivatePlanConflict
	}
	eligibilityRequest := PrivatePlanRequest{OperationID: saved.OperationID, FeatureID: saved.FeatureID, Model: saved.RequestedModel, BudgetMicros: saved.BudgetMicros, Harness: saved.Harness, CreditMultiplier: saved.CreditMultiplier}
	decision, err := s.eligible(ctx, eligibilityRequest)
	if err != nil {
		return PrivateReceipt{}, err
	}
	if !privateStreamRouteMatches(saved, decision.Routes) {
		return PrivateReceipt{}, ErrPrivatePlanNotEligible
	}
	if saved.CreditMultiplier <= 0 {
		return PrivateReceipt{}, errors.New("LPBS credit conversion is required")
	}
	feature := s.features[saved.FeatureID]
	request.Chat, err = s.streamChat(saved.FeatureID, saved.Model, request.Feature, request.Chat)
	if err != nil {
		return PrivateReceipt{}, err
	}

	var chat map[string]json.RawMessage
	if json.Unmarshal(request.Chat, &chat) != nil {
		return PrivateReceipt{}, errors.New("invalid chat request")
	}
	var model string
	if json.Unmarshal(chat["model"], &model) != nil || model != saved.Model {
		return PrivateReceipt{}, errors.New("chat model differs from pinned plan")
	}
	if len(request.Chat) > feature.MaxInputBytes {
		return PrivateReceipt{}, errors.New("chat exceeds input cap")
	}
	if err := boundStreamChat(chat, request.Chat, saved); err != nil {
		return PrivateReceipt{}, err
	}
	snapshot, err := s.snapshot(ctx, saved.FeatureID)
	if err != nil {
		return PrivateReceipt{}, err
	}
	// Authorize the very snapshot supplying downstream constraints. The earlier
	// read cannot lend authority to a reopened record or a different route/quote.
	decision, err = s.eligibleSnapshot(eligibilityRequest, snapshot)
	if err != nil {
		return PrivateReceipt{}, err
	}
	if !privateStreamRouteMatches(saved, decision.Routes) {
		return PrivateReceipt{}, ErrPrivatePlanNotEligible
	}
	downstreamFound := false
	for _, record := range snapshot.Records {
		if record.Supplier == saved.Supplier {
			if record.Terms.Downstream.DataCollection != "deny" {
				return PrivateReceipt{}, errors.New("private downstream policy unavailable")
			}
			// This public OpenAI-compatible envelope has no caller routing knobs.
			// Apply the authoritative downstream constraints to its upstream hop.
			provider := map[string]any{"data_collection": record.Terms.Downstream.DataCollection}
			if len(record.Terms.Downstream.PinnedHosts) > 0 {
				provider["only"] = record.Terms.Downstream.PinnedHosts
				provider["allow_fallbacks"] = false
			}
			chat["provider"], _ = json.Marshal(provider)
			downstreamFound = true
			break
		}
	}
	if !downstreamFound {
		return PrivateReceipt{}, ErrPrivatePlanNotEligible
	}
	chat["stream"] = json.RawMessage(`true`)
	chat["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	bounded, _ := json.Marshal(chat)
	if len(bounded) > saved.StreamQuote.MaxInputBytes || len(bounded) > feature.MaxInputBytes {
		return PrivateReceipt{}, errors.New("complete upstream input exceeds supplier quote")
	}
	endpoint, ok := s.stream.Suppliers[saved.Supplier]
	if s.stream.ResolveSupplier != nil {
		endpoint, err = s.stream.ResolveSupplier(ctx, saved.Supplier)
		if err != nil {
			return PrivateReceipt{}, err
		}
		ok = true
	}
	if !ok {
		return PrivateReceipt{}, errors.New("supplier endpoint unavailable")
	}
	fingerprint := sha256.Sum256(request.Chat)
	hash := hex.EncodeToString(fingerprint[:])
	receipt := PrivateReceipt{OperationID: request.OperationID, FeatureID: saved.FeatureID, State: "dispatching", RouteID: saved.RouteID, Supplier: saved.Supplier, Model: saved.Model, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	data, _ := json.Marshal(receipt)
	result, err := s.stream.DB.ExecContext(ctx, `INSERT INTO private_provider_receipts(operation_id,request_sha256,receipt_json,created_at) VALUES(?,?,?,?) ON CONFLICT(operation_id) DO NOTHING`, request.OperationID, hash, string(data), receipt.CreatedAt)
	if err != nil {
		return PrivateReceipt{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return PrivateReceipt{}, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	if count == 0 {
		var priorHash string
		if err := s.stream.DB.QueryRowContext(ctx, `SELECT request_sha256 FROM private_provider_receipts WHERE operation_id=?`, request.OperationID).Scan(&priorHash); err != nil {
			return PrivateReceipt{}, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
		}
		if priorHash != hash {
			return PrivateReceipt{}, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, ErrPrivateReceiptConflict)
		}
		prior, err := s.Receipt(ctx, request.OperationID)
		if err != nil {
			return PrivateReceipt{}, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
		}
		return prior, nil
	}
	// Cancellation stops delivery but cannot erase measured usage or its receipt.
	operation, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Minute)
	defer cancel()
	if request.Feature != nil {
		output := &featureOutput{header: http.Header{}}
		receipt = s.stream.execute(operation, endpoint, chat, receipt, saved.CreditMultiplier, feature.MaxOutputBytes, output)
		if receipt.State == "completed" {
			content, err := output.content(feature.Schema)
			if err != nil {
				receipt.State, receipt.ErrorCode = "unknown", "invalid_feature_output"
			} else {
				receipt.Content = content
			}
		}
	} else {
		receipt = s.stream.execute(operation, endpoint, chat, receipt, saved.CreditMultiplier, feature.MaxOutputBytes, w)
	}
	encoded, _ := json.Marshal(receipt)
	persist, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_, err = s.stream.DB.ExecContext(persist, `UPDATE private_provider_receipts SET receipt_json=? WHERE operation_id=? AND request_sha256=?`, string(encoded), receipt.OperationID, hash)
	if err != nil {
		return receipt, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	return receipt, nil
}

// privateStreamRouteMatches keeps both admission points bound to the exact
// persisted route and quote; an eligible fallback cannot substitute for it.
func privateStreamRouteMatches(plan PrivatePlan, routes []policy.Route) bool {
	if !plan.Eligible || len(routes) == 0 {
		return false
	}
	route := routes[0]
	return route.ID == plan.RouteID && route.Supplier == plan.Supplier && route.Model == plan.Model && route.QuotedCredits == plan.QuotedCredits && route.StreamQuote == plan.StreamQuote
}

// boundStreamChat admits only dimensions covered by one fixed supplier quote.
// Unsupported generation/routing parameters fail before the durable dispatch.
func boundStreamChat(chat map[string]json.RawMessage, raw []byte, plan PrivatePlan) error {
	quote := plan.StreamQuote
	if quote.TokenLimitField != "max_tokens" && quote.TokenLimitField != "max_completion_tokens" {
		return errors.New("supplier has no billable-output token limit")
	}
	if quote.MaxInputBytes <= 0 || quote.MaxOutputTokens <= 0 || quote.MaxCostMicros <= 0 || len(raw) > quote.MaxInputBytes || quote.MaxCostMicros > plan.BudgetMicros {
		return errors.New("request has no enforceable supplier quote")
	}
	// The entire envelope's upstream ceiling must fit the held customer charge.
	if quote.MaxCostMicros > (math.MaxInt64-9999)/plan.CreditMultiplier || (quote.MaxCostMicros*plan.CreditMultiplier+9999)/10000 > plan.QuotedCredits {
		return errors.New("supplier quote exceeds reserved credits")
	}
	allowed := map[string]bool{"model": true, "messages": true, "tools": true, "tool_choice": true, "parallel_tool_calls": true, "temperature": true, "top_p": true, "stop": true, "stream": true, "stream_options": true, "max_tokens": true, "max_completion_tokens": true, "n": true}
	for field := range chat {
		if !allowed[field] {
			return fmt.Errorf("unsupported quoted parameter: %s", field)
		}
	}
	if tools, ok := chat["tools"]; ok {
		var declarations []struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(tools, &declarations) != nil {
			return errors.New("invalid quoted tools")
		}
		for _, tool := range declarations {
			if tool.Type != "function" {
				return errors.New("built-in tools require a separate supplier quote")
			}
		}
	}
	if value, ok := chat["n"]; ok {
		var count int
		if json.Unmarshal(value, &count) != nil || count != 1 {
			return errors.New("supplier quote covers exactly one completion")
		}
	}
	limit := quote.MaxOutputTokens
	for _, field := range []string{"max_tokens", "max_completion_tokens"} {
		if value, ok := chat[field]; ok {
			var requested int
			if json.Unmarshal(value, &requested) != nil || requested <= 0 || requested > quote.MaxOutputTokens {
				return errors.New("requested tokens exceed supplier quote")
			}
			if requested < limit {
				limit = requested
			}
			delete(chat, field)
		}
	}
	if err := validateQuotedMessages(chat["messages"]); err != nil {
		return err
	}
	chat[quote.TokenLimitField], _ = json.Marshal(limit)
	chat["n"] = json.RawMessage(`1`)
	return nil
}

func validateQuotedMessages(raw json.RawMessage) error {
	// Text/tool JSON is sized by bytes. Remote media cannot be sized by its URL.
	var messages []map[string]json.RawMessage
	if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 {
		return errors.New("quoted messages required")
	}
	for _, message := range messages {
		for field := range message {
			switch field {
			case "role", "content", "name", "tool_calls", "tool_call_id", "refusal":
			default:
				return errors.New("unsupported quoted message field")
			}
		}
		content := message["content"]
		if len(content) == 0 || string(content) == "null" {
			continue
		}
		var text string
		if json.Unmarshal(content, &text) == nil {
			continue
		}
		var parts []map[string]json.RawMessage
		if json.Unmarshal(content, &parts) != nil {
			return errors.New("unsupported quoted content")
		}
		for _, part := range parts {
			var kind string
			if json.Unmarshal(part["type"], &kind) != nil || kind != "text" {
				return errors.New("media requires a separate supplier quote")
			}
			for field := range part {
				if field != "type" && field != "text" {
					return errors.New("unsupported quoted content field")
				}
			}
			if json.Unmarshal(part["text"], &text) != nil {
				return errors.New("invalid quoted text")
			}
		}
	}
	return nil
}

func (e *HTTPStreamExecutor) execute(ctx context.Context, endpoint SupplierEndpoint, chat map[string]json.RawMessage, receipt PrivateReceipt, multiplier int64, maxOutputBytes int, w http.ResponseWriter) PrivateReceipt {
	chat["stream"] = json.RawMessage(`true`)
	chat["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	// Internal markers and attribution are not provider request capabilities.
	for _, name := range []string{"outside_funded", "business_account_id", "use_class", "funded", "metadata"} {
		delete(chat, name)
	}
	body, _ := json.Marshal(chat)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint.URL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		receipt.State = "failed"
		receipt.Message = err.Error()
		return receipt
	}
	req.Header.Set("Content-Type", "application/json")
	if endpoint.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	}
	client := e.Client
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		receipt.State = "unknown"
		receipt.ErrorCode = "upstream_outcome_unknown"
		return receipt
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		receipt.State = "failed"
		receipt.CostKnown = true
		receipt.ErrorCode = "upstream_refused"
		receipt.Message = fmt.Sprintf("supplier HTTP %d", resp.StatusCode)
		return receipt
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Operation-ID", receipt.OperationID)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	delivered := true
	done := false
	outputBytes := 0
	for scanner.Scan() {
		line := scanner.Text()
		outputBytes += len(line) + 1
		if outputBytes > maxOutputBytes {
			break
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				done = true
				continue
			}
			var event struct {
				Usage *struct {
					PromptTokens     int      `json:"prompt_tokens"`
					CompletionTokens int      `json:"completion_tokens"`
					Cost             *float64 `json:"cost"`
				} `json:"usage"`
			}
			if json.Unmarshal([]byte(payload), &event) == nil && event.Usage != nil {
				receipt.PromptTokens = event.Usage.PromptTokens
				receipt.CompletionTokens = event.Usage.CompletionTokens
				if event.Usage.Cost != nil && *event.Usage.Cost >= 0 {
					receipt.CostMicros = int64(*event.Usage.Cost*1e6 + 0.5)
					if receipt.CostMicros >= 0 && receipt.CostMicros <= (math.MaxInt64-9999)/multiplier {
						receipt.Credits = (receipt.CostMicros*multiplier + 9999) / 10000
						receipt.CostKnown = true
					}
				}
			}
		}
		if delivered {
			_, err = io.WriteString(w, line+"\n")
			delivered = err == nil
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
	receipt.State = "completed"
	if scanner.Err() != nil || !done || !receipt.CostKnown {
		receipt.State = "unknown"
		receipt.ErrorCode = "upstream_outcome_unknown"
	}
	return receipt
}
