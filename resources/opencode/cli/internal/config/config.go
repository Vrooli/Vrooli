// Package config owns generation of the opencode.json the raw `opencode`
// binary reads (~/.config/opencode/opencode.json). It replaces the former
// nested-jq bash (`opencode::default_config_payload`, `ensure_config`,
// `ensure_ollama_provider`, `migrate_legacy_models`) with a typed builder.
//
// Two invariants the builder MUST hold:
//   - It preserves every key it does not manage byte-for-value — most
//     importantly `permission` (the governed bash map written by the
//     permissions adapter) and any operator/unknown keys.
//   - It is idempotent: rendering an already-current config yields the same
//     bytes, so callers only write on a real change.
//
// Model discovery and sampling are NOT decided here — they come from the
// Ollama probe SSOT (`resource-ollama models`/`policy`, see ensure.go). This
// package only assembles the document from concrete decisions.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Managed top-level keys.
const (
	keySchema       = "$schema"
	keyModel        = "model"
	keySmallModel   = "small_model"
	keyInstructions = "instructions"
	keyProvider     = "provider"

	schemaURL        = "https://opencode.ai/config.json"
	ollamaProviderID = "ollama"
	ollamaNPM        = "ollama-ai-provider-v2"
	ollamaName       = "Ollama (local)"

	// retiredOllamaModelSubstr is dropped from the provider block: its family
	// narrates tool calls as text instead of emitting structured tool_calls.
	retiredOllamaModelSubstr = "qwen2.5-coder"
)

// Sampling carries the resolved, clamped sampling triple for the local model.
// A nil field means "not pinned" — the key is omitted and the model default
// applies.
type Sampling struct {
	Temperature *float64
	TopP        *float64
	TopK        *int
}

// OllamaProvider describes the local provider block to write when the daemon
// is reachable (or to migrate in place when a stale block already exists).
type OllamaProvider struct {
	BaseURL    string // full base URL including /api suffix
	ChatModel  string
	SmallModel string
	NumCtx     int
	Sampling   Sampling
}

// Inputs is the fully-decided render request (decisions made in ensure.go).
type Inputs struct {
	// Active model selection (provider/model). Empty Provider leaves model
	// management to the merge/repoint flags below.
	Provider        string
	ChatModel       string
	CompletionModel string

	// Repoint forces model/small_model onto Provider/*; otherwise absent
	// model/small_model are filled but existing values are preserved.
	Repoint bool

	// LegacyChatTarget/LegacyCompletionTarget migrate the retired
	// openrouter/qwen3-coder slugs to the current default when present.
	MigrateLegacy bool
	LegacyTargets []string // slugs to rewrite (e.g. openrouter/qwen3-coder, openrouter/qwen/qwen3-coder)
	LegacyChat    string
	LegacySmall   string

	// Ollama, when non-nil, writes/refreshes the local provider block.
	Ollama *OllamaProvider
}

// Render produces the new opencode.json bytes from the existing file content
// (which may be empty/nil for a fresh config) and the decided inputs. It is
// pure and deterministic.
func Render(existing []byte, in Inputs) ([]byte, error) {
	top := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &top); err != nil {
			return nil, fmt.Errorf("parse existing opencode.json: %w", err)
		}
	}

	// $schema is always present.
	if _, ok := top[keySchema]; !ok {
		top[keySchema] = mustRaw(schemaURL)
	}
	// instructions defaults to ["AGENTS.md"] when absent.
	if _, ok := top[keyInstructions]; !ok {
		top[keyInstructions] = mustRaw([]string{"AGENTS.md"})
	}

	if in.MigrateLegacy {
		migrateLegacy(top, in)
	}

	// model / small_model management.
	if in.Provider != "" {
		desiredModel := in.Provider + "/" + in.ChatModel
		desiredSmall := in.Provider + "/" + in.CompletionModel
		if in.Repoint {
			top[keyModel] = mustRaw(desiredModel)
			top[keySmallModel] = mustRaw(desiredSmall)
		} else {
			if _, ok := top[keyModel]; !ok {
				top[keyModel] = mustRaw(desiredModel)
			}
			if _, ok := top[keySmallModel]; !ok {
				top[keySmallModel] = mustRaw(desiredSmall)
			}
		}
	}

	if in.Ollama != nil {
		if err := writeOllamaProvider(top, in.Ollama); err != nil {
			return nil, err
		}
	}

	return marshalDoc(top)
}

