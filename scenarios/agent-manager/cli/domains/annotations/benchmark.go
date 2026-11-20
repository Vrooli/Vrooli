package annotations

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// benchmarkGoldRow is one labelled run. A file may name one label per row or a
// labels map. The gold label is the desired behaviour; the stored annotation is
// the prediction.
type benchmarkGoldRow struct {
	RunID    string          `json:"run_id"`
	Label    string          `json:"label,omitempty"`
	Positive *bool           `json:"positive,omitempty"`
	Labels   map[string]bool `json:"labels,omitempty"`
}

type benchmarkResult struct {
	Label          string  `json:"label"`
	Threshold      float64 `json:"threshold"`
	DefinitionHash string  `json:"definition_hash,omitempty"`
	// Revisions is the count of distinct definition hashes among the scored
	// annotations. A gate is meaningful only for a single revision.
	Revisions int `json:"revisions"`
	// Gold is the number of rows the measurement actually scored. Labelled is
	// every gold row that named this label, including rows excluded because no
	// usable stored answer exists. Only Gold may satisfy gate.min_gold.
	Gold          int     `json:"gold"`
	Labelled      int     `json:"labelled"`
	TruePositive  int     `json:"true_positive"`
	FalsePositive int     `json:"false_positive"`
	FalseNegative int     `json:"false_negative"`
	TrueNegative  int     `json:"true_negative"`
	Unavailable   int     `json:"unavailable"`
	Precision     float64 `json:"precision"`
	Recall        float64 `json:"recall"`
	Note          string  `json:"note,omitempty"`
}

// benchmarkCommand scores stored annotations against a labelled gold file. It
// never calls a model: probabilities are already durable, so re-thresholding is
// free. The output is the gate.measured block the pack needs.
func benchmarkCommand() cliapp.Command {
	return cliapp.Command{
		Name: "benchmark", NeedsAPI: true,
		Description: "Score stored annotations against a labelled gold file (no inference)",
		Usage:       "agent-manager annotations benchmark --gold <file> [--label <id>] [--threshold P] [--json]",
		HelpText:    "Reads JSONL gold rows ({\"run_id\":...,\"label\":...,\"positive\":true} or {\"run_id\":...,\"labels\":{...}}), fetches each run's stored annotation and computes precision and recall. It never calls a decision model; the probabilities are already stored, so changing the threshold is free. A signed-score label applies its declared friction.max_score, so gold positive means the friction direction (for operator.valence, a negative score), never mere confidence. The threshold defaults to the label's declared friction or search threshold, so the gate measures the number consumers actually apply. The emitted gate.measured block carries that threshold and the definition_hash it measured; paste it under the label's gate in .vrooli/annotations/default.json. A measurement for another revision is dropped, so a definition change can never inherit its old proof.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "gold", Required: true, Description: "JSONL gold file"},
			{Name: "label", Description: "Label id (required unless every row names the same label)"},
			{Name: "threshold", Description: "Positive probability threshold; defaults to the label's declared threshold"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			rows, err := readGoldRows(ctx.Flag("gold"))
			if err != nil {
				return err
			}
			label := strings.TrimSpace(ctx.Flag("label"))
			if label == "" {
				label, err = singleGoldLabel(rows)
				if err != nil {
					return err
				}
			}
			threshold := 0.0
			if raw := strings.TrimSpace(ctx.Flag("threshold")); raw != "" {
				if _, err := fmt.Sscanf(raw, "%f", &threshold); err != nil {
					return fmt.Errorf("--threshold must be a number: %w", err)
				}
			} else {
				declared, err := declaredThreshold(ctx, label)
				if err != nil {
					return err
				}
				threshold = declared
			}
			if threshold <= 0 || threshold > 1 {
				return fmt.Errorf("threshold must be in (0,1], got %v", threshold)
			}
			views := map[string]runView{}
			for _, row := range rows {
				if _, seen := views[row.RunID]; seen {
					continue
				}
				body, err := ctx.Core().Request("GET", "annotations/run/"+url.PathEscape(row.RunID), nil, nil)
				if err != nil {
					return fmt.Errorf("read annotations for %s: %w", row.RunID, err)
				}
				var view runView
				if err := json.Unmarshal(body, &view); err != nil {
					return fmt.Errorf("decode annotations for %s: %w", row.RunID, err)
				}
				views[row.RunID] = view
			}
			result := scoreGold(rows, views, label, threshold)
			if ctx.JSON() {
				encoded, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(ctx.Stdout(), string(encoded))
				return err
			}
			renderBenchmark(ctx.Stdout(), result)
			return nil
		},
	}
}

// declaredThreshold reads the probability the label's consumers actually use:
// the friction threshold when friction is enabled, otherwise the search
// threshold. Measuring at a different number would produce a gate that says
// nothing about what search tags or friction counts.
func declaredThreshold(ctx cliapp.RunContext, label string) (float64, error) {
	body, err := ctx.Core().Request("GET", "annotations/coverage?window=all", nil, nil)
	if err != nil {
		return 0, fmt.Errorf("read label thresholds: %w", err)
	}
	var response coverageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("decode label thresholds: %w", err)
	}
	for _, candidate := range response.Labels {
		if candidate.ID != label {
			continue
		}
		if candidate.Friction.Enabled && candidate.Friction.MinProbability != nil {
			return *candidate.Friction.MinProbability, nil
		}
		if candidate.MinProbability != nil {
			return *candidate.MinProbability, nil
		}
		return 0, fmt.Errorf("label %s declares no probability threshold; pass --threshold", label)
	}
	return 0, fmt.Errorf("label %s is not in the loaded pack", label)
}

