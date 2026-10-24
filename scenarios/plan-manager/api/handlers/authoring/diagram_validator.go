package authoring

import (
	"context"

	internalauthoring "plan-manager/internal/authoring"
	"plan-manager/internal/knowledgeobservatory"
)

type diagramRequest struct {
	content string
	source  string
}

type diagramValidator struct {
	adapter knowledgeobservatory.Adapter[diagramRequest, internalauthoring.DiagramValidationResult]
}

func newDiagramValidator() diagramValidator {
	return diagramValidator{adapter: knowledgeobservatory.NewAdapter(
		func(request diagramRequest) knowledgeobservatory.Request {
			return knowledgeobservatory.Request{Content: request.content, Source: request.source}
		},
		func(result knowledgeobservatory.Result) internalauthoring.DiagramValidationResult {
			out := internalauthoring.DiagramValidationResult{Unverified: result.Unverified}
			for _, finding := range result.Findings {
				out.Findings = append(out.Findings, internalauthoring.DiagramFinding{Code: finding.Code, Message: finding.Message, Line: finding.Line})
			}
			return out
		},
	)}
}

func (v diagramValidator) ValidateMarkdownDiagrams(ctx context.Context, content, source string) (internalauthoring.DiagramValidationResult, error) {
	return v.adapter.ValidateMarkdownDiagrams(ctx, diagramRequest{content: content, source: source})
}
