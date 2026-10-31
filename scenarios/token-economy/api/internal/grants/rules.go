package grants

import (
	"context"
	"fmt"
	"slices"
	"time"
)

type RuleCondition string

const (
	RuleConditionCatalogScopeAllowed RuleCondition = "catalog_scope_allowed"
	RuleConditionCatalogScopeDenied  RuleCondition = "catalog_scope_denied"
	RuleConditionBeforeExpiry        RuleCondition = "before_expiry"
	RuleConditionRequiredEvidence    RuleCondition = "required_evidence"
	RuleConditionSufficientBalance   RuleCondition = "sufficient_balance"
)

type GrantRule struct {
	ID          string
	Condition   RuleCondition
	Operands    []string
	AmountLimit int64
}

type EvaluationRequest struct {
	CatalogScope           string
	Evidence               []string
	AvailableBalance       int64
	RequestedAmount        int64
	Now                    time.Time
	CallerClaimsAuthorized bool
}

type Decision struct {
	Allowed   bool
	RuleID    string
	Condition RuleCondition
	Reason    string
}

type Evaluator interface {
	Evaluate(context.Context, Grant, EvaluationRequest) (Decision, error)
}

type ruleEvaluator struct{}

func NewRuleEvaluator() Evaluator { return ruleEvaluator{} }

func (ruleEvaluator) Evaluate(_ context.Context, grant Grant, request EvaluationRequest) (Decision, error) {
	if request.RequestedAmount <= 0 {
		return Decision{}, &InvalidGrantError{Reason: "requested amount must be positive"}
	}
	for _, rule := range grant.Rules {
		refusal, err := evaluateRule(grant, rule, request)
		if err != nil {
			return Decision{}, err
		}
		if refusal != nil {
			return *refusal, nil
		}
	}
	return Decision{Allowed: true, Reason: "all server-side grant rules passed"}, nil
}

func evaluateRule(grant Grant, rule GrantRule, request EvaluationRequest) (*Decision, error) {
	refuse := func(detail string) *Decision {
		return &Decision{
			Allowed: false, RuleID: rule.ID, Condition: rule.Condition,
			Reason: fmt.Sprintf("rule %q (%s) refused: %s", rule.ID, rule.Condition, detail),
		}
	}
	switch rule.Condition {
	case RuleConditionCatalogScopeAllowed:
		if !slices.Contains(rule.Operands, request.CatalogScope) {
			return refuse(fmt.Sprintf("catalog scope %q is not allowed", request.CatalogScope)), nil
		}
	case RuleConditionCatalogScopeDenied:
		if slices.Contains(rule.Operands, request.CatalogScope) {
			return refuse(fmt.Sprintf("catalog scope %q is denied", request.CatalogScope)), nil
		}
	case RuleConditionBeforeExpiry:
		if !request.Now.Before(grant.ExpiresAt) {
			return refuse(fmt.Sprintf("grant expired at %s", grant.ExpiresAt.UTC().Format(time.RFC3339))), nil
		}
	case RuleConditionRequiredEvidence:
		for _, required := range rule.Operands {
			if !slices.Contains(request.Evidence, required) {
				return refuse(fmt.Sprintf("required evidence %q is missing", required)), nil
			}
		}
	case RuleConditionSufficientBalance:
		if request.RequestedAmount > request.AvailableBalance {
			return refuse(fmt.Sprintf("requested amount %d exceeds available balance %d", request.RequestedAmount, request.AvailableBalance)), nil
		}
		if rule.AmountLimit > 0 && request.RequestedAmount > rule.AmountLimit {
			return refuse(fmt.Sprintf("requested amount %d exceeds rule limit %d", request.RequestedAmount, rule.AmountLimit)), nil
		}
	default:
		return nil, fmt.Errorf("unknown closed rule condition %q", rule.Condition)
	}
	return nil, nil
}
