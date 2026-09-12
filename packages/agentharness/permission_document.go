package agentharness

// This file defines the portable, whole-document permission input shared by
// coding-agent resource CLIs. It deliberately describes intent only: each
// resource remains responsible for translating it to its native config.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const PermissionDocumentSchemaVersion = "v1"

type PermissionDocument struct {
	SchemaVersion string           `json:"schema_version"`
	Scope         string           `json:"scope,omitempty"`
	Rules         []PermissionRule `json:"rules"`
}

type PermissionRule struct {
	ID      string            `json:"id"`
	Action  string            `json:"action"`
	Matcher PermissionMatcher `json:"matcher"`
}

type PermissionMatcher struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
}

var permissionRuleID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// LoadPermissionDocument reads one strict declared desired-state document.
// The special path "-" consumes stdin, keeping the protocol useful to a
// process caller without leaking native configuration details.
func LoadPermissionDocument(path string, stdin io.Reader) (PermissionDocument, []byte, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		if stdin == nil {
			return PermissionDocument{}, nil, errors.New("permission document stdin is unavailable")
		}
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return PermissionDocument{}, nil, fmt.Errorf("read permission document %q: %w", path, err)
	}
	document, err := parsePermissionDocument(data, path)
	if err != nil {
		return PermissionDocument{}, nil, err
	}
	return document, data, nil
}

func parsePermissionDocument(data []byte, path string) (PermissionDocument, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var document PermissionDocument
	if err := decoder.Decode(&document); err != nil {
		return PermissionDocument{}, fmt.Errorf("parse permission document %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return PermissionDocument{}, fmt.Errorf("parse permission document %q: multiple JSON values", path)
		}
		return PermissionDocument{}, fmt.Errorf("parse permission document %q: %w", path, err)
	}
	if err := ValidatePermissionDocument(document); err != nil {
		return PermissionDocument{}, err
	}
	return document, nil
}

func ValidatePermissionDocument(document PermissionDocument) error {
	var problems []error
	if document.SchemaVersion != PermissionDocumentSchemaVersion {
		problems = append(problems, fmt.Errorf("schema_version must be %q", PermissionDocumentSchemaVersion))
	}
	if document.Scope != "" && document.Scope != "user" && document.Scope != "admin" {
		problems = append(problems, fmt.Errorf("scope must be user or admin when set"))
	}
	seen := make(map[string]struct{}, len(document.Rules))
	seenMatchers := make(map[string]struct{}, len(document.Rules))
	for index, rule := range document.Rules {
		prefix := fmt.Sprintf("rules[%d]", index)
		if !permissionRuleID.MatchString(rule.ID) {
			problems = append(problems, fmt.Errorf("%s.id must match %s", prefix, permissionRuleID.String()))
		}
		if _, ok := seen[rule.ID]; ok {
			problems = append(problems, fmt.Errorf("%s.id %q is duplicated", prefix, rule.ID))
		}
		seen[rule.ID] = struct{}{}
		if rule.Action != "allow" && rule.Action != "ask" && rule.Action != "deny" {
			problems = append(problems, fmt.Errorf("%s.action must be allow, ask, or deny", prefix))
		}
		if rule.Matcher.Kind != "bash" {
			problems = append(problems, fmt.Errorf("%s.matcher.kind %q is unsupported; only bash is portable", prefix, rule.Matcher.Kind))
		}
		if strings.TrimSpace(rule.Matcher.Pattern) == "" {
			problems = append(problems, fmt.Errorf("%s.matcher.pattern is required", prefix))
		}
		matcherKey := rule.Matcher.Kind + "\x00" + rule.Matcher.Pattern
		if _, ok := seenMatchers[matcherKey]; ok {
			problems = append(problems, fmt.Errorf("%s.matcher duplicates another rule; a matcher may have only one action", prefix))
		}
		seenMatchers[matcherKey] = struct{}{}
	}
	return errors.Join(problems...)
}

// PermissionPatterns returns portable Bash command patterns. Resource adapters
// translate these into their own native syntax (for example, Claude Code and
// Grok wrap them in Bash(...), while Codex and OpenCode persist raw patterns).
// The document validator ensures every matcher is representable before any
// resource plans or writes.
func PermissionPatterns(document PermissionDocument) (allow, ask, deny []string) {
	for _, rule := range document.Rules {
		pattern := rule.Matcher.Pattern
		switch rule.Action {
		case "allow":
			allow = append(allow, pattern)
		case "ask":
			ask = append(ask, pattern)
		case "deny":
			deny = append(deny, pattern)
		}
	}
	sort.Strings(allow)
	sort.Strings(ask)
	sort.Strings(deny)
	return allow, ask, deny
}

func PermissionDocumentDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
