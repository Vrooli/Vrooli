package agentspend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/clock"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	"treasury/internal/researchbudget"
	"treasury/internal/settlement"
	"treasury/internal/walletops"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
)

type MandateLister interface {
	List(context.Context) ([]mandate.Mandate, error)
}

type HeadroomReader interface {
	Headroom(context.Context, string) (budget.Headroom, error)
}

type MandateReader interface {
	Get(context.Context, string) (mandate.Mandate, error)
}

type ResearchBudgetPublisher interface {
	PutSnapshot(context.Context, string, researchbudget.Snapshot) error
	RecordSettlement(context.Context, string, researchbudget.Settlement) error
}

type connectHandler struct {
	service        *authorization.Service
	settlements    *settlement.Service
	mandates       MandateLister
	verifier       identity.Verifier
	headroom       HeadroomReader
	mandateRead    MandateReader
	researchBudget ResearchBudgetPublisher
	walletOps      *walletops.Service
	clock          clock.Clock
}

// AgentSpendDependencies is the complete capability set accepted by the
// agent-facing transport. In particular, it deliberately exposes budget
// reads, never the policy-mutating budget service used by TreasuryAdmin.
type AgentSpendDependencies struct {
	Authorization  *authorization.Service
	Settlements    *settlement.Service
	Mandates       MandateLister
	MandateReader  MandateReader
	Verifier       identity.Verifier
	Headroom       HeadroomReader
	WalletOps      *walletops.Service
	Clock          clock.Clock
	ResearchBudget ResearchBudgetPublisher
}

func NewConnectHandler(deps AgentSpendDependencies) authorizationconnect.AgentSpendHandler {
	return &connectHandler{
		service: deps.Authorization, settlements: deps.Settlements,
		mandates: deps.Mandates, mandateRead: deps.MandateReader,
		verifier: deps.Verifier, headroom: deps.Headroom,
		walletOps: deps.WalletOps, clock: deps.Clock,
		researchBudget: deps.ResearchBudget,
	}
}

func (h *connectHandler) GetBudgetHeadroom(ctx context.Context, request *connect.Request[authorizationv1.GetBudgetHeadroomRequest]) (*connect.Response[authorizationv1.GetBudgetHeadroomResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.headroom == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("headroom reporting is unavailable"))
	}
	claims, err := identity.VerifyRequest(ctx, h.verifier, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if h.clock == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization clock is unavailable"))
	}
	if err := identity.Authorize(claims, "treasury.read", "treasury:budget/"+request.Msg.GetBudgetId(), h.clock.Now()); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	value, err := h.headroom.Headroom(ctx, request.Msg.GetBudgetId())
	if err != nil {
		if errors.Is(err, budget.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&authorizationv1.GetBudgetHeadroomResponse{Headroom: &authorizationv1.AgentBudgetHeadroom{
		BudgetId: value.BudgetID, BookId: value.BookID, Currency: value.Currency,
		TotalCapMinor: value.TotalCapMinor, TotalUsedMinor: value.TotalUsedMinor, TotalRemainingMinor: value.TotalRemainingMinor,
		PeriodicCapMinor: value.PeriodicCapMinor, PeriodUsedMinor: value.PeriodUsedMinor, PeriodRemainingMinor: value.PeriodRemainingMinor,
		PerTransactionCapMinor: value.PerTransactionCapMinor, AvailableMinor: value.AvailableMinor,
		PeriodStartedAt: timestamppb.New(value.PeriodStartedAt), ComputedAt: timestamppb.New(value.ComputedAt),
		FundingRequired: value.FundingRequired, FundingCreditedMinor: value.FundingCreditedMinor,
		FundingRemainingMinor: value.FundingRemainingMinor, FundingAtomicAmount: value.FundingAtomicAmount,
		FundingAssetId: value.FundingAssetID,
	}}), nil
}

