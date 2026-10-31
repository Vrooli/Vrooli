package providers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	ProviderOllama       = "ollama"
	ProviderOpenRouter   = "openrouter"
	ProviderOrcaRouter   = "orcarouter"
	ProviderAbliteration = "abliteration"
	ProviderMetered      = "lpbs"
	ProviderPiiDetector  = "pii-detector"
	// ProviderOllaya is the decision-model resource: the local Ollaya daemon
	// plus the credential-gated hosted System One upstreams it owns.
	ProviderOllaya = "ollaya"
)

// Provider families describe the wire contract a resource implements. The
// gateway switches on the family, not on the provider id, so adding another
// OpenAI-compatible resource is one registry entry rather than another branch.
const (
	FamilyOllama           = "ollama"
	FamilyOpenAICompatible = "openai-compatible"
	FamilyPII              = "pii-detector"
	FamilyDecision         = "decision"
	FamilyMetered          = "metered"
)

// ProviderSpec is the static registration record for one provider resource.
type ProviderSpec struct {
	ID           string
	CommandName  string
	Locality     string
	Family       string
	Text         bool
	SpendCapable bool
}

// DefaultProviderSpecs returns the resource-backed providers the gateway knows
// how to execute. Adding a provider is an entry here plus its resource.
func DefaultProviderSpecs() []ProviderSpec {
	return []ProviderSpec{
		{ID: ProviderOllama, CommandName: "resource-ollama", Locality: "local", Family: FamilyOllama, Text: true},
		{ID: ProviderOpenRouter, CommandName: "resource-openrouter", Locality: "remote", Family: FamilyOpenAICompatible, Text: true, SpendCapable: true},
		{ID: ProviderOrcaRouter, CommandName: "resource-orcarouter", Locality: "remote", Family: FamilyOpenAICompatible, Text: true},
		{ID: ProviderAbliteration, CommandName: "resource-abliteration", Locality: "remote", Family: FamilyOpenAICompatible, Text: true},
		{ID: ProviderPiiDetector, CommandName: "resource-pii-detector", Locality: "local", Family: FamilyPII},
		{ID: ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Family: FamilyDecision},
	}
}

// SpecFor returns the registered spec for a provider id.
func SpecFor(provider string) (ProviderSpec, bool) {
	provider = strings.TrimSpace(provider)
	for _, spec := range DefaultProviderSpecs() {
		if spec.ID == provider {
			return spec, true
		}
	}
	return ProviderSpec{}, false
}

// TextProviderIDs returns the provider ids admissible for typed-value roles.
func TextProviderIDs() map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range DefaultProviderSpecs() {
		if spec.Text {
			out[spec.ID] = struct{}{}
		}
	}
	return out
}

// AllowedCommands returns the resource commands the runner may execute.
func AllowedCommands() map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range DefaultProviderSpecs() {
		out[spec.CommandName] = struct{}{}
	}
	return out
}

// Typed provider-limit codes. They are the provider-neutral seam between a
// native provider failure and the routing classifier so an account-credit
// exhaustion, a rate limit, or an overload is not flattened into a generic
// execution failure. The observed recovery window (Retry-After/reset) rides on
// CommandError beside the code.
const (
	CodeInsufficientCredits = "insufficient_credits"
	CodeRateLimited         = "rate_limited"
	CodeProviderOverloaded  = "provider_overloaded"
	CodeProviderFailed      = "provider_failed"
	// CodeStreamFailed is a provider response whose body failed mid-read after
	// an HTTP 2xx. It is transient, so it is classified like an overload.
	CodeStreamFailed = "stream_failed"
	// CodeUnreachable is a provider command that could not reach the provider at
	// all. It is an availability condition, not a request fault.
	CodeUnreachable   = "unreachable"
	CodeCommandFailed = "command_failed"
	// CodeUnconfigured is a provider role that cannot serve on this host yet:
	// its credential is not provisioned or its model is not pulled. The caller
	// skips the candidate; nothing failed.
	CodeUnconfigured = "unconfigured"
	// CodeContextOverflow is a request the provider would have to truncate to
	// answer; the provider refused rather than answer on partial input.
	CodeContextOverflow = "context_overflow"
)

