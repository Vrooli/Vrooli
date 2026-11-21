package annotations

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// goldFileRow is one labelled run in a gold file. A file may name one label per
// row or a labels map. The gold label is the desired behaviour; the stored
// annotation is the prediction.
type goldFileRow struct {
	RunID string `json:"run_id"`
	// ScopeID is the message event id for a message-scope label's row; the
	// run_id is the run it belongs to.
	ScopeID  string          `json:"scope_id,omitempty"`
	Label    string          `json:"label,omitempty"`
	Positive *bool           `json:"positive,omitempty"`
	Labels   map[string]bool `json:"labels,omitempty"`
}

// scoredRow is the wire row the API scores.
type scoredRow struct {
	RunID    string `json:"run_id"`
	ScopeID  string `json:"scope_id,omitempty"`
	Positive bool   `json:"positive"`
}

type reliabilityBin struct {
	Lo            float64 `json:"lo"`
	Hi            float64 `json:"hi"`
	Count         int     `json:"count"`
	MeanPredicted float64 `json:"mean_predicted"`
	ObservedRate  float64 `json:"observed_rate"`
}

type benchmarkGroup struct {
	Model        string `json:"model"`
	ResourceRole string `json:"resource_role"`
	Rows         int    `json:"rows"`
}

// measurement is the entry recorded under a label's gate.measurements.
type measurement struct {
	DefinitionHash string  `json:"definition_hash"`
	Model          string  `json:"model"`
	ResourceRole   string  `json:"resource_role"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	Gold           int     `json:"gold"`
	Positives      int     `json:"positives"`
	Threshold      float64 `json:"threshold"`
	Temperature    float64 `json:"temperature"`
	ECE            float64 `json:"ece"`
	Brier          float64 `json:"brier"`
	Split          string  `json:"split"`
	GoldFile       string  `json:"gold_file,omitempty"`
	MeasuredAt     string  `json:"measured_at,omitempty"`
}

// benchmarkResult mirrors the API's measurement.
type benchmarkResult struct {
	Label              string           `json:"label"`
	DefinitionHash     string           `json:"definition_hash"`
	Split              string           `json:"split"`
	Source             string           `json:"source"`
	Model              string           `json:"model"`
	ResourceRole       string           `json:"resource_role"`
	Labelled           int              `json:"labelled"`
	Gold               int              `json:"gold"`
	Positives          int              `json:"positives"`
	Unobserved         int              `json:"unobserved"`
	UnobservedSubjects []string         `json:"unobserved_subjects,omitempty"`
	Unavailable        int              `json:"unavailable"`
	FitRows            int              `json:"fit_rows"`
	TP                 int              `json:"true_positive"`
	FP                 int              `json:"false_positive"`
	FN                 int              `json:"false_negative"`
	TN                 int              `json:"true_negative"`
	Precision          float64          `json:"precision"`
	Recall             float64          `json:"recall"`
	Threshold          float64          `json:"threshold"`
	Temperature        float64          `json:"temperature"`
	Fitted             bool             `json:"fitted"`
	ECE                float64          `json:"ece"`
	Brier              float64          `json:"brier"`
	RawECE             float64          `json:"raw_ece"`
	RawBrier           float64          `json:"raw_brier"`
	Reliability        []reliabilityBin `json:"reliability"`
	RawReliability     []reliabilityBin `json:"raw_reliability"`
	StoredTemperature  float64          `json:"stored_temperature"`
	Measurement        *measurement     `json:"measurement"`
	Groups             []benchmarkGroup `json:"groups"`
	Refusal            string           `json:"refusal"`
	Note               string           `json:"note"`
}

// benchmarkRequest is the wire request.
type benchmarkRequest struct {
	Label        string      `json:"label"`
	Split        string      `json:"split,omitempty"`
	Rows         []scoredRow `json:"rows"`
	FitRows      []scoredRow `json:"fit_rows,omitempty"`
	GoldFile     string      `json:"gold_file,omitempty"`
	Threshold    *float64    `json:"threshold,omitempty"`
	Model        string      `json:"model,omitempty"`
	ResourceRole string      `json:"resource_role,omitempty"`
	Source       string      `json:"source,omitempty"`
}

// benchmarkFlags are shared by benchmark and calibration: which candidate's
// stored answers to read.
var candidateFlags = []cliapp.Flag{
	{Name: "model", Description: "Score only this model's stored answers (required when the gold rows span several)"},
	{Name: "resource-role", Description: "Score only answers served by this resource role (for example decide.local.accurate)"},
	{Name: "pinned", Bool: true, LocalOnly: true, Description: "Read the answers of a pinned bake-off (annotations evaluate --pin-resource-role) instead of production"},
}

func sourceFlag(ctx cliapp.RunContext) string {
	if ctx.BoolFlag("pinned") {
		return "pinned"
	}
	return ""
}

// benchmarkCommand measures stored annotations against a labelled gold file. It
// never calls a model: probabilities are already durable, so re-thresholding and
// recalibrating are free. The output is the gate measurement to record.
func benchmarkCommand() cliapp.Command {
	return cliapp.Command{
		Name: "benchmark", NeedsAPI: true,
		Description: "Measure stored annotations against a labelled gold file (no inference)",
		Usage:       "agent-manager annotations benchmark --gold <file> [--label <id>] [--split tuning|holdout] [--fit-from <tuning file>] [--model M --resource-role R] [--pinned] [--threshold P] [--json]",
		HelpText: "Reads JSONL gold rows ({\"run_id\":...,\"label\":...,\"positive\":true} or {\"run_id\":...,\"labels\":{...}}) and scores each run's stored probability vector. It never calls a decision model.\n\n" +
			"A measurement is for one (definition hash, model, resource role): rows whose stored answers span more than one are refused, so pass --model and --resource-role to pick the candidate. " +
			"With --split tuning (the default) the temperature is fitted on the rows by minimizing negative log-likelihood, the threshold is chosen from the calibrated curve against the label's gate targets, and ECE and Brier are reported before and after. " +
			"With --split holdout the rows are evaluated with the temperature and threshold fitted on the file named by --fit-from, without refitting; the two files must share no run. Only a hold-out measurement can prove a label, and a proof applies to that model alone.\n\n" +
			"A signed-score label applies its declared friction.max_score, so gold positive means the friction direction. Record the printed measurement under the label's gate.measurements in .vrooli/annotations/default.json; the stored vectors are never rewritten, the temperature is applied when they are read.",
		Args: cliapp.ArgSchema{Flags: append([]cliapp.Flag{
			{Name: "gold", Required: true, Description: "JSONL gold file (relative paths resolve against the gold root, runtime.gold_root)"},
			{Name: "label", Description: "Label id (required unless every row names the same label)"},
			{Name: "split", Default: "tuning", Description: "tuning (fit and evaluate on the rows) or holdout (evaluate a fit from --fit-from)"},
			{Name: "fit-from", Description: "Tuning gold file a hold-out's temperature and threshold are fitted on"},
			{Name: "threshold", Description: "Override the threshold chosen from the calibrated curve"},
		}, candidateFlags...)},
		RunCtx: func(ctx cliapp.RunContext) error {
			root, err := fetchGoldRoot(ctx)
			if err != nil {
				return err
			}
			request, err := benchmarkRequestFrom(root, ctx.Flag("gold"), ctx.Flag("label"), ctx.Flag("split"), ctx.Flag("fit-from"),
				ctx.Flag("threshold"), ctx.Flag("model"), ctx.Flag("resource-role"), sourceFlag(ctx))
			if err != nil {
				return err
			}
			result, err := postBenchmark(ctx, request)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				encoded, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(ctx.Stdout(), string(encoded))
				return err
			}
			renderBenchmark(ctx.Stdout(), result)
			if result.Refusal != "" {
				return fmt.Errorf("no measurement made: %s", result.Refusal)
			}
			return nil
		},
	}
}

// benchmarkRequestFrom reads the gold files, resolved against the gold root, and
// builds the wire request. It is pure apart from reading those files, so flag
// handling is testable.
func benchmarkRequestFrom(root goldRoot, goldPath, label, split, fitPath, threshold, model, role, source string) (benchmarkRequest, error) {
	resolved, err := root.resolve(goldPath)
	if err != nil {
		return benchmarkRequest{}, err
	}
	rows, err := readGoldRows(resolved)
	if err != nil {
		return benchmarkRequest{}, err
	}
	label = strings.TrimSpace(label)
	if label == "" {
		if label, err = singleGoldLabel(rows); err != nil {
			return benchmarkRequest{}, err
		}
	}
	split = strings.TrimSpace(split)
	if split == "" {
		split = "tuning"
	}
	request := benchmarkRequest{
		Label: label, Split: split, Rows: goldRowsFor(rows, label), GoldFile: root.name(resolved),
		Model: strings.TrimSpace(model), ResourceRole: strings.TrimSpace(role), Source: source,
	}
	if len(request.Rows) == 0 {
		return benchmarkRequest{}, fmt.Errorf("no gold row names label %s", label)
	}
	if fitPath = strings.TrimSpace(fitPath); fitPath != "" {
		if split != "holdout" {
			return benchmarkRequest{}, fmt.Errorf("--fit-from belongs to --split holdout; a tuning measurement fits on its own rows")
		}
		fit, err := root.readRows(fitPath)
		if err != nil {
			return benchmarkRequest{}, fmt.Errorf("--fit-from: %w", err)
		}
		if request.FitRows = goldRowsFor(fit, label); len(request.FitRows) == 0 {
			return benchmarkRequest{}, fmt.Errorf("--fit-from %s names no row for label %s", fitPath, label)
		}
	}
	if raw := strings.TrimSpace(threshold); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value <= 0 || value > 1 {
			return benchmarkRequest{}, fmt.Errorf("--threshold must be a number in (0,1]")
		}
		request.Threshold = &value
	}
	return request, nil
}

func postBenchmark(ctx cliapp.RunContext, request benchmarkRequest) (benchmarkResult, error) {
	body, err := ctx.Core().Request("POST", "annotations/benchmark", nil, request)
	if err != nil {
		return benchmarkResult{}, err
	}
	var result benchmarkResult
	if err := json.Unmarshal(body, &result); err != nil {
		return benchmarkResult{}, fmt.Errorf("decode benchmark: %w", err)
	}
	return result, nil
}

// goldRowsFor selects the rows naming a label, as the API's wire rows.
func goldRowsFor(rows []goldFileRow, label string) []scoredRow {
	var out []scoredRow
	for _, row := range rows {
		if positive, ok := goldPositive(row, label); ok {
			out = append(out, scoredRow{RunID: row.RunID, ScopeID: row.ScopeID, Positive: positive})
		}
	}
	return out
}

func readGoldRows(path string) ([]goldFileRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseGoldRows(file)
}

func parseGoldRows(reader io.Reader) ([]goldFileRow, error) {
	var rows []goldFileRow
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var row goldFileRow
		if err := json.Unmarshal([]byte(text), &row); err != nil {
			return nil, fmt.Errorf("gold line %d: %w", line, err)
		}
		if strings.TrimSpace(row.RunID) == "" {
			return nil, fmt.Errorf("gold line %d: run_id is required", line)
		}
		if row.Positive == nil && row.Labels == nil {
			return nil, fmt.Errorf("gold line %d: set positive to true or false; an unfilled draft row measures nothing", line)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("gold file has no rows")
	}
	return rows, nil
}

func singleGoldLabel(rows []goldFileRow) (string, error) {
	labels := map[string]bool{}
	for _, row := range rows {
		if strings.TrimSpace(row.Label) != "" {
			labels[row.Label] = true
		}
		for label := range row.Labels {
			labels[label] = true
		}
	}
	if len(labels) != 1 {
		names := make([]string, 0, len(labels))
		for label := range labels {
			names = append(names, label)
		}
		sort.Strings(names)
		return "", fmt.Errorf("gold file names %d labels (%s); pass --label", len(labels), strings.Join(names, ", "))
	}
	for label := range labels {
		return label, nil
	}
	return "", fmt.Errorf("gold file names no label")
}

func goldPositive(row goldFileRow, label string) (bool, bool) {
	if row.Labels != nil {
		value, ok := row.Labels[label]
		return value, ok
	}
	if row.Positive == nil {
		return false, false
	}
	if named := strings.TrimSpace(row.Label); named != "" && named != label {
		return false, false
	}
	return *row.Positive, true
}

func renderBenchmark(out io.Writer, result benchmarkResult) {
	var buffer bytes.Buffer
	candidate := "no stored answer"
	if result.Model != "" {
		candidate = result.Model
		if result.ResourceRole != "" {
			candidate += " @ " + result.ResourceRole
		}
	}
	fmt.Fprintf(&buffer, "Label %s · %s · %s rows · scored %d of %d labelled", result.Label, emptyAs(result.Split, "tuning"), emptyAs(result.Source, "production"), result.Gold, result.Labelled)
	if result.FitRows > 0 {
		fmt.Fprintf(&buffer, " · fitted on %d tuning row(s)", result.FitRows)
	}
	fmt.Fprintf(&buffer, "\n  candidate: %s\n", candidate)
	if result.Refusal != "" {
		fmt.Fprintf(&buffer, "  refused: %s\n", result.Refusal)
		for _, group := range result.Groups {
			fmt.Fprintf(&buffer, "    %s @ %s: %d row(s)\n", emptyAs(group.Model, "(no model)"), emptyAs(group.ResourceRole, "(no role)"), group.Rows)
		}
		_, _ = out.Write(buffer.Bytes())
		return
	}
	if result.Note != "" {
		fmt.Fprintf(&buffer, "  %s\n", result.Note)
	}
	if result.Measurement == nil {
		fmt.Fprintln(&buffer, "  no measurement: nothing could be scored.")
		if result.Unobserved > 0 {
			fmt.Fprintf(&buffer, "  %d labelled row(s) have no stored answer under the current definition: %s\n", result.Unobserved, strings.Join(result.UnobservedSubjects, ", "))
		}
		_, _ = out.Write(buffer.Bytes())
		return
	}
	fmt.Fprintf(&buffer, "  TP %d · FP %d · FN %d · TN %d · unobserved %d · unavailable %d · gold positives %d\n", result.TP, result.FP, result.FN, result.TN, result.Unobserved, result.Unavailable, result.Positives)
	fmt.Fprintf(&buffer, "  precision %.2f · recall %.2f at threshold %.4f\n", result.Precision, result.Recall, result.Threshold)
	fitted := "not fitted (kept 1: the rows cannot constrain it)"
	if result.Fitted {
		fitted = "fitted"
	}
	if result.Split == "holdout" {
		fitted = "applied from the tuning fit, not refitted"
	}
	fmt.Fprintf(&buffer, "  temperature %.4f (%s)\n", result.Temperature, fitted)
	fmt.Fprintf(&buffer, "  ECE %.4f -> %.4f · Brier %.4f -> %.4f (raw -> calibrated)\n", result.RawECE, result.ECE, result.RawBrier, result.Brier)
	encoded, err := json.Marshal(result.Measurement)
	if err == nil {
		fmt.Fprintf(&buffer, "  measurement: %s\n", encoded)
	}
	if result.Split == "holdout" {
		fmt.Fprintln(&buffer, "  Record it under the label's gate.measurements in .vrooli/annotations/default.json; a hold-out proves this model and definition only.")
	} else {
		fmt.Fprintln(&buffer, "  Tuning only fits the calibration and never proves a label. Measure a disjoint file with --split holdout --fit-from <this file>.")
	}
	_, _ = out.Write(buffer.Bytes())
}
