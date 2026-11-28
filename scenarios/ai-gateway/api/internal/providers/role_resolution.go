package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
)

// ResolveRole asks the resource for the concrete model selected by its
// policy. AI Gateway never reads provider model catalogs or credentials.
func (a Adapter) ResolveRole(ctx context.Context, role string) (ResolvedRole, error) {
	if a.Door != nil {
		return ResolvedRole{Provider: ProviderMetered, Role: role, Model: "policy-resolved", Locality: "remote"}, nil
	}
	if a.HTTP != nil {
		return a.HTTP.ResolveRole(ctx, role)
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("role is required")}
	}
	if a.Provider == ProviderMetered {
		if a.Metered == nil {
			return ResolvedRole{}, &CommandError{Code: "unavailable", Command: a.Provider, ExitCode: -1, Err: errors.New("metered inference client is not configured")}
		}
		return ResolvedRole{Provider: a.Provider, Role: role, Model: "policy-resolved", CanonicalModel: "policy-resolved", Locality: "remote"}, nil
	}
	result, err := a.runner().Run(ctx, Command{Name: a.CommandName, Args: []string{"policy", "resolve", "--role", role, "--json"}, Timeout: DefaultCommandTimeout, Fact: true})
	if err != nil {
		return ResolvedRole{}, mapCommandError(a.Provider, err)
	}
	response, err := decodeRoleResolution(a, result)
	if err != nil {
		return ResolvedRole{}, err
	}
	return a.resolvedRole(ctx, role, result, response)
}

type roleResolutionResponse struct {
	Role                 string            `json:"role"`
	Model                string            `json:"model"`
	Canonical            string            `json:"canonical_model"`
	Limits               *DecisionLimits   `json:"limits"`
	Placement            *ModelPlacement   `json:"placement"`
	Capabilities         []string          `json:"capabilities"`
	CoordinateConvention string            `json:"coordinate_convention"`
	SamplingSupport      map[string]string `json:"sampling_support"`
	MaxTokens            *policyInt32      `json:"max_tokens"`
	RequestDefaults      *struct {
		MaxTokens *policyInt32 `json:"max_tokens"`
	} `json:"request_defaults"`
	ContextWindow          *int32                                `json:"context_window"`
	Locality               string                                `json:"locality"`
	SchemaVersion          string                                `json:"schema_version"`
	PolicySHA256           string                                `json:"policy_sha256"`
	Rights                 *sharedpolicy.ArtifactRights          `json:"rights"`
	RuntimeIdentity        *sharedpolicy.RuntimeArtifactIdentity `json:"runtime_identity"`
	ObservedRuntimeDigest  string                                `json:"observed_runtime_digest"`
	ExpectedArtifactDigest string                                `json:"expected_artifact_digest"`
}

// policyInt32 accepts both JSON numbers and decimal strings. Resource policy
// contracts have emitted both forms across CLI versions; normalize them at
// this adapter boundary while still rejecting malformed and out-of-range caps.
type policyInt32 int32

func (n *policyInt32) UnmarshalJSON(raw []byte) error {
	value := strings.TrimSpace(string(raw))
	if len(value) > 0 && value[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return err
		}
		value = strings.TrimSpace(text)
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fmt.Errorf("max_tokens must be a 32-bit integer or decimal string: %w", err)
	}
	*n = policyInt32(parsed)
	return nil
}

func decodeRoleResolution(a Adapter, result Result) (roleResolutionResponse, error) {
	var response roleResolutionResponse
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return roleResolutionResponse{}, &CommandError{Code: "malformed_json", Command: a.command().String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	return response, nil
}

func (a Adapter) resolvedRole(ctx context.Context, role string, result Result, response roleResolutionResponse) (ResolvedRole, error) {
	if strings.TrimSpace(response.Role) != role || strings.TrimSpace(response.Model) == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_role_resolution", Command: a.command().String(), ExitCode: result.ExitCode, Err: fmt.Errorf("resource returned role %q and model %q for requested role %q", response.Role, response.Model, role)}
	}
	observedRuntimeDigest, err := a.resolvedRuntimeDigest(ctx, response, result)
	if err != nil {
		return ResolvedRole{}, err
	}
	var maxOutputTokens int32
	switch {
	case response.MaxTokens != nil:
		maxOutputTokens = int32(*response.MaxTokens)
	case response.RequestDefaults != nil && response.RequestDefaults.MaxTokens != nil:
		maxOutputTokens = int32(*response.RequestDefaults.MaxTokens)
	}
	return ResolvedRole{
		Provider: a.Provider, Role: role, Model: response.Model, CanonicalModel: response.Canonical,
		Limits: response.Limits, Placement: response.Placement, Rights: response.Rights, RuntimeIdentity: response.RuntimeIdentity, ObservedRuntimeDigest: observedRuntimeDigest,
		Capabilities: sortedUnique(response.Capabilities), CoordinateConvention: strings.TrimSpace(response.CoordinateConvention),
		SamplingSupport: parseSamplingSupport(response.SamplingSupport), MaxOutputTokens: maxOutputTokens,
		ContextWindow: valueOrZero(response.ContextWindow), Locality: firstNonEmpty(response.Locality, a.Locality),
		PolicySchemaVersion: strings.TrimSpace(response.SchemaVersion), PolicyFileSHA256: strings.TrimSpace(response.PolicySHA256),
	}, nil
}

func (a Adapter) resolvedRuntimeDigest(ctx context.Context, response roleResolutionResponse, result Result) (string, error) {
	if a.Provider == ProviderOllama {
		digests, err := a.ollamaModelDigests(ctx)
		if err != nil {
			return "", nil
		}
		return digests[strings.TrimSpace(response.Model)], nil
	}
	if a.Provider != ProviderOllaya {
		return "", nil
	}
	digest := strings.TrimSpace(response.ObservedRuntimeDigest)
	if !validOllayaManifestIdentity(response, digest) {
		return "", &CommandError{Code: "invalid_role_resolution", Command: a.command().String(), ExitCode: result.ExitCode, Err: errors.New("resource returned an inconsistent Ollaya manifest identity")}
	}
	return digest, nil
}

func validOllayaManifestIdentity(response roleResolutionResponse, digest string) bool {
	identity := response.RuntimeIdentity
	return identity == nil || (identity.Provider == ProviderOllaya && identity.Model == strings.TrimSpace(response.Model) && identity.Digest == digest && strings.TrimSpace(response.ExpectedArtifactDigest) == digest)
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
