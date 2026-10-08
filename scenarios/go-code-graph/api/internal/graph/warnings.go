package graph

import (
	"strings"

	"golang.org/x/tools/go/packages"
)

// packageWarnings translates p.Errors to typed Warnings. The
// classification heuristics are deliberately narrow — anything we don't
// recognize bubbles up as a TypeCheckFailure rather than masquerading
// as a "successful" extraction.
//
// Catastrophic project errors (no go.mod, multiple go.mod, go.work,
// path unreadable) are NOT emitted here — the Service detects those
// before calling the loader and returns ExtractError. This function
// only handles per-file/per-package partial failures.
func packageWarnings(p *packages.Package) []Warning {
	if p == nil || len(p.Errors) == 0 {
		return nil
	}
	out := make([]Warning, 0, len(p.Errors))
	for _, e := range p.Errors {
		out = append(out, Warning{
			Kind:    classifyPackageError(e),
			File:    parseErrorFile(e),
			Message: e.Msg,
		})
	}
	return out
}

// classifyPackageError maps a packages.Error to a WarningKind. The
// packages package exposes three kinds: ListError, ParseError,
// TypeError. We treat ListError as UnresolvedImport (the loader uses
// it for "could not load package <path>" failures), ParseError as
// ParseError, and TypeError as TypeCheckFailure.
func classifyPackageError(e packages.Error) WarningKind {
	switch e.Kind {
	case packages.ParseError:
		return WarningKindParseError
	case packages.ListError:
		return WarningKindUnresolvedImport
	case packages.TypeError:
		return WarningKindTypeCheckFailure
	default:
		return WarningKindTypeCheckFailure
	}
}

// parseErrorFile extracts the file path from a packages.Error position
// string of the form "file:line:col". Returns "" when no leading file
// component is present.
func parseErrorFile(e packages.Error) string {
	if e.Pos == "" {
		return ""
	}
	// Position strings look like "/abs/path/foo.go:12:3" or just "-".
	if i := strings.Index(e.Pos, ":"); i > 0 {
		return e.Pos[:i]
	}
	return ""
}
