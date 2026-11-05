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
	ErrPrivateFeatureNotFound = errors.New("declared AI feature was not found")
	ErrPrivatePlanNotEligible = errors.New("no eligible provider route")
	ErrPrivatePlanConflict    = errors.New("operation id is already bound to a different plan")
	ErrPrivateReceiptConflict = errors.New("operation id is already bound to a different request")
	ErrPrivateReceiptMissing  = errors.New("provider execution receipt was not found")
)

type PrivateFeature struct {
	ID                      string          `json:"id"`
	Role                    string          `json:"role"`
	Profile                 string          `json:"profile"`
	MaxInputBytes           int             `json:"max_input_bytes"`
	MaxOutputBytes          int             `json:"max_output_bytes"`
	MaxUserTurns            int             `json:"max_user_turns"`
	Schema                  json.RawMessage `json:"schema"`
	PinnedInstructionDigest string          `json:"pinned_instruction_digest,omitempty"`
}

type PrivatePlanRequest struct {
	OperationID      string `json:"operation_id"`
	FeatureID        string `json:"feature_id"`
	Model            string `json:"model"`
	BudgetMicros     int64  `json:"budget_micros"`
	Harness          string `json:"harness,omitempty"`
	CreditMultiplier int64  `json:"credit_multiplier"`
}

type PrivatePlan struct {
	OperationID      string             `json:"operation_id"`
	FeatureID        string             `json:"feature_id"`
	RequestedModel   string             `json:"requested_model"`
	BudgetMicros     int64              `json:"budget_micros"`
	Eligible         bool               `json:"eligible"`
	Harness          string             `json:"harness,omitempty"`
	QuotedCredits    int64              `json:"quoted_credits"`
	StreamQuote      policy.StreamQuote `json:"stream_quote"`
	CreditMultiplier int64              `json:"credit_multiplier"`
	RouteID          string             `json:"route_id,omitempty"`
	Supplier         string             `json:"supplier,omitempty"`
	Model            string             `json:"model,omitempty"`
	Reason           string             `json:"reason,omitempty"`
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
	if request.OperationID == "" || request.FeatureID == "" || (request.Model == "" && s.features[request.FeatureID].PinnedInstructionDigest == "") || request.BudgetMicros < 0 {
		return PrivatePlan{}, errors.New("operation_id, feature_id, model, and a non-negative budget_micros are required")
	}
	decision, err := s.eligible(ctx, request)
	if err != nil {
		return PrivatePlan{}, err
	}
	result := PrivatePlan{OperationID: request.OperationID, FeatureID: request.FeatureID, RequestedModel: request.Model, BudgetMicros: request.BudgetMicros, Harness: request.Harness, CreditMultiplier: request.CreditMultiplier}
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
	if s == nil || s.plans == nil {
		return PrivateReceipt{}, errors.New("provider route plan repository is unavailable")
	}
	savedPlan, err := s.plans.GetPlan(ctx, request.OperationID)
	if errors.Is(err, sql.ErrNoRows) {
		return PrivateReceipt{}, ErrPrivatePlanNotEligible
	}
	if err != nil {
		return PrivateReceipt{}, err
	}
	if savedPlan != request.Plan {
		return PrivateReceipt{}, ErrPrivatePlanConflict
	}
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
	if err != nil {
		return s.persistFailure(request, "policy_unavailable", "provider eligibility could not be verified")
	}
	if !savedPlan.Eligible || len(decision.Routes) == 0 {
		return s.persistFailure(request, "no_eligible_route", "the pinned route is no longer eligible")
	}
	selected := decision.Routes[0]
	if selected.ID != savedPlan.RouteID || selected.Supplier != savedPlan.Supplier || selected.Model != savedPlan.Model {
		return s.persistFailure(request, "pinned_route_changed", "the original route is no longer the selected eligible route")
	}
	return s.persistFailure(request, "route_executor_unavailable", "no production provider executor is configured for this policy route")
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
		bounded := make([]policy.Route, 0, len(snapshot.Routes))
		for _, route := range snapshot.Routes {
			quote := route.StreamQuote
			if (quote.TokenLimitField != "max_tokens" && quote.TokenLimitField != "max_completion_tokens") || quote.MaxInputBytes <= 0 || quote.MaxOutputTokens <= 0 || quote.MaxCostMicros <= 0 || (request.BudgetMicros > 0 && quote.MaxCostMicros > request.BudgetMicros) {
				continue
			}
			if request.CreditMultiplier > 0 && (quote.MaxCostMicros > (math.MaxInt64-9999)/request.CreditMultiplier || (quote.MaxCostMicros*request.CreditMultiplier+9999)/10000 > route.QuotedCredits) {
				continue
			}
			bounded = append(bounded, route)
		}
		snapshot.Routes = bounded
	}
	return policy.Eligible(policyRequest, snapshot.Account, snapshot.Records, snapshot.Agreements, snapshot.Routes, s.now()), nil
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

