// Package ai is the HTTP-handler home for the AI domain.
// It exposes the generated Connect-RPC AIService (proto schema:
// packages/proto/schemas/web-console/v1/ai).
package ai

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

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
		},
		Endpoints: Endpoints,
	}
}
