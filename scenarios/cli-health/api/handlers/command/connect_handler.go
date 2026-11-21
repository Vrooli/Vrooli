package command

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"cli-health/internal/commandref"

	commandv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command"
)

type Validator interface {
	Validate(ctx context.Context, req commandref.Request) commandref.Result
}

type Deps struct {
	Logger    *log.Logger
	Validator Validator
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateCommandReference(ctx context.Context, req *connect.Request[commandv1.ValidateCommandReferenceRequest]) (*connect.Response[commandv1.ValidateCommandReferenceResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("command reference validator not configured"))
	}
	result := h.deps.Validator.Validate(ctx, requestFromProto(req.Msg))
	return connect.NewResponse(&commandv1.ValidateCommandReferenceResponse{Result: resultToProto(result)}), nil
}

func (h *connectHandler) ValidateCommandReferences(ctx context.Context, req *connect.Request[commandv1.ValidateCommandReferencesRequest]) (*connect.Response[commandv1.ValidateCommandReferencesResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("command reference validator not configured"))
	}
	resp := &commandv1.ValidateCommandReferencesResponse{}
	for _, r := range req.Msg.GetRequests() {
		resp.Results = append(resp.Results, resultToProto(h.deps.Validator.Validate(ctx, requestFromProto(r))))
	}
	return connect.NewResponse(resp), nil
}

func requestFromProto(r *commandv1.ValidateCommandReferenceRequest) commandref.Request {
	if r == nil {
		return commandref.Request{}
	}
	return commandref.Request{
		CommandText:   r.GetCommandText(),
		Policy:        r.GetPolicy().String(),
		Qualifiers:    append([]string(nil), r.GetQualifiers()...),
		RefreshPolicy: r.GetRefreshPolicy().String(),
	}
}

func resultToProto(r commandref.Result) *commandv1.CommandReferenceValidationResult {
	out := &commandv1.CommandReferenceValidationResult{
		CommandText:      r.CommandText,
		Verdict:          verdictToProto(r.Verdict),
		ValidationLevel:  levelToProto(r.Level),
		CanonicalCommand: r.CanonicalCommand,
		Owner:            r.Owner,
		Source:           r.Source,
		Guidance:         append([]string(nil), r.Guidance...),
	}
	for _, issue := range r.Issues {
		out.Issues = append(out.Issues, &commandv1.CommandReferenceIssue{
			Code:     issue.Code,
			Message:  issue.Message,
			Severity: issue.Severity,
			Fix:      issue.Fix,
		})
	}
	for _, suggestion := range r.Suggestions {
		out.Suggestions = append(out.Suggestions, &commandv1.CommandReferenceSuggestion{
			Command: suggestion.Command,
			Reason:  suggestion.Reason,
		})
	}
	return out
}

func verdictToProto(v commandref.Verdict) commandv1.CommandReferenceVerdict {
	switch v {
	case commandref.VerdictValid:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_VALID
	case commandref.VerdictInvalid:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_INVALID
	case commandref.VerdictPartial:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_PARTIAL
	case commandref.VerdictSkipped:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_SKIPPED
	case commandref.VerdictUnsupported:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_UNSUPPORTED
	default:
		return commandv1.CommandReferenceVerdict_COMMAND_REFERENCE_VERDICT_UNKNOWN
	}
}

func levelToProto(v commandref.Level) commandv1.CommandReferenceValidationLevel {
	switch v {
	case commandref.LevelParsed:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_PARSED
	case commandref.LevelOwnerIdentified:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_OWNER_IDENTIFIED
	case commandref.LevelCommandExists:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_COMMAND_EXISTS
	case commandref.LevelArgumentShapeValidated:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_ARGUMENT_SHAPE_VALIDATED
	case commandref.LevelSkippedByQualifier:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_SKIPPED_BY_QUALIFIER
	case commandref.LevelUnsupportedSyntax:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_UNSUPPORTED_SYNTAX
	default:
		return commandv1.CommandReferenceValidationLevel_COMMAND_REFERENCE_VALIDATION_LEVEL_UNSPECIFIED
	}
}