type Adapter struct {
	Provider    string
	CommandName string
	Locality    string
	Family      string
	Runner      CommandRunner
	Metered     *MeteredClient
}

type Role struct {
	Provider        string
	Role            string
	Capabilities    []string
	InputModalities []sharedv1.Modality
	// Locality and Status are the resource's per-role declaration when it
	// makes one, else the adapter's: one resource may serve local and hosted
	// roles, and a hosted role may be unconfigured on this host.
	Locality            string
	Status              string
	PolicySchemaVersion string
}

type SmokeResult struct {
	Provider string
	Status   string
	Code     string
	Message  string
	ExitCode int
	Warnings []string
}

type Inventory struct {
	Roles    []Role
	Warnings []string
}

type ExecutionRequest struct {
	Kind            sharedv1.RequestKind
	Role            string
	Profile         sharedv1.Profile
	InputText       string
	MaxOutputTokens int32
	Timeout         time.Duration
	// Temperature requests a specific sampling temperature. It is a pointer so
	// that an explicit 0 (deterministic) is distinguishable from "unset, use
	// the resource default". Whether the resolved role's provider actually
	// applies it is a policy declaration, not something this request can know:
	// see ResolvedRole.TemperatureSupport.
	Temperature *float64
	// SchemaJSON is passed through the provider's native structured-output
	// request field. The gateway still validates the returned value locally.
	SchemaJSON           string
	Attachments          []*sharedv1.Attachment
	SpendLimit           *inferencev1.SpendLimit
	ExpectedWorkerDigest string
}

type ExecutionResult struct {
	OutputText         string
	ExitCode           int
	SpendAuthorization *inferencev1.SpendAuthorization
}

type ResolvedRole struct {
	Provider             string
	Role                 string
	Model                string
	CanonicalModel       string
	Capabilities         []string
	CoordinateConvention string
	// SamplingSupport is the resource role's per-parameter declaration of how
	// its provider treats an explicit sampling control. An absent entry means
	// SamplingUnknown.
	SamplingSupport map[string]SamplingSupport
	// MaxOutputTokens is the cap the resource role declares for callers that
	// send none, or 0 when the role declares none. Zero is not "unknown": it
	// means the resource imposes nothing and the provider's own default applies.
	MaxOutputTokens int32
	// ContextWindow is provider policy, not caller metadata. Zero means the
	// provider did not declare one and the gateway must not invent a refusal.
	ContextWindow int32
	// Locality is where the resolved role executes: the resource's per-role
	// declaration, else the adapter's locality.
	Locality string
}

// SamplingSupport states how a resolved role's provider treats an explicit
// sampling control. It is read from resource policy and never probed.
//
// Probing cannot work, and that is the whole reason this is a declaration:
// there are three real states and only one is detectable at runtime. A
// rejection surfaces as an error, but a provider that accepts a control and
// silently discards it is indistinguishable at the call site from one that
// honours it — a probe would report success and be wrong.
type SamplingSupport string

const (
	// SamplingUnknown means the role declared nothing. Callers treat it as
	// SamplingIgnored: best effort, no promise.
	SamplingUnknown SamplingSupport = "unknown"
	// SamplingHonored means the provider applies the value.
	SamplingHonored SamplingSupport = "honored"
	// SamplingIgnored means the provider accepts the field and discards it.
	SamplingIgnored SamplingSupport = "ignored"
	// SamplingRejected means the provider fails the request when the field is
	// present, so the gateway must omit it or route elsewhere.
	SamplingRejected SamplingSupport = "rejected"
)

// TemperatureSupport reports how the resolved role's provider treats an
// explicit temperature. It is read from resource policy, never probed.
//
// The metered provider is always SamplingRejected rather than SamplingUnknown,
// and the distinction is real: LPBS has no temperature field on its wire
// contract at all, so the gateway cannot transmit the value even in principle.
// That is provider incapacity, not an undeclared preference.
func (r ResolvedRole) TemperatureSupport() SamplingSupport {
	if r.Provider == ProviderMetered {
		return SamplingRejected
	}
	return r.samplingSupport("temperature")
}

