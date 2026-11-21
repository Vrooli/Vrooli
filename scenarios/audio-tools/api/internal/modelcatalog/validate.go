package modelcatalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Issue codes are stable identifiers a caller or test can assert on without
// matching prose.
const (
	IssueSchemaVersion        = "schema_version"
	IssueNoModels             = "no_models"
	IssueDuplicateID          = "duplicate_id"
	IssueMissingField         = "missing_field"
	IssueInvalidCapability    = "invalid_capability"
	IssueInvalidRuntime       = "invalid_runtime"
	IssueInvalidReadiness     = "invalid_readiness"
	IssueInvalidRole          = "invalid_role"
	IssueMissingProviderID    = "missing_provider_id"
	IssueUnknownProviderID    = "unknown_provider_id"
	IssueMissingEngineID      = "missing_engine_id"
	IssueUnexpectedEngineID   = "unexpected_engine_id"
	IssueInvalidLanguage      = "invalid_language"
	IssueInvalidCommercialUse = "invalid_commercial_use"
	IssueMissingLicenseNotes  = "missing_license_notes"
	IssueInvalidDefault       = "invalid_default"
	IssueInvalidHardware      = "invalid_hardware"
	IssueInvalidBenchmark     = "invalid_benchmark"
	IssueMissingCapability    = "missing_capability"
	IssueInvalidPolicy        = "invalid_policy"
	IssueMissingNotes         = "missing_notes"
	IssueInvalidArtifact      = "invalid_artifact"
)

// artifactProvidingRuntimes are the local runtimes whose weights must come
// from a resource data artifact. Remote runtimes (byok/gateway/browser) and
// ollama-managed records are excluded: their weights are owned elsewhere.
var artifactProvidingRuntimes = stringSet([]string{
	"whisper.cpp",
	"kyutai",
	"sherpa-onnx",
	"kokoro",
	"onnxruntime",
})

