package requirements

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	reqservice "test-genie/internal/requirements"
)

func runPhaseInspect(args []string) error {
	fs := flag.NewFlagSet("requirements phase", flag.ContinueOnError)
	dirFlag, scenarioFlag := parseCommonFlags(fs)
	phase := fs.String("phase", "", "Phase to inspect (required)")
	outputPath := fs.String("output", "", "Write output to file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*phase) == "" {
		return fmt.Errorf("--phase is required")
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	svc := reqservice.NewService()
	result, err := svc.PhaseInspect(context.Background(), dir, *phase)
	if err != nil {
		return fmt.Errorf("phase-inspect failed: %w", err)
	}

	result.Scenario = scenarioNameFromDir(dir, *scenarioFlag)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode phase-inspect output: %w", err)
	}

	if *outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		if err := os.WriteFile(*outputPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("write phase-inspect output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[requirements] wrote phase-inspect: %s\n", *outputPath)
		return nil
	}

	fmt.Println(string(data))
	return nil
}