func (r ResolvedRole) samplingSupport(parameter string) SamplingSupport {
	switch state := r.SamplingSupport[parameter]; state {
	case SamplingHonored, SamplingIgnored, SamplingRejected:
		return state
	default:
		return SamplingUnknown
	}
}

type rolePolicyReport struct {
	Roles []rolePolicyEntry `json:"roles"`
}

type rolePolicyEntry struct {
	SchemaVersion        string   `json:"schema_version"`
	Role                 string   `json:"role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Capabilities         []string `json:"capabilities"`
	Modalities           struct {
		Input []string `json:"input"`
	} `json:"modalities"`
	Locality string `json:"locality"`
	Status   string `json:"status"`
}

func NewDefaultAdapters(runner CommandRunner) []Adapter {
	if runner == nil {
		runner = ExecRunner{}
	}
	specs := DefaultProviderSpecs()
	adapters := make([]Adapter, 0, len(specs))
	for _, spec := range specs {
		adapters = append(adapters, Adapter{
			Provider:    spec.ID,
			CommandName: spec.CommandName,
			Locality:    spec.Locality,
			Family:      spec.Family,
			Runner:      runner,
		})
	}
	return adapters
}

func (a Adapter) family() string {
	if strings.TrimSpace(a.Family) != "" {
		return a.Family
	}
	if spec, ok := SpecFor(a.Provider); ok {
		return spec.Family
	}
	return ""
}

func (a Adapter) ListRoles(ctx context.Context) (Inventory, error) {
	if a.Provider == ProviderMetered {
		if a.Metered == nil {
			return Inventory{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
		}
		return Inventory{Roles: []Role{{Provider: a.Provider, Role: "metered", Locality: "remote", Status: "available"}}}, nil
	}
	result, err := a.runPolicyRoles(ctx)
	if err != nil {
		return Inventory{}, mapCommandError(a.Provider, err)
	}

	var report rolePolicyReport
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		return Inventory{}, &CommandError{
			Code:     "malformed_json",
			Command:  a.command().String(),
			ExitCode: result.ExitCode,
			Stderr:   result.Stderr,
			Err:      err,
		}
	}

	inventory := Inventory{Roles: make([]Role, 0, len(report.Roles))}
	for _, entry := range report.Roles {
		roleName := strings.TrimSpace(entry.Role)
		if roleName == "" {
			inventory.Warnings = append(inventory.Warnings, fmt.Sprintf("%s policy returned a role without a name", a.Provider))
			continue
		}
		capabilities := entry.RequiredCapabilities
		if len(capabilities) == 0 {
			capabilities = entry.Capabilities
		}
		if len(capabilities) == 0 {
			inventory.Warnings = append(inventory.Warnings, fmt.Sprintf("%s role %q has no declared capabilities", a.Provider, roleName))
		}
		inventory.Roles = append(inventory.Roles, Role{
			Provider:            a.Provider,
			Role:                roleName,
			Capabilities:        sortedUnique(capabilities),
			InputModalities:     parseModalities(entry.Modalities.Input),
			Locality:            firstNonEmpty(entry.Locality, a.Locality),
			Status:              firstNonEmpty(entry.Status, "available"),
			PolicySchemaVersion: strings.TrimSpace(entry.SchemaVersion),
		})
	}
	sort.Slice(inventory.Roles, func(i, j int) bool {
		return inventory.Roles[i].Role < inventory.Roles[j].Role
	})
	return inventory, nil
}

// ResolveRole asks the resource for the concrete model selected by its
// policy. AI Gateway never reads provider model catalogs or credentials.
func (a Adapter) ResolveRole(ctx context.Context, role string) (ResolvedRole, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("role is required")}
	}
	if a.Provider == ProviderMetered {
		if a.Metered == nil {
			return ResolvedRole{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
		}
		return ResolvedRole{Provider: a.Provider, Role: role, Model: "policy-resolved", CanonicalModel: "policy-resolved"}, nil
	}
	result, err := a.runner().Run(ctx, Command{Name: a.CommandName, Args: []string{"policy", "resolve", "--role", role, "--json"}, Timeout: DefaultCommandTimeout})
	if err != nil {
		return ResolvedRole{}, mapCommandError(a.Provider, err)
	}
	// The resources emit sampling and cap declarations alongside the model. The
	// vocabulary differs by provider — ollama says sampling_defaults/max_tokens
	// at role level, openrouter nests its cap under request_defaults — so both
	// shapes are decoded here and normalised into ResolvedRole.
	var response struct {
		Role                 string            `json:"role"`
		Model                string            `json:"model"`
		Canonical            string            `json:"canonical_model"`
		Capabilities         []string          `json:"capabilities"`
		CoordinateConvention string            `json:"coordinate_convention"`
		SamplingSupport      map[string]string `json:"sampling_support"`
		MaxTokens            *int32            `json:"max_tokens"`
		RequestDefaults      *struct {
			MaxTokens *int32 `json:"max_tokens"`
		} `json:"request_defaults"`
		ContextWindow *int32 `json:"context_window"`
		Locality      string `json:"locality"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return ResolvedRole{}, &CommandError{Code: "malformed_json", Command: a.command().String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	if strings.TrimSpace(response.Role) != role || strings.TrimSpace(response.Model) == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_role_resolution", Command: a.command().String(), ExitCode: result.ExitCode, Err: fmt.Errorf("resource returned role %q and model %q for requested role %q", response.Role, response.Model, role)}
	}
	var maxOutputTokens int32
	switch {
	case response.MaxTokens != nil:
		maxOutputTokens = *response.MaxTokens
	case response.RequestDefaults != nil && response.RequestDefaults.MaxTokens != nil:
		maxOutputTokens = *response.RequestDefaults.MaxTokens
	}
	return ResolvedRole{
		Provider:             a.Provider,
		Role:                 role,
		Model:                response.Model,
		CanonicalModel:       response.Canonical,
		Capabilities:         sortedUnique(response.Capabilities),
		CoordinateConvention: strings.TrimSpace(response.CoordinateConvention),
		SamplingSupport:      parseSamplingSupport(response.SamplingSupport),
		MaxOutputTokens:      maxOutputTokens,
		ContextWindow:        valueOrZero(response.ContextWindow),
		Locality:             firstNonEmpty(response.Locality, a.Locality),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func valueOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

// parseSamplingSupport normalises the resource's declared states. An
// unrecognised value degrades to SamplingUnknown rather than failing the
// resolution: the resources validate their own vocabulary on load, and a
// gateway that refused to route on an unfamiliar string would turn a future
// resource-side addition into a fleet outage.
func parseSamplingSupport(declared map[string]string) map[string]SamplingSupport {
	if len(declared) == 0 {
		return nil
	}
	out := make(map[string]SamplingSupport, len(declared))
	for parameter, state := range declared {
		switch SamplingSupport(strings.ToLower(strings.TrimSpace(state))) {
		case SamplingHonored:
			out[parameter] = SamplingHonored
		case SamplingIgnored:
			out[parameter] = SamplingIgnored
		case SamplingRejected:
			out[parameter] = SamplingRejected
		default:
			out[parameter] = SamplingUnknown
		}
	}
	return out
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (a Adapter) Smoke(ctx context.Context) SmokeResult {
	inventory, err := a.ListRoles(ctx)
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			return SmokeResult{
				Provider: a.Provider,
				Status:   "unavailable",
				Code:     cmdErr.Code,
				Message:  cmdErr.Error(),
				ExitCode: cmdErr.ExitCode,
			}
		}
		return SmokeResult{Provider: a.Provider, Status: "unavailable", Code: "unknown", Message: err.Error(), ExitCode: -1}
	}
	warnings := append([]string{}, inventory.Warnings...)
	if len(inventory.Roles) == 0 {
		warnings = append(warnings, "provider policy returned zero roles")
		return SmokeResult{Provider: a.Provider, Status: "degraded", Code: "empty_inventory", Message: "provider command responded but returned no roles", Warnings: warnings}
	}
	return SmokeResult{Provider: a.Provider, Status: "available", Code: "ok", Message: fmt.Sprintf("provider policy returned %d role(s)", len(inventory.Roles)), Warnings: warnings}
}

func (a Adapter) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if a.Provider == ProviderMetered {
		if a.Metered == nil {
			return ExecutionResult{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
		}
		images := make([]MeteredImage, 0, len(req.Attachments))
		for _, attachment := range req.Attachments {
			if attachment == nil || attachment.GetInlineBytes() == nil {
				return ExecutionResult{}, errors.New("metered route requires inline image references")
			}
			images = append(images, MeteredImage{Data: base64.StdEncoding.EncodeToString(attachment.GetInlineBytes()), MediaType: attachment.GetMediaType(), SHA256: digestBytes(attachment.GetInlineBytes())})
		}
		result, err := a.Metered.Run(ctx, MeteredRequest{
			Role: req.Role, Messages: []MeteredMessage{{Role: "user", Content: req.InputText}},
			ConstraintsJSON: req.SchemaJSON, MaxTokens: int(req.MaxOutputTokens),
			Profile: req.Profile, SpendLimit: req.SpendLimit, Images: images,
		})
		if err != nil {
			return ExecutionResult{}, mapCommandError(a.Provider, err)
		}
		encoded, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return ExecutionResult{}, fmt.Errorf("encode metered inference result: %w", marshalErr)
		}
		return ExecutionResult{OutputText: string(encoded), SpendAuthorization: result.SpendAuthorization}, nil
	}
	command, err := a.executionCommand(req)
	if err != nil {
		return ExecutionResult{}, err
	}
	result, err := a.runner().Run(ctx, command)
	if err != nil {
		return ExecutionResult{ExitCode: result.ExitCode}, mapCommandError(a.Provider, err)
	}
	output := strings.TrimSpace(result.Stdout)
	var spendAuthorization *inferencev1.SpendAuthorization
	if a.Provider == ProviderOpenRouter && req.SpendLimit != nil {
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal([]byte(output), &envelope); err != nil {
			return ExecutionResult{}, fmt.Errorf("decode OpenRouter capped response: %w", err)
		}
		authorizationJSON := envelope["vrooli_spend_authorization"]
		if len(authorizationJSON) == 0 {
			return ExecutionResult{}, errors.New("OpenRouter owner omitted spend authorization evidence")
		}
		spendAuthorization = &inferencev1.SpendAuthorization{}
		if err := json.Unmarshal(authorizationJSON, spendAuthorization); err != nil {
			return ExecutionResult{}, fmt.Errorf("decode OpenRouter spend authorization: %w", err)
		}
		delete(envelope, "vrooli_spend_authorization")
		clean, err := json.Marshal(envelope)
		if err != nil {
			return ExecutionResult{}, err
		}
		output = string(clean)
	}
	return ExecutionResult{OutputText: output, ExitCode: result.ExitCode, SpendAuthorization: spendAuthorization}, nil
}

