// Package models is the single source of truth for probing and validating
// the locally-installed Ollama models. Every "is this model installed / does
// it really tool-call / does it satisfy a role's required capabilities" check
// flows through here — agent-manager and the opencode resource call the
// `resource-ollama models {list,probe-tools,doctor}` verbs (this package)
// rather than reimplementing /api/tags, /api/show, or /api/chat probes.
//
// Design note — the live tool-call smoke is AUTHORITATIVE. A model can ship a
// stub prompt template (e.g. gemma4:12b's `{{ .Prompt }}`) and still emit
// clean structured tool_calls through Ollama's native /api/chat tool path. So
// the stub-template check is an advisory FLAG surfaced in the report, never
// the sole cause of a doctor failure: only a model that returns a chat
// response WITHOUT structured tool_calls (the real silent-success failure
// class) fails the tool-role verdict. This keeps the gate from over-blocking a
// genuinely-working model whose template merely looks unusual.
package models

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

// Daemon is the injectable seam over the live Ollama daemon. *ensure.Client
// satisfies it; tests provide a fake. It is intentionally narrow — the only
// daemon surface the probe SSOT needs.
type Daemon interface {
	ListModels(ctx context.Context) ([]string, error)
	ShowModel(ctx context.Context, model string) (ensure.ShowResponse, error)
	Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error)
}

// ToolCapability is the Ollama capability string that signals native
// tool-calling support on /api/show.
const ToolCapability = "tools"

// ListResult is the `models list` payload.
type ListResult struct {
	Models []string `json:"models"`
}

// ToolProbeResult is the `models probe-tools` payload.
type ToolProbeResult struct {
	Model         string   `json:"model"`
	SupportsTools bool     `json:"supports_tools"`
	ToolCalls     []string `json:"tool_calls,omitempty"`
	Evidence      string   `json:"evidence"`
	Error         string   `json:"error,omitempty"`
}

// CheckStatus is a per-check verdict.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
	StatusSkip CheckStatus = "skip"
)

// DoctorCheck is one named check within a model's doctor result.
type DoctorCheck struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

// DoctorModelResult is the per-(role/model) doctor verdict.
type DoctorModelResult struct {
	Role                 string        `json:"role,omitempty"`
	Model                string        `json:"model"`
	Pass                 bool          `json:"pass"`
	Installed            bool          `json:"installed"`
	RequiresTools        bool          `json:"requires_tools"`
	StubTemplate         bool          `json:"stub_template"`
	RequiredCapabilities []string      `json:"required_capabilities,omitempty"`
	LiveCapabilities     []string      `json:"live_capabilities,omitempty"`
	Checks               []DoctorCheck `json:"checks"`
	Reasons              []string      `json:"reasons,omitempty"`
}

// DoctorResult is the `models doctor` payload.
type DoctorResult struct {
	Pass   bool                `json:"pass"`
	Models []DoctorModelResult `json:"models"`
}

// List returns the installed model references via the daemon SSOT.
func List(ctx context.Context, d Daemon) (ListResult, error) {
	refs, err := d.ListModels(ctx)
	if err != nil {
		return ListResult{}, err
	}
	if refs == nil {
		refs = []string{}
	}
	return ListResult{Models: refs}, nil
}

// ProbeTools runs a live /api/chat write-file tool smoke against a model and
// reports whether it emitted structured tool_calls. This is the behavioral
// authority for "can this model tool-call".
func ProbeTools(ctx context.Context, d Daemon, model string) (ToolProbeResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return ToolProbeResult{}, fmt.Errorf("model is required")
	}
	resp, err := d.Chat(ctx, toolSmokeRequest(model))
	if err != nil {
		return ToolProbeResult{
			Model:    model,
			Evidence: "live /api/chat tool smoke could not be completed",
			Error:    err.Error(),
		}, err
	}
	calls := make([]string, 0, len(resp.Message.ToolCalls))
	for _, tc := range resp.Message.ToolCalls {
		if name := strings.TrimSpace(tc.Function.Name); name != "" {
			calls = append(calls, name)
		}
	}
	res := ToolProbeResult{
		Model:         model,
		SupportsTools: len(calls) > 0,
		ToolCalls:     calls,
	}
	if res.SupportsTools {
		res.Evidence = fmt.Sprintf("model emitted %d structured tool_call(s): %s", len(calls), strings.Join(calls, ", "))
	} else {
		res.Evidence = "model returned a chat response with no structured tool_calls (a tool call narrated as text would fail to execute)"
	}
	return res, nil
}

