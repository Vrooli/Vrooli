package providers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

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
	HTTP        *HTTPProvider
	Door        *HostedDoorClient
}

type Role struct {
	Provider              string
	Role                  string
	Model                 string
	Rights                *sharedpolicy.ArtifactRights
	RuntimeIdentity       *sharedpolicy.RuntimeArtifactIdentity
	ObservedRuntimeDigest string
	Capabilities          []string
	InputModalities       []sharedv1.Modality
	// Locality and Status are the resource's per-role declaration when it
	// makes one, else the adapter's: one resource may serve local and hosted
	// roles, and a hosted role may be unconfigured on this host.
	Locality            string
	Status              string
	PolicySchemaVersion string
	PolicyFileSHA256    string
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
	// RootOperationID binds a metered-provider reservation to the Gateway's
	// caller-visible recovery receipt. It is an idempotency key, not authority.
	RootOperationID              string
	Kind                         sharedv1.RequestKind
	Role                         string
	ExpectedModel                string
	ExpectedArtifactDigest       string
	ExpectedRightsDecisionSHA256 string
	ExpectedRightsEvaluatedAt    string
	ExpectedRightsReceiptSHA256  string
	ExpectedPolicyFileSHA256     string
	Profile                      sharedv1.Profile
	InputText                    string
	MaxOutputTokens              int32
	Timeout                      time.Duration
	// Temperature requests a specific sampling temperature. It is a pointer so
	// that an explicit 0 (deterministic) is distinguishable from "unset, use
	// the resource default". Whether the resolved role's provider actually
	// applies it is a policy declaration, not something this request can know:
	// see ResolvedRole.TemperatureSupport.
	Temperature *float64
	// SchemaJSON is passed through the provider's native structured-output
	// request field. The gateway still validates the returned value locally.
	SchemaJSON     string
	Attachments    []*sharedv1.Attachment
	SpendLimit     *inferencev1.SpendLimit
	BrowserSession *inferencev1.BrowserSessionBinding
	// IdempotencyKey is one Gateway-persisted HostedDoor child attempt identity.
	IdempotencyKey        string
	ExpectedKeyGeneration string
	RecoveryOnly          bool
	ExpectedWorkerDigest  string
}

type ExecutionResult struct {
	OutputText         string
	BillingOperationID string
	ExitCode           int
	SpendAuthorization *inferencev1.SpendAuthorization
	Recovery           *inferencev1.RecoveryReceipt
}

type ResolvedRole struct {
	Provider              string
	Role                  string
	Model                 string
	CanonicalModel        string
	Rights                *sharedpolicy.ArtifactRights
	RuntimeIdentity       *sharedpolicy.RuntimeArtifactIdentity
	ObservedRuntimeDigest string
	Capabilities          []string
	CoordinateConvention  string
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
	Locality             string
	PolicySchemaVersion  string
	PolicyFileSHA256     string
	RightsDecisionSHA256 string
	RightsEvaluatedAt    string
	RightsReceiptSHA256  string
	RightsContextSHA256  string
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
	SchemaVersion        string                                `json:"schema_version"`
	PolicySHA256         string                                `json:"policy_sha256"`
	Role                 string                                `json:"role"`
	Model                string                                `json:"model"`
	Rights               *sharedpolicy.ArtifactRights          `json:"rights"`
	RuntimeIdentity      *sharedpolicy.RuntimeArtifactIdentity `json:"runtime_identity"`
	RequiredCapabilities []string                              `json:"required_capabilities"`
	Capabilities         []string                              `json:"capabilities"`
	Modalities           struct {
		Input []string `json:"input"`
	} `json:"modalities"`
	Locality string `json:"locality"`
	Status   string `json:"status"`
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
	modelArgs := ollamaModelArgs(req)
	localOnlyArgs := ollamaLocalityArgs(req)
	switch req.Kind {
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING:
		return ollamaEmbeddingCommand(a.CommandName, role, input, timeout, modelArgs, localOnlyArgs), nil
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION:
		return ollamaGenerationCommand(a, req, role, input, timeout, modelArgs, localOnlyArgs)
	default:
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable", req.Kind.String())}
	}
}

func ollamaModelArgs(req ExecutionRequest) []string {
	args := []string{}
	for _, expected := range ollamaExpectedIdentity(req) {
		if value := strings.TrimSpace(expected.value); value != "" {
			args = append(args, expected.flag, value)
		}
	}
	return args
}

type expectedCLIArg struct{ value, flag string }

func ollamaExpectedIdentity(req ExecutionRequest) []expectedCLIArg {
	return []expectedCLIArg{
		{req.ExpectedModel, "--expected-model"},
		{req.ExpectedArtifactDigest, "--expected-artifact-digest"},
		{req.ExpectedRightsDecisionSHA256, "--expected-rights-decision-sha256"},
		{req.ExpectedRightsEvaluatedAt, "--expected-rights-evaluated-at"},
		{req.ExpectedRightsReceiptSHA256, "--expected-rights-receipt-sha256"},
		{req.ExpectedPolicyFileSHA256, "--expected-policy-file-sha256"},
	}
}