func readGoldRows(path string) ([]benchmarkGoldRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var rows []benchmarkGoldRow
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var row benchmarkGoldRow
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

func singleGoldLabel(rows []benchmarkGoldRow) (string, error) {
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

// scoreGold is pure so the arithmetic is testable without HTTP. A stored
// probability or noul value is the prediction; an observation without either
// is unavailable and excluded from the denominators. A signed-score label also
// applies its declared polarity ceiling, so praise is never scored as a
// friction positive.
func scoreGold(rows []benchmarkGoldRow, views map[string]runView, label string, threshold float64) benchmarkResult {
	result := benchmarkResult{Label: label, Threshold: threshold}
	revisions := map[string]struct{}{}
	for _, row := range rows {
		positive, ok := goldPositive(row, label)
		if !ok {
			continue
		}
		result.Labelled++
		view, ok := views[row.RunID]
		if !ok {
			result.Unavailable++
			continue
		}
		annotation, ok := findAnnotation(view.Labels, label)
		if !ok || (annotation.State != "observed" && annotation.State != "compatible") {
			result.Unavailable++
			continue
		}
		predicted, ok := annotationPositive(annotation, threshold)
		if !ok {
			result.Unavailable++
			continue
		}
		if hash := strings.TrimSpace(annotation.DefinitionHash); hash != "" {
			revisions[hash] = struct{}{}
		}
		result.Gold++
		switch {
		case positive && predicted:
			result.TruePositive++
		case positive && !predicted:
			result.FalseNegative++
		case !positive && predicted:
			result.FalsePositive++
		default:
			result.TrueNegative++
		}
	}
	if result.TruePositive+result.FalsePositive > 0 {
		result.Precision = float64(result.TruePositive) / float64(result.TruePositive+result.FalsePositive)
	}
	if result.TruePositive+result.FalseNegative > 0 {
		result.Recall = float64(result.TruePositive) / float64(result.TruePositive+result.FalseNegative)
	}
	result.Revisions = len(revisions)
	if result.Revisions == 1 {
		for hash := range revisions {
			result.DefinitionHash = hash
		}
	}
	if result.Labelled == 0 {
		result.Note = "no gold row names the requested label"
	} else if result.Unavailable > 0 {
		result.Note = fmt.Sprintf("%d of %d labelled row(s) had no usable stored annotation and are excluded from precision/recall and from the gate's gold count", result.Unavailable, result.Labelled)
	}
	return result
}

func goldPositive(row benchmarkGoldRow, label string) (bool, bool) {
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

func findAnnotation(annotations []annotationView, label string) (annotationView, bool) {
	for _, annotation := range annotations {
		if annotation.ID == label {
			return annotation, true
		}
	}
	return annotationView{}, false
}

func annotationPositive(annotation annotationView, threshold float64) (bool, bool) {
	if annotation.Friction.MinScore != nil || annotation.Friction.MaxScore != nil {
		// A score is friction only in the declared direction. Its confidence
		// is the probability mass in that region: using the winning level's
		// probability would hide a clear signal on a multi-level scale.
		if annotation.Value.Score == nil {
			return false, false
		}
		if !scoreInDirection(*annotation.Value.Score, annotation.Friction) {
			// A confident observation outside the friction direction is a
			// usable negative, not an unavailable row.
			return false, true
		}
		confidence := annotation.FrictionProbability
		if confidence == nil {
			confidence = annotation.Probability
		}
		if confidence == nil {
			return false, false
		}
		return *confidence >= threshold, true
	}
	confidence, hasConfidence := annotationConfidence(annotation)
	if !hasConfidence {
		return false, false
	}
	return confidence >= threshold, true
}

func scoreInDirection(score float64, friction friction) bool {
	if friction.MaxScore != nil {
		return score <= *friction.MaxScore
	}
	if friction.MinScore != nil {
		return score >= *friction.MinScore
	}
	return false
}

// annotationConfidence reads the calibrated value a label reports. A choice or
// score carries the winning level's probability; the older noul/bool shapes
// carry their value directly.
func annotationConfidence(annotation annotationView) (float64, bool) {
	if annotation.Probability != nil {
		return *annotation.Probability, true
	}
	if annotation.Value.Noul != nil {
		return *annotation.Value.Noul, true
	}
	if annotation.Value.Bool != nil {
		if *annotation.Value.Bool {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func renderBenchmark(out interface{ Write([]byte) (int, error) }, result benchmarkResult) {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "Label %s · threshold %.2f · scored %d of %d labelled\n", result.Label, result.Threshold, result.Gold, result.Labelled)
	fmt.Fprintf(&buffer, "  TP %d · FP %d · FN %d · TN %d · unavailable %d\n", result.TruePositive, result.FalsePositive, result.FalseNegative, result.TrueNegative, result.Unavailable)
	fmt.Fprintf(&buffer, "  precision %.2f · recall %.2f\n", result.Precision, result.Recall)
	if result.Note != "" {
		fmt.Fprintf(&buffer, "  %s\n", result.Note)
	}
	switch {
	case result.Revisions == 1 && result.DefinitionHash != "":
		fmt.Fprintf(&buffer, "  gate.measured: {\"precision\":%.2f,\"recall\":%.2f,\"gold\":%d,\"threshold\":%.2f,\"definition_hash\":%q}\n", result.Precision, result.Recall, result.Gold, result.Threshold, result.DefinitionHash)
		fmt.Fprintln(&buffer, "  Paste this under the label's gate in .vrooli/annotations/default.json; a measurement proves only the definition it carries.")
	case result.Revisions > 1:
		fmt.Fprintf(&buffer, "  gate not emitted: the scored annotations span %d definition revisions; re-annotate the gold set under one revision.\n", result.Revisions)
	default:
		fmt.Fprintln(&buffer, "  gate not emitted: no scored annotation carried a definition hash.")
	}
	_, _ = out.Write(buffer.Bytes())
}
