package focus

import (
	internalfocus "meta-optimization-manager/internal/focus"

	"github.com/vrooli/api-core/spacedoc"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"
)

// This file is the only translation point between the proto wire enums
// (vrooli...v1.shared) and the domain vocabulary (api-core/spacedoc +
// internal/focus). The domain layer never imports proto (api-steer §7).

func projFromProto(p sharedv1.Projection) internalfocus.Projection {
	switch p {
	case sharedv1.Projection_PROJECTION_ANSWER:
		return internalfocus.ProjectionAnswer
	case sharedv1.Projection_PROJECTION_VALIDATE:
		return internalfocus.ProjectionValidate
	case sharedv1.Projection_PROJECTION_GUIDE:
		return internalfocus.ProjectionGuide
	case sharedv1.Projection_PROJECTION_ACT:
		return internalfocus.ProjectionAct
	default:
		return "" // UNSPECIFIED => all / not projection-scoped
	}
}

func projToProto(p internalfocus.Projection) sharedv1.Projection {
	switch p {
	case internalfocus.ProjectionAnswer:
		return sharedv1.Projection_PROJECTION_ANSWER
	case internalfocus.ProjectionValidate:
		return sharedv1.Projection_PROJECTION_VALIDATE
	case internalfocus.ProjectionGuide:
		return sharedv1.Projection_PROJECTION_GUIDE
	case internalfocus.ProjectionAct:
		return sharedv1.Projection_PROJECTION_ACT
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

func axisToProto(axis internalfocus.Axis) sharedv1.GapAxis {
	switch axis {
	case internalfocus.AxisCoverage:
		return sharedv1.GapAxis_GAP_AXIS_COVERAGE
	case internalfocus.AxisEmpirical:
		return sharedv1.GapAxis_GAP_AXIS_EMPIRICAL
	default:
		return sharedv1.GapAxis_GAP_AXIS_UNSPECIFIED
	}
}