func ollamaLocalityArgs(req ExecutionRequest) []string {
	if req.Profile == sharedv1.Profile_PROFILE_LOCAL_ONLY {
		return []string{"--require-loopback"}
	}
	return nil
}

func ollamaEmbeddingCommand(name, role, input string, timeout time.Duration, modelArgs, localityArgs []string) Command {
	args := append(append(append([]string{"gateway", "embed", "--role", role}, modelArgs...), localityArgs...), "--json", "--input-stdin")
	return Command{Name: name, Args: args, Stdin: input, Timeout: timeout}
}

func ollamaGenerationCommand(a Adapter, req ExecutionRequest, role, input string, timeout time.Duration, modelArgs, localityArgs []string) (Command, error) {
	stdin, jsonEnvelope, err := multimodalStdin(input, req.Attachments)
	if err != nil {
		return Command{}, err
	}
	args := append(append([]string{"gateway", "generate", "--role", role}, modelArgs...), "--json", "--prompt-stdin")
	if jsonEnvelope {
		args = append(append([]string{"gateway", "generate", "--role", role}, modelArgs...), "--json", "--input-json-stdin")
	}
	args = append(args, localityArgs...)
	if digest := strings.TrimSpace(req.ExpectedWorkerDigest); digest != "" {
		args = append(args, "--require-worker-digest", digest)
	}
	if req.MaxOutputTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
	}
	if req.Temperature != nil {
		args = append(args, "--temperature", strconv.FormatFloat(*req.Temperature, 'g', -1, 64))
	}
	args, err = appendOllamaSchema(args, req, a.Provider)
	if err != nil {
		return Command{}, err
	}
	return Command{Name: a.CommandName, Args: args, Stdin: stdin, Timeout: timeout}, nil
}

func appendOllamaSchema(args []string, req ExecutionRequest, provider string) ([]string, error) {
	// vision.default embeds the schema in its prompt because Ollama returns no
	// visible response when its image template receives a native format.
	if strings.TrimSpace(req.SchemaJSON) == "" || strings.TrimSpace(req.Role) == "vision.default" {
		return args, nil
	}
	if !json.Valid([]byte(req.SchemaJSON)) {
		return nil, &CommandError{Code: "invalid_request", Command: provider, ExitCode: -1, Err: errors.New("schema_json must be valid JSON")}
	}
	return append(args, "--format", req.SchemaJSON), nil
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
	args, err = appendOpenAIRequestArgs(args, req, spec.SpendCapable, a.Provider)
	if err != nil {
		return Command{}, err
	}
	return Command{Name: a.CommandName, Args: args, Stdin: stdin, Timeout: timeout}, nil
}

func appendOpenAIRequestArgs(args []string, req ExecutionRequest, spendCapable bool, provider string) ([]string, error) {
	if req.MaxOutputTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
	}
	if req.SpendLimit != nil && spendCapable {
		args = append(args, "--max-cost-micros", strconv.FormatInt(req.SpendLimit.GetMaxCostMicros(), 10),
			"--operation-id", req.SpendLimit.GetOperationId(),
			"--input-fingerprint", req.SpendLimit.GetInputFingerprint(),
			"--funding-route", req.SpendLimit.GetFundingRoute())
		args = append(args, "--reference-sha256", strings.Join(req.SpendLimit.GetReferenceSha256(), ","))
		args = append(args, "--image-count", strconv.FormatInt(int64(req.SpendLimit.GetImageCount()), 10))
	}
	// Leaving temperature absent preserves the resource's declared default.
	if req.Temperature != nil {
		args = append(args, "--temperature", strconv.FormatFloat(*req.Temperature, 'g', -1, 64))
	}
	return appendOpenAISchema(args, req.SchemaJSON, provider)
}

func appendOpenAISchema(args []string, schemaJSON, provider string) ([]string, error) {
	if strings.TrimSpace(schemaJSON) == "" {
		return args, nil
	}
	if !json.Valid([]byte(schemaJSON)) {
		return nil, &CommandError{Code: "invalid_request", Command: provider, ExitCode: -1, Err: errors.New("schema_json must be valid JSON")}
	}
	format := fmt.Sprintf(`{"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":%s}}`, schemaJSON)
	return append(args, "--response-format", format), nil
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
	args := append([]string{"gateway", "decide", "--role", role, "--input-stdin", "--json", "--timeout", timeout.String()}, ollayaIdentityArgs(req)...)
	return Command{
		Name:    a.CommandName,
		Args:    args,
		Stdin:   input,
		Timeout: timeout + resourceCommandGrace,
	}, nil
}

func ollayaIdentityArgs(req ExecutionRequest) []string {
	if strings.TrimSpace(req.ExpectedArtifactDigest) == "" && strings.TrimSpace(req.ExpectedRightsDecisionSHA256) == "" && strings.TrimSpace(req.ExpectedRightsEvaluatedAt) == "" && strings.TrimSpace(req.ExpectedRightsReceiptSHA256) == "" && strings.TrimSpace(req.ExpectedPolicyFileSHA256) == "" {
		return nil
	}
	args := []string{}
	for _, expected := range ollamaExpectedIdentity(req) {
		if value := strings.TrimSpace(expected.value); value != "" {
			args = append(args, expected.flag, value)
		}
	}
	return args
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