func (a Adapter) runPolicyRoles(ctx context.Context) (Result, error) {
	return a.runner().Run(ctx, a.command())
}

func (a Adapter) command() Command {
	return Command{Name: a.CommandName, Args: []string{"policy", "roles", "--json"}, Timeout: DefaultCommandTimeout}
}

func (a Adapter) runner() CommandRunner {
	if a.Runner == nil {
		return ExecRunner{}
	}
	return a.Runner
}

func (a Adapter) executionCommand(req ExecutionRequest) (Command, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("role is required")}
	}
	input := req.InputText
	if strings.TrimSpace(input) == "" {
		return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("input_text is required")}
	}
	switch a.family() {
	case FamilyOllama:
		return a.ollamaExecutionCommand(req, role, input, timeout)
	case FamilyOpenAICompatible:
		return a.openAICompatibleExecutionCommand(req, role, input, timeout)
	case FamilyPII:
		return a.piiDetectorExecutionCommand(req, role, input, timeout)
	case FamilyDecision:
		return a.ollayaExecutionCommand(req, role, input, timeout)
	default:
		return Command{}, &CommandError{Code: "unsupported_provider", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("provider %q does not expose a gateway execution command", a.Provider)}
	}
}

func (a Adapter) ollamaExecutionCommand(req ExecutionRequest, role string, input string, timeout time.Duration) (Command, error) {
	switch req.Kind {
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING:
		return Command{
			Name:    a.CommandName,
			Args:    []string{"gateway", "embed", "--role", role, "--json", "--input-stdin"},
			Stdin:   input,
			Timeout: timeout,
		}, nil
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION:
		stdin, jsonEnvelope, err := multimodalStdin(input, req.Attachments)
		if err != nil {
			return Command{}, err
		}
		args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
		if jsonEnvelope {
			args = []string{"gateway", "generate", "--role", role, "--json", "--input-json-stdin"}
		}
		if req.Profile == sharedv1.Profile_PROFILE_LOCAL_ONLY {
			args = append(args, "--require-loopback")
		}
		if digest := strings.TrimSpace(req.ExpectedWorkerDigest); digest != "" {
			args = append(args, "--require-worker-digest", digest)
		}
		if req.MaxOutputTokens > 0 {
			args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
		}
		if req.Temperature != nil {
			args = append(args, "--temperature", strconv.FormatFloat(*req.Temperature, 'g', -1, 64))
		}
		// Ollama's qwen3-vl template returns an empty visible response when
		// /api/generate receives a structured-output format alongside images.
		// locate.visual still carries its provider schema in the prompt and the
		// gateway remains the authoritative validator, so omit the native format
		// only for this image role. Text structured inference keeps native format.
		if strings.TrimSpace(req.SchemaJSON) != "" && strings.TrimSpace(req.Role) != "vision.default" {
			if !json.Valid([]byte(req.SchemaJSON)) {
				return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("schema_json must be valid JSON")}
			}
			args = append(args, "--format", req.SchemaJSON)
		}
		return Command{Name: a.CommandName, Args: args, Stdin: stdin, Timeout: timeout}, nil
	default:
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable", req.Kind.String())}
	}
}

