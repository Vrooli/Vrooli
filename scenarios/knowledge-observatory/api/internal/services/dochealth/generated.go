package dochealth

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const generatedProgramTablePrefix = "<!-- BEGIN GENERATED PROGRAM TABLE"
const generatedHostToolMatrixPrefix = "<!-- BEGIN GENERATED HOST TOOL MATRIX"
const generatedNodeIndexPrefix = "<!-- BEGIN GENERATED NODE INDEX"
const generatedActionTypeTablePrefix = "<!-- BEGIN GENERATED ACTION TYPES"
const generatedBlockEnd = "<!-- END GENERATED PROGRAM TABLE -->"
const maxGeneratedSourceFiles = 128
const maxGeneratedSourceFileBytes = 1 << 20

var generatedBlockMetadata = regexp.MustCompile(`^<!-- BEGIN GENERATED ([A-Z0-9 _-]+) source=([A-Za-z0-9._/*-]+) source_sha256=([a-f0-9]{64}) block_sha256=([a-f0-9]{64}) -->$`)

// generatedBlockFindings fails marked generated content when either its
// authoritative inputs or its rendered block drift from the recorded hashes.
func generatedBlockFindings(files []string, root string) []Finding {
	var findings []Finding
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return []Finding{{Code: "stale_generated_block", Severity: SeverityFailure, Message: "cannot resolve documentation root: " + err.Error()}}
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fileAbs, err := filepath.Abs(file)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(rootAbs, fileAbs)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		lines := strings.Split(string(content), "\n")
		inFence := false
		for index := 0; index < len(lines); index++ {
			lineText := strings.TrimSuffix(lines[index], "\r")
			trimmed := strings.TrimSpace(lineText)
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if inFence || !(strings.HasPrefix(trimmed, generatedProgramTablePrefix) || strings.HasPrefix(trimmed, generatedHostToolMatrixPrefix) || strings.HasPrefix(trimmed, generatedNodeIndexPrefix) || strings.HasPrefix(trimmed, generatedActionTypeTablePrefix)) {
				continue
			}
			line := index + 1
			finding := func(message string) {
				findings = append(findings, Finding{Code: "stale_generated_block", Severity: SeverityFailure, Message: message, Path: rel, Line: line})
			}
			meta := generatedBlockMetadata.FindStringSubmatch(trimmed)
			if len(meta) != 5 {
				finding("Generated block marker must pin a safe source directory, source SHA-256, and block SHA-256.")
				continue
			}
			actualSourceHash, err := generatedSourceDigest(rootAbs, meta[2])
			if err != nil {
				finding("Generated block source cannot be verified: " + err.Error())
			} else if actualSourceHash != meta[3] {
				finding("Generated block source changed; regenerate the consumer from its authoritative inputs.")
			}
			endMarker := "<!-- END GENERATED " + strings.TrimSpace(meta[1]) + " -->"
			end := -1
			for candidate := index + 1; candidate < len(lines); candidate++ {
				if strings.TrimSpace(strings.TrimSuffix(lines[candidate], "\r")) == endMarker {
					end = candidate
					break
				}
			}
			if end < 0 {
				finding("Generated block has no matching end marker.")
				break
			}
			body := strings.Join(lines[index+1:end], "\n")
			actualBlockHash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
			if actualBlockHash != meta[4] {
				finding("Generated block content changed; regenerate it from its authoritative inputs.")
			}
			index = end
		}
	}
	return findings
}

func generatedSourceDigest(root, source string) (string, error) {
	if filepath.IsAbs(source) || strings.Contains(source, "\\") {
		return "", fmt.Errorf("source path must be relative with forward slashes")
	}
	sourceDir := source
	pattern := "*"
	exactFile := false
	if strings.ContainsAny(source, "*?[") {
		separator := strings.LastIndex(source, "/")
		if separator < 0 {
			return "", fmt.Errorf("source patterns must include a directory")
		}
		sourceDir = source[:separator]
		pattern = source[separator+1:]
	} else if info, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(source))); statErr == nil && !info.IsDir() {
		separator := strings.LastIndex(source, "/")
		if separator < 0 {
			return "", fmt.Errorf("exact source files must include a directory")
		}
		sourceDir = source[:separator]
		pattern = source[separator+1:]
		exactFile = true
	}
	if pattern == "" || strings.Contains(pattern, "/") {
		return "", fmt.Errorf("source pattern must match direct filenames")
	}
	if _, err := filepath.Match(pattern, ""); err != nil {
		return "", fmt.Errorf("invalid source pattern: %w", err)
	}
	for _, segment := range strings.Split(sourceDir, "/") {
		if segment == ".." || segment == "." || segment == "" {
			return "", fmt.Errorf("source path must not contain dot or empty components")
		}
	}
	clean := filepath.Clean(filepath.FromSlash(sourceDir))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source path escapes the scenario root")
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve scenario root: %w", err)
	}
	dir := filepath.Join(rootReal, clean)
	rel, err := filepath.Rel(rootReal, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source path escapes the scenario root")
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", fmt.Errorf("read source directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("source path must be a regular directory")
	}
	dirReal, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory: %w", err)
	}
	resolvedRel, err := filepath.Rel(rootReal, dirReal)
	if err != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source directory resolves outside the scenario root")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("list source directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		matched, matchErr := filepath.Match(pattern, entry.Name())
		if matchErr != nil {
			return "", fmt.Errorf("match source entry: %w", matchErr)
		}
		if !matched {
			continue
		}
		if entry.IsDir() {
			continue
		}
		if !entry.Type().IsRegular() {
			return "", fmt.Errorf("source entry %q is not a regular file", entry.Name())
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("source directory contains no regular files")
	}
	if exactFile && len(names) != 1 {
		return "", fmt.Errorf("exact source path did not resolve to one regular file")
	}
	if len(names) > maxGeneratedSourceFiles {
		return "", fmt.Errorf("source directory exceeds the %d-file bound", maxGeneratedSourceFiles)
	}
	digest := sha256.New()
	for _, name := range names {
		file, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return "", fmt.Errorf("open source %q: %w", name, err)
		}
		_, _ = io.WriteString(digest, name)
		_, _ = digest.Write([]byte{0})
		data, readErr := io.ReadAll(io.LimitReader(file, maxGeneratedSourceFileBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return "", fmt.Errorf("read source %q: %w", name, readErr)
		}
		if len(data) > maxGeneratedSourceFileBytes {
			return "", fmt.Errorf("source %q exceeds the %d-byte bound", name, maxGeneratedSourceFileBytes)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close source %q: %w", name, closeErr)
		}
		_, _ = digest.Write(data)
		_, _ = digest.Write([]byte{0})
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), nil
}
