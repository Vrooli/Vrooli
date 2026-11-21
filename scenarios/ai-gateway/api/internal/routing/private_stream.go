package routing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
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
	OperationID       string               `json:"operation_id"`
	BusinessAccountID string               `json:"business_account_id"`
	ActorID           string               `json:"actor_id"`
	AppKey            string               `json:"app_key"`
	Plan              PrivatePlan          `json:"plan"`
	Chat              json.RawMessage      `json:"chat,omitempty"`
	Feature           *PrivateFeatureInput `json:"feature,omitempty"`
}

func (s *PrivatePolicyService) SetStreamExecutor(executor *HTTPStreamExecutor) { s.stream = executor }

// Stream claims the operation durably before network dispatch. An interrupted
// dispatch is never automatically retried, including after process restart.
func (s *PrivatePolicyService) Stream(ctx context.Context, request PrivateStreamRequest, w http.ResponseWriter) (PrivateReceipt, error) {
	prepared, err := s.preparePrivateStream(ctx, request)
	if err != nil {
		return PrivateReceipt{}, err
	}
	return s.claimAndDispatchPrivateStream(ctx, prepared, w)
}

type preparedPrivateStream struct {
	request  PrivateStreamRequest
	saved    PrivatePlan
	feature  PrivateFeature
	chat     map[string]json.RawMessage
	endpoint SupplierEndpoint
	hash     string
	receipt  PrivateReceipt
}

