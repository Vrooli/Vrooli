package models

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

// fakeDaemon is the injected daemon seam for deterministic tests.
type fakeDaemon struct {
	models     []string
	listErr    error
	shows      map[string]ensure.ShowResponse
	showErr    map[string]error
	toolNames  map[string][]string // model -> tool_call names to return
	chatErr    map[string]error
	chatImages []string
	inventory  []ensure.ModelInfo
}

func (f *fakeDaemon) ListModels(ctx context.Context) ([]string, error) {
	return f.models, f.listErr
}

func (f *fakeDaemon) ListModelInventory(ctx context.Context) ([]ensure.ModelInfo, error) {
	return f.inventory, nil
}

func (f *fakeDaemon) ShowModel(ctx context.Context, model string) (ensure.ShowResponse, error) {
	if f.showErr != nil {
		if err := f.showErr[model]; err != nil {
			return ensure.ShowResponse{}, err
		}
	}
	return f.shows[model], nil
}

func (f *fakeDaemon) Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
	for _, message := range in.Messages {
		f.chatImages = append(f.chatImages, message.Images...)
	}
	if f.chatErr != nil {
		if err := f.chatErr[in.Model]; err != nil {
			return ensure.ChatResponse{}, err
		}
	}
	var resp ensure.ChatResponse
	for _, name := range f.toolNames[in.Model] {
		var tc ensure.ChatToolCall
		tc.Function.Name = name
		resp.Message.ToolCalls = append(resp.Message.ToolCalls, tc)
	}
	if len(resp.Message.ToolCalls) == 0 {
		resp.Message.Content = `{"name":"write_file","arguments":{}}` // narrated as text
	}
	return resp, nil
}

func (f *fakeDaemon) Generate(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
	if len(in.Images) > 0 {
		return ensure.GenerateResponse{Response: "generated vision response"}, nil
	}
	return ensure.GenerateResponse{}, nil
}

func TestList(t *testing.T) {
	d := &fakeDaemon{models: []string{"gemma4:12b", "qwen3.5:9b"}}
	res, err := List(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Models) != 2 {
		t.Fatalf("models=%v", res.Models)
	}
}

func TestInventoryAttributesPolicyReachabilityAndRegeneration(t *testing.T) {
	d := &fakeDaemon{inventory: []ensure.ModelInfo{
		{Name: "orphan:latest", Digest: "sha256:orphan", Size: 11},
		{Name: "primary:latest", Digest: "sha256:primary", Size: 22},
	}}
	p := policy.Policy{Roles: map[string]policy.Role{
		"code.local": {Model: "primary:latest", Fallbacks: []string{"fallback:latest"}},
	}}
	res, err := Inventory(context.Background(), d, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Models) != 2 || res.TotalBytes != 33 {
		t.Fatalf("inventory = %+v", res)
	}
	if res.Models[0].Name != "orphan:latest" || res.Models[0].PolicyReachable {
		t.Fatalf("orphan reachability = %+v", res.Models[0])
	}
	if res.Models[0].RegenerableReason == "" || !res.Models[1].PolicyReachable {
		t.Fatalf("regeneration/reachability = %+v", res.Models)
	}
}

func TestProbeTools_StructuredCallsPass(t *testing.T) {
	d := &fakeDaemon{toolNames: map[string][]string{"gemma4:12b": {"write_file"}}}
	res, err := ProbeTools(context.Background(), d, "gemma4:12b")
	if err != nil {
		t.Fatal(err)
	}
	if !res.SupportsTools {
		t.Errorf("expected supports_tools, got %+v", res)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0] != "write_file" {
		t.Errorf("tool calls = %v", res.ToolCalls)
	}
}

func TestProbeTools_TextOnlyFails(t *testing.T) {
	d := &fakeDaemon{toolNames: map[string][]string{}} // returns no tool_calls
	res, err := ProbeTools(context.Background(), d, "weakmodel")
	if err != nil {
		t.Fatal(err)
	}
	if res.SupportsTools {
		t.Errorf("expected supports_tools=false for a text-only response, got %+v", res)
	}
}

func TestProbeTools_InfraErrorPropagates(t *testing.T) {
	d := &fakeDaemon{chatErr: map[string]error{"m": errors.New("connection refused")}}
	res, err := ProbeTools(context.Background(), d, "m")
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	if res.SupportsTools {
		t.Error("infra error must not report supports_tools")
	}
}

