package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Indexer walks a configured filesystem root for *.tsx files,
// parses the canonical `@libraryId` header comment block, and upserts
// the result into the Repository. A final DeleteMissing call removes
// rows whose source files no longer exist, so deleted files leave the
// registry without manual intervention.
//
// The header contract (req CR-002, captured in PRD.md):
//
//	/**
//	 * @libraryId   react-component-library:Button
//	 * @displayName Button
//	 * @description Primary call-to-action button.
//	 * @version     1.0.0
//	 * @tags        ["form", "interactive"]
//	 * @deps        {"react": "^18"}
//	 * @warning     DO NOT REMOVE THIS HEADER...
//	 */
//
// Only `@libraryId` is required. Files without the header are ignored
// (they're not library components). Files with a malformed header
// return ErrInvalidHeader so the operator can fix and re-index.
type Indexer struct {
	repo Repository
	root string
	fs   fs.FS // injected for tests; nil means use os.DirFS(root)
}

// NewIndexer constructs an Indexer rooted at root. The root is the
// absolute path on disk; consumers resolve it via api-core/storage
// before calling. fsys may be nil — production passes nil and the
// indexer wraps os.DirFS(root); tests pass an in-memory fs.FS so they
// don't touch disk.
func NewIndexer(repo Repository, root string, fsys fs.FS) *Indexer {
	if fsys == nil && root != "" {
		fsys = os.DirFS(root)
	}
	return &Indexer{repo: repo, root: root, fs: fsys}
}

// IndexResult summarizes one Run.
type IndexResult struct {
	Scanned   int
	Indexed   int
	Skipped   int
	Deleted   int
	Errors    []error
	LibraryIDs []string // upserted IDs in walk order — useful for tests
}

// Run walks the root, upserts every file with a valid header, and
// returns a result. Files with malformed headers are recorded in
// Errors but do not stop the walk — a single broken file should not
// hide an otherwise healthy run.
func (idx *Indexer) Run(ctx context.Context) (IndexResult, error) {
	var result IndexResult
	if idx.fs == nil {
		return result, fmt.Errorf("indexer has no filesystem configured")
	}

	walkErr := fs.WalkDir(idx.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		result.Scanned++
		raw, readErr := fs.ReadFile(idx.fs, path)
		if readErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("read %s: %w", path, readErr))
			return nil
		}
		header, ok := extractHeaderBlock(string(raw))
		if !ok {
			result.Skipped++
			return nil
		}
		fields, perr := parseHeader(path, header)
		if perr != nil {
			result.Errors = append(result.Errors, perr)
			return nil
		}
		in, perr := buildUpsertInput(path, fields)
		if perr != nil {
			result.Errors = append(result.Errors, perr)
			return nil
		}
		if _, err := idx.repo.Upsert(ctx, in); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("upsert %s: %w", path, err))
			return nil
		}
		result.Indexed++
		result.LibraryIDs = append(result.LibraryIDs, in.LibraryID)
		return nil
	})
	if walkErr != nil {
		return result, fmt.Errorf("walk %s: %w", idx.root, walkErr)
	}

	deleted, derr := idx.repo.DeleteMissing(ctx, result.LibraryIDs)
	if derr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("delete missing: %w", derr))
	}
	result.Deleted = deleted

	return result, nil
}

// headerBlockRe captures the first /** … */ comment block in a file.
// (?s) flag makes . match newlines.
var headerBlockRe = regexp.MustCompile(`(?s)/\*\*(.*?)\*/`)

// headerFieldRe captures @field-name value pairs on each header line.
// The leading `*` and surrounding whitespace are stripped by the
// caller before this matches.
var headerFieldRe = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_-]*)\s+(.*)$`)

// extractHeaderBlock returns the inner text of the first JSDoc-style
// header block, or ok=false if no such block exists. Only blocks that
// contain `@libraryId` are treated as library headers; otherwise the
// indexer would claim every JSDoc-commented file.
func extractHeaderBlock(src string) (string, bool) {
	matches := headerBlockRe.FindStringSubmatch(src)
	if len(matches) < 2 {
		return "", false
	}
	body := matches[1]
	if !strings.Contains(body, "@libraryId") {
		return "", false
	}
	return body, true
}

// parseHeader extracts field/value pairs from a header block. Multi-
// line values are folded onto the @field line until the next @field
// or end-of-block (so JSDoc continuation lines are tolerated).
func parseHeader(path, body string) (map[string]string, error) {
	out := map[string]string{}
	var currentField string
	var currentValue strings.Builder

	flush := func() {
		if currentField == "" {
			return
		}
		out[currentField] = strings.TrimSpace(currentValue.String())
		currentField = ""
		currentValue.Reset()
	}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := headerFieldRe.FindStringSubmatch(line); m != nil {
			flush()
			currentField = m[1]
			currentValue.WriteString(m[2])
			continue
		}
		if currentField != "" {
			currentValue.WriteByte(' ')
			currentValue.WriteString(line)
		}
	}
	flush()

	if _, ok := out["libraryId"]; !ok {
		return nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	return out, nil
}

// buildUpsertInput projects parsed header fields into an UpsertInput.
// Unknown header fields are preserved in Headers for the registry
// row so future filters can query without re-parsing files.
func buildUpsertInput(path string, fields map[string]string) (UpsertInput, error) {
	libraryID := strings.TrimSpace(fields["libraryId"])
	if libraryID == "" {
		return UpsertInput{}, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	in := UpsertInput{
		LibraryID:   libraryID,
		DisplayName: fields["displayName"],
		Description: fields["description"],
		SourcePath:  filepath.ToSlash(path),
		Version:     fields["version"],
		Headers:     fields,
	}
	if raw, ok := fields["tags"]; ok && strings.TrimSpace(raw) != "" {
		tags, err := parseTags(raw)
		if err != nil {
			return UpsertInput{}, ErrInvalidHeader{SourcePath: path, Field: "tags", Reason: err.Error()}
		}
		in.Tags = tags
	}
	return in, nil
}

// parseTags accepts either a JSON array (`["a","b"]`) or a comma-
// separated bare list (`a, b`). Tags must be non-empty after trim and
// contain no commas (the sqlite repo uses comma as the in-row
// separator).
func parseTags(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil, fmt.Errorf("invalid JSON tag array: %w", err)
		}
		return validateTags(arr)
	}
	parts := strings.Split(raw, ",")
	return validateTags(parts)
}

func validateTags(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if strings.Contains(t, ",") {
			return nil, fmt.Errorf("tag %q contains comma", t)
		}
		out = append(out, t)
	}
	return out, nil
}
