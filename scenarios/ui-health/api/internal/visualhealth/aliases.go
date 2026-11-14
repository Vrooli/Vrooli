package visualhealth

import visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"

type (
	stepArtifact  = *visualpb.VisualStepArtifact
	visualFinding = visualpb.VisualFinding
)

const (
	severityWarning = visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING
	severityError   = visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR

	categoryDOM    = visualpb.VisualCategory_VISUAL_CATEGORY_DOM
	categoryLayout = visualpb.VisualCategory_VISUAL_CATEGORY_LAYOUT
	categoryAsset  = visualpb.VisualCategory_VISUAL_CATEGORY_ASSET
	categoryFocus  = visualpb.VisualCategory_VISUAL_CATEGORY_FOCUS
)
