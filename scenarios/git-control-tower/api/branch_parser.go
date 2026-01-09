package main

import (
	"fmt"
	"strings"
	"time"
)

const branchRefFormat = "%(refname)|%(refname:short)|%(upstream:short)|%(objectname:short)|%(committerdate:iso8601)"

// ParsedBranchRef represents a parsed for-each-ref entry.
type ParsedBranchRef struct {
	RefName      string
	ShortName    string
	Upstream     string
	OID          string
	LastCommitAt time.Time
	IsRemote     bool
}

// ParseBranchRefs parses the output of git for-each-ref with branchRefFormat.
func ParseBranchRefs(out []byte) ([]ParsedBranchRef, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return []ParsedBranchRef{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	refs := make([]ParsedBranchRef, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid branch ref line: %s", line)
		}
		refName := strings.TrimSpace(parts[0])
		shortName := strings.TrimSpace(parts[1])
		upstream := strings.TrimSpace(parts[2])
		oid := strings.TrimSpace(parts[3])
		dateStr := strings.TrimSpace(parts[4])

		isRemote := strings.HasPrefix(refName, "refs/remotes/")
		if isRemote && strings.HasSuffix(shortName, "/HEAD") {
			continue
		}

		lastCommitAt := parseGitISO8601(dateStr)

		refs = append(refs, ParsedBranchRef{
			RefName:      refName,
			ShortName:    shortName,
			Upstream:     upstream,
			OID:          oid,
			LastCommitAt: lastCommitAt,
			IsRemote:     isRemote,
		})
	}
	return refs, nil
}

func parseGitISO8601(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05 -0700", value); err == nil {
		return parsed
	}
	return time.Time{}
}
