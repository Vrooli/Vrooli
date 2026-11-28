package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
	"react-component-library/internal/gates"
)

func main() {
	output := flag.String("output", "", "test-genie configuration to update")
	check := flag.Bool("check", false, "fail when generated output differs")
	flag.Parse()
	if *output == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fatal(err)
		}
		*output = filepath.Join(cwd, "..", ".vrooli", "test-genie.json")
	}
	original, err := os.ReadFile(*output)
	if err != nil {
		fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(original, &document); err != nil {
		fatal(err)
	}
	capabilities := map[string]any{}
	for _, definition := range gates.Definitions() {
		capabilities[definition.ID] = map[string]any{"mode": "file-determined", "inputs": definition.DeterminismInputs}
	}
	document["determinism"] = map[string]any{"default": "observational", "reason": "Generated from the catalog gate registry; edit gate definitions rather than this block.", "capabilities": capabilities}
	generated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fatal(err)
	}
	generated = append(generated, '\n')
	if *check {
		if !bytes.Equal(original, generated) {
			fatal(fmt.Errorf("determinism block is stale; run make determinism"))
		}
		return
	}
	if err := writeGeneratedFile(*output, generated); err != nil {
		fatal(err)
	}
}

func writeGeneratedFile(output string, generated []byte) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	directory, name := filepath.Dir(absolute), filepath.Base(absolute)
	if !filepath.IsLocal(name) || name == "." {
		return fmt.Errorf("invalid generated output path: %s", output)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(absolute); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("generated output is not a regular file: %s", absolute)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect generated output: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return fmt.Errorf("open generated output directory: %w", err)
	}
	defer root.Close()
	if err := storage.WriteFileAtomicInRoot(root, name, generated, mode); err != nil {
		return fmt.Errorf("publish generated output: %w", err)
	}
	return nil
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
