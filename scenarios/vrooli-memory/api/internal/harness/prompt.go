package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	promptStart = "<!-- vrooli-memory:prompt-block:start -->"
	promptEnd   = "<!-- vrooli-memory:prompt-block:end -->"
)

func PromptBlock() string {
	return promptStart + "\n## Durable memory\nRecord durable rules, important environment facts, decisions, outcomes, and hard-won gotchas in your native memory when they will help a future agent. Do not record transient chat or telemetry.\n" + promptEnd + "\n"
}

func InstallPromptBlock(path string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(data)
	start, end := strings.Index(text, promptStart), strings.Index(text, promptEnd)
	if (start >= 0) != (end >= 0) || (start >= 0 && end < start) {
		return fmt.Errorf("malformed existing vrooli-memory prompt block in %s", path)
	}
	if start >= 0 {
		text = text[:start] + PromptBlock() + text[end+len(promptEnd):]
	} else {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += PromptBlock()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".vrooli-memory.tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// PromptTarget resolves a writable convention file independently of generated
// memory projections. VROOLI_MEMORY_WORKSPACE_ROOT makes the selection
// explicit in managed deployments; the process working directory is the safe
// local-development default.
func PromptTarget(runtime, root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("VROOLI_MEMORY_WORKSPACE_ROOT is required for prompt installation")
	}
	switch runtime {
	case "claude-code":
		return filepath.Join(root, "CLAUDE.md"), nil
	case "codex", "opencode":
		return filepath.Join(root, "AGENTS.md"), nil
	default:
		return "", fmt.Errorf("runtime %q has no independent prompt convention file", runtime)
	}
}
