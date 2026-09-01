package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var consumerAnchors = []string{"web-console", "agent-inbox", "document-manager"}

type matrixReport struct {
	Findings     []json.RawMessage `json:"findings"`
	RunnerErrors []json.RawMessage `json:"runner_errors"`
	Inspected    int               `json:"inspected_files"`
}

type preflightRow struct {
	Scenario string `json:"scenario"`
	Blocking bool   `json:"blocking"`
	Tokens   []any  `json:"unsatisfied_tokens"`
}

type measure struct {
	Name  string
	Value int
}

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: plan-harness <capture|compare> ...")
	}
	switch os.Args[1] {
	case "capture":
		capture(os.Args[2:])
	case "compare":
		compare(os.Args[2:])
	default:
		fatalf("unknown subcommand %q", os.Args[1])
	}
}

func capture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	out := fs.String("out", "", "output directory")
	root := fs.String("root", "", "repository root (defaults to the workspace root)")
	_ = fs.Parse(args)
	if *out == "" {
		fatalf("capture requires --out <dir>")
	}
	repoRoot := *root
	if repoRoot == "" {
		repoRoot = findRoot()
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	matrix, err := commandJSON(repoRoot, "react-component-library", "catalog", "gates", "--all", "--json")
	if err != nil {
		fatalf("capture gate matrix: %v", err)
	}
	if err := writeFile(filepath.Join(*out, "gate-matrix.json"), matrix); err != nil {
		fatalf("write gate matrix: %v", err)
	}
	corpus, err := commandJSONDir(filepath.Join(repoRoot, "scenarios", "react-component-library", "api"), "go", "run", "./cmd/corpus-report", "--root", "../../..")
	if err != nil {
		fatalf("capture corpus report: %v", err)
	}
	if err := writeFile(filepath.Join(*out, "corpus-report.json"), corpus); err != nil {
		fatalf("write corpus report: %v", err)
	}
	if err := capturePreflights(repoRoot, filepath.Join(*out, "preflight-sweep.tsv")); err != nil {
		fatalf("capture preflight sweep: %v", err)
	}
	if err := captureBuilds(repoRoot, filepath.Join(*out, "consumer-builds.tsv")); err != nil {
		fatalf("capture consumer builds: %v", err)
	}
}

func compare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	baseline := fs.String("baseline", "", "baseline capture directory")
	current := fs.String("current", "", "current capture directory")
	_ = fs.Parse(args)
	if *baseline == "" || *current == "" {
		fatalf("compare requires --baseline <dir> --current <dir>")
	}
	base, err := readMeasures(*baseline)
	if err != nil {
		fatalf("read baseline: %v", err)
	}
	now, err := readMeasures(*current)
	if err != nil {
		fatalf("read current: %v", err)
	}
	nowByName := measureMap(now)
	regressed := hasRegression(base, nowByName)
	fmt.Println("measure\tbaseline\tcurrent\tdelta\tstatus")
	for _, item := range base {
		currentValue := nowByName[item.Name]
		delta := currentValue - item.Value
		bad := regression(item.Name, item.Value, currentValue)
		status := "ok"
		if bad {
			status = "REGRESSION"
			regressed = true
		}
		fmt.Printf("%s\t%d\t%d\t%+d\t%s\n", item.Name, item.Value, currentValue, delta, status)
	}
	if regressed {
		os.Exit(1)
	}
}

func hasRegression(base []measure, current map[string]int) bool {
	for _, item := range base {
		if regression(item.Name, item.Value, current[item.Name]) {
			return true
		}
	}
	return false
}

func measureMap(values []measure) map[string]int {
	result := make(map[string]int, len(values))
	for _, value := range values {
		result[value.Name] = value.Value
	}
	return result
}

func regression(name string, baseline, current int) bool {
	if name == "inspected_files" {
		return current < baseline
	}
	return current > baseline
}

