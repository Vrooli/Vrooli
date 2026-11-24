package modelcatalog

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Issue codes are stable identifiers a caller or test can assert on without
// matching prose.
const (
	IssueSchemaVersion                = "schema_version"
	IssueNoModels                     = "no_models"
	IssueDuplicateID                  = "duplicate_id"
	IssueMissingField                 = "missing_field"
	IssueInvalidCapability            = "invalid_capability"
	IssueInvalidRuntime               = "invalid_runtime"
	IssueUnsupportedRuntimeCapability = "unsupported_runtime_capability"
	IssueInvalidBatchAdapter          = "invalid_batch_adapter"
	IssueInvalidReadiness             = "invalid_readiness"
	IssueInvalidRole                  = "invalid_role"
	IssueMissingProviderID            = "missing_provider_id"
	IssueUnknownProviderID            = "unknown_provider_id"
	IssueMissingEngineID              = "missing_engine_id"
	IssueUnexpectedEngineID           = "unexpected_engine_id"
	IssueInvalidEngineDefault         = "invalid_engine_default"
	IssueInvalidLanguage              = "invalid_language"
	IssueInvalidCommercialUse         = "invalid_commercial_use"
	IssueMissingLicenseNotes          = "missing_license_notes"
	IssueInvalidDefault               = "invalid_default"
	IssueInvalidHardware              = "invalid_hardware"
	IssueInvalidBenchmark             = "invalid_benchmark"
	IssueMissingCapability            = "missing_capability"
	IssueInvalidPolicy                = "invalid_policy"
	IssueMissingNotes                 = "missing_notes"
	IssueInvalidArtifact              = "invalid_artifact"
	IssueInvalidPricing               = "invalid_pricing"
	IssueInvalidPromotionEvidence     = "invalid_promotion_evidence"
)

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
	if c.seed.SchemaVersion != SupportedSchemaVersion {
		issues = append(issues, newIssue(IssueSchemaVersion, "", "schema_version", "got %q, want %q", c.seed.SchemaVersion, SupportedSchemaVersion))
	}
	if len(c.seed.Models) == 0 {
		issues = append(issues, newIssue(IssueNoModels, "", "models", "seed must declare at least one model"))
	}
	issues = append(issues, c.validatePolicy()...)
	issues = append(issues, c.validateProviderVocabulary()...)

	known := validationLookups{
		runtimes:     runtimeLookup(),
		roles:        stringSet(Roles),
		readinesses:  stringSet([]string{string(ReadinessAvailable), string(ReadinessCandidate), string(ReadinessExperimental), string(ReadinessWatch)}),
		capabilities: stringSet(capabilityStrings()),
	}
	seenIDs := make(map[string]struct{}, len(c.seed.Models))
	for _, m := range c.seed.Models {
		issues = append(issues, c.validateModel(m, seenIDs, known)...)
	}
	issues = append(issues, c.validateUniqueEngineDefaults()...)

	for _, capability := range DefaultCapabilities {
		if len(c.byCapability[capability]) == 0 {
			issues = append(issues, newIssue(IssueMissingCapability, "", "capability", "no model serves capability %q", capability))
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

type validationLookups struct {
	runtimes     map[string]Runtime
	roles        map[string]struct{}
	readinesses  map[string]struct{}
	capabilities map[string]struct{}
}

func newIssue(code, modelID, field, format string, args ...any) Issue {
	return Issue{Code: code, ModelID: modelID, Field: field, Message: fmt.Sprintf(format, args...)}
}

func (c *Catalog) validateModel(m Model, seenIDs map[string]struct{}, lookups validationLookups) []Issue {
	issues := validateModelIdentity(m, seenIDs, lookups.capabilities)
	issues = append(issues, validateModelRuntime(m, lookups, lookups.readinesses)...)
	issues = append(issues, validateModelLicense(m)...)
	issues = append(issues, c.validateModelProviders(m)...)
	issues = append(issues, validateModelEngines(m)...)
	issues = append(issues, validateModelEngineDefaults(m)...)
	issues = append(issues, validateModelLanguages(m)...)
	issues = append(issues, validateModelRoles(m, lookups.roles)...)
	issues = append(issues, validateModelPromotion(m)...)
	issues = append(issues, validateModelReadinessNotes(m)...)
	issues = append(issues, validateHardware(m)...)
	issues = append(issues, validateBenchmarks(m)...)
	issues = append(issues, validatePricing(m)...)
	return issues
}

func validateModelIdentity(m Model, seenIDs, capabilities map[string]struct{}) []Issue {
	var issues []Issue
	if strings.TrimSpace(m.ID) == "" {
		issues = append(issues, newIssue(IssueMissingField, "", "id", "model has empty id"))
	} else if _, dup := seenIDs[m.ID]; dup {
		issues = append(issues, newIssue(IssueDuplicateID, m.ID, "id", "duplicate model id %q", m.ID))
	} else {
		seenIDs[m.ID] = struct{}{}
	}
	if strings.TrimSpace(m.Name) == "" {
		issues = append(issues, newIssue(IssueMissingField, m.ID, "name", "model must have a display name"))
	}
	if _, ok := capabilities[string(m.Capability)]; !ok {
		issues = append(issues, newIssue(IssueInvalidCapability, m.ID, "capability", "unknown capability %q", m.Capability))
	}
	return issues
}

func validateModelRuntime(m Model, lookups validationLookups, readinesses map[string]struct{}) []Issue {
	var issues []Issue
	runtime, ok := lookups.runtimes[m.Runtime]
	if !ok {
		issues = append(issues, newIssue(IssueInvalidRuntime, m.ID, "runtime", "unknown runtime %q", m.Runtime))
	} else if !slices.Contains(runtime.Capabilities, m.Capability) {
		issues = append(issues, newIssue(IssueUnsupportedRuntimeCapability, m.ID, "runtime", "runtime %q does not support capability %q", m.Runtime, m.Capability))
	}
	if m.RuntimeModelID != "" && strings.TrimSpace(m.RuntimeModelID) != m.RuntimeModelID {
		issues = append(issues, newIssue(IssueInvalidRuntime, m.ID, "runtime_model_id", "runtime model id must not have surrounding whitespace"))
	}
	if m.BatchAdapterID != "" {
		knownAdapters := stringSet(BatchAdapterIDs)
		if _, ok := knownAdapters[m.BatchAdapterID]; !ok {
			issues = append(issues, newIssue(IssueInvalidBatchAdapter, m.ID, "batch_adapter_id", "unknown batch adapter id %q", m.BatchAdapterID))
		} else if m.Capability != CapabilitySTT || m.Runtime != "vllm" || m.RuntimeModelID == "" {
			issues = append(issues, newIssue(IssueInvalidBatchAdapter, m.ID, "batch_adapter_id", "batch adapter %q requires STT capability, vllm runtime, and runtime_model_id", m.BatchAdapterID))
		}
	}
	if m.Artifact != nil {
		if strings.TrimSpace(m.Artifact.Resource) == "" || strings.TrimSpace(m.Artifact.Name) == "" {
			issues = append(issues, newIssue(IssueInvalidArtifact, m.ID, "artifact", "artifact requires both resource and name"))
		}
	} else if m.Readiness == ReadinessAvailable {
		if runtime.ArtifactPolicy == RuntimeArtifactRequired {
			issues = append(issues, newIssue(IssueInvalidArtifact, m.ID, "artifact", "readiness=available with runtime=%s must name the resource artifact it ships", m.Runtime))
		}
	}
	if _, ok := readinesses[string(m.Readiness)]; !ok {
		issues = append(issues, newIssue(IssueInvalidReadiness, m.ID, "readiness", "unknown readiness %q", m.Readiness))
	}
	return issues
}

func validateModelLicense(m Model) []Issue {
	var issues []Issue
	if strings.TrimSpace(m.License) == "" {
		issues = append(issues, newIssue(IssueMissingField, m.ID, "license", "model must declare a licence"))
	}
	switch m.Commercial {
	case CommercialYes, CommercialNo, CommercialUnknown:
	default:
		issues = append(issues, newIssue(IssueInvalidCommercialUse, m.ID, "commercial_use", "unknown commercial_use %q", m.Commercial))
	}
	if (m.Commercial == CommercialNo || m.Commercial == CommercialUnknown) && strings.TrimSpace(m.LicenseNotes) == "" {
		issues = append(issues, newIssue(IssueMissingLicenseNotes, m.ID, "license_notes", "commercial_use=%s requires license_notes explaining the restriction or uncertainty", m.Commercial))
	}
	return issues
}

func (c *Catalog) validateModelProviders(m Model) []Issue {
	var issues []Issue
	if len(m.ProviderIDs) == 0 {
		issues = append(issues, newIssue(IssueMissingProviderID, m.ID, "provider_ids", "model must name at least one stable provider id"))
	}
	seen := make(map[string]struct{}, len(m.ProviderIDs))
	for _, id := range m.ProviderIDs {
		if strings.TrimSpace(id) == "" {
			issues = append(issues, newIssue(IssueMissingProviderID, m.ID, "provider_ids", "provider id must not be empty"))
			continue
		}
		if _, dup := seen[id]; dup {
			issues = append(issues, newIssue(IssueUnknownProviderID, m.ID, "provider_ids", "duplicate provider id %q", id))
		}
		seen[id] = struct{}{}
		if !contains(c.seed.KnownProviderIDs, id) {
			issues = append(issues, newIssue(IssueUnknownProviderID, m.ID, "provider_ids", "provider id %q is not in known_provider_ids", id))
		}
	}
	return issues
}

func validateModelEngines(m Model) []Issue {
	var issues []Issue
	seen := make(map[string]struct{}, len(m.EngineIDs))
	for _, id := range m.EngineIDs {
		if strings.TrimSpace(id) == "" {
			issues = append(issues, newIssue(IssueMissingEngineID, m.ID, "engine_ids", "engine id must not be empty"))
			continue
		}
		if _, dup := seen[id]; dup {
			issues = append(issues, newIssue(IssueMissingEngineID, m.ID, "engine_ids", "duplicate engine id %q", id))
		}
		seen[id] = struct{}{}
	}
	if m.Capability == CapabilitySTT {
		if remoteRuntime(m.Runtime) && len(m.EngineIDs) > 0 {
			issues = append(issues, newIssue(IssueUnexpectedEngineID, m.ID, "engine_ids", "runtime=%s is chain-routed and must not declare an sttengine stream-selection engine id", m.Runtime))
		}
	} else if len(m.EngineIDs) > 0 {
		issues = append(issues, newIssue(IssueUnexpectedEngineID, m.ID, "engine_ids", "engine_ids are STT stream-selection ids and must be empty for capability %q", m.Capability))
	}
	return issues
}

func validateModelEngineDefaults(m Model) []Issue {
	var issues []Issue
	seen := make(map[string]struct{}, len(m.EngineDefaults))
	engineIDs := stringSet(m.EngineIDs)
	for _, id := range m.EngineDefaults {
		if strings.TrimSpace(id) == "" {
			issues = append(issues, newIssue(IssueInvalidEngineDefault, m.ID, "engine_defaults", "engine default id must not be empty"))
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			issues = append(issues, newIssue(IssueInvalidEngineDefault, m.ID, "engine_defaults", "duplicate engine default %q", id))
		}
		seen[id] = struct{}{}
		if m.Capability != CapabilitySTT {
			issues = append(issues, newIssue(IssueInvalidEngineDefault, m.ID, "engine_defaults", "engine defaults are supported only for STT models"))
		}
		if _, declared := engineIDs[id]; !declared {
			issues = append(issues, newIssue(IssueInvalidEngineDefault, m.ID, "engine_defaults", "engine default %q must also appear in engine_ids", id))
		}
	}
	if len(m.EngineDefaults) > 0 && m.Readiness != ReadinessAvailable && !m.AllowDefault {
		issues = append(issues, newIssue(IssueInvalidEngineDefault, m.ID, "engine_defaults", "non-available engine default requires allow_default=true"))
	}
	if len(m.EngineDefaults) > 0 && !legacyExceptionAllowed(m.ID, m.Promotion.LegacyException) && !hasCompletePromotionReferences(m.Promotion) {
		issues = append(issues, newIssue(IssueInvalidPromotionEvidence, m.ID, "promotion_evidence", "engine default requires complete owner receipt references or an authorized legacy exception"))
	}
	return issues
}

func hasCompletePromotionReferences(e PromotionEvidence) bool {
	for _, facet := range promotionFacets(e) {
		if facet.facet.Status != "qualified" || strings.TrimSpace(facet.facet.Ref) == "" {
			return false
		}
	}
	return true
}

func (c *Catalog) validateUniqueEngineDefaults() []Issue {
	var issues []Issue
	seen := make(map[string]string)
	for _, model := range c.seed.Models {
		for _, engineID := range model.EngineDefaults {
			key := string(model.Capability) + "/" + engineID
			if prior, duplicate := seen[key]; duplicate {
				issues = append(issues, newIssue(IssueInvalidEngineDefault, model.ID, "engine_defaults", "capability %s engine %q also has default model %q", model.Capability, engineID, prior))
				continue
			}
			seen[key] = model.ID
		}
	}
	return issues
}

func validateModelLanguages(m Model) []Issue {
	var issues []Issue
	if len(m.Languages) == 0 {
		issues = append(issues, newIssue(IssueInvalidLanguage, m.ID, "languages", "model must declare at least one language tag"))
	}
	for _, tag := range m.Languages {
		if !languageTagPattern.MatchString(tag) {
			issues = append(issues, newIssue(IssueInvalidLanguage, m.ID, "languages", "invalid language tag %q", tag))
		}
	}
	return issues
}

func validateModelRoles(m Model, roles map[string]struct{}) []Issue {
	var issues []Issue
	if len(m.Roles) == 0 {
		issues = append(issues, newIssue(IssueInvalidRole, m.ID, "roles", "model must declare at least one selection role"))
	}
	seen := make(map[string]struct{}, len(m.Roles))
	for _, role := range m.Roles {
		if _, ok := roles[role]; !ok {
			issues = append(issues, newIssue(IssueInvalidRole, m.ID, "roles", "unknown role %q", role))
		}
		if _, dup := seen[role]; dup {
			issues = append(issues, newIssue(IssueInvalidRole, m.ID, "roles", "duplicate role %q", role))
		}
		seen[role] = struct{}{}
	}
	if m.HasRole("default") && m.Readiness != ReadinessAvailable && !m.AllowDefault {
		issues = append(issues, newIssue(IssueInvalidDefault, m.ID, "roles", "readiness=%s may not carry the default role without allow_default=true", m.Readiness))
	}
	return issues
}

func validateModelPromotion(m Model) []Issue {
	var issues []Issue
	legacyException := strings.TrimSpace(m.Promotion.LegacyException)
	if legacyException != "" && !legacyExceptionAllowed(m.ID, m.Promotion.LegacyException) {
		issues = append(issues, newIssue(IssueInvalidPromotionEvidence, m.ID, "promotion_evidence.legacy_exception", "legacy exception is not authorized for this model or does not match its code-owned reason"))
	}
	if m.HasRole("default") && !m.Promotion.Qualified() && !legacyExceptionAllowed(m.ID, m.Promotion.LegacyException) {
		issues = append(issues, newIssue(IssueInvalidPromotionEvidence, m.ID, "promotion_evidence", "default role requires owner-verified corpus_quality, latency, memory, platform, and license receipts, or the code-authorized exception for an existing operational default"))
	}
	for _, entry := range promotionFacets(m.Promotion) {
		if !validEvidenceStatus(entry.facet.Status) {
			issues = append(issues, newIssue(IssueInvalidPromotionEvidence, m.ID, "promotion_evidence."+entry.name, "unknown evidence status %q", entry.facet.Status))
		}
		if entry.facet.Status == "qualified" && strings.TrimSpace(entry.facet.Ref) == "" {
			issues = append(issues, newIssue(IssueInvalidPromotionEvidence, m.ID, "promotion_evidence."+entry.name, "qualified evidence must reference its owner receipt"))
		}
	}
	return issues
}

type namedEvidenceFacet struct {
	name  string
	facet EvidenceFacet
}

func promotionFacets(p PromotionEvidence) []namedEvidenceFacet {
	return []namedEvidenceFacet{
		{"corpus_quality", p.CorpusQuality},
		{"latency", p.Latency},
		{"memory", p.Memory},
		{"platform", p.Platform},
		{"license", p.License},
	}
}

func validEvidenceStatus(status string) bool {
	return status == "" || status == "qualified" || status == "unqualified" || status == "not_measured" || status == "unknown"
}

func validateModelReadinessNotes(m Model) []Issue {
	if (m.Readiness == ReadinessExperimental || m.Readiness == ReadinessWatch) && strings.TrimSpace(m.Notes) == "" {
		return []Issue{newIssue(IssueMissingNotes, m.ID, "notes", "readiness=%s requires notes stating what is unproven", m.Readiness)}
	}
	return nil
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
