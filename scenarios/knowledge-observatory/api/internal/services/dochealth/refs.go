package dochealth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/markedrefs"
	"github.com/vrooli/api-core/relationshiprefs"
)

// DOC: docs/internal/SEAMS.md#dochealth
// Bidirectional reference validation. Mirrors the test-genie logic but reads
// the filesystem directly under scenarioDir; the old workspace.Mapping
// indirection (used for BAS overlays) is intentionally absent here.

type codeRefTarget struct {
	File     string
	Ref      string
	FilePath string
	Line     int
}

type docRefTarget struct {
	File    string
	Ref     string
	DocPath string
	Line    int
}

type markedRefStatus int

const (
	markedRefOK markedRefStatus = iota
	markedRefSkipped
	markedRefUnknown
	markedRefBroken
	markedRefPartial
)

type markedRefTarget struct {
	File string
	Ref  markedrefs.Reference
}

type refSummary struct {
	CodeRefsFound     int
	CodeRefsBroken    int
	DocRefsFound      int
	DocRefsBroken     int
	CodeFilesScanned  int
	MarkedRefsFound   int
	MarkedRefsBroken  int
	MarkedRefsSkipped int
	MarkedRefsUnknown int
}

func validateBidirectionalRefs(ctx context.Context, scenarioDir string, markdownFiles []string, cfg effective, commandValidator CommandReferenceValidator) ([]Finding, refSummary) {
	var out []Finding
	var summary refSummary

	// [CODE: ...] references in markdown files
	for _, file := range markdownFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		refs := extractCodeRefs(file, string(content))
		summary.CodeRefsFound += len(refs)
		for _, ref := range refs {
			if err := validateCodeRef(scenarioDir, ref); err != nil {
				summary.CodeRefsBroken++
				out = append(out, Finding{
					Code:     "broken_code_ref",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d broken code reference [CODE: %s]: %v", ref.File, ref.Line, ref.Ref, err),
					Path:     ref.File,
					Line:     ref.Line,
					Target:   ref.FilePath,
				})
			}
		}
	}

	// Marked path/doc references in markdown files
	for _, file := range markdownFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		refs := extractMarkedRefs(file, string(content))
		summary.MarkedRefsFound += len(refs)
		for _, ref := range refs {
			status, err := validateMarkedRef(ctx, scenarioDir, ref, commandValidator)
			switch status {
			case markedRefSkipped:
				summary.MarkedRefsSkipped++
			case markedRefUnknown:
				summary.MarkedRefsUnknown++
				message := fmt.Sprintf("%s:%d unknown marked reference marker %q in %s", ref.File, ref.Ref.Line, ref.Ref.Marker, ref.Ref.Raw)
				code := "unknown_marked_ref"
				if err != nil {
					code = "unknown_cli_ref"
					message = fmt.Sprintf("%s:%d CLI reference %s could not be fully validated: %v", ref.File, ref.Ref.Line, ref.Ref.Raw, err)
				}
				out = append(out, Finding{
					Code:     code,
					Severity: SeverityWarning,
					Message:  message,
					Path:     ref.File,
					Line:     ref.Ref.Line,
				})
			case markedRefBroken:
				summary.MarkedRefsBroken++
				out = append(out, Finding{
					Code:     "broken_marked_ref",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d broken marked reference %s: %v", ref.File, ref.Ref.Line, ref.Ref.Raw, err),
					Path:     ref.File,
					Line:     ref.Ref.Line,
				})
			case markedRefPartial:
				out = append(out, Finding{
					Code:     "partial_cli_ref",
					Severity: SeverityInfo,
					Message:  fmt.Sprintf("%s:%d partially validated CLI reference %s: %v", ref.File, ref.Ref.Line, ref.Ref.Raw, err),
					Path:     ref.File,
					Line:     ref.Ref.Line,
					Target:   ref.Ref.Value,
				})
			}
		}
	}

	// // DOC: comments in code files
	docRefs, filesScanned, _ := scanCodeFilesForDocRefs(ctx, scenarioDir, cfg)
	summary.CodeFilesScanned = filesScanned
	summary.DocRefsFound = len(docRefs)
	for _, ref := range docRefs {
		if err := validateDocRef(scenarioDir, ref); err != nil {
			summary.DocRefsBroken++
			out = append(out, Finding{
				Code:     "broken_doc_ref",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s:%d broken doc reference // DOC: %s: %v", ref.File, ref.Line, ref.Ref, err),
				Path:     ref.File,
				Line:     ref.Line,
				Target:   ref.DocPath,
			})
		}
	}

	return out, summary
}