func (a Adapter) openAICompatibleExecutionCommand(req ExecutionRequest, role string, input string, timeout time.Duration) (Command, error) {
	if req.Kind != sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION &&
		req.Kind != sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION {
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable by %s", req.Kind.String(), a.Provider)}
	}
	spec, _ := SpecFor(a.Provider)
	if req.SpendLimit != nil && !spec.SpendCapable {
		return Command{}, &CommandError{Code: "unsupported_spend_cap", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("provider %q does not support spend-capped execution", a.Provider)}
	}
	args := []string{"generate", "--role", role, "--json"}
	stdin, jsonEnvelope, err := multimodalStdin(input, req.Attachments)
	if err != nil {
		return Command{}, err
	}
	if jsonEnvelope {
		args = append(args, "--input-json-stdin")
	} else {
		stdin = input
	}
	if req.MaxOutputTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
	}
	if req.SpendLimit != nil && spec.SpendCapable {
		args = append(args, "--max-cost-micros", strconv.FormatInt(req.SpendLimit.GetMaxCostMicros(), 10),
			"--operation-id", req.SpendLimit.GetOperationId(),
			"--input-fingerprint", req.SpendLimit.GetInputFingerprint(),
			"--funding-route", req.SpendLimit.GetFundingRoute())
		args = append(args, "--reference-sha256", strings.Join(req.SpendLimit.GetReferenceSha256(), ","))
		args = append(args, "--image-count", strconv.FormatInt(int64(req.SpendLimit.GetImageCount()), 10))
	}
	// Absent this flag the resource falls through to its role's declared
	// request_defaults.temperature. Passing it only when the gateway actually
	// has a value keeps that fallback reachable instead of pinning every call.
	if req.Temperature != nil {
		args = append(args, "--temperature", strconv.FormatFloat(*req.Temperature, 'g', -1, 64))
	}
	if strings.TrimSpace(req.SchemaJSON) != "" {
		if !json.Valid([]byte(req.SchemaJSON)) {
			return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("schema_json must be valid JSON")}
		}
		format := fmt.Sprintf(`{"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":%s}}`, req.SchemaJSON)
		args = append(args, "--response-format", format)
	}
	return Command{Name: a.CommandName, Args: args, Stdin: stdin, Timeout: timeout}, nil
}