func TestProbeVisionSendsImageAndPassesOnResponse(t *testing.T) {
	d := &fakeDaemon{}
	res, err := ProbeVision(context.Background(), d, "gemma4:12b", []byte("image-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.SupportsVision {
		t.Fatalf("expected supports_vision, got %+v", res)
	}
	if len(d.chatImages) != 1 || d.chatImages[0] == "" {
		t.Fatalf("expected one base64 image, got %v", d.chatImages)
	}
}

// testPolicy builds a minimal in-memory policy mirroring the real shape.
func testPolicy() policy.Policy {
	return policy.Policy{
		SchemaVersion: "test",
		Roles: map[string]policy.Role{
			"code.local": {
				Model:                "gemma4:12b",
				RequiredCapabilities: []string{"generate", "code", "tools"},
			},
			"chat.default": {
				Model:                "qwen3.5:9b",
				RequiredCapabilities: []string{"generate", "chat"},
			},
			"embedding.default": {
				Model:                "nomic-embed-text:latest",
				RequiredCapabilities: []string{"embedding"},
			},
		},
		Models: map[string]policy.Model{
			"gemma4:12b":              {Capabilities: []string{"completion", "tools"}, Modalities: policy.Modalities{Input: []policy.Modality{policy.ModalityText, policy.ModalityImage}, Output: []policy.Modality{policy.ModalityText}}},
			"qwen3.5:9b":              {Capabilities: []string{"completion", "tools"}, Modalities: policy.Modalities{Input: []policy.Modality{policy.ModalityText}, Output: []policy.Modality{policy.ModalityText}}},
			"nomic-embed-text:latest": {Capabilities: []string{"embedding"}, Modalities: policy.Modalities{Input: []policy.Modality{policy.ModalityText}, Output: []policy.Modality{policy.ModalityVector}}},
		},
	}
}

// TestDoctor_StubTemplateButToolsWork is the cornerstone: gemma4's stub
// template must be FLAGGED (warn) yet the role PASSES because the live tool
// smoke succeeds (behavioral authority). This is the regression that protects
// the proven-working production model from over-blocking.
func TestDoctor_StubTemplateButToolsWork(t *testing.T) {
	d := &fakeDaemon{
		models:    []string{"gemma4:12b"},
		shows:     map[string]ensure.ShowResponse{"gemma4:12b": {Template: "{{ .Prompt }}", Capabilities: []string{"completion", "tools", "vision"}}},
		toolNames: map[string][]string{"gemma4:12b": {"write_file"}},
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected overall pass, got %+v", res.Models)
	}
	mr := res.Models[0]
	if !mr.StubTemplate {
		t.Error("expected stub template to be flagged")
	}
	if !mr.Pass {
		t.Error("expected the role to pass (tool smoke is authoritative)")
	}
	if !hasCheck(mr, "template", StatusWarn) {
		t.Error("expected a warn-level template check")
	}
	if !hasCheck(mr, "tool_smoke", StatusPass) {
		t.Error("expected a passing tool_smoke check")
	}
}

// TestDoctor_ToolSmokeFailureFails: a tool role whose model does NOT emit
// structured tool_calls must FAIL — the real silent-success failure class.
func TestDoctor_ToolSmokeFailureFails(t *testing.T) {
	d := &fakeDaemon{
		models:    []string{"gemma4:12b"},
		shows:     map[string]ensure.ShowResponse{"gemma4:12b": {Template: "{{ .Messages }}", Capabilities: []string{"completion", "tools"}}},
		toolNames: map[string][]string{}, // no tool_calls -> text-only
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail when the tool smoke returns no structured tool_calls")
	}
	if !hasCheck(res.Models[0], "tool_smoke", StatusFail) {
		t.Error("expected a failing tool_smoke check")
	}
}

// TestDoctor_MissingCapabilityFails: live model lacking a required capability.
func TestDoctor_MissingCapabilityFails(t *testing.T) {
	d := &fakeDaemon{
		models:    []string{"gemma4:12b"},
		shows:     map[string]ensure.ShowResponse{"gemma4:12b": {Template: "{{ .Messages }}", Capabilities: []string{"completion"}}}, // no tools
		toolNames: map[string][]string{"gemma4:12b": {"write_file"}},
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail when live capabilities lack 'tools'")
	}
	if !hasCheck(res.Models[0], "capabilities", StatusFail) {
		t.Error("expected a failing capabilities check")
	}
}

func TestDoctor_MissingDeclaredImageInputFails(t *testing.T) {
	d := &fakeDaemon{
		models:    []string{"gemma4:12b"},
		shows:     map[string]ensure.ShowResponse{"gemma4:12b": {Template: "{{ .Messages }}", Capabilities: []string{"completion", "tools"}}},
		toolNames: map[string][]string{"gemma4:12b": {"write_file"}},
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass {
		t.Fatal("expected fail when declared image input is absent from live capabilities")
	}
	if !hasCheck(res.Models[0], "modalities.image_input", StatusFail) {
		t.Fatal("expected a failing image-input modality check")
	}
}

// TestDoctor_NotInstalledFails: role model not pulled.
func TestDoctor_NotInstalledFails(t *testing.T) {
	d := &fakeDaemon{models: []string{"qwen3.5:9b"}} // gemma4 absent
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass || res.Models[0].Installed {
		t.Fatal("expected fail for a non-installed role model")
	}
}

// TestDoctor_NonToolRoleSkipsSmoke: embedding role passes without a tool smoke.
func TestDoctor_NonToolRoleSkipsSmoke(t *testing.T) {
	d := &fakeDaemon{
		models: []string{"nomic-embed-text:latest"},
		shows:  map[string]ensure.ShowResponse{"nomic-embed-text:latest": {Template: "{{ .Prompt }}", Capabilities: []string{"embedding"}}},
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"embedding.default"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Fatalf("expected embedding role to pass, got %+v", res.Models)
	}
	if !hasCheck(res.Models[0], "tool_smoke", StatusSkip) {
		t.Error("expected tool_smoke to be skipped for a non-tool role")
	}
}

// TestDoctor_ShowInfraErrorSkips: a daemon /api/show error must not fail the
// model (can't prove it's bad).
func TestDoctor_ShowInfraErrorSkips(t *testing.T) {
	d := &fakeDaemon{
		models:  []string{"gemma4:12b"},
		showErr: map[string]error{"gemma4:12b": errors.New("connection refused")},
	}
	res, err := Doctor(context.Background(), d, testPolicy(), DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass {
		t.Error("an /api/show infra error must skip, not fail")
	}
}

// TestDoctor_ModelSwapInheritsValidation proves the gate is role-keyed, not
// model-keyed: swapping code.local's model to a different ref re-runs the same
// validation against the new model.
func TestDoctor_ModelSwapInheritsValidation(t *testing.T) {
	p := testPolicy()
	role := p.Roles["code.local"]
	role.Model = "qwen3.5:9b" // swap the model filling the role
	p.Roles["code.local"] = role

	d := &fakeDaemon{
		models:    []string{"qwen3.5:9b"},
		shows:     map[string]ensure.ShowResponse{"qwen3.5:9b": {Template: "{{ .Messages }}", Capabilities: []string{"completion", "tools"}}},
		toolNames: map[string][]string{"qwen3.5:9b": {"write_file"}},
	}
	res, err := Doctor(context.Background(), d, p, DoctorOptions{Roles: []string{"code.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pass || res.Models[0].Model != "qwen3.5:9b" {
		t.Fatalf("swapped model must inherit the role's validation, got %+v", res.Models)
	}
}

func TestIsStubTemplate(t *testing.T) {
	cases := []struct {
		tmpl string
		stub bool
	}{
		{"{{ .Prompt }}", true},
		{"", true},
		{"{{ .System }}\n{{ .Prompt }}", false},
		{"{{ range .Messages }}{{ .Content }}{{ end }}", false},
		{"{{ if .Tools }}tools{{ end }}{{ .Prompt }}", false},
	}
	for _, tc := range cases {
		if got := isStubTemplate(tc.tmpl); got != tc.stub {
			t.Errorf("isStubTemplate(%q)=%v want %v", tc.tmpl, got, tc.stub)
		}
	}
}

func TestMissingCapabilities(t *testing.T) {
	// generate/code map to completion; tools maps to tools.
	missing := missingCapabilities([]string{"generate", "code", "tools"}, []string{"completion"})
	if len(missing) != 1 || missing[0] != "tools" {
		t.Errorf("missing=%v want [tools]", missing)
	}
	if m := missingCapabilities([]string{"generate", "code", "tools"}, []string{"completion", "tools"}); len(m) != 0 {
		t.Errorf("expected no missing, got %v", m)
	}
}

func hasCheck(mr DoctorModelResult, name string, status CheckStatus) bool {
	for _, c := range mr.Checks {
		if c.Name == name && c.Status == status {
			return true
		}
	}
	return false
}
