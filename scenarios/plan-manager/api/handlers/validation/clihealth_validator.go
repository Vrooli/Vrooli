package validation

import (
	"plan-manager/internal/clihealth"
	internalvalidation "plan-manager/internal/validation"
)

type cliHealthCommandValidator = clihealth.Adapter[internalvalidation.CommandReferenceRequest, internalvalidation.CommandReferenceResult]

func newCLIHealthCommandValidator() cliHealthCommandValidator {
	return clihealth.NewAdapter(validationCommandReferenceRequest, validationCommandReferenceResult)
}

func validationCommandReferenceRequest(req internalvalidation.CommandReferenceRequest) clihealth.Request {
	return clihealth.Request{CommandText: req.CommandText, Qualifiers: req.Qualifiers}
}

func validationCommandReferenceResult(result clihealth.Result) internalvalidation.CommandReferenceResult {
	out := internalvalidation.CommandReferenceResult{
		Verdict:         result.Verdict,
		ValidationLevel: result.ValidationLevel,
		Suggestions:     append([]string(nil), result.Suggestions...),
		Guidance:        append([]string(nil), result.Guidance...),
	}
	for _, issue := range result.Issues {
		out.Issues = append(out.Issues, internalvalidation.CommandIssue{Code: issue.Code, Message: issue.Message})
	}
	return out
}
