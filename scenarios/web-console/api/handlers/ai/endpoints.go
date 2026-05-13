package ai

import (
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the AI module's public surface. Connect-RPC method
// paths reference generated *Procedure constants so adding or renaming an
// RPC in ai.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "ai_generate",
		Path:        aiconnect.AIServiceGenerateProcedure,
		Method:      "POST",
		Summary:     "Generate a shell command from natural language",
		Description: "Runs the provider chain (Ollama -> OpenRouter) to produce a single shell command for the given prompt.",
		Category:    "ai",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"prompt":  "string",
				"context": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"command":  "string",
				"provider": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Prompt missing"},
			{Status: 503, Code: "unavailable", Description: "All AI providers failed"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console ai generate", Args: []string{"--body-file"}},
	},
	{
		ID:          "ai_suggest",
		Path:        aiconnect.AIServiceSuggestProcedure,
		Method:      "POST",
		Summary:     "Suggest 1-3 candidate shell commands",
		Description: "Runs the provider chain to produce 1-3 shell-command suggestions for the given prompt.",
		Category:    "ai",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"prompt":  "string",
				"context": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"commands": "[]string",
				"provider": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Prompt missing"},
			{Status: 503, Code: "unavailable", Description: "All AI providers failed"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console ai suggest", Args: []string{"--body-file"}},
	},
	{
		ID:          "ai_config_get",
		Path:        aiconnect.AIServiceGetConfigProcedure,
		Method:      "POST",
		Summary:     "Get AI provider configuration and health",
		Description: "Returns the persisted provider configs plus runtime health snapshot.",
		Category:    "ai",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers": "[]ProviderConfig",
				"health":    "[]ProviderHealth",
			},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console ai config-get"},
	},
	{
		ID:          "ai_config_update",
		Path:        aiconnect.AIServiceUpdateConfigProcedure,
		Method:      "POST",
		Summary:     "Update an AI provider's configuration",
		Description: "Applies a partial update to one provider's config. Only fields with has_* = true are applied.",
		Category:    "ai",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers": "[]ProviderConfig",
				"health":    "[]ProviderHealth",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown provider or out-of-range value"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console ai config-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "ai_health_get",
		Path:        aiconnect.AIServiceGetHealthProcedure,
		Method:      "POST",
		Summary:     "Get AI provider health",
		Description: "Returns only the runtime provider health snapshot.",
		Category:    "ai",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"health": "[]ProviderHealth"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console ai health"},
	},
}
