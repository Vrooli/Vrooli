package operatingmode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	measures "github.com/vrooli/measures-go"
)

// FieldClassifier is the L2 rung of the resolution ladder: when deterministic
// extraction (L1) cannot recover a declared scalar/enum field from the agent's
// output, the ladder asks the classifier to reconstruct that one field from the
// raw text. It abstains (Found=false) rather than guessing, so an unresolved
// required field routes to an honest abstain instead of a fabricated value.
//
// The seam is deliberately per-field and transport-free: production wires an
// ollama-backed implementation (measures.LLMExtractor over the resource-ollama
// gateway), and unit tests inject a stub with no live model.
type FieldClassifier interface {
	ClassifyField(ctx context.Context, req ClassifyFieldRequest) (ClassifyFieldResult, error)
}

// ClassifyFieldRequest describes one declared field to reconstruct from raw
// agent output. Enum, when non-empty, hard-constrains the allowed answers.
type ClassifyFieldRequest struct {
	RawOutput   string
	FieldPath   string
	FieldType   string
	Description string
	Enum        []string
}

// ClassifyFieldResult is the classifier's answer. Found=false is an explicit
// abstention; Value is meaningful only when Found is true.
type ClassifyFieldResult struct {
	Found bool
	Value string
}

// ollamaFieldClassifier is the production L2 classifier. It reuses measures-go's
// constrained LLMExtractor (which prompts for a strict JSON reply and abstains
// on an unusable answer) over a Completer that shells one completion through
// `resource-ollama gateway generate --role classify.routing` — the same
// throttled resource CLI the rest of the fleet uses, so the daemon is never hit
// directly.
type ollamaFieldClassifier struct {
	extractor *measures.LLMExtractor
}

const (
	classifierBin  = "resource-ollama"
	classifierRole = "classify.routing"
	// classifierMaxTokens bounds the reply: a single field's JSON answer is
	// tiny, so a small cap keeps the call fast and cheap.
	classifierMaxTokens = 256
)

// newOllamaFieldClassifier builds the production classifier over the gateway
// CLI. The binary must be on $PATH; if it is not, ClassifyField fails per call
// and the ladder degrades to an honest abstain rather than crashing.
func newOllamaFieldClassifier() FieldClassifier {
	completer := measures.CompleterFunc(func(ctx context.Context, prompt string) (string, error) {
		return gatewayGenerate(ctx, classifierRole, prompt, classifierMaxTokens)
	})
	return &ollamaFieldClassifier{extractor: measures.NewLLMExtractor(completer)}
}

// ClassifyField maps the declared field onto a measures.Param and delegates to
// the constrained extractor. An enum becomes the hard `allowed` set; a boolean
// field is constrained to true/false so the model cannot invent a third value.
func (c *ollamaFieldClassifier) ClassifyField(ctx context.Context, req ClassifyFieldRequest) (ClassifyFieldResult, error) {
	allowed := req.Enum
	if len(allowed) == 0 && isBooleanType(req.FieldType) {
		allowed = []string{"true", "false"}
	}
	param := measures.Param{
		Name:        req.FieldPath,
		Type:        req.FieldType,
		Description: req.Description,
	}
	res, err := c.extractor.Extract(ctx, req.RawOutput, param, allowed)
	if err != nil {
		return ClassifyFieldResult{}, err
	}
	return ClassifyFieldResult{Found: res.Found, Value: strings.TrimSpace(res.Value)}, nil
}

// gatewayGenerate shells one deterministic completion through the resource-ollama
// gateway and unwraps the model's text, stripping any qwen3 reasoning block. The
// prompt is passed on stdin, never as an argument, so no injection is possible.
func gatewayGenerate(ctx context.Context, role, prompt string, maxTokens int) (string, error) {
	args := []string{
		"gateway", "generate",
		"--role", role,
		"--json",
		"--max-tokens", fmt.Sprintf("%d", maxTokens),
		"--temperature", "0",
		"--prompt-stdin",
	}
	// #nosec G204 -- no shell; argv[0] is the fixed resource-ollama binary and
	// the flags are internal constants. The prompt is delivered via stdin.
	cmd := exec.CommandContext(ctx, classifierBin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s: %w", strings.Join(strings.Fields(msg), " "), err)
		}
		return "", err
	}
	return unwrapGatewayResponse(stdout.Bytes()), nil
}

// unwrapGatewayResponse returns the model's completion text. The gateway wraps
// it as {"response":"…"}; a non-envelope payload is returned as-is. Any qwen3
// <think>…</think> reasoning block (which survives even under /no_think) is
// stripped so only the answer text reaches the extractor's JSON parser.
func unwrapGatewayResponse(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	var env struct {
		Response string `json:"response"`
	}
	text := string(trimmed)
	if err := json.Unmarshal(trimmed, &env); err == nil && env.Response != "" {
		text = env.Response
	}
	return stripThinkBlocks(text)
}

// stripThinkBlocks removes <think>…</think> spans; an unterminated block drops
// everything from it onward.
func stripThinkBlocks(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 || end < start {
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}

// isBooleanType reports whether a declared field type denotes a boolean.
func isBooleanType(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "boolean", "bool":
		return true
	default:
		return false
	}
}