func migrateLegacy(top map[string]json.RawMessage, in Inputs) {
	legacy := make(map[string]bool, len(in.LegacyTargets))
	for _, s := range in.LegacyTargets {
		legacy[s] = true
	}
	if cur := rawString(top[keyModel]); legacy[cur] && in.LegacyChat != "" {
		top[keyModel] = mustRaw(in.LegacyChat)
	}
	if cur := rawString(top[keySmallModel]); legacy[cur] && in.LegacySmall != "" {
		top[keySmallModel] = mustRaw(in.LegacySmall)
	}
}

// writeOllamaProvider merges the native local provider block into provider.ollama,
// preserving any other providers and dropping the retired model family.
func writeOllamaProvider(top map[string]json.RawMessage, op *OllamaProvider) error {
	providers := map[string]json.RawMessage{}
	if raw, ok := top[keyProvider]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &providers); err != nil {
			return fmt.Errorf("parse provider block: %w", err)
		}
	}

	// Existing ollama block (preserve unmanaged sub-keys / extra models).
	ollama := map[string]json.RawMessage{}
	if raw, ok := providers[ollamaProviderID]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &ollama); err != nil {
			return fmt.Errorf("parse ollama provider: %w", err)
		}
	}
	ollama["npm"] = mustRaw(ollamaNPM)
	ollama["name"] = mustRaw(ollamaName)

	// options.baseURL (preserve other options).
	opts := map[string]json.RawMessage{}
	if raw, ok := ollama["options"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &opts)
	}
	opts["baseURL"] = mustRaw(op.BaseURL)
	ollama["options"] = mustRaw(opts)

	// models map: drop retired family, then set chat/small entries.
	modelsMap := map[string]json.RawMessage{}
	if raw, ok := ollama["models"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &modelsMap)
	}
	for k := range modelsMap {
		if strings.Contains(k, retiredOllamaModelSubstr) {
			delete(modelsMap, k)
		}
	}
	entry := ollamaModelEntry(op)
	modelsMap[op.ChatModel] = entry
	if op.SmallModel != "" {
		modelsMap[op.SmallModel] = entry
	}
	ollama["models"] = mustRaw(modelsMap)

	providers[ollamaProviderID] = mustRaw(ollama)
	top[keyProvider] = mustRaw(providers)
	return nil
}

// ollamaModelEntry builds one models[<model>] entry: nested
// options.options.{num_ctx,sampling} (the native provider reads
// providerOptions.ollama.options.*) plus a matching limit budget.
func ollamaModelEntry(op *OllamaProvider) json.RawMessage {
	inner := map[string]json.RawMessage{
		"num_ctx": mustRaw(op.NumCtx),
	}
	if op.Sampling.Temperature != nil {
		inner["temperature"] = mustRaw(*op.Sampling.Temperature)
	}
	if op.Sampling.TopP != nil {
		inner["top_p"] = mustRaw(*op.Sampling.TopP)
	}
	if op.Sampling.TopK != nil {
		inner["top_k"] = mustRaw(*op.Sampling.TopK)
	}
	out := op.NumCtx / 2
	entry := map[string]json.RawMessage{
		"options": mustRaw(map[string]json.RawMessage{"options": mustRaw(inner)}),
		"limit":   mustRaw(map[string]int{"context": op.NumCtx, "output": out}),
	}
	return mustRaw(entry)
}

// DefaultPayload renders a fresh config (no existing file). Mirrors the former
// opencode::default_config_payload plus the local provider block when ollama
// inputs are supplied.
func DefaultPayload(in Inputs) ([]byte, error) {
	return Render(nil, in)
}

// marshalDoc emits deterministic, uniformly-indented JSON (Go map marshaling
// sorts keys), with a trailing newline.
func marshalDoc(top map[string]json.RawMessage) ([]byte, error) {
	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		// All inputs here are JSON-encodable; a failure is a programmer error.
		panic(fmt.Sprintf("config: marshal %T: %v", v, err))
	}
	return b
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
