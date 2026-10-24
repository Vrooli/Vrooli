package rewrite

import (
	"path"
	"sort"
	"strings"
)

// Normalize returns a deterministic, deduped, canonicalized copy of ops.
//
// The canonical key for each operation embeds the kind discriminator so
// FileMove and ImportRewrite never collide; the secondary key is the
// path tuple. After dedup the slice is sorted by canonical key so
// identical input multisets always produce identical output orderings —
// which is what makes PlanID derivation stable.
//
// Path canonicalization for FileMove (paths are module-root-relative):
// trailing slashes trimmed, "./" prefixes collapsed, internal "//"
// collapsed. ".." segments and absolute paths are rejected up the call
// stack in validate(); Normalize assumes its input has already passed
// validation.
func Normalize(ops []Operation) []Operation {
	if len(ops) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ops))
	out := make([]Operation, 0, len(ops))
	for _, op := range ops {
		var normalized Operation
		switch o := op.(type) {
		case FileMove:
			normalized = FileMove{From: canonicalPath(o.From), To: canonicalPath(o.To)}
		case ImportRewrite:
			normalized = ImportRewrite{Old: canonicalImport(o.Old), New: canonicalImport(o.New)}
		default:
			continue
		}
		key := canonicalKey(normalized)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return canonicalKey(out[i]) < canonicalKey(out[j])
	})
	return out
}

// canonicalKey serializes an operation to the stable sort/dedup key.
// Format: "<kind>\x00<field1>\x00<field2>". The NUL delimiter is safe
// because validated operation fields cannot contain NUL bytes.
func canonicalKey(op Operation) string {
	switch o := op.(type) {
	case FileMove:
		return string(OperationKindFileMove) + "\x00" + o.From + "\x00" + o.To
	case ImportRewrite:
		return string(OperationKindImportRewrite) + "\x00" + o.Old + "\x00" + o.New
	}
	return ""
}

// canonicalPath collapses "./" prefixes, internal "//", and trailing
// slashes. It uses path.Clean which is the right tool for
// forward-slash, relative module-root paths.
func canonicalPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	cleaned := path.Clean(p)
	// path.Clean turns "" into "." — undo that.
	if cleaned == "." {
		return ""
	}
	return cleaned
}

// canonicalImport strips whitespace and trailing slashes from import
// paths. Go import paths are forward-slash-separated and don't carry
// "./" prefixes; we just normalize whitespace + trailing slash.
func canonicalImport(p string) string {
	p = strings.TrimSpace(p)
	return strings.TrimRight(p, "/")
}

// ValidateOperations checks every operation in the input list is
// well-formed: non-empty fields, no absolute paths in FileMove, no
// ".." segments. Returns a typed RewriteError on the first failure.
//
// Empty operations lists are NOT rejected here — the Service checks
// that separately (different typed-error kind).
func ValidateOperations(ops []Operation) error {
	for i, op := range ops {
		switch o := op.(type) {
		case FileMove:
			if err := validateRelPath("file_move", "from_path", o.From, i); err != nil {
				return err
			}
			if err := validateRelPath("file_move", "to_path", o.To, i); err != nil {
				return err
			}
			if canonicalPath(o.From) == canonicalPath(o.To) {
				return RewriteError{
					Kind:    RewriteErrorMalformedOperation,
					Message: "file_move from_path and to_path are identical",
				}
			}
		case ImportRewrite:
			if strings.TrimSpace(o.Old) == "" || strings.TrimSpace(o.New) == "" {
				return RewriteError{
					Kind:    RewriteErrorMalformedOperation,
					Message: "import_rewrite old_path and new_path must be non-empty",
				}
			}
			if canonicalImport(o.Old) == canonicalImport(o.New) {
				return RewriteError{
					Kind:    RewriteErrorMalformedOperation,
					Message: "import_rewrite old_path and new_path are identical",
				}
			}
		default:
			return RewriteError{
				Kind:    RewriteErrorMalformedOperation,
				Message: "unknown operation type",
			}
		}
	}
	return nil
}

// validateRelPath rejects empty fields, absolute paths, and ".."
// segments. Used only by FileMove fields.
func validateRelPath(kind, field, p string, _ int) error {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return RewriteError{
			Kind:    RewriteErrorMalformedOperation,
			Message: kind + " " + field + " is required",
		}
	}
	if strings.HasPrefix(trimmed, "/") {
		return RewriteError{
			Kind:    RewriteErrorMalformedOperation,
			Message: kind + " " + field + " must be module-root-relative, not absolute",
		}
	}
	for _, seg := range strings.Split(trimmed, "/") {
		if seg == ".." {
			return RewriteError{
				Kind:    RewriteErrorMalformedOperation,
				Message: kind + " " + field + " must not contain '..' segments",
			}
		}
	}
	return nil
}
