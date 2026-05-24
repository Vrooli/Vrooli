package graph

import "typescript-code-graph/internal/sidecar"

// fromSidecarWarnings converts the sidecar's wire-shape Warning slice
// into the domain Warning type, classifying the string Code into a
// typed WarningKind. Unknown codes map to TypeCheckFailure rather than
// silently dropping the warning — visible-but-classified is always
// safer than invisible.
func fromSidecarWarnings(ws []sidecar.Warning) []Warning {
	if len(ws) == 0 {
		return nil
	}
	out := make([]Warning, 0, len(ws))
	for _, w := range ws {
		out = append(out, Warning{
			Kind:    classifyWarningCode(w.Code),
			File:    w.Path,
			Message: w.Message,
		})
	}
	return out
}

// classifyWarningCode maps a sidecar warning code to a WarningKind.
// The closed set lives in plan §8.4; anything outside it is treated as
// a type-check failure so the caller still sees it.
func classifyWarningCode(code string) WarningKind {
	switch code {
	case "parse_failure", "parse_error":
		return WarningKindParseError
	case "unresolved_import":
		return WarningKindUnresolvedImport
	case "type_check_failure":
		return WarningKindTypeCheckFailure
	default:
		return WarningKindTypeCheckFailure
	}
}