func (s *PrivatePolicyService) preparePrivateStream(ctx context.Context, request PrivateStreamRequest) (preparedPrivateStream, error) {
	if s.stream == nil {
		return preparedPrivateStream{}, errors.New("provider stream executor unavailable")
	}
	saved, err := s.loadEligiblePrivatePlan(ctx, request)
	if err != nil {
		return preparedPrivateStream{}, err
	}
	feature := s.features[saved.FeatureID]
	chat, raw, err := s.preparePrivateChat(ctx, request, saved, feature)
	if err != nil {
		return preparedPrivateStream{}, err
	}
	request.Chat = raw
	endpoint, err := s.resolvePrivateEndpoint(ctx, saved.Supplier)
	if err != nil {
		return preparedPrivateStream{}, err
	}
	fingerprint := sha256.Sum256(request.Chat)
	hash := hex.EncodeToString(fingerprint[:])
	receipt := PrivateReceipt{OperationID: request.OperationID, FeatureID: saved.FeatureID, State: "dispatching", RouteID: saved.RouteID, Supplier: saved.Supplier, Model: saved.Model, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	return preparedPrivateStream{request: request, saved: saved, feature: feature, chat: chat, endpoint: endpoint, hash: hash, receipt: receipt}, nil
}

func (s *PrivatePolicyService) loadEligiblePrivatePlan(ctx context.Context, request PrivateStreamRequest) (PrivatePlan, error) {
	saved, err := s.plans.GetPlan(ctx, request.OperationID)
	if err != nil {
		return PrivatePlan{}, err
	}
	if saved != request.Plan || saved.OperationID != request.OperationID || saved.BusinessAccountID != request.BusinessAccountID || saved.ActorID != request.ActorID || saved.AppKey != request.AppKey {
		return PrivatePlan{}, ErrPrivatePlanConflict
	}
	eligibilityRequest := PrivatePlanRequest{OperationID: saved.OperationID, FeatureID: saved.FeatureID, Model: saved.RequestedModel, BudgetMicros: saved.BudgetMicros, Harness: saved.Harness, CreditMultiplier: saved.CreditMultiplier}
	decision, err := s.eligible(ctx, eligibilityRequest)
	if err != nil {
		return PrivatePlan{}, err
	}
	if !privateStreamRouteMatches(saved, decision.Routes) {
		return PrivatePlan{}, ErrPrivatePlanNotEligible
	}
	if saved.CreditMultiplier <= 0 {
		return PrivatePlan{}, errors.New("LPBS credit conversion is required")
	}
	return saved, nil
}

func (s *PrivatePolicyService) preparePrivateChat(ctx context.Context, request PrivateStreamRequest, saved PrivatePlan, feature PrivateFeature) (map[string]json.RawMessage, []byte, error) {
	raw, err := s.streamChat(saved.FeatureID, saved.Model, request.Feature, request.Chat)
	if err != nil {
		return nil, nil, err
	}
	var chat map[string]json.RawMessage
	if json.Unmarshal(raw, &chat) != nil {
		return nil, nil, errors.New("invalid chat request")
	}
	var model string
	if json.Unmarshal(chat["model"], &model) != nil || model != saved.Model {
		return nil, nil, errors.New("chat model differs from pinned plan")
	}
	if len(raw) > feature.MaxInputBytes {
		return nil, nil, errors.New("chat exceeds input cap")
	}
	if err := boundStreamChat(chat, raw, saved); err != nil {
		return nil, nil, err
	}
	if err := s.applyCurrentPrivateDownstream(ctx, saved, feature, chat); err != nil {
		return nil, nil, err
	}
	return chat, raw, nil
}

func (s *PrivatePolicyService) applyCurrentPrivateDownstream(ctx context.Context, saved PrivatePlan, feature PrivateFeature, chat map[string]json.RawMessage) error {
	snapshot, err := s.snapshot(ctx, saved.FeatureID)
	if err != nil {
		return err
	}
	eligibilityRequest := PrivatePlanRequest{OperationID: saved.OperationID, FeatureID: saved.FeatureID, Model: saved.RequestedModel, BudgetMicros: saved.BudgetMicros, Harness: saved.Harness, CreditMultiplier: saved.CreditMultiplier}
	decision, err := s.eligibleSnapshot(eligibilityRequest, snapshot)
	if err != nil {
		return err
	}
	if !privateStreamRouteMatches(saved, decision.Routes) {
		return ErrPrivatePlanNotEligible
	}
	if err := setPrivateDownstreamProvider(chat, snapshot, saved.Supplier); err != nil {
		return err
	}
	chat["stream"] = json.RawMessage(`true`)
	chat["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	bounded, _ := json.Marshal(chat)
	if len(bounded) > saved.StreamQuote.MaxInputBytes || len(bounded) > feature.MaxInputBytes {
		return errors.New("complete upstream input exceeds supplier quote")
	}
	return nil
}

func setPrivateDownstreamProvider(chat map[string]json.RawMessage, snapshot PrivatePolicySnapshot, supplier string) error {
	for _, record := range snapshot.Records {
		if record.Supplier != supplier {
			continue
		}
		if record.Terms.Downstream.DataCollection != "deny" {
			return errors.New("private downstream policy unavailable")
		}
		provider := map[string]any{"data_collection": record.Terms.Downstream.DataCollection}
		if len(record.Terms.Downstream.PinnedHosts) > 0 {
			provider["only"] = record.Terms.Downstream.PinnedHosts
			provider["allow_fallbacks"] = false
		}
		chat["provider"], _ = json.Marshal(provider)
		return nil
	}
	return ErrPrivatePlanNotEligible
}

func (s *PrivatePolicyService) resolvePrivateEndpoint(ctx context.Context, supplier string) (SupplierEndpoint, error) {
	endpoint, ok := s.stream.Suppliers[supplier]
	if s.stream.ResolveSupplier != nil {
		resolved, err := s.stream.ResolveSupplier(ctx, supplier)
		if err != nil {
			return SupplierEndpoint{}, err
		}
		endpoint, ok = resolved, true
	}
	if !ok {
		return SupplierEndpoint{}, errors.New("supplier endpoint unavailable")
	}
	return endpoint, nil
}

func (s *PrivatePolicyService) claimAndDispatchPrivateStream(ctx context.Context, prepared preparedPrivateStream, w http.ResponseWriter) (PrivateReceipt, error) {
	receipt, duplicate, err := s.claimPrivateStream(ctx, prepared)
	if err != nil {
		return PrivateReceipt{}, err
	}
	if duplicate {
		return receipt, nil
	}
	receipt = s.dispatchPrivateStream(ctx, prepared, receipt, w)
	if err := s.persistPrivateReceipt(prepared, receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func (s *PrivatePolicyService) claimPrivateStream(ctx context.Context, prepared preparedPrivateStream) (PrivateReceipt, bool, error) {
	data, _ := json.Marshal(prepared.receipt)
	result, err := s.stream.DB.ExecContext(ctx, `INSERT INTO private_provider_receipts(operation_id,request_sha256,receipt_json,created_at) VALUES(?,?,?,?) ON CONFLICT(operation_id) DO NOTHING`, prepared.request.OperationID, prepared.hash, string(data), prepared.receipt.CreatedAt)
	if err != nil {
		return PrivateReceipt{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return PrivateReceipt{}, false, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	if count == 1 {
		return prepared.receipt, false, nil
	}
	return s.loadDuplicatePrivateReceipt(ctx, prepared)
}

func (s *PrivatePolicyService) loadDuplicatePrivateReceipt(ctx context.Context, prepared preparedPrivateStream) (PrivateReceipt, bool, error) {
	var priorHash string
	if err := s.stream.DB.QueryRowContext(ctx, `SELECT request_sha256 FROM private_provider_receipts WHERE operation_id=?`, prepared.request.OperationID).Scan(&priorHash); err != nil {
		return PrivateReceipt{}, false, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	if priorHash != prepared.hash {
		return PrivateReceipt{}, false, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, ErrPrivateReceiptConflict)
	}
	prior, err := s.Receipt(ctx, prepared.request.OperationID)
	if err != nil {
		return PrivateReceipt{}, false, fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	return prior, true, nil
}

func (s *PrivatePolicyService) dispatchPrivateStream(ctx context.Context, prepared preparedPrivateStream, receipt PrivateReceipt, w http.ResponseWriter) PrivateReceipt {
	operation, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Minute)
	defer cancel()
	if prepared.request.Feature == nil {
		return s.stream.execute(operation, prepared.endpoint, prepared.chat, receipt, prepared.saved.CreditMultiplier, prepared.feature.MaxOutputBytes, w)
	}
	output := &featureOutput{header: http.Header{}}
	receipt = s.stream.execute(operation, prepared.endpoint, prepared.chat, receipt, prepared.saved.CreditMultiplier, prepared.feature.MaxOutputBytes, output)
	if receipt.State == "completed" {
		content, err := output.content(prepared.feature.Schema)
		if err != nil {
			receipt.State, receipt.ErrorCode = "unknown", "invalid_feature_output"
		} else {
			receipt.Content = content
		}
	}
	return receipt
}

func (s *PrivatePolicyService) persistPrivateReceipt(prepared preparedPrivateStream, receipt PrivateReceipt) error {
	encoded, _ := json.Marshal(receipt)
	persist, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_, err := s.stream.DB.ExecContext(persist, `UPDATE private_provider_receipts SET receipt_json=? WHERE operation_id=? AND request_sha256=?`, string(encoded), receipt.OperationID, prepared.hash)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPrivateDispatchUncertain, err)
	}
	return nil
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
	if err := validateStreamQuote(quote, raw, plan); err != nil {
		return err
	}
	if err := validateStreamFields(chat); err != nil {
		return err
	}
	limit, err := quotedTokenLimit(chat, quote.MaxOutputTokens)
	if err != nil {
		return err
	}
	if err := validateQuotedMessages(chat["messages"]); err != nil {
		return err
	}
	chat[quote.TokenLimitField], _ = json.Marshal(limit)
	chat["n"] = json.RawMessage(`1`)
	return nil
}

func validateStreamQuote(quote policy.StreamQuote, raw []byte, plan PrivatePlan) error {
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
	return nil
}

func validateStreamFields(chat map[string]json.RawMessage) error {
	allowed := map[string]bool{"model": true, "messages": true, "tools": true, "tool_choice": true, "parallel_tool_calls": true, "temperature": true, "top_p": true, "stop": true, "stream": true, "stream_options": true, "max_tokens": true, "max_completion_tokens": true, "n": true}
	for field := range chat {
		if !allowed[field] {
			return fmt.Errorf("unsupported quoted parameter: %s", field)
		}
	}
	if err := validateQuotedTools(chat["tools"]); err != nil {
		return err
	}
	return validateQuotedCount(chat["n"])
}

func validateQuotedTools(tools json.RawMessage) error {
	if len(tools) > 0 {
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
	return nil
}

func validateQuotedCount(value json.RawMessage) error {
	if len(value) > 0 {
		var count int
		if json.Unmarshal(value, &count) != nil || count != 1 {
			return errors.New("supplier quote covers exactly one completion")
		}
	}
	return nil
}

func quotedTokenLimit(chat map[string]json.RawMessage, maximum int) (int, error) {
	limit := maximum
	for _, field := range []string{"max_tokens", "max_completion_tokens"} {
		if value, ok := chat[field]; ok {
			var requested int
			if json.Unmarshal(value, &requested) != nil || requested <= 0 || requested > maximum {
				return 0, errors.New("requested tokens exceed supplier quote")
			}
			if requested < limit {
				limit = requested
			}
			delete(chat, field)
		}
	}
	return limit, nil
}

func validateQuotedMessages(raw json.RawMessage) error {
	// Text/tool JSON is sized by bytes. Remote media cannot be sized by its URL.
	var messages []map[string]json.RawMessage
	if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 {
		return errors.New("quoted messages required")
	}
	for _, message := range messages {
		if err := validateQuotedMessage(message); err != nil {
			return err
		}
	}
	return nil
}

func validateQuotedMessage(message map[string]json.RawMessage) error {
	for field := range message {
		switch field {
		case "role", "content", "name", "tool_calls", "tool_call_id", "refusal":
		default:
			return errors.New("unsupported quoted message field")
		}
	}
	return validateQuotedContent(message["content"])
}

func validateQuotedContent(content json.RawMessage) error {
	if len(content) == 0 || string(content) == "null" {
		return nil
	}
	var text string
	if json.Unmarshal(content, &text) == nil {
		return nil
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
		if err := validateQuotedTextPart(part); err != nil {
			return err
		}
	}
	return nil
}

func validateQuotedTextPart(part map[string]json.RawMessage) error {
	for field := range part {
		if field != "type" && field != "text" {
			return errors.New("unsupported quoted content field")
		}
	}
	var text string
	if json.Unmarshal(part["text"], &text) != nil {
		return errors.New("invalid quoted text")
	}
	return nil
}
