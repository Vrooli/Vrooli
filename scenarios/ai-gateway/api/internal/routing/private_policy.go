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
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

var (
	ErrPrivateFeatureNotFound      = errors.New("declared AI feature was not found")
	ErrPrivatePlanNotEligible      = errors.New("no eligible provider route")
	ErrPrivatePlanConflict         = errors.New("operation id is already bound to a different plan")
	ErrPrivateReceiptConflict      = errors.New("operation id is already bound to a different request")
	ErrPrivateReceiptMissing       = errors.New("provider execution receipt was not found")
	ErrPrivateFeatureScopeMismatch = errors.New("private feature is outside the admitted app scope")
)

type PrivateFeature struct {
	ID                      string          `json:"id"`
	AppKey                  string          `json:"app_key,omitempty"`
	Role                    string          `json:"role"`
	Profile                 string          `json:"profile"`
	MaxInputBytes           int             `json:"max_input_bytes"`
	MaxOutputBytes          int             `json:"max_output_bytes"`
	MaxUserTurns            int             `json:"max_user_turns"`
	Schema                  json.RawMessage `json:"schema"`
	PinnedInstructionDigest string          `json:"pinned_instruction_digest,omitempty"`
}

type PrivatePlanRequest struct {
	OperationID       string `json:"operation_id"`
	FeatureID         string `json:"feature_id"`
	Model             string `json:"model"`
	BudgetMicros      int64  `json:"budget_micros"`
	Harness           string `json:"harness,omitempty"`
	CreditMultiplier  int64  `json:"credit_multiplier"`
	BusinessAccountID string `json:"business_account_id"`
	ActorID           string `json:"actor_id"`
	AppKey            string `json:"app_key"`
	MaxCredits        int64  `json:"max_credits"`
}

type PrivatePlan struct {
	OperationID       string             `json:"operation_id"`
	FeatureID         string             `json:"feature_id"`
	RequestedModel    string             `json:"requested_model"`
	BudgetMicros      int64              `json:"budget_micros"`
	Eligible          bool               `json:"eligible"`
	Harness           string             `json:"harness,omitempty"`
	QuotedCredits     int64              `json:"quoted_credits"`
	StreamQuote       policy.StreamQuote `json:"stream_quote"`
	CreditMultiplier  int64              `json:"credit_multiplier"`
	RouteID           string             `json:"route_id,omitempty"`
	Supplier          string             `json:"supplier,omitempty"`
	Model             string             `json:"model,omitempty"`
	Reason            string             `json:"reason,omitempty"`
	BusinessAccountID string             `json:"business_account_id"`
	ActorID           string             `json:"actor_id"`
	AppKey            string             `json:"app_key"`
	MaxCredits        int64              `json:"max_credits"`
}

type PrivateFeatureInput struct {
	Instruction string          `json:"instruction"`
	Turn        PrivateTurn     `json:"turn"`
	Schema      json.RawMessage `json:"schema"`
}

type PrivateTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PrivateExecuteRequest struct {
	OperationID string              `json:"operation_id"`
	Plan        PrivatePlan         `json:"plan"`
	Feature     PrivateFeatureInput `json:"feature"`
}

