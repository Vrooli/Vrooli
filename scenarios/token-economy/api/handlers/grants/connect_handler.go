package grants

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/grants"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	grantsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/grants"
)

type connectHandler struct {
	service domain.Service
	logger  *log.Logger
}

func NewConnectHandler(service domain.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) CreateGrant(ctx context.Context, req *connect.Request[grantsv1.CreateGrantRequest]) (*connect.Response[grantsv1.CreateGrantResponse], error) {
	created, err := h.service.Create(ctx, domain.CreateInput{
		TokenTypeID: req.Msg.TokenTypeId, GrantSourceID: req.Msg.GrantSourceId,
		Authorizer: req.Msg.Authorizer, HolderID: req.Msg.HolderId, AmountMinor: req.Msg.AmountMinor,
		AllowedCatalogScopes: req.Msg.AllowedCatalogScopes, DeniedCatalogScopes: req.Msg.DeniedCatalogScopes,
		ExpiresAt: timestampValue(req.Msg.ExpiresAt), IdempotencyKey: req.Msg.IdempotencyKey,
		RequiredEvidence: req.Msg.RequiredEvidence, Rules: rulesFromProto(req.Msg.Rules),
	})
	if err != nil {
		return nil, h.mapError("CreateGrant", err)
	}
	return connect.NewResponse(&grantsv1.CreateGrantResponse{Grant: grantToProto(created)}), nil
}

func (h *connectHandler) GetGrant(ctx context.Context, req *connect.Request[grantsv1.GetGrantRequest]) (*connect.Response[grantsv1.GetGrantResponse], error) {
	grant, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapError("GetGrant", err)
	}
	return connect.NewResponse(&grantsv1.GetGrantResponse{Grant: grantToProto(grant)}), nil
}

