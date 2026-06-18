package genprune

import (
	"fmt"
	"os"
	"path/filepath"
)

func PruneBeforeGenerate(protoRoot string) error {
	if protoRoot == "" {
		protoRoot = "."
	}
	genRoot := filepath.Join(protoRoot, "gen")
	for _, rel := range []string{"go", "python"} {
		if err := os.RemoveAll(filepath.Join(genRoot, rel)); err != nil {
			return fmt.Errorf("prune gen/%s: %w", rel, err)
		}
	}

	tsRoot := filepath.Join(genRoot, "typescript")
	entries, err := os.ReadDir(tsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read gen/typescript: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == "package.json" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(tsRoot, entry.Name())); err != nil {
			return fmt.Errorf("prune gen/typescript/%s: %w", entry.Name(), err)
		}
	}
	return nil
}
