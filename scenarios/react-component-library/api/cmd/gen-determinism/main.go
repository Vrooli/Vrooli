package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
	if err := os.WriteFile(*output, generated, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
