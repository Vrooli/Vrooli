package codecs

import (
	"fmt"
	"strings"

	"agent-manager/internal/domain"
)

func codexControlArgs(cfg *domain.RunConfig) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	args := make([]string, 0, 8)
	model := strings.TrimSpace(cfg.Model)
	{
		bareModel, isOllama := splitOllamaModel(model)
		if isOllama {
			args = append(args, "--oss", "--local-provider", "ollama")
		}
		if bareModel != "" {
			args = append(args, "-m", bareModel)
		}
		if cfg.Effort != "" && cfg.Effort != domain.EffortMax {
			args = append(args, "-c", "model_reasoning_effort="+string(cfg.Effort))
		}
	}
	return args, nil
}

func claudeControlArgs(cfg *domain.RunConfig) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	args := make([]string, 0, 8)
	model := strings.TrimSpace(cfg.Model)
	{
		if model != "" {
			args = append(args, "--model", model)
		}
		if cfg.Effort != "" {
			args = append(args, "--effort", string(cfg.Effort))
		}
		if tools, err := translateCanonicalTools(claudeToolTranslations, cfg.AllowedTools); err != nil {
			return nil, err
		} else if len(tools) > 0 {
			args = append(args, "--allowedTools", strings.Join(tools, ","))
		}
		if tools, err := translateCanonicalTools(claudeToolTranslations, cfg.DeniedTools); err != nil {
			return nil, err
		} else if len(tools) > 0 {
			args = append(args, "--disallowedTools", strings.Join(tools, ","))
		}
	}
	return args, nil
}

func grokControlArgs(cfg *domain.RunConfig) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	args := make([]string, 0, 8)
	model := strings.TrimSpace(cfg.Model)
	{
		if model != "" {
			args = append(args, "-m", model)
		}
		if cfg.Effort != "" {
			args = append(args, "--effort", string(cfg.Effort))
		}
		if tools, err := translateCanonicalTools(grokToolTranslations, cfg.AllowedTools); err != nil {
			return nil, err
		} else {
			for _, tool := range tools {
				args = append(args, "--allow", tool)
			}
		}
		if tools, err := translateCanonicalTools(grokToolTranslations, cfg.DeniedTools); err != nil {
			return nil, err
		} else {
			for _, tool := range tools {
				args = append(args, "--deny", tool)
			}
		}
	}
	return args, nil
}

func opencodeControlArgs(cfg *domain.RunConfig) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	args := make([]string, 0, 4)
	model := strings.TrimSpace(cfg.Model)
	{
		if model != "" {
			args = append(args, "-m", model)
		}
		if cfg.Effort != "" {
			if !openCodeVariantSupported(model, cfg.Effort) {
				return nil, fmt.Errorf("opencode model %q has no documented variant for effort %q", model, cfg.Effort)
			}
			args = append(args, "--variant", string(cfg.Effort))
		}
	}
	return args, nil
}

// openCodeVariantSupported is deliberately conservative. OpenCode documents
// variants by provider/model rather than as one global CLI domain: Anthropic
// supports high/max, OpenAI supports low/medium/high/xhigh (model-dependent),
// and Google supports low/high. Unknown and local providers must configure a
// custom variant explicitly before Agent Manager can safely emit one.
func openCodeVariantSupported(model string, effort domain.Effort) bool {
	provider, _, ok := strings.Cut(strings.ToLower(strings.TrimSpace(model)), "/")
	if !ok || provider == "" {
		return false
	}
	switch provider {
	case "anthropic":
		return effort == domain.EffortHigh || effort == domain.EffortMax
	case "openai":
		return effort == domain.EffortLow || effort == domain.EffortMedium || effort == domain.EffortHigh || effort == domain.EffortXHigh
	case "google":
		return effort == domain.EffortLow || effort == domain.EffortHigh
	default:
		return false
	}
}