// piiDetectorExecutionCommand maps a DETECT_PII request to the pii-detector
// resource CLI gateway. The CLI emits the detect result ({spans,...}) as JSON on
// stdout, which Execute returns verbatim as OutputText. The role is carried for
// provenance only; the detector's active model (policy/role) governs the NER
// pass, and the deterministic regex/checksum pass always runs.
func (a Adapter) piiDetectorExecutionCommand(req ExecutionRequest, _ string, input string, timeout time.Duration) (Command, error) {
	if req.Kind != sharedv1.RequestKind_REQUEST_KIND_DETECT_PII {
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable by pii-detector", req.Kind.String())}
	}
	return Command{
		Name:    a.CommandName,
		Args:    []string{"gateway", "detect", "--text-stdin", "--json"},
		Stdin:   input,
		Timeout: timeout,
	}, nil
}

// ollayaExecutionCommand maps a DECISION request to the decision-model
// resource. InputText is the System One body ({state, questions}); the resource
// picks the upstream and model from its role policy and prints a normalized
// answer set. The deadline is passed down so a slow model load surfaces as the
// resource's own typed timeout rather than a killed process.
func (a Adapter) ollayaExecutionCommand(req ExecutionRequest, role string, input string, timeout time.Duration) (Command, error) {
	if req.Kind != sharedv1.RequestKind_REQUEST_KIND_DECISION {
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable by ollaya", req.Kind.String())}
	}
	return Command{
		Name:    a.CommandName,
		Args:    []string{"gateway", "decide", "--role", role, "--input-stdin", "--json", "--timeout", timeout.String()},
		Stdin:   input,
		Timeout: timeout + resourceCommandGrace,
	}, nil
}

