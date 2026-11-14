// Responsibility: per-process native harness model source configuration.
package cliutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ModelSource describes a deliberate selection, not permission to spend. The
// credential issuer and inference door enforce the session's model and cap.
// SecretEnv names a credential supplied by the credential authority at spawn;
// neither this record nor the generated configuration contains the credential.
type ModelSource struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Model     string `json:"model,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	SecretEnv string `json:"secret_env,omitempty"`
}

const ModelSourceEnvironment = "VROOLI_MODEL_SOURCE"

// SupportsModelSource reports an implemented native adapter, not billing or
// supplier permission. Eligibility is still evaluated by the inference door.
func SupportsModelSource(runner string) bool { return runner == "opencode" || runner == "codex" }

func modelSourceFromEnvironment(environment []string) (*ModelSource, error) {
	raw := environmentValue(environment, ModelSourceEnvironment)
	if raw == "" {
		return nil, nil
	}
	var source ModelSource
	if err := json.Unmarshal([]byte(raw), &source); err != nil {
		return nil, errors.New("invalid model source selection")
	}
	return &source, nil
}

// ResolveModelSource applies native per-process settings without writing the
// user's configuration. Subscription selection is deliberately an identity
// operation. Callers must obtain eligible sources from the route authority.
func ResolveModelSource(runner string, source ModelSource, args, environment []string) ([]string, []string, error) {
	if source.Kind == "subscription" {
		return args, environment, nil
	}
	if source.Kind != "own_key" && source.Kind != "local" && source.Kind != "vrooli_credits" {
		return nil, nil, errors.New("unknown model source kind")
	}
	if source.Model == "" || strings.ContainsAny(source.Model, "\r\n\x00") {
		return nil, nil, errors.New("model source requires a model")
	}
	base, err := url.Parse(source.BaseURL)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, nil, errors.New("model source requires a clean inference URL")
	}
	loopback := base.Hostname() == "localhost"
	if ip := net.ParseIP(base.Hostname()); ip != nil {
		loopback = ip.IsLoopback()
	}
	if base.Scheme != "https" && !(base.Scheme == "http" && loopback) {
		return nil, nil, errors.New("model source requires HTTPS or loopback HTTP")
	}
	if source.Kind != "local" && (source.SecretEnv == "" || environmentValue(environment, source.SecretEnv) == "") {
		return nil, nil, errors.New("model source credential is unavailable")
	}
	if source.SecretEnv != "" && !validSourceEnv(source.SecretEnv) {
		return nil, nil, errors.New("invalid model source credential reference")
	}
	// Copy before extending so a caller can reuse the original own-account launch.
	out := withoutNativeModelOverride(args)
	env := append([]string(nil), environment...)
	switch runner {
	case "opencode":
		options := map[string]any{"baseURL": strings.TrimRight(source.BaseURL, "/")}
		if source.SecretEnv != "" {
			options["apiKey"] = "{env:" + source.SecretEnv + "}"
		}
		config, err := json.Marshal(map[string]any{
			"model": "vrooli-source/" + source.Model,
			"provider": map[string]any{"vrooli-source": map[string]any{
				"npm": "@ai-sdk/openai-compatible", "name": source.ID,
				"options": options,
				"models":  map[string]any{source.Model: map[string]any{"name": source.Model}},
			}},
		})
		if err != nil {
			return nil, nil, err
		}
		env = withEnvironmentValue(env, "OPENCODE_CONFIG_CONTENT", string(config))
		out = append(out, "--model", "vrooli-source/"+source.Model)
	case "codex":
		// JSON string quoting is compatible with TOML basic strings here and
		// prevents a URL or model from introducing another configuration key.
		quote := func(value string) string { encoded, _ := json.Marshal(value); return string(encoded) }
		out = append(out, "--model", source.Model, "-c", "model_provider=\"vrooli-source\"", "-c", "model="+quote(source.Model),
			"-c", "web_search=\"disabled\"",
			"-c", "model_providers.vrooli-source.name="+quote(source.ID),
			"-c", "model_providers.vrooli-source.base_url="+quote(strings.TrimRight(source.BaseURL, "/")),
			"-c", "model_providers.vrooli-source.wire_api=\"responses\"",
			"-c", "model_providers.vrooli-source.requires_openai_auth=false")
		if source.SecretEnv != "" {
			out = append(out, "-c", "model_providers.vrooli-source.env_key="+quote(source.SecretEnv))
		}
	default:
		return nil, nil, fmt.Errorf("%s has no qualified model source adapter", runner)
	}
	return out, env, nil
}

func validSourceEnv(name string) bool {
	for i, ch := range name {
		if ch != '_' && (ch < 'A' || ch > 'Z') && (i == 0 || ch < '0' || ch > '9') {
			return false
		}
	}
	return name != ""
}

// An explicit source owns the model. Native -m/--model flags otherwise beat
// configuration overrides (notably Codex), routing a saved model by mistake.
func withoutNativeModelOverride(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}
		if arg == "-m" || arg == "--model" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--model=") || strings.HasPrefix(arg, "-m=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}
