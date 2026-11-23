package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	apiidentity "github.com/vrooli/api-core/identity"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/proto"

	"ai-gateway/internal/caller"
	gatewayidentity "ai-gateway/internal/identity"
	internalinference "ai-gateway/internal/inference"
	"ai-gateway/internal/providers"
)

type connectHandler struct {
	service           *internalinference.Service
	runCallerVerifier RunCallerVerifier
}

type RunCallerVerifier interface {
	VerifyRunCallerAssertion(*http.Request, string, string, string) (gatewayidentity.RunCaller, error)
}

func NewConnectHandler(service *internalinference.Service, verifiers ...RunCallerVerifier) *connectHandler {
	handler := &connectHandler{service: service}
	if len(verifiers) > 0 {
		handler.runCallerVerifier = verifiers[0]
	}
	return handler
}

func (h *connectHandler) Run(ctx context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = caller.Observe(ctx, req.Peer().Addr)
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	ctx = providers.WithCreditCallerAuthorization(ctx, req.Header().Get(providers.CreditCallerAuthorizationHeader))
	ctx = providers.WithSource(ctx, req.Header().Get("X-Vrooli-AI-Source"))
	ctx = providers.WithPrivacyConsent(ctx, req.Header().Get("X-Vrooli-AI-Consent-Grant"))
	ctx = providers.WithGatewayHop(ctx, req.Header().Get("X-Vrooli-Gateway-Hop"))
	if assertion := req.Header().Get(gatewayidentity.RunCallerAssertionHeader); assertion != "" {
		if h.runCallerVerifier == nil {
			return nil, connect.NewError(connect.CodeUnavailable, gatewayidentity.ErrRunCallerVerifierUnavailable)
		}
		authRequest := &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/ai-gateway.v1.inference.InferenceService/Run"}, Header: req.Header()}
		verified, err := h.runCallerVerifier.VerifyRunCallerAssertion(authRequest, assertion, req.Msg.GetRole(), req.Msg.GetRootOperationId())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		ctx = gatewayidentity.WithRunCaller(ctx, verified)
		ctx = apiidentity.WithPrincipal(ctx, apiidentity.Principal{Kind: apiidentity.ActorHuman, Subject: verified.ActorID, Owner: verified.BusinessAccountID, Verified: true, Source: apiidentity.SourceConsumerAccessToken})
	}
	request := internalinference.ProviderRequest{
		Source: req.Msg.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(),
		Profile: req.Msg.GetProfile(), PrivacyClass: req.Msg.GetPrivacyClass(),
		PrivacyProvenance: req.Msg.GetPrivacyProvenance(), AllowSemanticEscalation: req.Msg.GetAllowSemanticEscalation(),
		Turns: req.Msg.GetTurns(), Attachments: req.Msg.GetAttachments(),
		MaxOutputTokens:  req.Msg.GetMaxOutputTokens(),
		MaxProviderCalls: req.Msg.GetMaxProviderCalls(),
		SpendLimit:       req.Msg.GetSpendLimit(),
		RootOperationID:  req.Msg.GetRootOperationId(),
		BrowserSession:   req.Msg.GetBrowserSession(),
	}
	digestRequest := proto.Clone(req.Msg).(*inferencev1.RunRequest)
	digestRequest.RootOperationId = ""
	if encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(digestRequest); err == nil {
		// A verified LPBS caller is part of the recovery identity. Without this
		// binding, another account that learns a root operation id and submits
		// identical content could read its accounting-only recovery receipt.
		// Keep generic internal Run calls compatible when no consumer principal
		// was supplied; the LPBS door assertion is the authority for this scope.
		if runCaller, ok := gatewayidentity.RunCallerFromContext(ctx); ok {
			encoded = append(encoded, []byte("\x00lpbs-run-caller\x00"+runCaller.BusinessAccountID+"\x00"+runCaller.ActorID+"\x00"+runCaller.AppKey)...)
		}
		digest := sha256.Sum256(encoded)
		request.RequestDigest = hex.EncodeToString(digest[:])
	}
	// A present SamplingControls with no temperature is not the same as an
	// explicit 0, so read presence through the optional accessor rather than the
	// zero value.
	if sampling := req.Msg.GetSampling(); sampling != nil && sampling.Temperature != nil {
		request.Temperature = proto.Float64(sampling.GetTemperature())
	}
	response := h.service.Run(ctx, request)
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RunBatch(ctx context.Context, req *connect.Request[inferencev1.RunBatchRequest]) (*connect.Response[inferencev1.RunBatchResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = caller.Observe(ctx, req.Peer().Addr)
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	ctx = providers.WithCreditCallerAuthorization(ctx, req.Header().Get(providers.CreditCallerAuthorizationHeader))
	ctx = providers.WithSource(ctx, req.Header().Get("X-Vrooli-AI-Source"))
	ctx = providers.WithPrivacyConsent(ctx, req.Header().Get("X-Vrooli-AI-Consent-Grant"))
	ctx = providers.WithGatewayHop(ctx, req.Header().Get("X-Vrooli-Gateway-Hop"))
	requests := make([]internalinference.ProviderRequest, 0, len(req.Msg.GetItems()))
	for _, item := range req.Msg.GetItems() {
		requests = append(requests, internalinference.ProviderRequest{
			Source: item.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(),
			Profile: req.Msg.GetProfile(), PrivacyClass: req.Msg.GetPrivacyClass(),
			PrivacyProvenance: req.Msg.GetPrivacyProvenance(),
		})
	}
	return connect.NewResponse(h.service.RunBatch(ctx, requests)), nil
}

func (h *connectHandler) Embed(ctx context.Context, req *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = caller.Observe(ctx, req.Peer().Addr)
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	ctx = providers.WithCreditCallerAuthorization(ctx, req.Header().Get(providers.CreditCallerAuthorizationHeader))
	response := h.service.Embed(ctx, internalinference.EmbeddingRequest{
		Role: req.Msg.GetRole(), Texts: req.Msg.GetTexts(), Sampling: req.Msg.GetSampling() != nil,
		Profile: req.Msg.GetProfile(), PrivacyClass: req.Msg.GetPrivacyClass(),
		PrivacyProvenance: req.Msg.GetPrivacyProvenance(),
	})
	return connect.NewResponse(response), nil
}

func (h *connectHandler) Decide(ctx context.Context, req *connect.Request[inferencev1.DecideRequest]) (*connect.Response[inferencev1.DecideResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	// The enum fallback runs the typed path, which may reach the metered
	// provider on the caller's behalf.
	ctx = caller.Observe(ctx, req.Peer().Addr)
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	ctx = providers.WithCreditCallerAuthorization(ctx, req.Header().Get(providers.CreditCallerAuthorizationHeader))
	ctx = providers.WithSource(ctx, req.Header().Get("X-Vrooli-AI-Source"))
	ctx = providers.WithPrivacyConsent(ctx, req.Header().Get("X-Vrooli-AI-Consent-Grant"))
	ctx = providers.WithGatewayHop(ctx, req.Header().Get("X-Vrooli-Gateway-Hop"))
	return connect.NewResponse(h.service.Decide(ctx, internalinference.DecisionRequest{
		Role: req.Msg.GetRole(), State: req.Msg.GetState(), StateJSON: req.Msg.GetStateJson(),
		Questions: req.Msg.GetQuestions(), Profile: req.Msg.GetProfile(), PrivacyClass: req.Msg.GetPrivacyClass(),
		PrivacyProvenance: req.Msg.GetPrivacyProvenance(), PinResourceRole: req.Msg.GetPinResourceRole(),
	})), nil
}

func (h *connectHandler) DecideBatch(ctx context.Context, req *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = caller.Observe(ctx, req.Peer().Addr)
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	ctx = providers.WithCreditCallerAuthorization(ctx, req.Header().Get(providers.CreditCallerAuthorizationHeader))
	ctx = providers.WithSource(ctx, req.Header().Get("X-Vrooli-AI-Source"))
	ctx = providers.WithPrivacyConsent(ctx, req.Header().Get("X-Vrooli-AI-Consent-Grant"))
	ctx = providers.WithGatewayHop(ctx, req.Header().Get("X-Vrooli-Gateway-Hop"))
	items := make([]internalinference.DecisionRequest, 0, len(req.Msg.GetItems()))
	for _, item := range req.Msg.GetItems() {
		items = append(items, internalinference.DecisionRequest{State: item.GetState(), StateJSON: item.GetStateJson()})
	}
	return connect.NewResponse(h.service.DecideBatch(ctx, internalinference.DecisionRequest{
		Role: req.Msg.GetRole(), Questions: req.Msg.GetQuestions(), Profile: req.Msg.GetProfile(), PrivacyClass: req.Msg.GetPrivacyClass(),
		PrivacyProvenance: req.Msg.GetPrivacyProvenance(), PinResourceRole: req.Msg.GetPinResourceRole(),
	}, items)), nil
}