func extractCodeRefs(file, content string) []codeRefTarget {
	var refs []codeRefTarget
	for _, ref := range relationshiprefs.ExtractMarkdownRefs(content) {
		if ref.Kind != relationshiprefs.KindCode {
			continue
		}
		rawRef := strings.TrimSpace(ref.Value)
		refs = append(refs, codeRefTarget{
			File:     file,
			Ref:      rawRef,
			FilePath: relationshiprefs.TargetPath(rawRef),
			Line:     ref.Line,
		})
	}
	return refs
}

func extractDocRefsFromFile(path string) ([]docRefTarget, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var refs []docRefTarget
	for _, ref := range relationshiprefs.ExtractDocCommentRefs(string(content)) {
		rawRef := strings.TrimSpace(ref.Value)
		refs = append(refs, docRefTarget{
			File:    path,
			Ref:     rawRef,
			DocPath: relationshiprefs.TargetPath(rawRef),
			Line:    ref.Line,
		})
	}
	return refs, nil
}

func extractMarkedRefs(file, content string) []markedRefTarget {
	var refs []markedRefTarget
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMarker := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceMatch := codeFencePattern.FindStringSubmatch(trimmed); fenceMatch != nil {
			marker := fenceMatch[1]
			if inFence && marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			} else if !inFence {
				inFence = true
				fenceMarker = marker
			}
			continue
		}
		if inFence {
			continue
		}
		for _, ref := range markedrefs.ParseInlineCode(line, i+1) {
			if markedrefs.UnknownMarker(ref) && !looksLikeIntentionalUnknownMarkedRef(ref) {
				continue
			}
			refs = append(refs, markedRefTarget{File: file, Ref: ref})
		}
	}
	return refs
}

func looksLikeIntentionalUnknownMarkedRef(ref markedrefs.Reference) bool {
	if len(ref.Qualifiers) > 0 {
		return true
	}
	value := strings.TrimSpace(ref.Value)
	if value == "" || strings.ContainsAny(value, " \t\r\n\"'<>") {
		return false
	}
	if strings.HasPrefix(value, "//") {
		return false
	}
	return strings.Contains(value, "/") || strings.Contains(value, "*")
}

func resolveScenarioTarget(scenarioDir, target string) (string, bool) {
	target = strings.TrimPrefix(target, "./")
	resolved := filepath.Clean(filepath.Join(scenarioDir, target))
	if _, err := os.Stat(resolved); err != nil {
		return resolved, false
	}
	return resolved, true
}

func validateMarkedRef(ctx context.Context, scenarioDir string, ref markedRefTarget, commandValidator CommandReferenceValidator) (markedRefStatus, error) {
	if markedrefs.UnknownMarker(ref.Ref) {
		return markedRefUnknown, nil
	}
	if ref.Ref.Marker == markedrefs.MarkerCLI {
		return validateCLIRef(ctx, ref, commandValidator)
	}
	if ref.Ref.Marker != markedrefs.MarkerPath && ref.Ref.Marker != markedrefs.MarkerDoc {
		return markedRefSkipped, nil
	}
	if !markedrefs.RequiresExistence(ref.Ref) {
		return markedRefSkipped, nil
	}
	targetValue := relationshiprefs.TargetPath(ref.Ref.Value)
	if targetValue == "" {
		return markedRefBroken, fmt.Errorf("empty reference target")
	}
	target, ok := resolveScenarioTarget(scenarioDir, targetValue)
	if !ok {
		return markedRefBroken, fmt.Errorf("target not found: %s", targetValue)
	}
	info, err := os.Stat(target)
	if err != nil {
		return markedRefBroken, fmt.Errorf("target not found: %s", targetValue)
	}
	if ref.Ref.Marker == markedrefs.MarkerDoc {
		if info.IsDir() {
			return markedRefBroken, fmt.Errorf("doc reference points to directory, not file: %s", targetValue)
		}
		ext := strings.ToLower(filepath.Ext(targetValue))
		if ext != ".md" && ext != ".mdx" {
			return markedRefBroken, fmt.Errorf("doc reference must point to .md or .mdx file: %s", targetValue)
		}
		return markedRefOK, nil
	}
	return markedRefOK, nil
}

