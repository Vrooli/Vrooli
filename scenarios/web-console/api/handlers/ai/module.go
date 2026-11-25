// Package ai is the HTTP-handler home for the AI domain.
// It exposes the generated Connect-RPC AIService (proto schema:
// packages/proto/schemas/web-console/v1/ai).
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"

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
			r.Handle("/api/v1/ai/privacy-consent", privacyConsentProxy()).Methods(http.MethodGet, http.MethodPost)
			r.Handle("/api/v1/ai/privacy-consent/{id}", privacyConsentProxy()).Methods(http.MethodDelete)
		},
		Endpoints: Endpoints,
	}
}

// privacyConsentProxy keeps browser requests same-origin while Gateway remains
// the shared consent authority. It forwards only the original authentication
// material, which Gateway independently verifies to derive the grant owner.
func privacyConsentProxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base, err := discovery.ResolveScenarioURLDefault(r.Context(), "ai-gateway")
		if err != nil {
			http.Error(w, "AI consent authority is unavailable", http.StatusServiceUnavailable)
			return
		}
		url := strings.TrimRight(base, "/") + r.URL.RequestURI()
		request, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
		if err != nil {
			http.Error(w, "AI consent request could not be prepared", http.StatusBadGateway)
			return
		}
		for _, name := range []string{"Authorization", "Cookie", "Content-Type"} {
			for _, value := range r.Header.Values(name) {
				request.Header.Add(name, value)
			}
		}
		response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
		if err != nil {
			http.Error(w, "AI consent authority is unavailable", http.StatusServiceUnavailable)
			return
		}
		defer response.Body.Close()
		for _, value := range response.Header.Values("Content-Type") {
			w.Header().Add("Content-Type", value)
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	})
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
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil || strings.TrimSpace(request.Prompt) == "" {
			writeRESTError(w, http.StatusBadRequest, "invalid_argument", "prompt is required", "")
			return
		}
		request.Prompt = strings.TrimSpace(request.Prompt)
		ctx := aidomain.WithConsumerToken(r.Context(), r.Header.Get("Authorization"))
		ctx = aidomain.WithPrivacyConsentGrant(ctx, r.Header.Get("X-Vrooli-AI-Consent-Grant"), r.Header.Get("X-Vrooli-AI-Consent-Source"))
		ctx = aidomain.WithRootOperationID(ctx, r.Header.Get("X-Idempotency-Key"))
		command, provider, err := svc.Generate(ctx, request.Prompt, request.Context)
		if err != nil {
			if errors.Is(err, aidomain.ErrCreditsRequired) {
				upgradePath, _ := discovery.ResolveExternalURL(r.Context(), "landing-page-business-suite", discovery.ExternalHost(r))
				writeRESTError(w, http.StatusPaymentRequired, monetization.ErrorCreditsRequired, "credits are required for routed inference", upgradePath)
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
