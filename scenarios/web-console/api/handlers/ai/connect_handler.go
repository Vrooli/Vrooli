package ai

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	internalai "web-console/internal/ai"
)

// Deps wires the seams the Connect AI handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// AIServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ErrInvalidBody is the sentinel for malformed/empty request fields. Mapped
// to CodeInvalidArgument.
var ErrInvalidBody = errors.New("invalid request body")

// ErrProviderUnavailable is the sentinel the Service implementation returns
// when no AI provider produced a response. Mapped to CodeUnavailable.
var ErrProviderUnavailable = errors.New("ai provider unavailable")

// ErrUnknownProvider is returned by UpdateConfig when name does not match a
// known provider. Mapped to CodeInvalidArgument.
var ErrUnknownProvider = errors.New("unknown provider")

func (h *connectHandler) Generate(ctx context.Context, req *connect.Request[aiv1.GenerateRequest]) (*connect.Response[aiv1.GenerateResponse], error) {
	ctx = internalai.WithConsumerToken(ctx, req.Header().Get("Authorization"))
	prompt := strings.TrimSpace(req.Msg.GetPrompt())
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("prompt is required"))
	}
	command, provider, err := h.deps.Service.Generate(ctx, prompt, req.Msg.GetContext())
	if err != nil {
		h.deps.Logger.Printf("ai.Generate: %v", err)
		if errors.Is(err, internalai.ErrCreditsRequired) {
			refusal := connect.NewError(connect.CodeResourceExhausted, errors.New("credits required"))
			refusal.Meta().Set("X-Vrooli-Error-Type", "credits_required")
			return nil, refusal
		}
		return nil, connect.NewError(connect.CodeUnavailable, ErrProviderUnavailable)
	}
	return connect.NewResponse(&aiv1.GenerateResponse{Command: command, Provider: provider}), nil
}

func (h *connectHandler) Suggest(ctx context.Context, req *connect.Request[aiv1.SuggestRequest]) (*connect.Response[aiv1.SuggestResponse], error) {
	ctx = internalai.WithConsumerToken(ctx, req.Header().Get("Authorization"))
	prompt := strings.TrimSpace(req.Msg.GetPrompt())
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("prompt is required"))
	}
	commands, provider, err := h.deps.Service.Suggest(ctx, prompt, req.Msg.GetContext())
	if err != nil {
		h.deps.Logger.Printf("ai.Suggest: %v", err)
		if errors.Is(err, internalai.ErrCreditsRequired) {
			refusal := connect.NewError(connect.CodeResourceExhausted, errors.New("credits required"))
			refusal.Meta().Set("X-Vrooli-Error-Type", "credits_required")
			return nil, refusal
		}
		return nil, connect.NewError(connect.CodeUnavailable, ErrProviderUnavailable)
	}
	return connect.NewResponse(&aiv1.SuggestResponse{Commands: commands, Provider: provider}), nil
}

func (h *connectHandler) GetConfig(ctx context.Context, _ *connect.Request[aiv1.GetConfigRequest]) (*connect.Response[aiv1.GetConfigResponse], error) {
	snap := h.deps.Service.GetConfig(ctx)
	return connect.NewResponse(&aiv1.GetConfigResponse{
		Providers: configsToProto(snap.Providers),
		Health:    healthsToProto(snap.Health),
	}), nil
}

func (h *connectHandler) UpdateConfig(ctx context.Context, req *connect.Request[aiv1.UpdateConfigRequest]) (*connect.Response[aiv1.UpdateConfigResponse], error) {
	if strings.TrimSpace(req.Msg.GetName()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider name is required"))
	}
	in := UpdateConfigRequest{
		Name:          req.Msg.GetName(),
		Enabled:       req.Msg.GetEnabled(),
		HasEnabled:    req.Msg.GetHasEnabled(),
		Priority:      int(req.Msg.GetPriority()),
		HasPriority:   req.Msg.GetHasPriority(),
		TimeoutSec:    int(req.Msg.GetTimeoutSec()),
		HasTimeoutSec: req.Msg.GetHasTimeoutSec(),
		MaxRetries:    int(req.Msg.GetMaxRetries()),
		HasMaxRetries: req.Msg.GetHasMaxRetries(),
	}
	snap, err := h.deps.Service.UpdateConfig(ctx, in)
	if err != nil {
		if errors.Is(err, ErrUnknownProvider) || errors.Is(err, ErrInvalidBody) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.deps.Logger.Printf("ai.UpdateConfig: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&aiv1.UpdateConfigResponse{
		Providers: configsToProto(snap.Providers),
		Health:    healthsToProto(snap.Health),
	}), nil
}

func (h *connectHandler) GetHealth(ctx context.Context, _ *connect.Request[aiv1.GetHealthRequest]) (*connect.Response[aiv1.GetHealthResponse], error) {
	return connect.NewResponse(&aiv1.GetHealthResponse{
		Health: healthsToProto(h.deps.Service.GetHealth(ctx)),
	}), nil
}

func configToProto(c ProviderConfig) *aiv1.ProviderConfig {
	return &aiv1.ProviderConfig{
		Name:          c.Name,
		Enabled:       c.Enabled,
		Priority:      int32(c.Priority),
		TimeoutSec:    int32(c.TimeoutSec),
		MaxRetries:    int32(c.MaxRetries),
		KeyConfigured: c.KeyConfigured,
		KeySource:     c.KeySource,
	}
}

func configsToProto(in []ProviderConfig) []*aiv1.ProviderConfig {
	out := make([]*aiv1.ProviderConfig, 0, len(in))
	for _, c := range in {
		out = append(out, configToProto(c))
	}
	return out
}

func healthToProto(h ProviderHealth) *aiv1.ProviderHealth {
	return &aiv1.ProviderHealth{
		Name:         h.Name,
		Available:    h.Available,
		LastCheck:    h.LastCheck,
		LastLatency:  h.LastLatency,
		ErrorCount:   h.ErrorCount,
		SuccessCount: h.SuccessCount,
		ErrorRate:    h.ErrorRate,
	}
}

func healthsToProto(in []ProviderHealth) []*aiv1.ProviderHealth {
	out := make([]*aiv1.ProviderHealth, 0, len(in))
	for _, h := range in {
		out = append(out, healthToProto(h))
	}
	return out
}
