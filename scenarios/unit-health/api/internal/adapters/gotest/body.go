package gotest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxTestBodyBytes = 4096

// BodyRequest identifies a source test body inside one already-resolved Go
// workspace. The caller owns scenario and workspace resolution.
type BodyRequest struct {
	Root      string
	Workspace string
	File      string
	TestID    string
	MaxBytes  uint32
}

// BodyResponse is a bounded, redacted source excerpt suitable for governed
// review. BodyBytes is the unredacted body size before the excerpt cap.
type BodyResponse struct {
	TestIdentity  string
	BodyExcerpt   string
	BodyBytes     uint32
	Redactions    uint32
	Refused       bool
	RefusalReason string
}

var (
	accessKeyPattern        = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	secretAssignmentPattern = regexp.MustCompile(`(?i)((?:api[_-]?key|access[_-]?token|client[_-]?secret|password|secret|token)\s*[:=]+\s*["']?)[A-Za-z0-9_./+=-]+`)
)

// ReadTestBody extracts one top-level Go test function and applies the same
// privacy boundary used by sampled review. It never returns the full file.
func ReadTestBody(req BodyRequest) (BodyResponse, error) {
	if strings.TrimSpace(req.Root) == "" || strings.TrimSpace(req.File) == "" || strings.TrimSpace(req.TestID) == "" {
		return BodyResponse{}, fmt.Errorf("root, file, and test id are required")
	}
	if privacyPath(req.File) {
		return BodyResponse{Refused: true, RefusalReason: "privacy_pattern"}, nil
	}
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return BodyResponse{}, err
	}
	file := filepath.Clean(filepath.Join(root, filepath.FromSlash(req.File)))
	if !withinRoot(root, file) {
		return BodyResponse{}, fmt.Errorf("file escapes workspace")
	}
	if filepath.Ext(file) != ".go" {
		return BodyResponse{}, fmt.Errorf("unsupported source profile: %s", filepath.Ext(file))
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return BodyResponse{}, err
	}
	set := token.NewFileSet()
	tree, err := parser.ParseFile(set, file, data, parser.ParseComments)
	if err != nil {
		return BodyResponse{}, fmt.Errorf("parse test source: %w", err)
	}
	topLevelID := strings.SplitN(req.TestID, "/", 2)[0]
	var body []byte
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name != nil && fn.Name.Name == topLevelID {
			start := set.Position(fn.Pos()).Offset
			end := set.Position(fn.End()).Offset
			if start < 0 || end < start || end > len(data) {
				return BodyResponse{}, fmt.Errorf("invalid source span for %s", req.TestID)
			}
			body = data[start:end]
			break
		}
	}
	if len(body) == 0 {
		return BodyResponse{}, fmt.Errorf("test %q not found", req.TestID)
	}
	limit := int(req.MaxBytes)
	if limit <= 0 || limit > maxTestBodyBytes {
		limit = maxTestBodyBytes
	}
	redacted, redactions := redact(body)
	if len(redacted) > limit {
		redacted = redacted[:limit]
	}
	identity := strings.Join([]string{req.Workspace, filepath.ToSlash(req.File), req.TestID}, ":")
	return BodyResponse{TestIdentity: identity, BodyExcerpt: string(redacted), BodyBytes: uint32(len(body)), Redactions: redactions}, nil
}

func privacyPath(path string) bool {
	for _, part := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		lower := strings.ToLower(part)
		if lower == ".env" || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
			return true
		}
	}
	return false
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func redact(body []byte) ([]byte, uint32) {
	redactions := uint32(0)
	redacted := secretAssignmentPattern.ReplaceAllFunc(body, func(match []byte) []byte {
		parts := secretAssignmentPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		redactions++
		return append(append([]byte(nil), parts[1]...), []byte("[redacted]")...)
	})
	redacted = accessKeyPattern.ReplaceAllFunc(redacted, func([]byte) []byte {
		redactions++
		return []byte("[redacted]")
	})
	return redacted, redactions
}