// toolSmokeRequest builds the deterministic write-file tool smoke. Temperature
// 0 keeps the probe reproducible.
func toolSmokeRequest(model string) ensure.ChatRequest {
	zero := 0.0
	maxTok := 256
	return ensure.ChatRequest{
		Model: model,
		Messages: []ensure.ChatMessage{
			{Role: "system", Content: "You are a coding agent. When asked to create or modify a file, you MUST call the write_file tool. Do not describe the call in prose."},
			{Role: "user", Content: "Create a file named hello.txt containing the text \"hi\" in the current directory. Use the write_file tool."},
		},
		Temperature: &zero,
		NumPredict:  &maxTok,
		Tools: []ensure.ChatTool{{
			Type: "function",
			Function: ensure.ChatToolFunction{
				Name:        "write_file",
				Description: "Write text content to a file at the given path.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":    map[string]any{"type": "string", "description": "File path to write."},
						"content": map[string]any{"type": "string", "description": "File contents."},
					},
					"required": []string{"path", "content"},
				},
			},
		}},
	}
}

// DoctorOptions selects which roles to validate.
type DoctorOptions struct {
	Roles []string // explicit roles; ignored when All is set
	All   bool     // validate every role in policy
}

// Doctor validates roles against the live daemon: required-capability subset,
// stub-template flag, and (for tool-requiring roles) the live tool-call smoke.
// The returned DoctorResult.Pass is the AND of every model's Pass.
func Doctor(ctx context.Context, d Daemon, p policy.Policy, opts DoctorOptions) (DoctorResult, error) {
	roles := opts.Roles
	if opts.All || len(roles) == 0 {
		roles = p.RoleNames()
	}
	installed, err := d.ListModels(ctx)
	if err != nil {
		return DoctorResult{}, fmt.Errorf("list installed models: %w", err)
	}
	installedSet := make(map[string]bool, len(installed))
	for _, m := range installed {
		installedSet[m] = true
	}

	result := DoctorResult{Pass: true}
	for _, roleName := range roles {
		role, ok := p.Roles[roleName]
		if !ok {
			return DoctorResult{}, fmt.Errorf("unknown role %q", roleName)
		}
		mr := doctorRole(ctx, d, roleName, role, installedSet)
		if !mr.Pass {
			result.Pass = false
		}
		result.Models = append(result.Models, mr)
	}
	return result, nil
}

func doctorRole(ctx context.Context, d Daemon, roleName string, role policy.Role, installed map[string]bool) DoctorModelResult {
	mr := DoctorModelResult{
		Role:                 roleName,
		Model:                role.Model,
		Pass:                 true,
		RequiresTools:        roleRequiresTools(role),
		RequiredCapabilities: append([]string{}, role.RequiredCapabilities...),
	}

	// 1. Installed?
	if !installed[role.Model] && !installed[withLatestTag(role.Model)] {
		mr.Installed = false
		mr.Pass = false
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "installed", Status: StatusFail, Detail: "model is not pulled on this host"})
		mr.Reasons = append(mr.Reasons, fmt.Sprintf("model %q is not installed (run `ollama pull %s`)", role.Model, role.Model))
		return mr
	}
	mr.Installed = true
	mr.Checks = append(mr.Checks, DoctorCheck{Name: "installed", Status: StatusPass})

	// 2. Live /api/show: capabilities + template.
	show, err := d.ShowModel(ctx, role.Model)
	if err != nil {
		// Infra error — cannot prove the model is bad. Skip (do not fail).
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "show", Status: StatusSkip, Detail: err.Error()})
		return mr
	}
	mr.LiveCapabilities = append([]string{}, show.Capabilities...)

	// 3. Required-capability subset (mapped policy vocab -> Ollama vocab).
	if missing := missingCapabilities(role.RequiredCapabilities, show.Capabilities); len(missing) > 0 {
		mr.Pass = false
		mr.Checks = append(mr.Checks, DoctorCheck{
			Name: "capabilities", Status: StatusFail,
			Detail: fmt.Sprintf("live model lacks required capability(ies): %s", strings.Join(missing, ", ")),
		})
		mr.Reasons = append(mr.Reasons, fmt.Sprintf("required capabilities %v not all satisfied by live capabilities %v", role.RequiredCapabilities, show.Capabilities))
	} else {
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "capabilities", Status: StatusPass})
	}

	// 4. Stub-template flag (ADVISORY — never the sole cause of failure).
	mr.StubTemplate = isStubTemplate(show.Template)
	if mr.StubTemplate {
		mr.Checks = append(mr.Checks, DoctorCheck{
			Name: "template", Status: StatusWarn,
			Detail: "model ships a stub prompt template (no message/system/tool rendering); relying on Ollama's native tool path",
		})
	} else {
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "template", Status: StatusPass})
	}

	// 5. Live tool-call smoke (AUTHORITATIVE) — only for tool-requiring roles.
	if !mr.RequiresTools {
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "tool_smoke", Status: StatusSkip, Detail: "role does not require tool-calling"})
		return mr
	}
	probe, probeErr := ProbeTools(ctx, d, role.Model)
	switch {
	case probeErr != nil:
		// Infra error reaching the daemon — skip, do not fail.
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "tool_smoke", Status: StatusSkip, Detail: probeErr.Error()})
	case probe.SupportsTools:
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "tool_smoke", Status: StatusPass, Detail: probe.Evidence})
	default:
		mr.Pass = false
		mr.Checks = append(mr.Checks, DoctorCheck{Name: "tool_smoke", Status: StatusFail, Detail: probe.Evidence})
		mr.Reasons = append(mr.Reasons, fmt.Sprintf("model %q failed the live tool-call smoke for tool role %q: %s", role.Model, roleName, probe.Evidence))
	}
	return mr
}

