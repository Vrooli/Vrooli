package main

import (
	"context"
	"fmt"

	aiH "web-console/handlers/ai"
	"web-console/internal/events"
)

// commandSystemPrompt and suggestSystemPrompt are the base system
// instructions used by buildCommandSystemPrompt / buildSuggestSystemPrompt
// in ai_system_context.go to produce the system-context-enriched prompts.
const (
	commandSystemPrompt = "You are a command-line assistant. Given a natural language description, output ONLY the shell command. No explanation, no markdown, no backticks."
	suggestSystemPrompt = "You are a command-line assistant. Given a natural language description, output 1 to 3 shell commands (one per line) that accomplish the task. If there is only one reasonable command, output just that one. No explanation, no numbering, no markdown, no backticks."
)

// aiServiceShim adapts package-main *Server (provider chain, AIConfigStore,
// system context, events bus, metrics) to the transport-neutral aiH.Backend
// interface. All internal↔transport conversion happens here so handlers/ai
// stays free of package-main type references.
type aiServiceShim struct {
	s *Server
}

func newAIServiceShim(s *Server) *aiServiceShim { return &aiServiceShim{s: s} }

func (a *aiServiceShim) ExecuteCommand(ctx context.Context, userPrompt string) (string, string, error) {
	return a.s.executeAI(ctx, buildCommandSystemPrompt(a.s.systemContext), userPrompt)
}

func (a *aiServiceShim) ExecuteSuggest(ctx context.Context, userPrompt string) (string, string, error) {
	return a.s.executeAI(ctx, buildSuggestSystemPrompt(a.s.systemContext), userPrompt)
}

func (a *aiServiceShim) EmitGenerate(provider, prompt string) {
	a.s.events.Emit(events.AIGenerate, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
	})
}

func (a *aiServiceShim) EmitSuggest(provider, prompt string, count int) {
	a.s.events.Emit(events.AISuggest, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
		"count":    fmt.Sprintf("%d", count),
	})
}

func (a *aiServiceShim) IncrGenerations() { a.s.metrics.AIGenerations.Add(1) }
func (a *aiServiceShim) IncrSuggestions() { a.s.metrics.AISuggestions.Add(1) }

func (a *aiServiceShim) GetConfigs() []aiH.ProviderConfig {
	return providerConfigsToTransport(a.s.aiConfig.GetConfigs())
}

func (a *aiServiceShim) GetHealth() []aiH.ProviderHealth {
	return providerHealthsToTransport(a.s.aiConfig.GetHealth())
}

func (a *aiServiceShim) UpdateProviderConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
	return a.s.aiConfig.UpdateConfig(name, enabled, priority, timeoutSec, maxRetries)
}

func providerConfigToTransport(c ProviderConfig) aiH.ProviderConfig {
	return aiH.ProviderConfig{
		Name:       c.Name,
		Enabled:    c.Enabled,
		Priority:   c.Priority,
		TimeoutSec: c.TimeoutSec,
		MaxRetries: c.MaxRetries,
	}
}

func providerConfigsToTransport(in []ProviderConfig) []aiH.ProviderConfig {
	out := make([]aiH.ProviderConfig, 0, len(in))
	for _, c := range in {
		out = append(out, providerConfigToTransport(c))
	}
	return out
}

func providerHealthToTransport(h ProviderHealth) aiH.ProviderHealth {
	return aiH.ProviderHealth{
		Name:         h.Name,
		Available:    h.Available,
		LastCheck:    h.LastCheck,
		LastLatency:  h.LastLatency,
		ErrorCount:   h.ErrorCount,
		SuccessCount: h.SuccessCount,
		ErrorRate:    h.ErrorRate,
	}
}

func providerHealthsToTransport(in []ProviderHealth) []aiH.ProviderHealth {
	out := make([]aiH.ProviderHealth, 0, len(in))
	for _, h := range in {
		out = append(out, providerHealthToTransport(h))
	}
	return out
}
