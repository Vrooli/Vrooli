package pricing

import "strings"

// DefaultModelAliases maps short model names to their canonical OpenRouter IDs.
var DefaultModelAliases = map[string]struct {
	Canonical string
	Provider  string
}{
	// OpenAI models
	"gpt-5.1-codex":      {"openai/gpt-5.1-codex", "openrouter"},
	"gpt-5.1-codex-max":  {"openai/gpt-5.1-codex-max", "openrouter"},
	"gpt-5.1-codex-mini": {"openai/gpt-5.1-codex-mini", "openrouter"},
	"gpt-5-codex":        {"openai/gpt-5-codex", "openrouter"},
	"codex-mini":         {"openai/codex-mini", "openrouter"},
	"gpt-5.2":            {"openai/gpt-5.2", "openrouter"},
	"gpt-4o":             {"openai/gpt-4o", "openrouter"},
	"gpt-4o-mini":        {"openai/gpt-4o-mini", "openrouter"},
	"gpt-4-turbo":        {"openai/gpt-4-turbo", "openrouter"},

	// Anthropic models
	"claude-opus-4-5":          {"anthropic/claude-opus-4-5", "openrouter"},
	"claude-opus-4-5-20251101": {"anthropic/claude-opus-4-5-20251101", "openrouter"},
	"claude-sonnet-4":          {"anthropic/claude-sonnet-4", "openrouter"},
	"claude-sonnet-4-20250514": {"anthropic/claude-sonnet-4-20250514", "openrouter"},
	"claude-3.5-sonnet":        {"anthropic/claude-3.5-sonnet", "openrouter"},
	"claude-3-opus":            {"anthropic/claude-3-opus", "openrouter"},
	"claude-3-sonnet":          {"anthropic/claude-3-sonnet", "openrouter"},
	"claude-3-haiku":           {"anthropic/claude-3-haiku", "openrouter"},

	// Google models
	"gemini-2.5-pro":   {"google/gemini-2.5-pro", "openrouter"},
	"gemini-2.0-flash": {"google/gemini-2.0-flash", "openrouter"},
	"gemini-1.5-pro":   {"google/gemini-1.5-pro", "openrouter"},
	"gemini-1.5-flash": {"google/gemini-1.5-flash", "openrouter"},

	// Meta models
	"llama-3.3-70b":  {"meta/llama-3.3-70b", "openrouter"},
	"llama-3.1-405b": {"meta/llama-3.1-405b", "openrouter"},
}

// ResolveModelAlias resolves a model name to its canonical form.
// Returns the canonical name, provider, and whether it was found.
func ResolveModelAlias(model string) (canonical, provider string, found bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", false
	}

	// If already canonical (has provider prefix), return as-is
	if strings.Contains(model, "/") {
		return model, "openrouter", true
	}

	// Check default aliases
	if alias, ok := DefaultModelAliases[model]; ok {
		return alias.Canonical, alias.Provider, true
	}

	// Try common prefixes
	prefixes := []string{"openai/", "anthropic/", "google/", "meta/"}
	for _, prefix := range prefixes {
		candidate := prefix + model
		if _, ok := DefaultModelAliases[candidate]; ok {
			return candidate, "openrouter", true
		}
	}

	// Default: assume openai prefix for codex models
	if strings.Contains(strings.ToLower(model), "codex") || strings.Contains(strings.ToLower(model), "gpt") {
		return "openai/" + model, "openrouter", true
	}

	// Default: assume anthropic prefix for claude models
	if strings.Contains(strings.ToLower(model), "claude") {
		return "anthropic/" + model, "openrouter", true
	}

	return model, "openrouter", false
}
