// ui-health-trust-replay evaluates frozen BrowserObservation records through
// ui-health's current client-observation evaluator without writing to its DB.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	"ui-health/internal/visualhealth"
)

type inputRow struct {
	CaseID      string          `json:"case_id"`
	Phase       string          `json:"phase"`
	RuleCode    string          `json:"rule_code"`
	Observation json.RawMessage `json:"observation"`
}

type outputRow struct {
	CaseID       string   `json:"case_id"`
	Phase        string   `json:"phase"`
	Verdict      string   `json:"verdict"`
	RuleCode     string   `json:"rule_code"`
	FindingCodes []string `json:"finding_codes"`
	Evaluator    string   `json:"evaluator"`
}

func main() {
	inputPath := flag.String("input", "-", "input corpus JSONL; '-' means stdin")
	outputPath := flag.String("output", "-", "prediction JSONL; '-' means stdout")
	flag.Parse()

	if err := run(*inputPath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, outputPath string) error {
	in, closeIn, err := openInput(inputPath)
	if err != nil {
		return err
	}
	if closeIn != nil {
		defer closeIn()
	}
	out, closeOut, err := openOutput(outputPath)
	if err != nil {
		return err
	}
	if closeOut != nil {
		defer closeOut()
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	writer := bufio.NewWriter(out)
	defer writer.Flush()
	seen := map[string]struct{}{}
	line := 0
	for scanner.Scan() {
		line++
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var row inputRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return fmt.Errorf("input line %d: invalid JSON: %w", line, err)
		}
		if row.CaseID == "" || row.Phase == "" || row.RuleCode == "" || len(row.Observation) == 0 {
			return fmt.Errorf("input line %d: case_id, phase, rule_code, and observation are required", line)
		}
		if _, ok := seen[row.CaseID]; ok {
			return fmt.Errorf("input line %d: duplicate case_id %q", line, row.CaseID)
		}
		seen[row.CaseID] = struct{}{}

		observation := &pb.BrowserObservation{}
		if err := protojson.Unmarshal(row.Observation, observation); err != nil {
			return fmt.Errorf("input line %d: invalid BrowserObservation: %w", line, err)
		}
		findings := visualhealth.ReplayBrowserObservationFindings(observation)
		codes := make([]string, 0, len(findings))
		verdict := "passed"
		for _, finding := range findings {
			codes = append(codes, finding.GetCode())
			if finding.GetCode() == row.RuleCode {
				verdict = "failed"
			}
		}
		result := outputRow{CaseID: row.CaseID, Phase: row.Phase, Verdict: verdict, RuleCode: row.RuleCode, FindingCodes: codes, Evaluator: "ui-health.browserObservationFindings/current"}
		encoded, err := json.Marshal(result)
		if err != nil {
			return err
		}
		if _, err = writer.Write(append(encoded, '\n')); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if line == 0 {
		return errors.New("input corpus is empty")
	}
	return writer.Flush()
}

func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}

func openOutput(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, nil, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}
