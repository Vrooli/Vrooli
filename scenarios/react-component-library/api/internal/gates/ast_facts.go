package gates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type sourceFacts struct {
	File                string              `json:"file"`
	Attributes          map[string][]string `json:"attributes"`
	Elements            []sourceFactElement `json:"elements"`
	Calls               []string            `json:"calls"`
	InlineStyleElements int                 `json:"inlineStyleElements"`
}

type sourceFactElement struct {
	Tag        string              `json:"tag"`
	Attributes map[string][]string `json:"attributes"`
}

func readSourceFacts(root, path string) ([]sourceFacts, error) {
	script := filepath.Join(root, "packages", "react-component-library", "scripts", "resolve-imports.mjs")
	if _, err := os.Stat(script); err != nil {
		return nil, err
	}
	command := exec.CommandContext(context.Background(), "node", script, "--facts", path)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("analyze structured TSX facts for %s: %w", path, err)
	}
	var facts []sourceFacts
	if err := json.Unmarshal(output, &facts); err != nil {
		return nil, fmt.Errorf("decode structured TSX facts for %s: %w", path, err)
	}
	return facts, nil
}

func readSourceFactsIndex(root string) (map[string]sourceFacts, error) {
	script := filepath.Join(root, "packages", "react-component-library", "scripts", "resolve-imports.mjs")
	if _, err := os.Stat(script); err != nil {
		return nil, err
	}
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	command := exec.CommandContext(context.Background(), "node", script, "--facts-root", libraryRoot)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("analyze structured source facts for %s: %w", libraryRoot, err)
	}
	var facts []sourceFacts
	if err := json.Unmarshal(output, &facts); err != nil {
		return nil, fmt.Errorf("decode structured source facts for %s: %w", libraryRoot, err)
	}
	index := make(map[string]sourceFacts, len(facts))
	for _, fact := range facts {
		index[filepath.Clean(fact.File)] = fact
	}
	return index, nil
}

// sourceHasOverlayRole consumes the shared AST facts emitted by the package's
// ast-grep parser. The byte fallback exists only for sparse Go fixtures that
// intentionally do not carry the repository package tree; production scans
// always use the structured facts path.
func sourceHasOverlayRole(root, path string, source []byte) (bool, error) {
	script := filepath.Join(root, "packages", "react-component-library", "scripts", "resolve-imports.mjs")
	if _, err := os.Stat(script); err != nil {
		if os.IsNotExist(err) {
			for _, role := range []string{`role="dialog"`, `role='dialog'`, `role="alertdialog"`, `role='alertdialog'`, `role="menu"`, `role='menu'`} {
				if bytes.Contains(source, []byte(role)) {
					return true, nil
				}
			}
			return false, nil
		}
		return false, err
	}
	facts, err := readSourceFacts(root, path)
	if err != nil {
		return false, err
	}
	for _, fact := range facts {
		for _, value := range fact.Attributes["role"] {
			value = strings.ToLower(value)
			for _, role := range []string{`"dialog"`, `"alertdialog"`, `"menu"`, `'dialog'`, `'alertdialog'`, `'menu'`} {
				if strings.Contains(value, role) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
