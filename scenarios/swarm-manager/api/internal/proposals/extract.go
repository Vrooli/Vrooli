package proposals

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	fencedProposalBlockRE = regexp.MustCompile("(?si)```\\s*json\\b[^\\n]*\\n(.*?)```")
	genericFencedBlockRE  = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")
	proposalSentinelRE    = regexp.MustCompile(`(?si)PROPOSAL\s*:[^\{]*?(\{.*\})`)
)

// Extract finds a mutation proposal envelope in an agent response. It is
// deliberately tolerant of surrounding prose, but never invents a proposal:
// callers receive parse warnings and can ask the same session for a revision.
func Extract(body string) (*Proposal, string, []string) {
	if strings.TrimSpace(body) == "" {
		return nil, "", nil
	}
	var warnings []string
	tryParse := func(raw string) (*Proposal, string, bool) {
		raw = strings.TrimSpace(raw)
		if !strings.HasPrefix(raw, "{") {
			return nil, "", false
		}
		var proposal Proposal
		if err := json.Unmarshal([]byte(raw), &proposal); err != nil {
			warnings = append(warnings, fmt.Sprintf("parse proposal block: %s", err))
			return nil, "", false
		}
		return &proposal, raw, true
	}
	for _, match := range fencedProposalBlockRE.FindAllStringSubmatch(body, -1) {
		if proposal, raw, ok := tryParse(match[1]); ok {
			return proposal, raw, warnings
		}
	}
	for _, match := range genericFencedBlockRE.FindAllStringSubmatch(body, -1) {
		if proposal, raw, ok := tryParse(match[1]); ok {
			return proposal, raw, warnings
		}
	}
	if match := proposalSentinelRE.FindStringSubmatch(body); len(match) == 2 {
		if balanced := firstBalancedJSON(match[1]); balanced != "" {
			if proposal, raw, ok := tryParse(balanced); ok {
				return proposal, raw, warnings
			}
		}
	}
	if balanced := firstBalancedJSON(body); balanced != "" {
		if proposal, raw, ok := tryParse(balanced); ok {
			return proposal, raw, warnings
		}
	}
	return nil, "", warnings
}

func firstBalancedJSON(value string) string {
	start := strings.IndexByte(value, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inString, escaped := false, false
	for i := start; i < len(value); i++ {
		ch := value[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return value[start : i+1]
			}
		}
	}
	return ""
}
