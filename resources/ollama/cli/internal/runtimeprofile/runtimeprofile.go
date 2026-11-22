// Package runtimeprofile is the single source of truth for the Ollama runtime
// options that decide whether a loaded runner can be reused.
//
// Ollama compares a request's runner options (num_ctx, num_batch, num_gpu,
// main_gpu, use_mmap, num_thread) with the loaded runner and restarts the
// runner when any differ. On a nearly full GPU that restart evicts and
// cold-starts models, so two callers that disagree about one option make the
// daemon thrash. The fix is structural: callers never choose these options.
// The server applies the declared defaults to every runner (resource.json
// carries them as daemon environment, verified against profile.json by test).
// A model whose declared context exceeds the server default gets it on every
// request from the two owner paths that reach Ollama: the gateway sends
// RequestOptions, and the public activity edge replaces whatever load options
// a caller sent with the same RequestOptions.
package runtimeprofile

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

//go:embed profile.json
var profileJSON []byte

// Profile is the decoded profile.json.
type Profile struct {
	SchemaVersion      int                       `json:"schema_version"`
	Purpose            string                    `json:"purpose"`
	RequestLoadOptions []string                  `json:"request_load_options"`
	Server             Server                    `json:"server"`
	Variants           map[string]map[string]any `json:"variants"`
	Embedding          Embedding                 `json:"embedding"`
	Models             map[string]Model          `json:"models"`
}

// Server holds the daemon-wide runner settings every model receives.
type Server struct {
	ContextTokens int `json:"context_tokens"`
	CPUThreads    int `json:"cpu_threads"`
	// Environment maps each daemon environment variable to the Server field
	// that supplies its value.
	Environment map[string]string `json:"environment"`
}

// Embedding declares how inputs longer than a model's runner context behave.
type Embedding struct {
	Overflow string `json:"overflow"`
}

// Model records per-model runner facts that differ from the server defaults.
type Model struct {
	ContextTokens int    `json:"context_tokens"`
	Reason        string `json:"reason"`
}

// OverflowTruncateTail keeps the head of an embedding input and drops tokens
// beyond the runner context (Ollama /api/embed with truncate=true).
const OverflowTruncateTail = "truncate_tail"

// VariantCPUOnly is the one sanctioned placement variant. It forces a separate
// CPU runner for the model and therefore replaces any GPU runner.
const VariantCPUOnly = "cpu-only"

var current = mustDecode(profileJSON)

// Current returns the embedded profile.
func Current() Profile { return current }

func mustDecode(raw []byte) Profile {
	p, err := Decode(raw)
	if err != nil {
		panic(err)
	}
	return p
}

// Decode parses and validates a profile document.
func Decode(raw []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(raw, &p); err != nil {
		return Profile{}, fmt.Errorf("decode Ollama runtime profile: %w", err)
	}
	if p.SchemaVersion != 1 {
		return Profile{}, fmt.Errorf("Ollama runtime profile schema_version %d is unsupported", p.SchemaVersion)
	}
	if len(p.RequestLoadOptions) == 0 {
		return Profile{}, fmt.Errorf("Ollama runtime profile declares no request_load_options")
	}
	if p.Server.ContextTokens <= 0 || p.Server.CPUThreads <= 0 {
		return Profile{}, fmt.Errorf("Ollama runtime profile server context_tokens and cpu_threads must be positive")
	}
	if _, err := p.ServerEnvironment(); err != nil {
		return Profile{}, err
	}
	if p.Embedding.Overflow != OverflowTruncateTail {
		return Profile{}, fmt.Errorf("Ollama runtime profile embedding overflow %q is unsupported", p.Embedding.Overflow)
	}
	for name, options := range p.Variants {
		for key := range options {
			if !p.IsLoadOption(key) {
				return Profile{}, fmt.Errorf("Ollama runtime profile variant %s sets non-load option %q", name, key)
			}
		}
	}
	for ref, model := range p.Models {
		if model.ContextTokens <= 0 || strings.TrimSpace(model.Reason) == "" {
			return Profile{}, fmt.Errorf("Ollama runtime profile model %s needs positive context_tokens and a reason", ref)
		}
		if modelretention.CanonicalName(ref) != ref {
			return Profile{}, fmt.Errorf("Ollama runtime profile model %s must be written with its tag", ref)
		}
	}
	return p, nil
}

// ServerEnvironment returns the daemon environment that applies the server
// settings, as resource.json must declare it.
func (p Profile) ServerEnvironment() (map[string]string, error) {
	env := make(map[string]string, len(p.Server.Environment))
	for name, field := range p.Server.Environment {
		switch field {
		case "context_tokens":
			env[name] = strconv.Itoa(p.Server.ContextTokens)
		case "cpu_threads":
			env[name] = strconv.Itoa(p.Server.CPUThreads)
		default:
			return nil, fmt.Errorf("Ollama runtime profile environment %s names unknown server field %q", name, field)
		}
	}
	if len(env) == 0 {
		return nil, fmt.Errorf("Ollama runtime profile declares no server environment")
	}
	return env, nil
}

// IsLoadOption reports whether an Ollama request option changes runner reuse.
func (p Profile) IsLoadOption(key string) bool {
	key = strings.TrimSpace(key)
	for _, candidate := range p.RequestLoadOptions {
		if candidate == key {
			return true
		}
	}
	return false
}

// StripLoadOptions deletes every load-affecting key from options and returns
// the removed keys in sorted order.
func (p Profile) StripLoadOptions(options map[string]any) []string {
	var removed []string
	for key := range options {
		if p.IsLoadOption(key) {
			removed = append(removed, key)
			delete(options, key)
		}
	}
	sort.Strings(removed)
	return removed
}

// RequestOptions returns the load options every request for model carries.
// Only a declared context above the server default is sent; a smaller declared
// context is a model limit Ollama applies itself (its trained context), so
// sending it would only create a second runner shape for direct callers.
func (p Profile) RequestOptions(model string) map[string]any {
	options := map[string]any{}
	if declared, ok := p.DeclaredContextTokens(model); ok && declared > p.Server.ContextTokens {
		options["num_ctx"] = declared
	}
	return options
}

// GatewayOptions returns RequestOptions plus an optional sanctioned variant.
func (p Profile) GatewayOptions(model, variant string) (map[string]any, error) {
	options := p.RequestOptions(model)
	if variant == "" {
		return options, nil
	}
	extra, err := p.VariantOptions(variant)
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		options[key] = value
	}
	return options, nil
}

// VariantOptions returns a copy of a sanctioned variant's load options.
func (p Profile) VariantOptions(name string) (map[string]any, error) {
	options, ok := p.Variants[name]
	if !ok {
		return nil, fmt.Errorf("Ollama runtime profile has no %q variant", name)
	}
	out := make(map[string]any, len(options))
	for key, value := range options {
		// JSON numbers decode as float64; Ollama options are integers here.
		if f, ok := value.(float64); ok && f == float64(int(f)) {
			out[key] = int(f)
			continue
		}
		out[key] = value
	}
	return out, nil
}

// ContextTokens returns the runner context every caller of model receives.
func (p Profile) ContextTokens(model string) int {
	if m, ok := p.Models[modelretention.CanonicalName(model)]; ok {
		return m.ContextTokens
	}
	return p.Server.ContextTokens
}

// DeclaredContextTokens returns a per-model override and whether one exists.
func (p Profile) DeclaredContextTokens(model string) (int, bool) {
	m, ok := p.Models[modelretention.CanonicalName(model)]
	return m.ContextTokens, ok
}
