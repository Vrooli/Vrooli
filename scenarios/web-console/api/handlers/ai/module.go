// Package ai is the HTTP-handler home for the AI domain.
// It exposes the generated Connect-RPC AIService (proto schema:
// packages/proto/schemas/web-console/v1/ai).
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"

	aidomain "web-console/internal/ai"
	"web-console/internal/module"
)

// ProviderConfig and ProviderHealth re-export the canonical types from
// internal/ai so handler code can keep its short local names.
type (
	ProviderConfig = aidomain.Config
	ProviderHealth = aidomain.Health
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (adapts the AI provider chain +
// AIConfigStore to satisfy this interface).
type Service interface {
	Generate(ctx context.Context, prompt, terminalContext string) (command, provider string, err error)
	Suggest(ctx context.Context, prompt, terminalContext string) (commands []string, provider string, err error)
	GetConfig(ctx context.Context) ConfigSnapshot
	UpdateConfig(ctx context.Context, req UpdateConfigRequest) (ConfigSnapshot, error)
	GetHealth(ctx context.Context) []ProviderHealth
}

// ConfigSnapshot bundles current configs and health.
type ConfigSnapshot struct {
	Providers []ProviderConfig
	Health    []ProviderHealth
}

// UpdateConfigRequest carries the provider name plus optional field overrides.
// Each Has* flag indicates whether the paired field should be applied.
type UpdateConfigRequest struct {
	Name string

	Enabled       bool
	HasEnabled    bool
	Priority      int
	HasPriority   bool
	TimeoutSec    int
	HasTimeoutSec bool
	MaxRetries    int
	HasMaxRetries bool
}

// Module wires the AI domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := aiconnect.NewAIServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "ai",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.Handle("/api/v1/ai/generate", restGenerateHandler(svc)).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

type restGenerateRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context"`
}

// restGenerateHandler is the browser-safe REST exception for clients that
// need payment-required semantics. Connect maps resource exhaustion to 429;
// this explicit feature endpoint preserves the paid-surface 402 contract.
func restGenerateHandler(svc Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request restGenerateRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeRESTError(w, http.StatusBadRequest, "invalid_argument", "prompt is required", "")
			return
		}
		request.Prompt = strings.TrimSpace(request.Prompt)
		if request.Prompt == "" {
			writeRESTError(w, http.StatusBadRequest, "invalid_argument", "prompt is required", "")
			return
		}
		ctx := aidomain.WithConsumerToken(r.Context(), r.Header.Get("Authorization"))
		command, provider, err := svc.Generate(ctx, request.Prompt, request.Context)
		if err != nil {
			if errors.Is(err, aidomain.ErrCreditsRequired) {
				upgradePath, _ := discovery.ResolveExternalURL(r.Context(), "landing-page-business-suite", r.Host)
				writeRESTError(w, http.StatusPaymentRequired, "credits_required", "credits are required for routed inference", upgradePath)
				return
			}
			writeRESTError(w, http.StatusServiceUnavailable, "unavailable", "AI provider unavailable", "")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"command": command, "provider": provider})
	})
}

func writeRESTError(w http.ResponseWriter, status int, errorType, message, upgradePath string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": message, "error_type": errorType, "retryable": status >= 500, "upgrade_path": upgradePath,
	})
}
