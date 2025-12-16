package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func ParsePorcelainV2Status(output []byte) (*RepoStatus, error) {
	status := &RepoStatus{
		Scopes: map[string][]string{},
		Raw:    map[string]interface{}{},
	}

	records := bytes.Split(output, []byte{0})
	for i := 0; i < len(records); i++ {
		record := strings.TrimSpace(string(records[i]))
		if record == "" {
			continue
		}

		switch {
		case strings.HasPrefix(record, "#"):
			parseBranchHeader(status, record)
		case strings.HasPrefix(record, "1 "):
			if err := parseChangedRecord(status, record); err != nil {
				return nil, err
			}
		case strings.HasPrefix(record, "2 "):
			var orig string
			if i+1 < len(records) {
				orig = strings.TrimSpace(string(records[i+1]))
				i++
			}
			if err := parseRenamedRecord(status, record, orig); err != nil {
				return nil, err
			}
		case strings.HasPrefix(record, "u "):
			if err := parseUnmergedRecord(status, record); err != nil {
				return nil, err
			}
		case strings.HasPrefix(record, "? "):
			path := strings.TrimSpace(strings.TrimPrefix(record, "?"))
			status.Files.Untracked = append(status.Files.Untracked, unquoteGitPath(path))
		case strings.HasPrefix(record, "! "):
			path := strings.TrimSpace(strings.TrimPrefix(record, "!"))
			status.Files.Ignored = append(status.Files.Ignored, unquoteGitPath(path))
		default:
			// Keep for debugging; avoid hard failing on unrecognized future record types.
			status.Raw["unparsed"] = appendRaw(status.Raw["unparsed"], record)
		}
	}

	return status, nil
}

func parseBranchHeader(status *RepoStatus, record string) {
	record = strings.TrimSpace(strings.TrimPrefix(record, "#"))
	if record == "" {
		return
	}
	if strings.HasPrefix(record, "branch.head ") {
		status.Branch.Head = strings.TrimSpace(strings.TrimPrefix(record, "branch.head "))
		return
	}
	if strings.HasPrefix(record, "branch.upstream ") {
		status.Branch.Upstream = strings.TrimSpace(strings.TrimPrefix(record, "branch.upstream "))
		return
	}
	if strings.HasPrefix(record, "branch.oid ") {
		status.Branch.OID = strings.TrimSpace(strings.TrimPrefix(record, "branch.oid "))
		return
	}
	if strings.HasPrefix(record, "branch.ab ") {
		ab := strings.TrimSpace(strings.TrimPrefix(record, "branch.ab "))
		parseAheadBehind(&status.Branch, ab)
		return
	}
	status.Raw["branch"] = appendRaw(status.Raw["branch"], record)
}

func parseAheadBehind(branch *RepoBranchStatus, ab string) {
	fields := strings.Fields(ab)
	for _, token := range fields {
		if strings.HasPrefix(token, "+") {
			if n, err := strconv.Atoi(strings.TrimPrefix(token, "+")); err == nil {
				branch.Ahead = n
			}
		}
		if strings.HasPrefix(token, "-") {
			if n, err := strconv.Atoi(strings.TrimPrefix(token, "-")); err == nil {
				branch.Behind = n
			}
		}
	}
}

func parseChangedRecord(status *RepoStatus, record string) error {
	xy, ok := nthField(record, 1)
	if !ok || len(xy) < 2 {
		return fmt.Errorf("invalid status record: %q", record)
	}
	path, ok := restAfterFields(record, 7)
	if !ok {
		return fmt.Errorf("invalid status record (missing path): %q", record)
	}
	addPathByXY(status, xy[:2], unquoteGitPath(path))
	return nil
}

func parseRenamedRecord(status *RepoStatus, record string, _ string) error {
	xy, ok := nthField(record, 1)
	if !ok || len(xy) < 2 {
		return fmt.Errorf("invalid rename record: %q", record)
	}
	path, ok := restAfterFields(record, 8)
	if !ok {
		return fmt.Errorf("invalid rename record (missing path): %q", record)
	}
	addPathByXY(status, xy[:2], unquoteGitPath(path))
	return nil
}

func parseUnmergedRecord(status *RepoStatus, record string) error {
	xy, ok := nthField(record, 1)
	if !ok || len(xy) < 2 {
		return fmt.Errorf("invalid unmerged record: %q", record)
	}
	path, ok := restAfterFields(record, 9)
	if !ok {
		return fmt.Errorf("invalid unmerged record (missing path): %q", record)
	}
	unquotedPath := unquoteGitPath(path)
	addPathByXY(status, xy[:2], unquotedPath)
	status.Files.Conflicts = append(status.Files.Conflicts, unquotedPath)
	return nil
}

func addPathByXY(status *RepoStatus, xy string, path string) {
	if path == "" {
		return
	}

	index := xy[0]
	worktree := xy[1]

	if index != '.' {
		status.Files.Staged = append(status.Files.Staged, path)
	}
	if worktree != '.' {
		status.Files.Unstaged = append(status.Files.Unstaged, path)
	}
}

func unquoteGitPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "\"") {
		if unquoted, err := strconv.Unquote(path); err == nil {
			return unquoted
		}
	}
	return path
}

func nthField(s string, fieldIndex int) (string, bool) {
	start, end, ok := fieldSpan(s, fieldIndex)
	if !ok {
		return "", false
	}
	return s[start:end], true
}

func restAfterFields(s string, fields int) (string, bool) {
	_, end, ok := fieldSpan(s, fields)
	if !ok {
		return "", false
	}
	rest := strings.TrimLeft(s[end:], " \t")
	if rest == "" {
		return "", false
	}
	return rest, true
}

func fieldSpan(s string, fieldIndex int) (int, int, bool) {
	i := 0
	for current := 0; current <= fieldIndex; current++ {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			return 0, 0, false
		}
		start := i
		for i < len(s) && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		end := i
		if current == fieldIndex {
			return start, end, true
		}
	}
	return 0, 0, false
}

func appendRaw(existing any, value string) []string {
	if existing == nil {
		return []string{value}
	}
	if slice, ok := existing.([]string); ok {
		return append(slice, value)
	}
	return []string{value}
}
