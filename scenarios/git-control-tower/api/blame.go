package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"git-control-tower/internal/provenance"
)

const (
	defaultBlameMaxPaths = 32
	defaultBlameMaxLines = 2000
	defaultBlameMaxBytes = 4 << 20
)

// BlameRunner is the read-only Git seam used by the blame engine. Keeping it
// separate from GitRunner lets the long-lived runner interface evolve without
// invalidating every existing test double during the proto migration.
type BlameRunner interface {
	Blame(ctx context.Context, repoDir, revision, path string, maxBytes int) ([]byte, error)
}

type BlameRequest struct {
	RepoDir   string
	Revision  string
	Paths     []string
	StartLine int
	EndLine   int
	MaxPaths  int
	MaxLines  int
	MaxBytes  int
	Enrich    bool
	Evidence  map[string][]provenance.Evidence
}

type BlameResponse struct {
	Revision  string                   `json:"revision"`
	Files     []BlameFile              `json:"files"`
	Truncated bool                     `json:"truncated"`
	Warnings  []string                 `json:"warnings,omitempty"`
	Bundles   []ProvenanceChangeBundle `json:"change_bundles,omitempty"`
}

type ProvenanceChangeBundle struct {
	RunID          string               `json:"runId,omitempty"`
	SandboxID      string               `json:"sandboxId,omitempty"`
	Files          []string             `json:"files"`
	RunOutcome     string               `json:"runOutcome,omitempty"`
	ConversationID string               `json:"conversationId,omitempty"`
	CostUSD        float64              `json:"costUsd,omitempty"`
	WorkReferences []provenance.WorkRef `json:"workReferences,omitempty"`
	Gaps           []string             `json:"gaps,omitempty"`
}

type BlameFile struct {
	Path             string                `json:"path"`
	Status           string                `json:"status"`
	Lines            []BlameLine           `json:"lines,omitempty"`
	ContentDigest    string                `json:"content_digest,omitempty"`
	Reason           string                `json:"reason,omitempty"`
	Standing         provenance.Standing   `json:"standing,omitempty"`
	DowngradeReasons []string              `json:"downgrade_reasons,omitempty"`
	Evidence         []provenance.Evidence `json:"evidence,omitempty"`
}

type BlameLine struct {
	Number       int    `json:"line"`
	Content      string `json:"content"`
	Commit       string `json:"commit,omitempty"`
	Author       string `json:"author,omitempty"`
	AuthorTime   string `json:"author_time,omitempty"`
	Subject      string `json:"subject,omitempty"`
	OriginalLine int    `json:"original_line,omitempty"`
	OriginalPath string `json:"original_path,omitempty"`
}

// ReadBlame obtains deterministic, bounded native blame data. It only accepts
// repository-relative paths and delegates process execution to BlameRunner.
func ReadBlame(ctx context.Context, runner BlameRunner, req BlameRequest) (*BlameResponse, error) {
	if runner == nil {
		return nil, errors.New("blame runner is required")
	}
	root, err := filepath.Abs(strings.TrimSpace(req.RepoDir))
	if err != nil || root == "." || strings.TrimSpace(req.RepoDir) == "" {
		return nil, errors.New("repository directory is required")
	}
	maxPaths := req.MaxPaths
	if maxPaths <= 0 {
		maxPaths = defaultBlameMaxPaths
	}
	maxLines := req.MaxLines
	if maxLines <= 0 {
		maxLines = defaultBlameMaxLines
	}
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultBlameMaxBytes
	}
	if maxPaths > defaultBlameMaxPaths {
		maxPaths = defaultBlameMaxPaths
	}
	if maxLines > defaultBlameMaxLines {
		maxLines = defaultBlameMaxLines
	}
	if maxBytes > defaultBlameMaxBytes {
		maxBytes = defaultBlameMaxBytes
	}
	if req.StartLine < 0 || req.EndLine < 0 || (req.EndLine > 0 && req.StartLine > req.EndLine) {
		return nil, errors.New("invalid blame line range")
	}

	paths, truncated, err := expandBlamePaths(root, req.Paths, maxPaths)
	if err != nil {
		return nil, err
	}
	response := &BlameResponse{Revision: normalizeBlameRevision(req.Revision)}
	response.Truncated = truncated
	for _, path := range paths {
		file := BlameFile{Path: path, Status: "unavailable"}
		out, runErr := runner.Blame(ctx, root, req.Revision, path, maxBytes)
		if runErr != nil {
			file.Status, file.Reason = classifyBlameFailure(root, path, runErr)
			response.Files = append(response.Files, file)
			continue
		}
		parsed, lineTruncated, parseErr := parseBlamePorcelain(out, path, maxLines, req.StartLine, req.EndLine)
		if parseErr != nil {
			file.Status, file.Reason = "unavailable", "malformed native blame output: "+parseErr.Error()
			response.Files = append(response.Files, file)
			continue
		}
		file.Status = "native_commit"
		file.Lines = parsed
		response.Truncated = response.Truncated || lineTruncated
		if req.StartLine == 0 && req.EndLine == 0 && !lineTruncated {
			file.ContentDigest = digestBlameContent(parsed)
		}
		if len(parsed) == 0 {
			file.Status = "empty"
		}
		if req.Enrich {
			joined := JoinBlameEvidence(file, req.StartLine == 0 && req.EndLine == 0 && !lineTruncated, req.Evidence[path])
			file.Standing, file.DowngradeReasons, file.Evidence = joined.Standing, joined.DowngradeReasons, joined.Evidence
		}
		response.Files = append(response.Files, file)
	}
	sort.Slice(response.Files, func(i, j int) bool { return response.Files[i].Path < response.Files[j].Path })
	return response, nil
}