func validateCLIRef(ctx context.Context, ref markedRefTarget, commandValidator CommandReferenceValidator) (markedRefStatus, error) {
	if !markedrefs.RequiresExistence(ref.Ref) {
		return markedRefSkipped, nil
	}
	if strings.TrimSpace(ref.Ref.Value) == "" {
		return markedRefBroken, fmt.Errorf("empty CLI reference target")
	}
	if commandValidator == nil {
		return markedRefUnknown, fmt.Errorf("CLI Health command validator is not configured")
	}
	result, err := commandValidator.ValidateCommandReference(ctx, CommandReferenceRequest{
		CommandText: ref.Ref.Value,
		Qualifiers:  append([]string(nil), ref.Ref.Qualifiers...),
	})
	if err != nil {
		return markedRefUnknown, err
	}
	switch result.Verdict {
	case "valid", "skipped":
		return markedRefOK, nil
	case "partial":
		return markedRefPartial, formatCommandReferenceResult(result)
	case "invalid", "unsupported":
		return markedRefBroken, formatCommandReferenceResult(result)
	default:
		return markedRefUnknown, formatCommandReferenceResult(result)
	}
}

func formatCommandReferenceResult(result CommandReferenceResult) error {
	var parts []string
	if result.Verdict != "" {
		if result.ValidationLevel != "" {
			parts = append(parts, fmt.Sprintf("verdict=%s level=%s", result.Verdict, result.ValidationLevel))
		} else {
			parts = append(parts, "verdict="+result.Verdict)
		}
	}
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion.Command != "" {
			parts = append(parts, "suggestion: "+suggestion.Command)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		return fmt.Errorf("CLI Health returned no detail")
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

func validateCodeRef(scenarioDir string, ref codeRefTarget) error {
	target, ok := resolveScenarioTarget(scenarioDir, ref.FilePath)
	if !ok {
		return fmt.Errorf("file not found: %s", ref.FilePath)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("file not found: %s", ref.FilePath)
	}
	if info.IsDir() {
		return fmt.Errorf("reference points to directory, not file: %s", ref.FilePath)
	}
	return nil
}

func validateDocRef(scenarioDir string, ref docRefTarget) error {
	target, ok := resolveScenarioTarget(scenarioDir, ref.DocPath)
	if !ok {
		return fmt.Errorf("doc not found: %s", ref.DocPath)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("doc not found: %s", ref.DocPath)
	}
	if info.IsDir() {
		return fmt.Errorf("reference points to directory, not file: %s", ref.DocPath)
	}
	return nil
}

func scanCodeFilesForDocRefs(ctx context.Context, scenarioDir string, cfg effective) ([]docRefTarget, int, error) {
	var refs []docRefTarget
	var filesScanned int

	extensions := make(map[string]struct{})
	for _, ext := range cfg.codeExtensions {
		extensions[ext] = struct{}{}
	}
	customSkipDirs := make(map[string]struct{})
	for _, dir := range cfg.referencesSkip {
		customSkipDirs[dir] = struct{}{}
	}

	err := filepath.WalkDir(scenarioDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if shouldExcludePath(scenarioDir, p, d.IsDir(), cfg) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if _, skip := defaultSkipDirs[name]; skip {
				return filepath.SkipDir
			}
			if _, skip := customSkipDirs[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if _, ok := extensions[ext]; !ok {
			return nil
		}
		filesScanned++
		fileRefs, err := extractDocRefsFromFile(p)
		if err != nil {
			return nil
		}
		refs = append(refs, fileRefs...)
		return nil
	})
	return refs, filesScanned, err
}
