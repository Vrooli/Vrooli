// Package fsx owns engine-independent filesystem predicates and JSON
// serialization. It intentionally does not import config: ownership, modes,
// and atomic replacement remain policy owned by config.
package fsx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Within reports lexical containment after resolving both paths against one
// captured working directory. Equal paths and descendants are contained;
// prefix siblings and parent traversal are not.
func Within(root, target string) (bool, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
		return false, errors.New("fsx: root and target must not be empty")
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("fsx: capture working directory: %w", err)
	}
	rootPath, err := absoluteFrom(workingDir, root)
	if err != nil {
		return false, err
	}
	targetPath, err := absoluteFrom(workingDir, target)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false, fmt.Errorf("fsx: compare paths: %w", err)
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)), nil
}

// WithinResolved is the symlink-aware counterpart to Within. Resolution
// errors are returned because callers using this operation are making an
// authorization decision.
func WithinResolved(root, target string) (bool, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
		return false, errors.New("fsx: root and target must not be empty")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false, fmt.Errorf("fsx: resolve root: %w", err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return false, fmt.Errorf("fsx: resolve target: %w", err)
	}
	return Within(resolvedRoot, resolvedTarget)
}

func absoluteFrom(workingDir, value string) (string, error) {
	if !filepath.IsAbs(value) {
		value = filepath.Join(workingDir, value)
	}
	return filepath.Abs(filepath.Clean(value))
}

// Exists distinguishes absence from other filesystem failures.
func Exists(path string) (bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, errors.New("fsx: path must not be empty")
	}
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// MarshalJSON produces the repository's human-readable JSON file form: two
// spaces, no HTML escaping surprises beyond encoding/json defaults, and one
// trailing newline.
func MarshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ReadJSON reads exactly one non-empty JSON value. Unknown fields are allowed;
// domain schemas decide whether additional fields are valid.
func ReadJSON(path string, destination any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("fsx: path must not be empty")
	}
	if destination == nil {
		return errors.New("fsx: destination must not be nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return errors.New("fsx: JSON file is empty")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("fsx: JSON file contains more than one value")
		}
		return err
	}
	return nil
}