func readMeasures(dir string) ([]measure, error) {
	matrixBytes, err := os.ReadFile(filepath.Join(dir, "gate-matrix.json"))
	if err != nil {
		return nil, err
	}
	var matrix matrixReport
	if err := json.Unmarshal(matrixBytes, &matrix); err != nil {
		return nil, err
	}
	preflight, err := readPreflight(filepath.Join(dir, "preflight-sweep.tsv"))
	if err != nil {
		return nil, err
	}
	buildFailures, err := readBuildFailures(filepath.Join(dir, "consumer-builds.tsv"))
	if err != nil {
		return nil, err
	}
	return []measure{
		{"runner_errors", len(matrix.RunnerErrors)},
		{"findings", len(matrix.Findings)},
		{"inspected_files", matrix.Inspected},
		{"preflight_nonzero", preflight.nonzero},
		{"blocking_preflights", preflight.blocking},
		{"unsatisfied_tokens", preflight.tokens},
		{"consumer_build_failures", buildFailures},
	}, nil
}

type preflightSummary struct{ nonzero, blocking, tokens int }

func readPreflight(path string) (preflightSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return preflightSummary{}, err
	}
	defer file.Close()
	var summary preflightSummary
	scanner := bufio.NewScanner(file)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 6 {
			continue
		}
		if fields[1] != "0" {
			summary.nonzero++
		}
		if fields[2] == "true" {
			summary.blocking++
		}
		if n, parseErr := strconv.Atoi(fields[3]); parseErr == nil {
			summary.tokens += n
		}
	}
	return summary, scanner.Err()
}

func readBuildFailures(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	failures := 0
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) > 1 && fields[1] != "pass" {
			failures++
		}
	}
	return failures, scanner.Err()
}

func capturePreflights(root, path string) error {
	entries, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		return err
	}
	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Test Genie creates disposable positive fixtures under the scenarios
			// root. They are gate inputs, not adopting consumer scenarios.
			if strings.HasPrefix(entry.Name(), "rcl-fixture-") {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(root, "scenarios", entry.Name(), "ui")); statErr == nil {
				scenarios = append(scenarios, entry.Name())
			}
		}
	}
	sort.Strings(scenarios)
	var out strings.Builder
	out.WriteString("scenario\texit_status\tblocking\tunsatisfied_tokens\tmaturity_rung\tmaturity_floor\n")
	for _, scenario := range scenarios {
		data, runErr := commandJSONAllowFailure(root, "react-component-library", "adoptions", "preflight", "controls.button", scenario, "--json")
		row := preflightRow{}
		_ = json.Unmarshal(data, &row)
		exitStatus := "0"
		if runErr != nil {
			exitStatus = "1"
		}
		out.WriteString(strings.Join([]string{scenario, exitStatus, strconv.FormatBool(row.Blocking), strconv.Itoa(len(row.Tokens)), jsonString(data, "maturity_rung"), jsonString(data, "maturity_floor")}, "\t"))
		out.WriteByte('\n')
	}
	return writeFile(path, []byte(out.String()))
}

func captureBuilds(root, path string) error {
	var out strings.Builder
	out.WriteString("scenario\tstatus\n")
	for _, scenario := range consumerAnchors {
		cmd := exec.Command("make", "-C", filepath.Join(root, "scenarios", scenario), "build")
		if err := cmd.Run(); err != nil {
			out.WriteString(scenario + "\tfail\n")
		} else {
			out.WriteString(scenario + "\tpass\n")
		}
	}
	return writeFile(path, []byte(out.String()))
}

func commandJSON(root string, command string, args ...string) ([]byte, error) {
	return commandJSONAllowFailure(root, command, args...)
}

func commandJSONAllowFailure(root string, command string, args ...string) ([]byte, error) {
	return commandJSONDir(root, command, args...)
}

func commandJSONDir(dir string, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stdout.Len() == 0 && err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), err
}

func jsonString(data []byte, key string) string {
	var object map[string]any
	if json.Unmarshal(data, &object) != nil {
		return ""
	}
	if value, ok := object[key].(string); ok {
		return value
	}
	return ""
}

func findRoot() string {
	current, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, statErr := os.Stat(filepath.Join(current, "scenarios", "react-component-library", "api")); statErr == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return current
		}
		current = parent
	}
}

func writeFile(path string, data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("command returned empty output")
	}
	return os.WriteFile(path, data, 0o644)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