func (h *connectHandler) ProposeCharge(ctx context.Context, request *connect.Request[authorizationv1.ProposeChargeRequest]) (*connect.Response[authorizationv1.ProposeChargeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service is unavailable"))
	}
	msg := request.Msg
	ctx = approval.WithProducerToken(ctx, identityTokenFromHeaders(request.Header()))
	result, err := h.service.Propose(ctx, authorization.ProposeInput{
		ID: msg.GetId(), IdempotencyKey: msg.GetIdempotencyKey(), MandateID: msg.GetMandateId(),
		IdentityToken: identityTokenFromHeaders(request.Header()), OperationToken: strings.TrimSpace(request.Header().Get(identity.HeaderAgentOperationToken)),
		AmountMinor: msg.GetAmountMinor(), Currency: msg.GetCurrency(), Counterparty: msg.GetCounterparty(),
		SettlementID: msg.GetSettlementId(), InstrumentID: msg.GetInstrumentId(), SettlementIdempotencyKey: msg.GetSettlementIdempotencyKey(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authorizationv1.ProposeChargeResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) GetAuthorization(ctx context.Context, request *connect.Request[authorizationv1.GetAuthorizationRequest]) (*connect.Response[authorizationv1.GetAuthorizationResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.service == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service is unavailable"))
	}
	claims, err := identity.VerifyRequest(ctx, h.verifier, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	result, err := h.service.Get(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	if h.clock == nil || result.RequestingAgent != claims.Subject || identity.Authorize(claims, "treasury.read", "treasury:budget/"+result.BudgetID, h.clock.Now()) != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("authorization belongs to another owner"))
	}
	return connect.NewResponse(&authorizationv1.GetAuthorizationResponse{Authorization: toProto(result)}), nil
}

func (h *connectHandler) ListMandates(ctx context.Context, request *connect.Request[authorizationv1.AgentListMandatesRequest]) (*connect.Response[authorizationv1.AgentListMandatesResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.mandates == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("mandate listing is unavailable"))
	}
	claims, err := identity.VerifyRequest(ctx, h.verifier, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	values, err := h.mandates.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &authorizationv1.AgentListMandatesResponse{Mandates: make([]*authorizationv1.AgentMandate, 0, len(values))}
	for _, value := range values {
		if h.clock == nil || value.AgentSubject != claims.Subject || value.AgentRunID != claims.RunID || identity.Authorize(claims, "treasury.read", "treasury:budget/"+value.BudgetID, h.clock.Now()) != nil {
			continue
		}
		response.Mandates = append(response.Mandates, mandateToProto(value))
	}
	return connect.NewResponse(response), nil
}

func mandateToProto(value mandate.Mandate) *authorizationv1.AgentMandate {
	result := &authorizationv1.AgentMandate{Id: value.ID, BookId: value.BookID, BudgetId: value.BudgetID, CapMinor: value.CapMinor, Currency: value.Currency, AllowedCounterparties: value.AllowedCounterparties, DeniedCounterparties: value.DeniedCounterparties, ExpiresAt: timestamppb.New(value.ExpiresAt), Status: string(value.Status), RequiredEvidence: value.RequiredEvidence, RecurrenceSeconds: int64(value.RecurrenceInterval / time.Second)}
	if !value.NextChargeAt.IsZero() {
		result.NextChargeAt = timestamppb.New(value.NextChargeAt)
	}
	return result
}

func (h *connectHandler) ReportOutcome(ctx context.Context, request *connect.Request[authorizationv1.ReportOutcomeRequest]) (*connect.Response[authorizationv1.ReportOutcomeResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	msg := request.Msg
	if strings.TrimSpace(msg.GetOutcome()) != "" || strings.TrimSpace(msg.GetRailReference()) != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("caller-asserted outcome and rail_reference are forbidden; Treasury reads both from the registered rail"))
	}
	if h.settlements == nil || h.service == nil || h.walletOps == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("mandate session settlement authority is unavailable"))
	}
	claims, err := identity.VerifyRequest(ctx, h.verifier, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	auth, err := h.service.Get(ctx, msg.GetAuthorizationId())
	if err != nil {
		return nil, mapError(err)
	}
	intent := walletops.AgentSettlementIntent{
		AuthorizationID: auth.ID, MandateID: auth.MandateID, BudgetID: auth.BudgetID,
		AgentSubject: claims.Subject, AgentRunID: claims.RunID, SettlementID: msg.GetSettlementId(),
		InstrumentID: msg.GetInstrumentId(), IdempotencyKey: msg.GetIdempotencyKey(),
		AmountMinor: auth.AmountMinor, Currency: auth.Currency, Counterparty: auth.Counterparty,
	}
	if err = h.walletOps.AuthorizeAgentSettlement(ctx, claims, intent, request.Header().Get(walletops.HeaderAgentSessionProof)); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	settleInput := settlement.SettleInput{
		ID: msg.GetSettlementId(), AuthorizationID: msg.GetAuthorizationId(), InstrumentID: msg.GetInstrumentId(),
		IdempotencyKey: msg.GetIdempotencyKey(), IdentityToken: identityTokenFromHeaders(request.Header()),
		OperationToken: strings.TrimSpace(request.Header().Get(identity.HeaderAgentOperationToken)),
	}
	// An exact retry from the same authorized run can reconcile a durable
	// unknown outcome. A first submission still only dispatches once; the
	// follow-up ReportOutcome invokes the registered rail's exact recovery
	// procedure after the mandate/session/grant checks above have succeeded.
	charge, existingErr := h.settlements.Get(ctx, settleInput.ID)
	if existingErr == nil && charge.Outcome == settlement.OutcomeUnknown && charge.AuthorizationID == auth.ID && charge.InstrumentID == settleInput.InstrumentID && charge.IdempotencyKey == settleInput.IdempotencyKey {
		charge, err = h.settlements.ResolveUnknown(ctx, charge.ID)
	} else {
		charge, err = h.settlements.Settle(ctx, settleInput)
	}
	if err != nil {
		return nil, mapSettlementError(err)
	}
	auth, err = h.service.Get(ctx, charge.AuthorizationID)
	if err != nil {
		return nil, mapError(err)
	}
	if err := h.publishResearchBudgetEvidence(ctx, claims, auth, charge); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Treasury settled the x402 charge but Agent Manager budget evidence was not accepted; retry this exact ReportOutcome: %w", err))
	}
	return connect.NewResponse(&authorizationv1.ReportOutcomeResponse{Authorization: toProto(auth), Settlement: settlementToProto(charge)}), nil
}