// ToolRoles returns the policy role names whose required capabilities include
// tool-calling — the role-keyed scope the fail-closed admission gate enforces.
func ToolRoles(p policy.Policy) []string {
	var out []string
	for name, role := range p.Roles {
		if roleRequiresTools(role) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// roleRequiresTools reports whether a role's declared required capabilities
// include tool-calling. This is the role-keyed scope of the admission gate —
// keyed to the role, never to a specific model.
func roleRequiresTools(role policy.Role) bool {
	for _, c := range role.RequiredCapabilities {
		if strings.EqualFold(strings.TrimSpace(c), ToolCapability) {
			return true
		}
	}
	return false
}

// ollamaCapabilityAlias maps policy-vocabulary capabilities (generate, chat,
// code, …) to the Ollama /api/show capability vocabulary (completion, tools,
// embedding, …). Unknown capabilities map to themselves.
var ollamaCapabilityAlias = map[string]string{
	"generate":  "completion",
	"chat":      "completion",
	"code":      "completion",
	"summarize": "completion",
	"rerank":    "completion",
	"classify":  "completion",
	"tools":     "tools",
	"embedding": "embedding",
	"vision":    "vision",
	"thinking":  "thinking",
}

func mapPolicyCapability(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	if mapped, ok := ollamaCapabilityAlias[c]; ok {
		return mapped
	}
	return c
}

// missingCapabilities returns the required capabilities (in policy vocab) whose
// mapped Ollama capability is absent from the live capability set.
func missingCapabilities(required, live []string) []string {
	liveSet := make(map[string]bool, len(live))
	for _, c := range live {
		liveSet[strings.ToLower(strings.TrimSpace(c))] = true
	}
	var missing []string
	for _, req := range required {
		if !liveSet[mapPolicyCapability(req)] {
			missing = append(missing, req)
		}
	}
	sort.Strings(missing)
	return missing
}

// isStubTemplate reports whether a model's prompt template is a stub that only
// echoes the prompt (e.g. `{{ .Prompt }}`) rather than rendering structured
// messages/system/tools. Advisory only.
func isStubTemplate(template string) bool {
	t := strings.TrimSpace(template)
	if t == "" {
		return true
	}
	// A real chat/tool template references the message stream, a system slot,
	// or iterates messages/tools.
	for _, marker := range []string{".Messages", ".System", ".Tools", ".ToolCalls", "range"} {
		if strings.Contains(t, marker) {
			return false
		}
	}
	return true
}

func withLatestTag(ref string) string {
	if strings.Contains(ref, ":") {
		return ref
	}
	return ref + ":latest"
}
