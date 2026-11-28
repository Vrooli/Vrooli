package workflows

import (
	"context"

	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// NewDefaultValidator wraps the in-process workflow validator into the
// Validator seam, returning proto-shaped results to the service layer.
func NewDefaultValidator(v *workflowvalidator.Validator) Validator {
	return &defaultValidator{inner: v}
}

type defaultValidator struct {
	inner *workflowvalidator.Validator
}

func (d *defaultValidator) ValidateDefinition(_ context.Context, def *basworkflows.WorkflowDefinitionV2) *basapi.WorkflowValidationResult {
	if d == nil || d.inner == nil || def == nil {
		return &basapi.WorkflowValidationResult{Valid: false}
	}
	result := d.inner.ValidateV2(def)
	return protoconv.WorkflowValidationResultToProto(result)
}
