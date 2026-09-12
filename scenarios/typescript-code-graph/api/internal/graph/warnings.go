package graph

import "typescript-code-graph/internal/sidecar"

// fromSidecarWarnings converts the sidecar's wire-shape Warning slice
// into the domain Warning type, decoding the numeric
// common.v1.CodeGraphWarningKind into a typed WarningKind. Unknown
// kinds map to TypeCheckFailure rather than silently dropping the
// warning — visible-but-classified is always safer than invisible.
func fromSidecarWarnings(ws []sidecar.Warning) []Warning {
	if len(ws) == 0 {
		return nil
	}
	out := make([]Warning, 0, len(ws))
	for _, w := range ws {
		out = append(out, Warning{
			Kind:    classifyWarningKind(w.Kind),
			File:    w.File,
			Message: w.Message,
		})
	}
	return out
}

// classifyWarningKind maps a numeric common.v1.CodeGraphWarningKind to a
// domain WarningKind. The enum values are stable
// (1=PARSE_ERROR, 2=UNRESOLVED_IMPORT, 3=TYPE_CHECK_FAILURE); anything
// outside the known set is treated as a type-check failure so the
// caller still sees it.
func classifyWarningKind(kind int32) WarningKind {
	switch kind {
	case 1:
		return WarningKindParseError
	case 2:
		return WarningKindUnresolvedImport
	case 3:
		return WarningKindTypeCheckFailure
	default:
		return WarningKindTypeCheckFailure
	}
}
