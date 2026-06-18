// Package validation applies proto-health policy to a scenario-scoped proto
// surface. It does not compute fleet graphs or dependency drift.
package validation

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"proto-health/internal/protosurface"

	"github.com/vrooli/maturity-go/assessment"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

const (
	CodeCycle                         = "proto.cycle"
	CodeGenManifestMissing            = "proto.gen_manifest_missing"
	CodeGenOutOfSync                  = "proto.gen_out_of_sync"
	CodeGenToolchainDrift             = "proto.gen_toolchain_drift"
	CodePackageMismatch               = "proto.package_mismatch"
	CodeStabilityDishonest            = "proto.stability_dishonest"
	CodeCrossDomainImport             = "proto.cross_domain_import"
	CodeUnsupportedAnnotation         = "proto.unsupported_annotation"
	CodeTemplateSource                = "proto.template_source"
	CodeHandRolledTransport           = "proto.hand_rolled_transport"
	CodeVersionNaming                 = "proto.version_naming"
	CodeDomainMismatch                = "proto.domain_mismatch"
	CodeMissingHealthProto            = "proto.missing_health_proto"
	CodePossiblyUnused                = "proto.possibly_unused"
	CodeRESTPayloadMissingDeclaration = "proto.rest_payload_missing_declaration"
	CodeRESTPayloadUnknownMessage     = "proto.rest_payload_unknown_message"
	CodeRESTPayloadInvalidConformance = "proto.rest_payload_invalid_conformance"
	CodeStabilityDependencyMismatch   = "proto.stability_dependency_mismatch"
	CodeSharedTypeMisplaced           = "proto.shared_type_misplaced"
	CodeImportKindUnknown             = "proto.import_kind_unknown"
	CodeCodeFactsUnavailable          = "proto.code_facts_unavailable"
	CodeProtoAdoptionMissing          = "proto.proto_adoption_missing"
	CodeProtoAdoptionUnsupported      = "proto.proto_adoption_unsupported"
	CodeProtoAdoptionContradicted     = "proto.proto_adoption_contradicted"
	CodeEndpointProofMissing          = "proto.endpoint_proof_missing"
	CodeEndpointProofUnsupported      = "proto.endpoint_proof_unsupported"
	CodeEndpointProofContradicted     = "proto.endpoint_proof_contradicted"
)

type Finding struct {
	Severity   Severity
	Code       string
	Location   string
	Message    string
	Suggestion string
}

type Summary struct {
	Errors   int
	Warnings int
	Infos    int
}

type Report struct {
	Scenario string
	Passed   bool
	Findings []Finding
	Summary  Summary
}

type FindingMetadata struct {
	Severity            Severity
	LocalLevelImpact    string
	GlobalImpact        assessment.GlobalImpact
	Dimension           string
	RecommendedSkillIDs []string
}

type FindingCatalog map[string]FindingMetadata

func NewFindingCatalog(spec *assessment.Spec) (FindingCatalog, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	out := make(FindingCatalog, len(spec.Findings))
	for _, code := range AllFindingCodes() {
		mapping, ok := spec.Findings[code]
		if !ok {
			return nil, fmt.Errorf("maturity spec missing finding %s", code)
		}
		severity, err := severityFromToken(mapping.SeverityDefault)
		if err != nil {
			return nil, fmt.Errorf("maturity spec finding %s: %w", code, err)
		}
		out[code] = FindingMetadata{
			Severity:            severity,
			LocalLevelImpact:    mapping.LocalLevelImpact,
			GlobalImpact:        mapping.GlobalImpact,
			Dimension:           mapping.Dimension,
			RecommendedSkillIDs: append([]string(nil), mapping.RecommendedSkillIDs...),
		}
	}
	for code := range spec.Findings {
		if _, ok := out[code]; !ok {
			return nil, fmt.Errorf("maturity spec has orphaned finding %s", code)
		}
	}
	return out, nil
}

func (c FindingCatalog) ResolveSeverity(code string) (Severity, error) {
	meta, ok := c[code]
	if !ok {
		return "", fmt.Errorf("finding code %s is not in maturity catalog", code)
	}
	return meta.Severity, nil
}

func severityFromToken(token string) (Severity, error) {
	switch strings.TrimSpace(token) {
	case "SEVERITY_ERROR", "ERROR", "FINDING_SEVERITY_ERROR":
		return SeverityError, nil
	case "SEVERITY_WARNING", "WARNING", "FINDING_SEVERITY_WARNING":
		return SeverityWarning, nil
	case "SEVERITY_INFO", "INFO", "FINDING_SEVERITY_INFO":
		return SeverityInfo, nil
	default:
		return "", fmt.Errorf("unsupported severity_default %q", token)
	}
}

func AllFindingCodes() []string {
	return []string{
		CodeCycle,
		CodeGenManifestMissing,
		CodeGenOutOfSync,
		CodeGenToolchainDrift,
		CodePackageMismatch,
		CodeStabilityDishonest,
		CodeCrossDomainImport,
		CodeUnsupportedAnnotation,
		CodeTemplateSource,
		CodeHandRolledTransport,
		CodeVersionNaming,
		CodeDomainMismatch,
		CodeMissingHealthProto,
		CodePossiblyUnused,
		CodeRESTPayloadMissingDeclaration,
		CodeRESTPayloadUnknownMessage,
		CodeRESTPayloadInvalidConformance,
		CodeStabilityDependencyMismatch,
		CodeSharedTypeMisplaced,
		CodeImportKindUnknown,
		CodeCodeFactsUnavailable,
		CodeProtoAdoptionMissing,
		CodeProtoAdoptionUnsupported,
		CodeProtoAdoptionContradicted,
		CodeEndpointProofMissing,
		CodeEndpointProofUnsupported,
		CodeEndpointProofContradicted,
	}
}

type SurfaceResult struct {
	Scenario string
	Surface  protosurface.Surface
	Error    string
}

type SurfaceLoader interface {
	LoadScenario(scenario string) (protosurface.Surface, error)
	ListScenarios() ([]string, error)
}

type GenSyncStatus struct {
	InSync          bool
	ManifestMissing bool
	ToolchainDrift  bool
	Drift           []string
	Detail          string
	Skipped         bool
	SkipMessage     string
}

type GenSyncChecker interface {
	CheckScenario(ctx context.Context, scenario string) (GenSyncStatus, error)
}

type CodeFactsClient interface {
	CheckProtoAdoption(ctx context.Context, scenario string) (*factsv1.ProofReport, error)
	CheckEndpointProof(ctx context.Context, scenario string, endpointIDs []string) (*factsv1.ProofReport, error)
}

func summarize(findings []Finding) Summary {
	var s Summary
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			s.Errors++
		case SeverityWarning:
			s.Warnings++
		case SeverityInfo:
			s.Infos++
		}
	}
	return s
}

func finalize(scenario string, findings []Finding) Report {
	summary := summarize(findings)
	return Report{
		Scenario: scenario,
		Passed:   summary.Errors == 0,
		Findings: findings,
		Summary:  summary,
	}
}

func SortedCatalogCodes(c FindingCatalog) []string {
	out := make([]string, 0, len(c))
	for code := range c {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}
