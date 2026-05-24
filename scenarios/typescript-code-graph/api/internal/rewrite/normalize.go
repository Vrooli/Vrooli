package rewrite

import (
	"path"
	"sort"
	"strings"
)

// Normalize canonicalizes ops:
//
//  1. Each operation is validated structurally (exactly-one-oneof, paths
//     non-empty, paths != each other).
//  2. Each path is canonicalized: leading "./" stripped, separators
//     normalized to forward-slash, redundant "." / "//" collapsed.
//     Absolute paths and ".." segments are rejected as invalid.
//  3. The resulting slice is sorted by (oneof_tag, primary_path,
//     secondary_path) so identical operation sets produce identical
//     byte sequences (precondition for the PlanID hash).
//  4. Exact duplicates (post-canonicalization) are deduplicated.
//
// Returns a fresh slice; ops is not mutated.
//
// Validation errors return RewriteError{Kind:RewriteErrorInvalidOperation}
// so handlers map them to InvalidArgument.
func Normalize(ops []Operation) ([]Operation, error) {
	if len(ops) == 0 {
		return nil, RewriteError{
			Kind:    RewriteErrorInvalidInput,
			Message: "operations list is empty",
		}
	}

	out := make([]Operation, 0, len(ops))
	for i, op := range ops {
		canon, err := canonicalizeOperation(op)
		if err != nil {
			// Wrap with the failing index so the caller can pinpoint
			// the bad operation in a long list.
			if re, ok := err.(RewriteError); ok {
				re.Message = appendIndex(re.Message, i)
				return nil, re
			}
			return nil, err
		}
		out = append(out, canon)
	}

	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := out[i].OperationTag(), out[j].OperationTag()
		if ti != tj {
			return ti < tj
		}
		pi, pj := out[i].PrimaryPath(), out[j].PrimaryPath()
		if pi != pj {
			return pi < pj
		}
		return out[i].SecondaryPath() < out[j].SecondaryPath()
	})

	// Dedup adjacent duplicates after sort. The compare is structural:
	// same tag, same primary, same secondary, same pointer-shape (and
	// thus same effective operation).
	dedup := out[:0]
	for i, op := range out {
		if i == 0 || !operationsEqual(out[i-1], op) {
			dedup = append(dedup, op)
		}
	}
	return dedup, nil
}

// canonicalizeOperation validates one Operation and returns a copy
// with canonicalized paths. Pointer arms are reallocated so the
// caller's input is not aliased.
func canonicalizeOperation(op Operation) (Operation, error) {
	switch op.OperationTag() {
	case "file_move":
		fm := *op.FileMove
		from, err := canonicalizePath(fm.FromPath)
		if err != nil {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: fm.FromPath, Message: "from_path: " + err.Error()}
		}
		to, err := canonicalizePath(fm.ToPath)
		if err != nil {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: fm.ToPath, Message: "to_path: " + err.Error()}
		}
		if from == to {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: from, Message: "file_move from_path equals to_path"}
		}
		return Operation{FileMove: &FileMove{FromPath: from, ToPath: to}}, nil

	case "import_rewrite":
		ir := *op.ImportRewrite
		oldP, err := canonicalizePath(ir.OldPath)
		if err != nil {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: ir.OldPath, Message: "old_path: " + err.Error()}
		}
		newP, err := canonicalizePath(ir.NewPath)
		if err != nil {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: ir.NewPath, Message: "new_path: " + err.Error()}
		}
		if oldP == newP {
			return Operation{}, RewriteError{Kind: RewriteErrorInvalidOperation, Path: oldP, Message: "import_rewrite old_path equals new_path"}
		}
		return Operation{ImportRewrite: &ImportRewrite{OldPath: oldP, NewPath: newP}}, nil

	default:
		// Either both arms set or neither — both invalid.
		return Operation{}, RewriteError{
			Kind:    RewriteErrorInvalidOperation,
			Message: "operation must have exactly one of file_move or import_rewrite set",
		}
	}
}

// canonicalizePath enforces the path discipline declared in the plan:
// strip leading "./", normalize separators to "/", reject empty inputs,
// reject absolute paths, reject any segment equal to "..".
func canonicalizePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errPathEmpty
	}
	// Normalize separators first so the rest of the checks see a
	// uniform form. We intentionally do not call filepath.Clean —
	// platform-specific path handling would let a Windows-style path
	// sneak past validation on Linux. path.Clean works on slash paths
	// only, which is what the sidecar consumes.
	p = strings.ReplaceAll(p, "\\", "/")
	if strings.HasPrefix(p, "/") {
		return "", errPathAbsolute
	}
	// Strip leading "./" repeatedly so "././x" → "x".
	for strings.HasPrefix(p, "./") {
		p = p[2:]
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == "" {
		return "", errPathEmpty
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.Contains("/"+cleaned+"/", "/../") {
		return "", errPathParentEscape
	}
	return cleaned, nil
}

// operationsEqual returns true iff two ops are structurally identical
// after canonicalization (used for dedup).
func operationsEqual(a, b Operation) bool {
	if a.OperationTag() != b.OperationTag() {
		return false
	}
	switch a.OperationTag() {
	case "file_move":
		return a.FileMove.FromPath == b.FileMove.FromPath && a.FileMove.ToPath == b.FileMove.ToPath
	case "import_rewrite":
		return a.ImportRewrite.OldPath == b.ImportRewrite.OldPath && a.ImportRewrite.NewPath == b.ImportRewrite.NewPath
	}
	return false
}

// errPath* are validation sentinels; canonicalizePath wraps them with
// the path in a RewriteError up the call stack.
type pathErr string

func (e pathErr) Error() string { return string(e) }

const (
	errPathEmpty        = pathErr("path is empty")
	errPathAbsolute     = pathErr("path must be scenario-relative, not absolute")
	errPathParentEscape = pathErr("path must not contain .. segments")
)

func appendIndex(msg string, i int) string {
	if msg == "" {
		return "operation index " + itoa(i)
	}
	return msg + " (operation index " + itoa(i) + ")"
}

func itoa(i int) string {
	// crypto/sha256 path uses encoding/hex — we deliberately avoid
	// pulling in strconv here for one call. Manual digit conversion
	// keeps the package's import surface minimal and signals that this
	// is a leaf utility, not a general-purpose helper.
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