type PrivateReceipt struct {
	OperationID      string `json:"operation_id"`
	FeatureID        string `json:"feature_id"`
	Content          string `json:"content,omitempty"`
	State            string `json:"state"`
	ErrorCode        string `json:"error_code,omitempty"`
	Message          string `json:"message,omitempty"`
	RouteID          string `json:"route_id,omitempty"`
	Supplier         string `json:"supplier,omitempty"`
	Model            string `json:"model,omitempty"`
	CreatedAt        string `json:"created_at"`
	CostMicros       int64  `json:"cost_micros"`
	Credits          int64  `json:"credits"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	CostKnown        bool   `json:"cost_known"`
}

type PrivatePolicySnapshot struct {
	Account    policy.Account
	Records    []policy.RuleRecord
	Agreements []policy.Agreement
	Routes     []policy.Route
}

type PrivatePolicySnapshotSource func(context.Context, string) (PrivatePolicySnapshot, error)

type PrivateReceiptRepository interface {
	Persist(context.Context, string, string, PrivateReceipt) (PrivateReceipt, error)
	Get(context.Context, string) (PrivateReceipt, error)
}

type PrivatePlanRepository interface {
	PersistPlan(context.Context, string, string, PrivatePlan) (PrivatePlan, error)
	GetPlan(context.Context, string) (PrivatePlan, error)
}

type PrivatePolicyService struct {
	features map[string]PrivateFeature
	snapshot PrivatePolicySnapshotSource
	plans    PrivatePlanRepository
	receipts PrivateReceiptRepository
	clock    func() time.Time
	stream   *HTTPStreamExecutor
}

func NewPrivatePolicyService(features []PrivateFeature, snapshot PrivatePolicySnapshotSource, plans PrivatePlanRepository, receipts PrivateReceiptRepository) *PrivatePolicyService {
	byID := make(map[string]PrivateFeature, len(features))
	for _, feature := range features {
		if id := strings.TrimSpace(feature.ID); id != "" {
			byID[id] = feature
		}
	}
	if snapshot == nil {
		snapshot = func(context.Context, string) (PrivatePolicySnapshot, error) { return PrivatePolicySnapshot{}, nil }
	}
	return &PrivatePolicyService{features: byID, snapshot: snapshot, plans: plans, receipts: receipts, clock: time.Now}
}

func (s *PrivatePolicyService) Plan(ctx context.Context, request PrivatePlanRequest) (PrivatePlan, error) {
	request.OperationID = strings.TrimSpace(request.OperationID)
	request.FeatureID = strings.TrimSpace(request.FeatureID)
	request.Model = strings.TrimSpace(request.Model)
	if feature := s.features[request.FeatureID]; feature.AppKey != "" && feature.AppKey != strings.TrimSpace(request.AppKey) {
		return PrivatePlan{}, ErrPrivateFeatureScopeMismatch
	}
	if request.OperationID == "" || request.FeatureID == "" || (request.Model == "" && s.features[request.FeatureID].PinnedInstructionDigest == "") || request.BudgetMicros < 0 {
		return PrivatePlan{}, errors.New("operation_id, feature_id, model, and a non-negative budget_micros are required")
	}
	decision, err := s.eligible(ctx, request)
	if err != nil {
		return PrivatePlan{}, err
	}
	result := PrivatePlan{OperationID: request.OperationID, FeatureID: request.FeatureID, RequestedModel: request.Model, BudgetMicros: request.BudgetMicros, Harness: request.Harness, CreditMultiplier: request.CreditMultiplier, BusinessAccountID: request.BusinessAccountID, ActorID: request.ActorID, AppKey: request.AppKey, MaxCredits: request.MaxCredits}
	if len(decision.Routes) == 0 {
		result.Reason = "no_eligible_route"
		return s.persistPlan(request, result)
	}
	selected := decision.Routes[0]
	result.Eligible = true
	result.RouteID, result.Supplier, result.Model = selected.ID, selected.Supplier, selected.Model
	result.QuotedCredits = selected.QuotedCredits
	result.StreamQuote = selected.StreamQuote
	return s.persistPlan(request, result)
}

// Execute re-evaluates the exact same policy path as Plan. It persists a
// terminal no-route receipt with an independent context so a disconnected
// caller cannot erase the outcome. A provider call is deliberately unavailable
// until the owner route has a verified record reader and concrete executor.
func (s *PrivatePolicyService) Execute(ctx context.Context, request PrivateExecuteRequest) (PrivateReceipt, error) {
	request.OperationID = strings.TrimSpace(request.OperationID)
	if request.OperationID == "" || request.Plan.OperationID != request.OperationID {
		return PrivateReceipt{}, errors.New("operation_id must match the pinned plan")
	}
	// Once the execute contract is accepted, caller cancellation must not erase
	// its policy decision or durable receipt. Provider work added by the owning
	// route path must use this detached operation context as well.
	ctx = context.WithoutCancel(ctx)
	savedPlan, err := s.acceptedExecutionPlan(ctx, request)
	if err != nil {
		return PrivateReceipt{}, err
	}
	return s.executeAcceptedPlan(ctx, request, savedPlan)
}

func (s *PrivatePolicyService) acceptedExecutionPlan(ctx context.Context, request PrivateExecuteRequest) (PrivatePlan, error) {
	if s == nil || s.plans == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is unavailable")
	}
	savedPlan, err := s.plans.GetPlan(ctx, request.OperationID)
	if errors.Is(err, sql.ErrNoRows) {
		return PrivatePlan{}, ErrPrivatePlanNotEligible
	}
	if err != nil {
		return PrivatePlan{}, err
	}
	if savedPlan != request.Plan {
		return PrivatePlan{}, ErrPrivatePlanConflict
	}
	return savedPlan, nil
}

func (s *PrivatePolicyService) executeAcceptedPlan(ctx context.Context, request PrivateExecuteRequest, savedPlan PrivatePlan) (PrivateReceipt, error) {
	planRequest := PrivatePlanRequest{
		OperationID:  request.OperationID,
		FeatureID:    savedPlan.FeatureID,
		Model:        savedPlan.RequestedModel,
		BudgetMicros: savedPlan.BudgetMicros,
	}
	// Fields carried in a pinned plan are copied only after validation below;
	// the actual policy decision is reconstructed from the registered feature.
	if planRequest.FeatureID == "" || planRequest.Model == "" {
		return s.persistFailure(request, "invalid_plan", "the pinned plan is incomplete")
	}
	if err := s.validateFeatureInput(planRequest.FeatureID, request.Feature); err != nil {
		return s.persistFailure(request, "invalid_feature_request", err.Error())
	}
	decision, err := s.eligible(ctx, planRequest)
	if failure := privateExecutionEligibilityFailure(err, savedPlan, decision); failure != nil {
		return s.persistFailure(request, failure.code, failure.message)
	}
	return s.persistFailure(request, "route_executor_unavailable", "no production provider executor is configured for this policy route")
}

type privateExecutionRejection struct{ code, message string }

func privateExecutionEligibilityFailure(err error, saved PrivatePlan, decision policy.EligibilityResult) *privateExecutionRejection {
	if err != nil {
		return &privateExecutionRejection{code: "policy_unavailable", message: "provider eligibility could not be verified"}
	}
	if !saved.Eligible || len(decision.Routes) == 0 {
		return &privateExecutionRejection{code: "no_eligible_route", message: "the pinned route is no longer eligible"}
	}
	selected := decision.Routes[0]
	if selected.ID != saved.RouteID || selected.Supplier != saved.Supplier || selected.Model != saved.Model {
		return &privateExecutionRejection{code: "pinned_route_changed", message: "the original route is no longer the selected eligible route"}
	}
	return nil
}

func (s *PrivatePolicyService) persistPlan(request PrivatePlanRequest, plan PrivatePlan) (PrivatePlan, error) {
	if s == nil || s.plans == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is unavailable")
	}
	canonicalRequest, err := json.Marshal(request)
	if err != nil {
		return PrivatePlan{}, err
	}
	fingerprint := sha256.Sum256(canonicalRequest)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.plans.PersistPlan(ctx, request.OperationID, hex.EncodeToString(fingerprint[:]), plan)
}

func (s *PrivatePolicyService) Receipt(ctx context.Context, operationID string) (PrivateReceipt, error) {
	if s == nil || s.receipts == nil || s.plans == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is unavailable")
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return PrivateReceipt{}, errors.New("operation_id is required")
	}
	receipt, err := s.receipts.Get(ctx, operationID)
	if errors.Is(err, sql.ErrNoRows) {
		return PrivateReceipt{}, ErrPrivateReceiptMissing
	}
	if err != nil {
		return PrivateReceipt{}, err
	}
	// The accepted immutable plan owns feature attribution, including receipts
	// written before feature_id was carried on the receipt wire contract.
	plan, err := s.plans.GetPlan(ctx, operationID)
	if err != nil {
		return PrivateReceipt{}, err
	}
	if receipt.FeatureID != "" && receipt.FeatureID != plan.FeatureID {
		return PrivateReceipt{}, ErrPrivateReceiptConflict
	}
	receipt.FeatureID = plan.FeatureID
	return receipt, nil
}

func (s *PrivatePolicyService) eligible(ctx context.Context, request PrivatePlanRequest) (policy.EligibilityResult, error) {
	if s == nil {
		return policy.EligibilityResult{}, errors.New("private policy service is unavailable")
	}
	snapshot, err := s.snapshot(ctx, request.FeatureID)
	if err != nil {
		return policy.EligibilityResult{}, fmt.Errorf("load provider policy snapshot: %w", err)
	}
	return s.eligibleSnapshot(request, snapshot)
}

func (s *PrivatePolicyService) eligibleSnapshot(request PrivatePlanRequest, snapshot PrivatePolicySnapshot) (policy.EligibilityResult, error) {
	feature, found := s.features[request.FeatureID]
	if !found {
		return policy.EligibilityResult{}, ErrPrivateFeatureNotFound
	}
	contract := policy.FeatureContract{
		ID: feature.ID, MaxInputBytes: feature.MaxInputBytes, MaxOutputBytes: feature.MaxOutputBytes,
		MaxUserTurns: feature.MaxUserTurns, PinnedInstructionDigest: feature.PinnedInstructionDigest,
	}
	check, err := policy.CheckFeatureInvocation(contract, policy.FeatureInvocation{
		InstructionHash: feature.PinnedInstructionDigest, Endpoint: feature.Role,
		UserTurns: 1, InputBytes: 0, OutputBytes: 0, SchemaValid: true,
	})
	if err != nil {
		return policy.EligibilityResult{}, fmt.Errorf("validate registered AI feature: %w", err)
	}
	policyRequest, err := policy.NewEligibleRequest(check,
		request.Model, feature.Profile, "private", request.Harness, 0)
	if err != nil {
		return policy.EligibilityResult{}, err
	}
	if feature.PinnedInstructionDigest == "" {
		snapshot.Routes = boundedPrivateStreamRoutes(snapshot.Routes, request.BudgetMicros, request.CreditMultiplier)
	}
	return policy.Eligible(policyRequest, snapshot.Account, snapshot.Records, snapshot.Agreements, snapshot.Routes, s.now()), nil
}

func boundedPrivateStreamRoutes(routes []policy.Route, budgetMicros, creditMultiplier int64) []policy.Route {
	bounded := make([]policy.Route, 0, len(routes))
	for _, route := range routes {
		if streamQuoteWithinBudget(route, budgetMicros, creditMultiplier) {
			bounded = append(bounded, route)
		}
	}
	return bounded
}

func streamQuoteWithinBudget(route policy.Route, budgetMicros, creditMultiplier int64) bool {
	quote := route.StreamQuote
	if !streamQuoteHasBillableBounds(quote) {
		return false
	}
	if budgetMicros > 0 && quote.MaxCostMicros > budgetMicros {
		return false
	}
	return streamQuoteCreditsFit(quote.MaxCostMicros, creditMultiplier, route.QuotedCredits)
}

func streamQuoteHasBillableBounds(quote policy.StreamQuote) bool {
	if quote.TokenLimitField != "max_tokens" && quote.TokenLimitField != "max_completion_tokens" {
		return false
	}
	return quote.MaxInputBytes > 0 && quote.MaxOutputTokens > 0 && quote.MaxCostMicros > 0
}

func streamQuoteCreditsFit(costMicros, creditMultiplier, quotedCredits int64) bool {
	if creditMultiplier <= 0 {
		return true
	}
	if costMicros > (math.MaxInt64-9999)/creditMultiplier {
		return false
	}
	return (costMicros*creditMultiplier+9999)/10000 <= quotedCredits
}

func (s *PrivatePolicyService) validateFeatureInput(featureID string, input PrivateFeatureInput) error {
	feature, found := s.features[featureID]
	if !found {
		return ErrPrivateFeatureNotFound
	}
	if feature.PinnedInstructionDigest == "" {
		return errors.New("registered feature has no pinned instruction digest")
	}
	instructionDigest := sha256.Sum256([]byte(input.Instruction))
	if hex.EncodeToString(instructionDigest[:]) != feature.PinnedInstructionDigest {
		return errors.New("instruction digest does not match the published feature")
	}
	if input.Turn.Role != "user" || strings.TrimSpace(input.Turn.Content) == "" || len(input.Turn.Content) > feature.MaxInputBytes {
		return errors.New("feature request must contain one bounded user turn")
	}
	if len(input.Schema) == 0 || !json.Valid(input.Schema) || !sameJSON(input.Schema, feature.Schema) {
		return errors.New("request schema does not match the published feature")
	}
	return nil
}

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	leftJSON, leftErr := json.Marshal(leftValue)
	rightJSON, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func (s *PrivatePolicyService) persistFailure(request PrivateExecuteRequest, code, message string) (PrivateReceipt, error) {
	if s == nil || s.receipts == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is unavailable")
	}
	now := time.Now().UTC()
	if s.clock != nil {
		now = s.clock().UTC()
	}
	receipt := PrivateReceipt{OperationID: request.OperationID, FeatureID: request.Plan.FeatureID, State: "failed", ErrorCode: code, Message: message, CreatedAt: now.Format(time.RFC3339Nano)}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return PrivateReceipt{}, err
	}
	var requestValue any
	if err := json.Unmarshal(requestJSON, &requestValue); err != nil {
		return PrivateReceipt{}, err
	}
	canonicalRequest, err := json.Marshal(requestValue)
	if err != nil {
		return PrivateReceipt{}, err
	}
	fingerprint := sha256.Sum256(canonicalRequest)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.receipts.Persist(ctx, request.OperationID, hex.EncodeToString(fingerprint[:]), receipt)
}

func (s *PrivatePolicyService) now() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock().UTC()
}

// Models shares the execution eligibility function; preview never grants a
// route that execution would refuse after a rule or trust change.
func (s *PrivatePolicyService) Models(ctx context.Context, featureID, harness string) ([]string, string, error) {
	snapshot, err := s.snapshot(ctx, featureID)
	if err != nil {
		return nil, "policy_unavailable", err
	}
	result := []string{}
	reason := "no_eligible_route"
	seen := map[string]bool{}
	for _, route := range snapshot.Routes {
		if seen[route.Model] {
			continue
		}
		decision, err := s.eligibleSnapshot(PrivatePlanRequest{FeatureID: featureID, Model: route.Model, Harness: harness}, snapshot)
		if err != nil {
			return nil, "policy_unavailable", err
		}
		if len(decision.Routes) == 0 && len(decision.Refusals) > 0 {
			reason = decision.Refusals[0].Reason
		}
		if len(decision.Routes) > 0 {
			seen[route.Model] = true
			result = append(result, route.Model)
		}
	}
	if len(result) > 0 {
		reason = ""
	}
	return result, reason, nil
}