func (h *connectHandler) publishResearchBudgetEvidence(ctx context.Context, claims identity.Claims, auth authorization.Record, charge settlement.Record) error {
	if h.researchBudget == nil || charge.Outcome != settlement.OutcomeSettled || charge.Rail != "x402" {
		return nil
	}
	runID := strings.TrimSpace(claims.RunID)
	if runID == "" {
		return nil
	}
	if h.mandateRead == nil || h.headroom == nil {
		return errors.New("run-bound mandate and Treasury headroom readers are unavailable")
	}
	boundMandate, err := h.mandateRead.Get(ctx, auth.MandateID)
	if err != nil {
		return fmt.Errorf("read mandate for run-bound settlement: %w", err)
	}
	if boundMandate.ID != auth.MandateID || boundMandate.AgentRunID != runID || boundMandate.AgentSubject != claims.Subject || auth.BudgetID != boundMandate.BudgetID {
		return errors.New("settled mandate does not match the verified Agent Manager run and subject")
	}
	if boundMandate.IssuedAt.IsZero() {
		return errors.New("run-bound mandate has no stable issuance time for the budget snapshot")
	}
	headroom, err := h.headroom.Headroom(ctx, auth.BudgetID)
	if err != nil {
		return fmt.Errorf("read Treasury budget headroom for run snapshot: %w", err)
	}
	currency := strings.ToUpper(strings.TrimSpace(headroom.Currency))
	if headroom.BudgetID != auth.BudgetID || currency == "" || currency != strings.ToUpper(strings.TrimSpace(auth.Currency)) || currency != strings.ToUpper(strings.TrimSpace(charge.Currency)) || charge.AmountMinor < 0 || strings.TrimSpace(charge.ExternalID) == "" {
		return errors.New("Treasury headroom and settled receipt do not match the authorized run budget")
	}
	unitCode := currency + "_MINOR"
	snapshot := researchbudget.Snapshot{
		TreasuryBudgetID: auth.BudgetID, MandateID: boundMandate.ID, AgentSubject: claims.Subject,
		CapMinor: headroom.TotalCapMinor, Currency: currency, UnitCode: unitCode,
		PolicyRevision: "treasury-budget-headroom-v1", CreatedAt: boundMandate.IssuedAt.UTC(),
	}
	if err := h.researchBudget.PutSnapshot(ctx, runID, snapshot); err != nil {
		return fmt.Errorf("publish immutable Treasury budget snapshot: %w", err)
	}
	benefitState := "pending"
	benefitSummary := "Treasury settlement is confirmed; bounded paid-resource result evidence is unavailable."
	if strings.Contains(charge.Detail, "; response_bytes=") && strings.Contains(charge.Detail, "; response_sha256=") {
		benefitState = "measured"
		benefitSummary = charge.Detail
	}
	settledAt := charge.OccurredAt
	if settledAt.IsZero() {
		settledAt = charge.UpdatedAt
	}
	if settledAt.IsZero() {
		return errors.New("settled Treasury record has no stable settlement time")
	}
	settlementEvidence := researchbudget.Settlement{
		ID: "treasury:" + charge.ID, SettlementID: charge.ID, TreasuryReceiptID: charge.ExternalID,
		IntegrationID: "x402:" + strings.ToLower(strings.TrimSpace(charge.Counterparty)), Category: "x402_call",
		AmountMinor: charge.AmountMinor, Currency: currency, UnitCode: unitCode,
		ResultRef: "treasury:settlement/" + charge.ID, BenefitState: benefitState,
		BenefitSummary: benefitSummary, SettledAt: settledAt.UTC(),
	}
	if err := h.researchBudget.RecordSettlement(ctx, runID, settlementEvidence); err != nil {
		return fmt.Errorf("publish Treasury receipt-backed settlement evidence: %w", err)
	}
	return nil
}

