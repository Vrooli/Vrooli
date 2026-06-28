package coverage

import (
	internalcoverage "meta-optimization-manager/internal/coverage"

	"github.com/vrooli/api-core/spacedoc"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"
)

// This file is the only translation point between the proto wire enums
// (vrooli...v1.shared) and the domain vocabulary (api-core/spacedoc +
// internal/coverage). The domain layer never imports proto (api-steer §7).

func projFromProto(p sharedv1.Projection) internalcoverage.Projection {
	switch p {
	case sharedv1.Projection_PROJECTION_ANSWER:
		return internalcoverage.ProjectionAnswer
	case sharedv1.Projection_PROJECTION_VALIDATE:
		return internalcoverage.ProjectionValidate
	case sharedv1.Projection_PROJECTION_GUIDE:
		return internalcoverage.ProjectionGuide
	default:
		return "" // UNSPECIFIED => all projections
	}
}

func projToProto(p internalcoverage.Projection) sharedv1.Projection {
	switch p {
	case internalcoverage.ProjectionAnswer:
		return sharedv1.Projection_PROJECTION_ANSWER
	case internalcoverage.ProjectionValidate:
		return sharedv1.Projection_PROJECTION_VALIDATE
	case internalcoverage.ProjectionGuide:
		return sharedv1.Projection_PROJECTION_GUIDE
	default:
		return sharedv1.Projection_PROJECTION_UNSPECIFIED
	}
}

func statusFromProto(s sharedv1.CellStatus) spacedoc.CellStatus {
	switch s {
	case sharedv1.CellStatus_CELL_STATUS_NOW:
		return spacedoc.StatusNow
	case sharedv1.CellStatus_CELL_STATUS_IN_REACH:
		return spacedoc.StatusInReach
	case sharedv1.CellStatus_CELL_STATUS_MISSING:
		return spacedoc.StatusMissing
	default:
		return "" // UNSPECIFIED => no filter
	}
}

func statusToProto(s spacedoc.CellStatus) sharedv1.CellStatus {
	switch s {
	case spacedoc.StatusNow:
		return sharedv1.CellStatus_CELL_STATUS_NOW
	case spacedoc.StatusInReach:
		return sharedv1.CellStatus_CELL_STATUS_IN_REACH
	case spacedoc.StatusMissing:
		return sharedv1.CellStatus_CELL_STATUS_MISSING
	default:
		return sharedv1.CellStatus_CELL_STATUS_UNSPECIFIED
	}
}

func basisToProto(b spacedoc.Basis) commonv1.Basis {
	switch b {
	case spacedoc.BasisDerived:
		return commonv1.Basis_BASIS_DERIVED
	case spacedoc.BasisValidated:
		return commonv1.Basis_BASIS_VALIDATED
	case spacedoc.BasisDeclaredUnverified:
		return commonv1.Basis_BASIS_DECLARED_UNVERIFIED
	case spacedoc.BasisContradicted:
		return commonv1.Basis_BASIS_CONTRADICTED
	case spacedoc.BasisAbsent:
		return commonv1.Basis_BASIS_ABSENT
	default:
		return commonv1.Basis_BASIS_UNSPECIFIED
	}
}

func confToProto(c spacedoc.DenominatorConfidence) sharedv1.DenominatorConfidence {
	switch c {
	case spacedoc.ConfidenceAuthoritative:
		return sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_AUTHORITATIVE
	case spacedoc.ConfidencePartial:
		return sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_PARTIAL
	case spacedoc.ConfidenceSketch:
		return sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_SKETCH
	default:
		return sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_UNSPECIFIED
	}
}

func sufficiencyToProto(s string) commonv1.Sufficiency {
	switch s {
	case "full":
		return commonv1.Sufficiency_SUFFICIENCY_FULL
	case "partial":
		return commonv1.Sufficiency_SUFFICIENCY_PARTIAL
	case "insufficient":
		return commonv1.Sufficiency_SUFFICIENCY_INSUFFICIENT
	default:
		return commonv1.Sufficiency_SUFFICIENCY_UNSPECIFIED
	}
}

func severityToProto(s internalcoverage.Severity) sharedv1.Severity {
	switch s {
	case internalcoverage.SeverityError:
		return sharedv1.Severity_SEVERITY_ERROR
	case internalcoverage.SeverityWarn:
		return sharedv1.Severity_SEVERITY_WARN
	case internalcoverage.SeverityInfo:
		return sharedv1.Severity_SEVERITY_INFO
	default:
		return sharedv1.Severity_SEVERITY_UNSPECIFIED
	}
}
