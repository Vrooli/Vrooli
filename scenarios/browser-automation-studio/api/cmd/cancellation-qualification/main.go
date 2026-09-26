package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("cancellation-qualification", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	build := flags.String("build", "", "managed API build identity")
	observationsPath := flags.String("observations", "", "owner-emitted JSONL observations")
	outputPath := flags.String("output", "", "new cancellation receipt path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *build == "" || *observationsPath == "" || *outputPath == "" {
		return errors.New("usage: cancellation-qualification --build <identity> --observations <jsonl> --output <new-receipt.json>")
	}

	observationsFile, err := os.Open(*observationsPath)
	if err != nil {
		return fmt.Errorf("open owner observations: %w", err)
	}
	defer observationsFile.Close()

	var records []cancellationqualification.ObservationRecord
	scanner := bufio.NewScanner(observationsFile)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var record cancellationqualification.ObservationRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("decode owner observation line %d: %w", line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read owner observations: %w", err)
	}

	scenarioRoot, err := findScenarioRoot(".")
	if err != nil {
		return fmt.Errorf("resolve scenario root: %w", err)
	}
	outputPathAbs, err := filepath.Abs(*outputPath)
	if err != nil {
		return fmt.Errorf("resolve receipt output path: %w", err)
	}
	evidenceRoot := filepath.Join(scenarioRoot, ".vrooli/runtime/rehabilitation-evidence")
	relativeOutput, err := filepath.Rel(evidenceRoot, outputPathAbs)
	if err != nil || relativeOutput == ".." || strings.HasPrefix(relativeOutput, ".."+string(filepath.Separator)) {
		return fmt.Errorf("receipt output must stay under %s", evidenceRoot)
	}
	if filepath.Ext(outputPathAbs) != ".json" || !strings.HasPrefix(filepath.Base(outputPathAbs), "cancellation-recovery-") {
		return errors.New("receipt output must be a cancellation-recovery-*.json file")
	}

	receipt, err := cancellationqualification.AssembleReceipt(scenarioRoot, *build, records)
	if err != nil {
		return fmt.Errorf("assemble cancellation receipt: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPathAbs), 0o755); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}
	file, err := os.OpenFile(outputPathAbs, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create new receipt: %w", err)
	}
	encodeErr := json.NewEncoder(file).Encode(receipt)
	closeErr := file.Close()
	if encodeErr != nil {
		_ = os.Remove(outputPathAbs)
		return fmt.Errorf("write cancellation receipt: %w", encodeErr)
	}
	if closeErr != nil {
		_ = os.Remove(outputPathAbs)
		return fmt.Errorf("close cancellation receipt: %w", closeErr)
	}
	_, err = fmt.Fprintln(stdout, outputPathAbs)
	return err
}

func findScenarioRoot(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "docs/internal/REFRACTOR_CONTRACT.json")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("no BAS scenario root above %s", start)
		}
		directory = parent
	}
}
