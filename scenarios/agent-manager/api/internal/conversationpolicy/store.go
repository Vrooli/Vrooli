package conversationpolicy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileEnvVar overrides the on-disk policy file location. When unset the
// resolver falls back to the scenario's .vrooli/conversation-policy.json, the
// same convention search.json and the other scenario config files follow.
const FileEnvVar = "AGENT_MANAGER_CONVERSATION_POLICY_FILE"

// DefaultFileName is the policy file basename under .vrooli/.
const DefaultFileName = "conversation-policy.json"

// Document is the on-disk shape. Keeping rules under a top-level key leaves room
// for future policy metadata without breaking the file format.
type Document struct {
	Rules []Rule `json:"rules"`
}

// ResolveFilePath returns the policy file path. repoRoot is the Vrooli
// repository root; when the env override is set it wins. The path is returned
// even if the file does not exist yet (set writes it, load tolerates absence).
func ResolveFilePath(repoRoot string) string {
	if override := strings.TrimSpace(os.Getenv(FileEnvVar)); override != "" {
		return override
	}
	return filepath.Join(repoRoot, "scenarios", "agent-manager", ".vrooli", DefaultFileName)
}

// LoadDocument reads and validates the policy file. A missing file is not an
// error: it yields an empty document so an unconfigured host simply has no
// policy. Malformed or structurally invalid content IS an error so a typo can
// never silently drop a privacy rule.
func LoadDocument(path string) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, nil
		}
		return Document{}, fmt.Errorf("read conversation policy %s: %w", path, err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return Document{}, nil
	}
	var doc Document
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("parse conversation policy %s: %w", path, err)
	}
	for i, rule := range doc.Rules {
		if err := rule.Validate(); err != nil {
			return Document{}, fmt.Errorf("conversation policy %s rule %d (%s): %w", path, i, ruleLabel(rule, i), err)
		}
	}
	return doc, nil
}

// SaveDocument validates then atomically writes the policy file, creating the
// parent directory if needed. Validation happens before the write so a rejected
// rule never half-replaces the live policy.
func SaveDocument(path string, doc Document) error {
	for i, rule := range doc.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("conversation policy rule %d (%s): %w", i, ruleLabel(rule, i), err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create conversation policy directory: %w", err)
	}
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conversation policy: %w", err)
	}
	payload = append(payload, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write conversation policy: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install conversation policy: %w", err)
	}
	return nil
}

// MutateDocument performs a read-modify-write of the policy file under an
// exclusive advisory file lock, closing the add/remove TOCTOU window: two
// concurrent `policy add` invocations would otherwise each LoadDocument the
// same base, mutate independently, and the second SaveDocument's atomic rename
// would silently drop the first writer's rule. The lock serializes the whole
// load→mutate→save so the second invocation observes the first's write.
//
// The mutate callback receives the current document and returns the document
// to persist. A nil error from mutate triggers the save; any error aborts
// without writing. The lock is a sidecar file (path + ".lock") so it never
// perturbs the atomic rename of the policy file itself.
func MutateDocument(path string, mutate func(Document) (Document, error)) error {
	unlock, err := lockPolicyFile(path)
	if err != nil {
		return err
	}
	defer unlock()
	doc, err := LoadDocument(path)
	if err != nil {
		return err
	}
	next, err := mutate(doc)
	if err != nil {
		return err
	}
	return SaveDocument(path, next)
}

// LoadPolicy resolves, reads, validates, and compiles the policy file in one
// call. A missing file yields a disabled policy with no error.
func LoadPolicy(repoRoot string) (*Policy, error) {
	doc, err := LoadDocument(ResolveFilePath(repoRoot))
	if err != nil {
		return nil, err
	}
	return New(doc.Rules)
}
