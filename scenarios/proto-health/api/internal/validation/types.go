// Package validation applies proto-health policy to a scenario-scoped proto
// surface. It does not compute fleet graphs or dependency drift.
package validation

import (
	"context"

	"proto-health/internal/protosurface"

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
	CodeGenOutOfSync                  = "proto.gen_out_of_sync"
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
	InSync bool
	Drift  []string
	Detail string
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
