package feedback

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"swarm-manager/internal/proposals"
)

// fencedProposalBlockRE finds a fenced code block whose info string is
// case-insensitively "json" (with optional surrounding whitespace). Used
// as the first-pass extractor before falling back to looser strategies.
var fencedProposalBlockRE = regexp.MustCompile("(?si)```\\s*json\\b[^\\n]*\\n(.*?)```")

// genericFencedBlockRE finds any fenced block, including ones with no
// language tag or a non-json language. Used after the json-fenced pass
// fails so prose like ```\n{...}\n``` or ```yaml-like-but-actually-json
// still parses.
var genericFencedBlockRE = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")

// proposalSentinelRE matches `PROPOSAL:` (case-insensitive) followed by an
// optional fence then a JSON object. Lets agents use the explicit
// sentinel pattern documented in the skill prompt.
var proposalSentinelRE = regexp.MustCompile(`(?si)PROPOSAL\s*:[^\{]*?(\{.*\})`)

// extractProposal pulls a JSON proposal envelope out of an agent message
// using a lenient-then-strict strategy:
//  1. ```json fenced blocks (case-insensitive on the language tag)
//  2. any fenced block whose contents start with `{`
//  3. a `PROPOSAL:` sentinel followed by a JSON object
//  4. the first balanced `{...}` substring in the message
//
// All four strategies feed the same parser, so a single message can be
// noisy as long as one extraction succeeds. Returns nil + warnings when
// no extraction parses — the round still records the turn so the user
// can ask for a revision, which is the documented failure mode.
func extractProposal(body string) (*proposals.Proposal, string, []string) {
	if strings.TrimSpace(body) == "" {
		return nil, "", nil
	}
	var warnings []string

	tryParse := func(raw string) (*proposals.Proposal, string, bool) {
		raw = strings.TrimSpace(raw)
		if !strings.HasPrefix(raw, "{") {
			return nil, "", false
		}
		var p proposals.Proposal
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			warnings = append(warnings, fmt.Sprintf("parse proposal block: %s", err.Error()))
			return nil, "", false
		}
		return &p, raw, true
	}

	// Strategy 1: ```json fenced blocks.
	for _, m := range fencedProposalBlockRE.FindAllStringSubmatch(body, -1) {
		if p, raw, ok := tryParse(m[1]); ok {
			return p, raw, warnings
		}
	}

	// Strategy 2: any fenced block.
	for _, m := range genericFencedBlockRE.FindAllStringSubmatch(body, -1) {
		if p, raw, ok := tryParse(m[1]); ok {
			return p, raw, warnings
		}
	}

	// Strategy 3: PROPOSAL: sentinel.
	if m := proposalSentinelRE.FindStringSubmatch(body); len(m) == 2 {
		if balanced := extractFirstBalancedJSON(m[1]); balanced != "" {
			if p, raw, ok := tryParse(balanced); ok {
				return p, raw, warnings
			}
		}
	}

	// Strategy 4: any balanced JSON object in the body — last resort,
	// useful when the agent emits raw JSON with no markdown wrapping.
	if balanced := extractFirstBalancedJSON(body); balanced != "" {
		if p, raw, ok := tryParse(balanced); ok {
			return p, raw, warnings
		}
	}

	return nil, "", warnings
}

// extractFirstBalancedJSON returns the substring starting at the first '{'
// up to (and including) the matching closing '}'. Counts balanced braces
// honoring strings and escapes. Returns "" when no balanced object is
// found. Tolerates leading prose, attribute lists, etc.
func extractFirstBalancedJSON(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// deriveSlugFromText produces a slug from a free-form submission. Falls
// back to "round" when the text contains no word characters.
func deriveSlugFromText(text string) string {
	s := Sanitize(text)
	if s == "" {
		return "round"
	}
	// Trim to the first few dash-separated tokens for a readable folder name.
	tokens := strings.Split(s, "-")
	if len(tokens) > 5 {
		tokens = tokens[:5]
	}
	return strings.Join(tokens, "-")
}

// ComputeSlug derives the canonical slug a round will get, given a slug
// hint and submission text. Exposed so callers that need to place files
// on disk *before* StartRound runs (HTTP multipart handlers) can agree on
// the same folder name the service will pick.
func ComputeSlug(slugHint, text string) string {
	if s := Sanitize(slugHint); s != "" {
		return s
	}
	return deriveSlugFromText(text)
}