type SQLPrivatePlanRepository struct{ db *sql.DB }

func NewSQLPrivatePlanRepository(db *sql.DB) *SQLPrivatePlanRepository {
	return &SQLPrivatePlanRepository{db: db}
}

func (r *SQLPrivatePlanRepository) PersistPlan(ctx context.Context, operationID, fingerprint string, plan PrivatePlan) (PrivatePlan, error) {
	if r == nil || r.db == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is not configured")
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return PrivatePlan{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO private_provider_plans (operation_id, request_sha256, plan_json, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(operation_id) DO NOTHING`, operationID, fingerprint, string(encoded), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PrivatePlan{}, err
	}
	var existingFingerprint, existingJSON string
	if err := r.db.QueryRowContext(ctx, `SELECT request_sha256, plan_json FROM private_provider_plans WHERE operation_id = ?`, operationID).Scan(&existingFingerprint, &existingJSON); err != nil {
		return PrivatePlan{}, err
	}
	if existingFingerprint != fingerprint {
		return PrivatePlan{}, ErrPrivatePlanConflict
	}
	var existing PrivatePlan
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return PrivatePlan{}, fmt.Errorf("decode provider route plan: %w", err)
	}
	return existing, nil
}

func (r *SQLPrivatePlanRepository) GetPlan(ctx context.Context, operationID string) (PrivatePlan, error) {
	if r == nil || r.db == nil {
		return PrivatePlan{}, errors.New("provider route plan repository is not configured")
	}
	var encoded string
	if err := r.db.QueryRowContext(ctx, `SELECT plan_json FROM private_provider_plans WHERE operation_id = ?`, operationID).Scan(&encoded); err != nil {
		return PrivatePlan{}, err
	}
	var plan PrivatePlan
	if err := json.Unmarshal([]byte(encoded), &plan); err != nil {
		return PrivatePlan{}, fmt.Errorf("decode provider route plan: %w", err)
	}
	return plan, nil
}

type SQLPrivateReceiptRepository struct{ db *sql.DB }

func NewSQLPrivateReceiptRepository(db *sql.DB) *SQLPrivateReceiptRepository {
	return &SQLPrivateReceiptRepository{db: db}
}

func (r *SQLPrivateReceiptRepository) Persist(ctx context.Context, operationID, fingerprint string, receipt PrivateReceipt) (PrivateReceipt, error) {
	if r == nil || r.db == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is not configured")
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return PrivateReceipt{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO private_provider_receipts (operation_id, request_sha256, receipt_json, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(operation_id) DO NOTHING`, operationID, fingerprint, string(encoded), receipt.CreatedAt)
	if err != nil {
		return PrivateReceipt{}, err
	}
	var existingFingerprint, existingJSON string
	if err := r.db.QueryRowContext(ctx, `SELECT request_sha256, receipt_json FROM private_provider_receipts WHERE operation_id = ?`, operationID).Scan(&existingFingerprint, &existingJSON); err != nil {
		return PrivateReceipt{}, err
	}
	if existingFingerprint != fingerprint {
		return PrivateReceipt{}, ErrPrivateReceiptConflict
	}
	var existing PrivateReceipt
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return PrivateReceipt{}, fmt.Errorf("decode provider execution receipt: %w", err)
	}
	return existing, nil
}

func (r *SQLPrivateReceiptRepository) Get(ctx context.Context, operationID string) (PrivateReceipt, error) {
	if r == nil || r.db == nil {
		return PrivateReceipt{}, errors.New("provider execution receipt repository is not configured")
	}
	var encoded string
	if err := r.db.QueryRowContext(ctx, `SELECT receipt_json FROM private_provider_receipts WHERE operation_id = ?`, operationID).Scan(&encoded); err != nil {
		return PrivateReceipt{}, err
	}
	var receipt PrivateReceipt
	if err := json.Unmarshal([]byte(encoded), &receipt); err != nil {
		return PrivateReceipt{}, fmt.Errorf("decode provider execution receipt: %w", err)
	}
	return receipt, nil
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
