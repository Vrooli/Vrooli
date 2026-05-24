package rewrite

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FSExecutor is the production RewriteExecutor. It performs operations
// directly against the filesystem under scenarioRoot using stdlib
// primitives: os.Rename for FileMove and go/parser + go/format for
// ImportRewrite. It NEVER invokes git, go, or any subprocess —
// no_external_command_test.go enforces this at the test level.
type FSExecutor struct{}

// NewFSExecutor returns a ready-to-use FSExecutor.
func NewFSExecutor() *FSExecutor { return &FSExecutor{} }

// Execute applies op against scenarioRoot. It rejects operations that
// would escape scenarioRoot (path traversal, absolute paths) before
// touching disk.
func (e *FSExecutor) Execute(ctx context.Context, scenarioRoot string, op Operation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	absRoot, err := filepath.Abs(scenarioRoot)
	if err != nil {
		return fmt.Errorf("resolve scenario root: %w", err)
	}
	switch o := op.(type) {
	case FileMove:
		return e.executeFileMove(absRoot, o)
	case ImportRewrite:
		return e.executeImportRewrite(absRoot, o)
	default:
		return fmt.Errorf("unsupported operation kind %q", op.Kind())
	}
}

// executeFileMove resolves From and To inside absRoot, ensures both
// stay within the root, creates the destination directory if needed,
// and os.Rename's the file.
func (e *FSExecutor) executeFileMove(absRoot string, m FileMove) error {
	fromAbs, err := joinWithinRoot(absRoot, m.From)
	if err != nil {
		return err
	}
	toAbs, err := joinWithinRoot(absRoot, m.To)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil {
		return fmt.Errorf("mkdir for to_path: %w", err)
	}
	if err := os.Rename(fromAbs, toAbs); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", m.From, m.To, err)
	}
	return nil
}

// executeImportRewrite walks every .go file under absRoot and rewrites
// import specs whose path == m.Old to m.New. Files without a matching
// import are left untouched. Each modified file is written atomically
// via a sibling tempfile + rename.
func (e *FSExecutor) executeImportRewrite(absRoot string, m ImportRewrite) error {
	return filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			// Skip vendor, testdata, .git, and dot-directories — the
			// loader skips them and we shouldn't be rewriting inside.
			if p != absRoot && (base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		return e.rewriteImportInFile(p, m.Old, m.New)
	})
}

// rewriteImportInFile parses path, mutates matching imports, and writes
// the result back if (and only if) at least one import was changed.
func (e *FSExecutor) rewriteImportInFile(path, oldPath, newPath string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		// Per PRD: parse errors flow as warnings during Extract; during
		// Apply they're per-op failures. Bubble up unchanged.
		return fmt.Errorf("parse %s: %w", path, err)
	}
	changed := false
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		unquoted, qerr := strconv.Unquote(imp.Path.Value)
		if qerr != nil {
			continue
		}
		if unquoted == oldPath {
			imp.Path.Value = strconv.Quote(newPath)
			// Also update the file-level imports declarations: the
			// AST keeps decls in sync because imp is the same pointer
			// the GenDecl's specs reference.
			changed = true
		}
	}
	if !changed {
		return nil
	}
	// Re-sort import decls so the rewrite produces gofmt-clean output.
	ast.SortImports(fset, file)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	return writeAtomic(path, buf.Bytes())
}

// joinWithinRoot resolves rel against root and asserts the result is
// inside root (no traversal escape via symlinks resolved by Abs +
// prefix check).
func joinWithinRoot(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q must be scenario-root-relative", rel)
	}
	joined := filepath.Join(root, rel)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", rel, err)
	}
	rootWithSep := root + string(filepath.Separator)
	if abs != root && !strings.HasPrefix(abs, rootWithSep) {
		return "", fmt.Errorf("path %q escapes scenario root", rel)
	}
	return abs, nil
}

// writeAtomic writes data to path via a sibling tempfile + rename so
// readers never see a half-written file.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".rewrite-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp -> %s: %w", path, err)
	}
	return nil
}

// Compile-time assertion.
var _ RewriteExecutor = (*FSExecutor)(nil)