// Issue is one validation finding.
type Issue struct {
	Code    string `json:"code"`
	ModelID string `json:"model_id,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (i Issue) String() string {
	if i.ModelID != "" {
		return fmt.Sprintf("%s: %s: %s", i.Code, i.ModelID, i.Message)
	}
	return fmt.Sprintf("%s: %s", i.Code, i.Message)
}

var languageTagPattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// ValidationIssues returns every finding, in deterministic order. An empty
// slice means the seed satisfies the registry contract.
func (c *Catalog) ValidationIssues() []Issue {
	var issues []Issue
	add := func(code, modelID, field, format string, args ...any) {
		issues = append(issues, Issue{
			Code:    code,
			ModelID: modelID,
			Field:   field,
			Message: fmt.Sprintf(format, args...),
		})
	}

	if c.seed.SchemaVersion != SupportedSchemaVersion {
		add(IssueSchemaVersion, "", "schema_version", "got %q, want %q", c.seed.SchemaVersion, SupportedSchemaVersion)
	}
	if len(c.seed.Models) == 0 {
		add(IssueNoModels, "", "models", "seed must declare at least one model")
	}
	issues = append(issues, c.validatePolicy()...)
	issues = append(issues, c.validateProviderVocabulary()...)

	runtimes := stringSet(Runtimes)
	roles := stringSet(Roles)
	readinesses := stringSet([]string{
		string(ReadinessAvailable),
		string(ReadinessCandidate),
		string(ReadinessExperimental),
		string(ReadinessWatch),
	})
	capabilities := stringSet(capabilityStrings())

	seenIDs := make(map[string]struct{}, len(c.seed.Models))
	for _, m := range c.seed.Models {
		if strings.TrimSpace(m.ID) == "" {
			add(IssueMissingField, "", "id", "model has empty id")
		} else if _, dup := seenIDs[m.ID]; dup {
			add(IssueDuplicateID, m.ID, "id", "duplicate model id %q", m.ID)
		} else {
			seenIDs[m.ID] = struct{}{}
		}
		if strings.TrimSpace(m.Name) == "" {
			add(IssueMissingField, m.ID, "name", "model must have a display name")
		}
		if _, ok := capabilities[string(m.Capability)]; !ok {
			add(IssueInvalidCapability, m.ID, "capability", "unknown capability %q", m.Capability)
		}
		if _, ok := runtimes[m.Runtime]; !ok {
			add(IssueInvalidRuntime, m.ID, "runtime", "unknown runtime %q", m.Runtime)
		}
		if m.Artifact != nil {
			if strings.TrimSpace(m.Artifact.Resource) == "" || strings.TrimSpace(m.Artifact.Name) == "" {
				add(IssueInvalidArtifact, m.ID, "artifact",
					"artifact requires both resource and name")
			}
		} else if m.Readiness == ReadinessAvailable {
			if _, local := artifactProvidingRuntimes[m.Runtime]; local {
				add(IssueInvalidArtifact, m.ID, "artifact",
					"readiness=available with runtime=%s must name the resource artifact it ships", m.Runtime)
			}
		}
		if _, ok := readinesses[string(m.Readiness)]; !ok {
			add(IssueInvalidReadiness, m.ID, "readiness", "unknown readiness %q", m.Readiness)
		}
		if strings.TrimSpace(m.License) == "" {
			add(IssueMissingField, m.ID, "license", "model must declare a licence")
		}
		switch m.Commercial {
		case CommercialYes, CommercialNo, CommercialUnknown:
		default:
			add(IssueInvalidCommercialUse, m.ID, "commercial_use", "unknown commercial_use %q", m.Commercial)
		}
		if (m.Commercial == CommercialNo || m.Commercial == CommercialUnknown) && strings.TrimSpace(m.LicenseNotes) == "" {
			add(IssueMissingLicenseNotes, m.ID, "license_notes",
				"commercial_use=%s requires license_notes explaining the restriction or uncertainty", m.Commercial)
		}
		if len(m.ProviderIDs) == 0 {
			add(IssueMissingProviderID, m.ID, "provider_ids", "model must name at least one stable provider id")
		}
		seenProviders := make(map[string]struct{}, len(m.ProviderIDs))
		for _, id := range m.ProviderIDs {
			if strings.TrimSpace(id) == "" {
				add(IssueMissingProviderID, m.ID, "provider_ids", "provider id must not be empty")
				continue
			}
			if _, dup := seenProviders[id]; dup {
				add(IssueUnknownProviderID, m.ID, "provider_ids", "duplicate provider id %q", id)
			}
			seenProviders[id] = struct{}{}
			if !contains(c.seed.KnownProviderIDs, id) {
				add(IssueUnknownProviderID, m.ID, "provider_ids", "provider id %q is not in known_provider_ids", id)
			}
		}
		seenEngines := make(map[string]struct{}, len(m.EngineIDs))
		for _, id := range m.EngineIDs {
			if strings.TrimSpace(id) == "" {
				add(IssueMissingEngineID, m.ID, "engine_ids", "engine id must not be empty")
				continue
			}
			if _, dup := seenEngines[id]; dup {
				add(IssueMissingEngineID, m.ID, "engine_ids", "duplicate engine id %q", id)
			}
			seenEngines[id] = struct{}{}
		}
		if m.Capability == CapabilitySTT {
			if remoteRuntime(m.Runtime) && len(m.EngineIDs) > 0 {
				add(IssueUnexpectedEngineID, m.ID, "engine_ids",
					"runtime=%s is chain-routed and must not declare an sttengine stream-selection engine id", m.Runtime)
			}
		} else if len(m.EngineIDs) > 0 {
			add(IssueUnexpectedEngineID, m.ID, "engine_ids",
				"engine_ids are STT stream-selection ids and must be empty for capability %q", m.Capability)
		}
		if len(m.Languages) == 0 {
			add(IssueInvalidLanguage, m.ID, "languages", "model must declare at least one language tag")
		}
		for _, tag := range m.Languages {
			if !languageTagPattern.MatchString(tag) {
				add(IssueInvalidLanguage, m.ID, "languages", "invalid language tag %q", tag)
			}
		}
		if len(m.Roles) == 0 {
			add(IssueInvalidRole, m.ID, "roles", "model must declare at least one selection role")
		}
		seenRoles := make(map[string]struct{}, len(m.Roles))
		for _, role := range m.Roles {
			if _, ok := roles[role]; !ok {
				add(IssueInvalidRole, m.ID, "roles", "unknown role %q", role)
			}
			if _, dup := seenRoles[role]; dup {
				add(IssueInvalidRole, m.ID, "roles", "duplicate role %q", role)
			}
			seenRoles[role] = struct{}{}
		}
		if m.HasRole("default") && m.Readiness != ReadinessAvailable && !m.AllowDefault {
			add(IssueInvalidDefault, m.ID, "roles",
				"readiness=%s may not carry the default role without allow_default=true", m.Readiness)
		}
		if m.Readiness == ReadinessExperimental || m.Readiness == ReadinessWatch {
			if strings.TrimSpace(m.Notes) == "" {
				add(IssueMissingNotes, m.ID, "notes",
					"readiness=%s requires notes stating what is unproven", m.Readiness)
			}
		}
		issues = append(issues, validateHardware(m)...)
		issues = append(issues, validateBenchmarks(m)...)
	}

	for _, capability := range Capabilities {
		if len(c.byCapability[capability]) == 0 {
			add(IssueMissingCapability, "", "capability", "no model serves capability %q", capability)
		}
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].ModelID != issues[j].ModelID {
			return issues[i].ModelID < issues[j].ModelID
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Message < issues[j].Message
	})
	return issues
}

// Validate returns a non-nil error when any validation issue exists. The error
// names the first findings so a broken seed fails with an actionable message.
func (c *Catalog) Validate() error {
	issues := c.ValidationIssues()
	if len(issues) == 0 {
		return nil
	}
	const shown = 6
	parts := make([]string, 0, shown)
	for i, issue := range issues {
		if i == shown {
			break
		}
		parts = append(parts, issue.String())
	}
	suffix := ""
	if len(issues) > shown {
		suffix = fmt.Sprintf(" (and %d more)", len(issues)-shown)
	}
	return fmt.Errorf("modelcatalog: %d validation issue(s): %s%s", len(issues), strings.Join(parts, "; "), suffix)
}

func (c *Catalog) validatePolicy() []Issue {
	var issues []Issue
	policy := c.seed.SelectionPolicy
	check := func(field string, got, want []string) {
		if !sameStrings(got, want) {
			issues = append(issues, Issue{
				Code:    IssueInvalidPolicy,
				Field:   "selection_policy." + field,
				Message: fmt.Sprintf("got %v, want the complete set %v", got, want),
			})
		}
	}
	check("readiness_rank", policy.ReadinessRank, []string{
		string(ReadinessAvailable), string(ReadinessCandidate),
		string(ReadinessExperimental), string(ReadinessWatch),
	})
	check("role_order", policy.RoleOrder, Roles)
	if policy.LanguageFallback != "none" {
		issues = append(issues, Issue{
			Code:    IssueInvalidPolicy,
			Field:   "selection_policy.language_fallback",
			Message: fmt.Sprintf("unknown fallback %q; only \"none\" is implemented", policy.LanguageFallback),
		})
	}
	return issues
}

func (c *Catalog) validateProviderVocabulary() []Issue {
	var issues []Issue
	if len(c.seed.KnownProviderIDs) == 0 {
		issues = append(issues, Issue{
			Code:    IssueMissingProviderID,
			Field:   "known_provider_ids",
			Message: "provider vocabulary must declare at least one id",
		})
	}
	seen := make(map[string]struct{}, len(c.seed.KnownProviderIDs))
	for _, id := range c.seed.KnownProviderIDs {
		if strings.TrimSpace(id) == "" {
			issues = append(issues, Issue{
				Code: IssueMissingProviderID, Field: "known_provider_ids", Message: "provider id must not be empty",
			})
			continue
		}
		if _, dup := seen[id]; dup {
			issues = append(issues, Issue{
				Code: IssueUnknownProviderID, Field: "known_provider_ids", Message: fmt.Sprintf("duplicate provider id %q", id),
			})
		}
		seen[id] = struct{}{}
	}
	return issues
}

func validateHardware(m Model) []Issue {
	var issues []Issue
	h := m.Hardware
	if h.GPURequired && h.CPUCapable {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "gpu_required and cpu_capable are mutually exclusive",
		})
	}
	if !h.GPURequired && !h.CPUCapable {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "model must be cpu_capable or gpu_required (remote models declare cpu_capable for the provider host)",
		})
	}
	if h.GPURequired && h.MinVRAMGB <= 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware.min_vram_gb",
			Message: "gpu_required models must declare a positive min_vram_gb",
		})
	}
	if h.MinVRAMGB < 0 || h.MinRAMGB < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware",
			Message: "min_vram_gb and min_ram_gb must not be negative",
		})
	}
	if m.SizeMBApprox < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidHardware, ModelID: m.ID, Field: "size_mb_approx",
			Message: "size_mb_approx must not be negative",
		})
	}
	for _, entry := range h.OSArch {
		osName, arch, ok := strings.Cut(entry, "/")
		if !ok || !oneOf(osName, "linux", "darwin", "windows", "any") || !oneOf(arch, "amd64", "arm64", "any") {
			issues = append(issues, Issue{
				Code: IssueInvalidHardware, ModelID: m.ID, Field: "hardware.os_arch",
				Message: fmt.Sprintf("invalid os/arch %q; use linux|darwin|windows|any / amd64|arm64|any", entry),
			})
		}
	}
	return issues
}

func validateBenchmarks(m Model) []Issue {
	var issues []Issue
	b := m.Benchmarks
	if b.WER != nil && *b.WER < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.wer",
			Message: "WER percentage must not be negative",
		})
	}
	if b.SIM != nil && (*b.SIM < 0 || *b.SIM > 1) {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.sim",
			Message: "SIM must be a 0..1 similarity score",
		})
	}
	if b.RTF != nil && *b.RTF < 0 {
		issues = append(issues, Issue{
			Code: IssueInvalidBenchmark, ModelID: m.ID, Field: "benchmarks.rtf",
			Message: "RTF must not be negative",
		})
	}
	return issues
}

func remoteRuntime(runtime string) bool {
	switch runtime {
	case "byok", "gateway", "browser":
		return true
	default:
		return false
	}
}

func capabilityStrings() []string {
	out := make([]string, len(Capabilities))
	for i, capability := range Capabilities {
		out[i] = string(capability)
	}
	return out
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := stringSet(got)
	for _, value := range want {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
