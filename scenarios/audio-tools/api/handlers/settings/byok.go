package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/providercatalog"

	"connectrpc.com/connect"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

func (h *connectHandler) ListBYOKCredentials(ctx context.Context, _ *connect.Request[settv1.ListBYOKCredentialsRequest]) (*connect.Response[settv1.ListBYOKCredentialsResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	creds, err := h.deps.BYOK.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*settv1.BYOKCredentialSummary, 0, len(creds))
	for _, c := range creds {
		out = append(out, credToProto(c))
	}
	return connect.NewResponse(&settv1.ListBYOKCredentialsResponse{Credentials: out}), nil
}

func (h *connectHandler) UpsertBYOKCredential(ctx context.Context, req *connect.Request[settv1.UpsertBYOKCredentialRequest]) (*connect.Response[settv1.UpsertBYOKCredentialResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	m := req.Msg
	if err := validateCapability(m.GetCapability()); err != nil {
		return nil, err
	}
	providerID := strings.TrimSpace(m.GetProviderId())
	if providerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id required"))
	}
	// A credential stored under the wrong provider/capability pair can never be
	// selected by the chain: the register lookup is (provider_id, capability).
	// Reject it at the edge instead of persisting a row that silently does
	// nothing (or, worse, looks configured in the settings list).
	if err := validateBYOKPair(providerID, m.GetCapability()); err != nil {
		return nil, err
	}
	key := strings.TrimSpace(m.GetApiKey())
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("api_key required"))
	}
	c, err := h.deps.BYOK.Upsert(ctx, providerID, m.GetCapability(), key)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.UpsertBYOKCredentialResponse{Credential: credToProto(byokstore.Credential{
		ProviderID:  c.ProviderID,
		Capability:  c.Capability,
		Fingerprint: c.Fingerprint,
		CreatedAt:   c.CreatedAt,
		LastUsedAt:  c.LastUsedAt,
	})}), nil
}

func (h *connectHandler) DeleteBYOKCredential(ctx context.Context, req *connect.Request[settv1.DeleteBYOKCredentialRequest]) (*connect.Response[settv1.DeleteBYOKCredentialResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	m := req.Msg
	if err := validateCapability(m.GetCapability()); err != nil {
		return nil, err
	}
	if _, err := h.deps.BYOK.Delete(ctx, m.GetProviderId(), m.GetCapability()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.DeleteBYOKCredentialResponse{}), nil
}

// validateBYOKPair proves the provider exists, is credential-backed, and
// declares the capability the credential is being stored for.
func validateBYOKPair(providerID, capability string) error {
	provider, ok := providercatalog.ByID(providerID)
	if !ok {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown provider %q", providerID))
	}
	if provider.Kind != providercatalog.KindBYOKAPI && provider.Kind != providercatalog.KindGateway {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("provider %q does not accept a BYOK credential", providerID))
	}
	for _, supported := range provider.Capabilities {
		if string(supported) == capability {
			return nil
		}
	}
	return connect.NewError(connect.CodeInvalidArgument,
		fmt.Errorf("provider %q does not support capability %q", providerID, capability))
}

func validateCapability(cap string) error {
	switch cap {
	case "stt", "tts", "summarize":
		return nil
	default:
		return connect.NewError(connect.CodeInvalidArgument, errors.New("capability must be one of stt|tts|summarize"))
	}
}