// resourceCommandGrace lets a resource report its own deadline before the
// runner kills it.
const resourceCommandGrace = 2 * time.Second

type multimodalEnvelope struct {
	Prompt string `json:"prompt"`
	Images []struct {
		MediaType string `json:"media_type"`
		DataB64   string `json:"data_b64"`
	} `json:"images"`
}

func multimodalStdin(prompt string, attachments []*sharedv1.Attachment) (string, bool, error) {
	if len(attachments) == 0 {
		return prompt, false, nil
	}
	envelope := multimodalEnvelope{Prompt: prompt}
	for _, attachment := range attachments {
		if attachment == nil || len(attachment.GetInlineBytes()) == 0 {
			return "", false, &CommandError{Code: "attachment_unresolved", Command: "provider execution", ExitCode: -1, Err: errors.New("provider execution requires resolved inline attachment bytes")}
		}
		mediaType := strings.TrimSpace(attachment.GetMediaType())
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		envelope.Images = append(envelope.Images, struct {
			MediaType string `json:"media_type"`
			DataB64   string `json:"data_b64"`
		}{MediaType: mediaType, DataB64: base64.StdEncoding.EncodeToString(attachment.GetInlineBytes())})
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", false, fmt.Errorf("marshal multimodal request: %w", err)
	}
	return string(raw), true, nil
}

func parseModalities(values []string) []sharedv1.Modality {
	out := make([]sharedv1.Modality, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "text":
			out = append(out, sharedv1.Modality_MODALITY_TEXT)
		case "image":
			out = append(out, sharedv1.Modality_MODALITY_IMAGE)
		case "vector":
			out = append(out, sharedv1.Modality_MODALITY_VECTOR)
		case "video":
			out = append(out, sharedv1.Modality_MODALITY_VIDEO)
		case "audio":
			out = append(out, sharedv1.Modality_MODALITY_AUDIO)
		}
	}
	return out
}

func mapCommandError(provider string, err error) error {
	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code == "" {
			cmdErr.Code = "command_failed"
		}
		return cmdErr
	}
	return &CommandError{Code: "command_failed", Command: provider, ExitCode: -1, Err: err}
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