func (h *connectHandler) ListGrants(ctx context.Context, req *connect.Request[accessv1.ListGrantsRequest]) (*connect.Response[accessv1.ListGrantsResponse], error) {
	values, err := h.service.List(ctx, req.Msg.HolderId, req.Msg.TokenTypeId, req.Msg.IncludeInactive)
	if err != nil {
		return nil, h.mapError("ListGrants", err)
	}
	out := &accessv1.ListGrantsResponse{Grants: make([]*accessv1.Grant, 0, len(values))}
	for _, value := range values {
		out.Grants = append(out.Grants, accessGrant(value))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RevokeGrant(ctx context.Context, req *connect.Request[accessv1.RevokeGrantRequest]) (*connect.Response[accessv1.RevokeGrantResponse], error) {
	value, err := h.service.Revoke(ctx, domain.RevokeInput{ID: req.Msg.Id, Reason: req.Msg.Reason, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, h.mapError("RevokeGrant", err)
	}
	return connect.NewResponse(&accessv1.RevokeGrantResponse{Grant: accessGrant(value)}), nil
}

func (h *connectHandler) mapError(operation string, err error) error {
	var invalid *domain.InvalidGrantError
	switch {
	case errors.Is(err, domain.ErrGrantNotFound), errors.Is(err, domain.ErrTokenTypeNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrTokenTypeRetired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.logger.Printf("grants.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func timestampValue(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}

func rulesFromProto(values []*grantsv1.GrantRule) []domain.GrantRule {
	out := make([]domain.GrantRule, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		out = append(out, domain.GrantRule{
			ID: value.Id, Condition: ruleConditionFromProto(value.Condition),
			Operands: value.Operands, AmountLimit: value.AmountLimit,
		})
	}
	return out
}

func ruleConditionFromProto(value grantsv1.RuleCondition) domain.RuleCondition {
	switch value {
	case grantsv1.RuleCondition_RULE_CONDITION_CATALOG_SCOPE_ALLOWED:
		return domain.RuleConditionCatalogScopeAllowed
	case grantsv1.RuleCondition_RULE_CONDITION_CATALOG_SCOPE_DENIED:
		return domain.RuleConditionCatalogScopeDenied
	case grantsv1.RuleCondition_RULE_CONDITION_BEFORE_EXPIRY:
		return domain.RuleConditionBeforeExpiry
	case grantsv1.RuleCondition_RULE_CONDITION_REQUIRED_EVIDENCE:
		return domain.RuleConditionRequiredEvidence
	case grantsv1.RuleCondition_RULE_CONDITION_SUFFICIENT_BALANCE:
		return domain.RuleConditionSufficientBalance
	default:
		return ""
	}
}

func ruleConditionToProto(value domain.RuleCondition) grantsv1.RuleCondition {
	switch value {
	case domain.RuleConditionCatalogScopeAllowed:
		return grantsv1.RuleCondition_RULE_CONDITION_CATALOG_SCOPE_ALLOWED
	case domain.RuleConditionCatalogScopeDenied:
		return grantsv1.RuleCondition_RULE_CONDITION_CATALOG_SCOPE_DENIED
	case domain.RuleConditionBeforeExpiry:
		return grantsv1.RuleCondition_RULE_CONDITION_BEFORE_EXPIRY
	case domain.RuleConditionRequiredEvidence:
		return grantsv1.RuleCondition_RULE_CONDITION_REQUIRED_EVIDENCE
	case domain.RuleConditionSufficientBalance:
		return grantsv1.RuleCondition_RULE_CONDITION_SUFFICIENT_BALANCE
	default:
		return grantsv1.RuleCondition_RULE_CONDITION_UNSPECIFIED
	}
}

func grantStatusToProto(value domain.GrantStatus) grantsv1.GrantStatus {
	switch value {
	case domain.GrantStatusDraft:
		return grantsv1.GrantStatus_GRANT_STATUS_DRAFT
	case domain.GrantStatusLive:
		return grantsv1.GrantStatus_GRANT_STATUS_LIVE
	case domain.GrantStatusExhausted:
		return grantsv1.GrantStatus_GRANT_STATUS_EXHAUSTED
	case domain.GrantStatusExpired:
		return grantsv1.GrantStatus_GRANT_STATUS_EXPIRED
	case domain.GrantStatusRevoked:
		return grantsv1.GrantStatus_GRANT_STATUS_REVOKED
	default:
		return grantsv1.GrantStatus_GRANT_STATUS_UNSPECIFIED
	}
}

func grantToProto(value domain.Grant) *grantsv1.Grant {
	out := &grantsv1.Grant{
		Id: value.ID, TokenTypeId: value.TokenTypeID, GrantSourceId: value.GrantSourceID,
		Authorizer: value.Authorizer, HolderId: value.HolderID, AmountMinor: value.AmountMinor,
		AllowedCatalogScopes: value.AllowedCatalogScopes, DeniedCatalogScopes: value.DeniedCatalogScopes,
		ExpiresAt: timestamppb.New(value.ExpiresAt), IssuedAt: timestamppb.New(value.IssuedAt),
		Status: grantStatusToProto(value.Status), IdempotencyKey: value.IdempotencyKey,
		RequiredEvidence: value.RequiredEvidence, RecurrenceSeconds: value.RecurrenceSeconds,
		Rules: make([]*grantsv1.GrantRule, 0, len(value.Rules)),
	}
	if value.NextIssueAt != nil {
		out.NextIssueAt = timestamppb.New(*value.NextIssueAt)
	}
	if value.CancelledAt != nil {
		out.CancelledAt = timestamppb.New(*value.CancelledAt)
	}
	for _, rule := range value.Rules {
		out.Rules = append(out.Rules, &grantsv1.GrantRule{
			Id: rule.ID, Condition: ruleConditionToProto(rule.Condition),
			Operands: rule.Operands, AmountLimit: rule.AmountLimit,
		})
	}
	return out
}

func accessGrant(value domain.Grant) *accessv1.Grant {
	grant := grantToProto(value)
	out := &accessv1.Grant{
		Id: grant.Id, TokenTypeId: grant.TokenTypeId, GrantSourceId: grant.GrantSourceId,
		Authorizer: grant.Authorizer, HolderId: grant.HolderId, AmountMinor: grant.AmountMinor,
		AllowedCatalogScopes: grant.AllowedCatalogScopes, DeniedCatalogScopes: grant.DeniedCatalogScopes,
		ExpiresAt: grant.ExpiresAt, IssuedAt: grant.IssuedAt, Status: accessv1.GrantStatus(grant.Status),
		IdempotencyKey: grant.IdempotencyKey, RequiredEvidence: grant.RequiredEvidence,
		RecurrenceSeconds: grant.RecurrenceSeconds, NextIssueAt: grant.NextIssueAt, CancelledAt: grant.CancelledAt,
		Rules: make([]*accessv1.GrantRule, 0, len(grant.Rules)),
	}
	for _, rule := range grant.Rules {
		out.Rules = append(out.Rules, &accessv1.GrantRule{
			Id: rule.Id, Condition: accessv1.RuleCondition(rule.Condition), Operands: rule.Operands, AmountLimit: rule.AmountLimit,
		})
	}
	return out
}
