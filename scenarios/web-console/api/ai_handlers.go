package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aiH "web-console/handlers/ai"
)

// DOC: docs/concepts/ARCHITECTURE.md#ai-command-generation
// [REQ:P0-005a] AI Command Generation API
// [REQ:P1-003a] Provider Configuration Storage
// [REQ:P1-003b] Provider Health Dashboard

// commandSystemPrompt is the system instruction for command generation.
const commandSystemPrompt = "You are a command-line assistant. Given a natural language description, output ONLY the shell command. No explanation, no markdown, no backticks."

// suggestSystemPrompt is the system instruction for multi-command suggestion.
const suggestSystemPrompt = "You are a command-line assistant. Given a natural language description, output 1 to 3 shell commands (one per line) that accomplish the task. If there is only one reasonable command, output just that one. No explanation, no numbering, no markdown, no backticks."

// knownCodeFences lists the markdown code fence prefixes that AI providers
// commonly wrap shell commands in. Order matters: longer/more-specific prefixes
// are tried first so that e.g. "```bash" is removed before the generic "```".
// Only shell-related fences are stripped; fences for other languages (python,
// ruby, etc.) are left intact so the caller sees that the AI returned the
// wrong type of output.
var knownCodeFences = []string{"```bash", "```sh", "```"}

// extractCommand cleans up AI output by stripping known markdown code fences,
// trimming whitespace, and selecting only the first line. This is the single
// place where the "raw AI text → executable command" decision is made.
func extractCommand(raw string) string {
	s := strings.TrimSpace(raw)
	for _, fence := range knownCodeFences {
		s = strings.TrimPrefix(s, fence)
	}
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// extractCommands splits raw AI output into 1–3 individual commands.
// Each line is cleaned of markdown fences and whitespace; empty lines are dropped.
func extractCommands(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var commands []string
	for _, line := range lines {
		cmd := extractCommand(line)
		if cmd != "" {
			commands = append(commands, cmd)
		}
		if len(commands) >= 3 {
			break
		}
	}
	return commands
}

// aiAdapter implements aiH.Service against the server's provider chain
// and config store.
type aiAdapter struct {
	srv *Server
}

func newAIAdapter(s *Server) *aiAdapter { return &aiAdapter{srv: s} }

func (a *aiAdapter) Generate(ctx context.Context, prompt, terminalContext string) (string, string, error) {
	userPrompt := prompt
	if terminalContext != "" {
		userPrompt = fmt.Sprintf("%s\n\nTerminal context: %s", prompt, terminalContext)
	}
	raw, provider, err := a.srv.executeAI(ctx, buildCommandSystemPrompt(a.srv.systemContext), userPrompt)
	if err != nil {
		return "", "", err
	}
	command := extractCommand(raw)
	a.srv.events.Emit(EventAIGenerate, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
	})
	a.srv.metrics.AIGenerations.Add(1)
	return command, provider, nil
}

func (a *aiAdapter) Suggest(ctx context.Context, prompt, terminalContext string) ([]string, string, error) {
	userPrompt := prompt
	if terminalContext != "" {
		userPrompt = fmt.Sprintf("%s\n\nTerminal context: %s", prompt, terminalContext)
	}
	raw, provider, err := a.srv.executeAI(ctx, buildSuggestSystemPrompt(a.srv.systemContext), userPrompt)
	if err != nil {
		return nil, "", err
	}
	commands := extractCommands(raw)
	a.srv.events.Emit(EventAISuggest, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
		"count":    fmt.Sprintf("%d", len(commands)),
	})
	a.srv.metrics.AISuggestions.Add(1)
	return commands, provider, nil
}

func (a *aiAdapter) GetConfig() aiH.ConfigSnapshot {
	return aiH.ConfigSnapshot{
		Providers: providerConfigsToTransport(a.srv.aiConfig.GetConfigs()),
		Health:    providerHealthsToTransport(a.srv.aiConfig.GetHealth()),
	}
}

func (a *aiAdapter) UpdateConfig(req aiH.UpdateConfigRequest) (aiH.ConfigSnapshot, error) {
	configs := a.srv.aiConfig.GetConfigs()
	var current *ProviderConfig
	for i := range configs {
		if configs[i].Name == req.Name {
			current = &configs[i]
			break
		}
	}
	if current == nil {
		return aiH.ConfigSnapshot{}, fmt.Errorf("%w: %s", aiH.ErrUnknownProvider, req.Name)
	}

	enabled := current.Enabled
	if req.HasEnabled {
		enabled = req.Enabled
	}
	priority := current.Priority
	if req.HasPriority {
		priority = req.Priority
	}
	timeoutSec := current.TimeoutSec
	if req.HasTimeoutSec {
		if req.TimeoutSec < 1 || req.TimeoutSec > 120 {
			return aiH.ConfigSnapshot{}, fmt.Errorf("%w: timeout must be between 1 and 120 seconds", aiH.ErrInvalidBody)
		}
		timeoutSec = req.TimeoutSec
	}
	maxRetries := current.MaxRetries
	if req.HasMaxRetries {
		if req.MaxRetries < 0 || req.MaxRetries > 5 {
			return aiH.ConfigSnapshot{}, fmt.Errorf("%w: max_retries must be between 0 and 5", aiH.ErrInvalidBody)
		}
		maxRetries = req.MaxRetries
	}

	if !a.srv.aiConfig.UpdateConfig(req.Name, enabled, priority, timeoutSec, maxRetries) {
		return aiH.ConfigSnapshot{}, errors.New("failed to update provider config")
	}
	return a.GetConfig(), nil
}

func (a *aiAdapter) GetHealth() []aiH.ProviderHealth {
	return providerHealthsToTransport(a.srv.aiConfig.GetHealth())
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
