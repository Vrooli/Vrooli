package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// knownCodeFences lists the markdown code fence prefixes that AI providers
// commonly wrap shell commands in. Longer/more-specific prefixes are tried
// first so that e.g. "```bash" is removed before the generic "```". Only
// shell-related fences are stripped.
var KnownCodeFences = []string{"```bash", "```sh", "```"}

// Backend is the seam the Adapter depends on. Methods speak in transport-
// neutral ai-package types; package main provides the shim that adapts
// the provider chain + AIConfigStore + events bus to this interface.
type Backend interface {
	// ExecuteCommand runs the provider chain with the command system prompt
	// (system-context augmented) and the user prompt. Returns raw output,
	// the provider name that served the request, and any error.
	ExecuteCommand(ctx context.Context, userPrompt string) (raw string, provider string, err error)
	// ExecuteSuggest is the multi-command variant.
	ExecuteSuggest(ctx context.Context, userPrompt string) (raw string, provider string, err error)

	EmitGenerate(provider, prompt string)
	EmitSuggest(provider, prompt string, count int)
	IncrGenerations()
	IncrSuggestions()

	GetConfigs(ctx context.Context) []ProviderConfig
	GetHealth(ctx context.Context) []ProviderHealth
	UpdateProviderConfig(ctx context.Context, name string, enabled bool, priority, timeoutSec, maxRetries int) bool
}

// Adapter is the production Service implementation. Constructed in
// api/main.go with a typed Backend and passed to Module.
type Adapter struct {
	Backend Backend
}

// ExtractCommand cleans up AI output by stripping known markdown code
// fences, trimming whitespace, and selecting only the first line. This
// is the single place where "raw AI text → executable command" is
// decided.
func ExtractCommand(raw string) string {
	s := strings.TrimSpace(raw)
	for _, fence := range KnownCodeFences {
		s = strings.TrimPrefix(s, fence)
	}
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// ExtractCommands splits raw AI output into 1–3 individual commands.
// Each line is cleaned of markdown fences and whitespace; empty lines
// are dropped.
func ExtractCommands(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var commands []string
	for _, line := range lines {
		cmd := ExtractCommand(line)
		if cmd != "" {
			commands = append(commands, cmd)
		}
		if len(commands) >= 3 {
			break
		}
	}
	return commands
}

func (a *Adapter) Generate(ctx context.Context, prompt, terminalContext string) (string, string, error) {
	userPrompt := prompt
	if terminalContext != "" {
		userPrompt = fmt.Sprintf("%s\n\nTerminal context: %s", prompt, terminalContext)
	}
	raw, provider, err := a.Backend.ExecuteCommand(ctx, userPrompt)
	if err != nil {
		return "", "", err
	}
	command := ExtractCommand(raw)
	a.Backend.EmitGenerate(provider, prompt)
	a.Backend.IncrGenerations()
	return command, provider, nil
}

func (a *Adapter) Suggest(ctx context.Context, prompt, terminalContext string) ([]string, string, error) {
	userPrompt := prompt
	if terminalContext != "" {
		userPrompt = fmt.Sprintf("%s\n\nTerminal context: %s", prompt, terminalContext)
	}
	raw, provider, err := a.Backend.ExecuteSuggest(ctx, userPrompt)
	if err != nil {
		return nil, "", err
	}
	commands := ExtractCommands(raw)
	a.Backend.EmitSuggest(provider, prompt, len(commands))
	a.Backend.IncrSuggestions()
	return commands, provider, nil
}

func (a *Adapter) GetConfig(ctx context.Context) ConfigSnapshot {
	return ConfigSnapshot{
		Providers: a.Backend.GetConfigs(ctx),
		Health:    a.Backend.GetHealth(ctx),
	}
}

func (a *Adapter) UpdateConfig(ctx context.Context, req UpdateConfigRequest) (ConfigSnapshot, error) {
	configs := a.Backend.GetConfigs(ctx)
	var current *ProviderConfig
	for i := range configs {
		if configs[i].Name == req.Name {
			current = &configs[i]
			break
		}
	}
	if current == nil {
		return ConfigSnapshot{}, fmt.Errorf("%w: %s", ErrUnknownProvider, req.Name)
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
			return ConfigSnapshot{}, fmt.Errorf("%w: timeout must be between 1 and 120 seconds", ErrInvalidBody)
		}
		timeoutSec = req.TimeoutSec
	}
	maxRetries := current.MaxRetries
	if req.HasMaxRetries {
		if req.MaxRetries < 0 || req.MaxRetries > 5 {
			return ConfigSnapshot{}, fmt.Errorf("%w: max_retries must be between 0 and 5", ErrInvalidBody)
		}
		maxRetries = req.MaxRetries
	}

	if !a.Backend.UpdateProviderConfig(ctx, req.Name, enabled, priority, timeoutSec, maxRetries) {
		return ConfigSnapshot{}, errors.New("failed to update provider config")
	}
	return a.GetConfig(ctx), nil
}

func (a *Adapter) GetHealth(ctx context.Context) []ProviderHealth {
	return a.Backend.GetHealth(ctx)
}