func expandBlamePaths(root string, requested []string, maxPaths int) ([]string, bool, error) {
	if len(requested) == 0 {
		return nil, false, errors.New("at least one path or glob is required")
	}
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(requested))
	truncated := false
	for _, raw := range requested {
		path := filepath.ToSlash(strings.TrimSpace(raw))
		if path == "" || filepath.IsAbs(path) || path == "." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
			return nil, false, fmt.Errorf("unsafe blame path %q", raw)
		}
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, false, fmt.Errorf("invalid blame glob %q: %w", raw, err)
		}
		if len(matches) == 0 && !hasGlob(path) {
			matches = []string{filepath.Join(root, filepath.FromSlash(path))}
		}
		sort.Strings(matches)
		for _, match := range matches {
			rel, err := filepath.Rel(root, match)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, false, fmt.Errorf("glob escaped repository root: %q", raw)
			}
			if resolved, resolveErr := filepath.EvalSymlinks(match); resolveErr == nil {
				resolvedRoot, _ := filepath.EvalSymlinks(root)
				resolvedRel, relErr := filepath.Rel(resolvedRoot, resolved)
				if relErr != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) {
					return nil, false, fmt.Errorf("blame path symlink escaped repository root: %q", raw)
				}
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seen[rel]; ok {
				continue
			}
			if len(paths) >= maxPaths {
				truncated = true
				continue
			}
			seen[rel] = struct{}{}
			paths = append(paths, rel)
		}
	}
	sort.Strings(paths)
	return paths, truncated, nil
}

func hasGlob(path string) bool { return strings.ContainsAny(path, "*?[") }

func normalizeBlameRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return "WORKTREE"
	}
	if strings.EqualFold(revision, "index") {
		return "INDEX"
	}
	return revision
}

func classifyBlameFailure(root, path string, err error) (string, string) {
	reason := err.Error()
	_, fileErr := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if strings.Contains(strings.ToLower(reason), "binary") {
		return "binary", reason
	}
	if fileErr == nil && (strings.Contains(strings.ToLower(reason), "no such path") || strings.Contains(strings.ToLower(reason), "does not exist") || strings.Contains(strings.ToLower(reason), "cannot stat")) {
		return "untracked", reason
	}
	if strings.Contains(strings.ToLower(reason), "no such path") || strings.Contains(strings.ToLower(reason), "does not exist") {
		return "deleted", reason
	}
	return "unavailable", reason
}

func parseBlamePorcelain(data []byte, path string, maxLines, start, end int) ([]BlameLine, bool, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	// Native porcelain lines can be large, but a bounded scanner prevents a
	// malformed fixture from allocating unbounded memory.
	scanner.Buffer(make([]byte, 4096), defaultBlameMaxBytes)
	var lines []BlameLine
	selectedTotal := 0
	var current *BlameLine
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			if current == nil {
				return nil, false, errors.New("content without header")
			}
			current.Content = strings.TrimPrefix(line, "\t")
			if current.Number >= start && (end == 0 || current.Number <= end) {
				selectedTotal++
				if len(lines) < maxLines {
					lines = append(lines, *current)
				}
			}
			current = nil
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 && len(fields[0]) >= 8 && isHex(fields[0]) {
			if current != nil {
				return nil, false, errors.New("header before content")
			}
			final, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, false, errors.New("invalid final line")
			}
			current = &BlameLine{Number: final, Commit: fields[0]}
			if len(fields) >= 4 {
				current.OriginalLine, _ = strconv.Atoi(fields[1])
			}
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "author "):
			current.Author = strings.TrimPrefix(line, "author ")
		case strings.HasPrefix(line, "author-time "):
			seconds, err := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64)
			if err == nil {
				current.AuthorTime = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
			}
		case strings.HasPrefix(line, "summary "):
			current.Subject = strings.TrimPrefix(line, "summary ")
		case strings.HasPrefix(line, "filename "):
			current.OriginalPath = strings.TrimPrefix(line, "filename ")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	if current != nil {
		return nil, false, errors.New("unterminated blame record")
	}
	return lines, selectedTotal > maxLines, nil
}

func isHex(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func digestBlameContent(lines []BlameLine) string {
	h := sha256.New()
	for _, line := range lines {
		h.Write([]byte(line.Content))
		h.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