func identityTokenFromHeaders(headers http.Header) string {
	if token := strings.TrimSpace(headers.Get(identity.HeaderAgentIdentityToken)); token != "" {
		return token
	}
	parts := strings.Fields(strings.TrimSpace(headers.Get("Authorization")))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func mapSettlementError(err error) error {
	switch {
	case errors.Is(err, settlement.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, settlement.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func settlementToProto(value settlement.Record) *settlementv1.Charge {
	outcome := settlementv1.ChargeOutcome_CHARGE_OUTCOME_UNSPECIFIED
	switch value.Outcome {
	case settlement.OutcomeReady:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_READY
	case settlement.OutcomeCalling:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_CALLING
	case settlement.OutcomeSettled:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_SETTLED
	case settlement.OutcomeFailed:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_FAILED
	case settlement.OutcomeUnknown:
		outcome = settlementv1.ChargeOutcome_CHARGE_OUTCOME_UNKNOWN
	}
	receiptReference := value.ReceiptReference
	if (value.Rail == "x402" || value.Rail == "base-usdc") && value.Outcome == settlement.OutcomeUnknown {
		// A pending x402 Credential Authority reference locates replay
		// authorization. Keep it private to Treasury's service boundary.
		receiptReference = ""
	}
	result := &settlementv1.Charge{Id: value.ID, AuthorizationId: value.AuthorizationID, MandateId: value.MandateID, InstrumentId: value.InstrumentID, Rail: value.Rail, IdempotencyKey: value.IdempotencyKey, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty, Outcome: outcome, ExternalId: value.ExternalID, ReceiptReference: receiptReference, Basis: value.Basis, Detail: value.Detail, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), RetainUntil: timestamppb.New(value.RetainUntil)}
	if !value.OccurredAt.IsZero() {
		result.OccurredAt = timestamppb.New(value.OccurredAt)
	}
	if value.Outcome == settlement.OutcomeSettled && !value.OccurredAt.IsZero() {
		result.SettledAt = timestamppb.New(value.OccurredAt)
	}
	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, authorization.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, authorization.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProto(value authorization.Record) *authorizationv1.AuthorizationRecord {
	verdict := authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_UNSPECIFIED
	switch value.Verdict {
	case authorization.VerdictRefused:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED
	case authorization.VerdictPending:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_PENDING
	case authorization.VerdictApproved:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED
	case authorization.VerdictReleased:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_RELEASED
	case authorization.VerdictSettled:
		verdict = authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED
	}
	return &authorizationv1.AuthorizationRecord{
		Id: value.ID, IdempotencyKey: value.IdempotencyKey, BookId: value.BookID, MandateId: value.MandateID, BudgetId: value.BudgetID,
		RequestingAgent: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty,
		Verdict: verdict, ViolatedConstraint: value.ViolatedConstraint, Remediation: value.Remediation, HoldMinor: value.HoldMinor,
		CreatedAt: timestamppb.New(value.CreatedAt), ExpiresAt: timestamppb.New(value.ExpiresAt),
		SettlementId: value.SettlementID, InstrumentId: value.InstrumentID, SettlementIdempotencyKey: value.SettlementIdempotencyKey,
	}
}
