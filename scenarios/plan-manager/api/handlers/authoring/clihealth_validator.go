package authoring

import (
	internalauthoring "plan-manager/internal/authoring"
	"plan-manager/internal/clihealth"
)

type cliHealthCommandValidator = clihealth.Adapter[internalauthoring.CommandReferenceRequest, internalauthoring.CommandReferenceResult]

func newCLIHealthCommandValidator() cliHealthCommandValidator {
	return clihealth.NewAdapter(authoringCommandReferenceRequest, authoringCommandReferenceResult)
}

func authoringCommandReferenceRequest(req internalauthoring.CommandReferenceRequest) clihealth.Request {
	return clihealth.Request{CommandText: req.CommandText, Qualifiers: req.Qualifiers}
}

func authoringCommandReferenceResult(result clihealth.Result) internalauthoring.CommandReferenceResult {
	out := internalauthoring.CommandReferenceResult{
		Verdict:         result.Verdict,
		ValidationLevel: result.ValidationLevel,
		Suggestions:     append([]string(nil), result.Suggestions...),
		Guidance:        append([]string(nil), result.Guidance...),
	}
	for _, issue := range result.Issues {
		out.Issues = append(out.Issues, internalauthoring.CommandIssue{Code: issue.Code, Message: issue.Message})
	}
	return out
}
