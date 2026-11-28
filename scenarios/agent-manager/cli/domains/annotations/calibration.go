package annotations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// calibrationCommand prints a label's reliability table from stored probability
// vectors and gold. It runs no inference and records nothing.
func calibrationCommand() cliapp.Command {
	return cliapp.Command{
		Name: "calibration", NeedsAPI: true,
		Description: "Print a label's reliability table, ECE and Brier from stored vectors and gold (no inference)",
		Usage:       "agent-manager annotations calibration --label <id> [--gold <file>]... [--model M] [--resource-role R] [--pinned] [--json]",
		HelpText: "Bins the stored probabilities of one label into ten equal-width bins and reports, for each, the number of rows, the mean predicted score and the observed positive rate, then ECE and Brier before and after the fitted temperature. " +
			"The rows come from one or more --gold files; without --gold the files recorded in the label's gate measurements are read from the gold root. " +
			"It is free: the vectors are already stored, and nothing is written.",
		Args: cliapp.ArgSchema{Flags: append([]cliapp.Flag{
			{Name: "label", Required: true, Description: "Label id"},
			{Name: "gold", Repeated: true, Description: "Gold file (repeatable); default: the gold files named by the label's measurements"},
		}, candidateFlags...)},
		RunCtx: func(ctx cliapp.RunContext) error {
			label := strings.TrimSpace(ctx.Flag("label"))
			root, err := fetchGoldRoot(ctx)
			if err != nil {
				return err
			}
			paths := ctx.FlagValues("gold")
			if len(paths) == 0 {
				discovered, err := measuredGoldFiles(ctx, root, label, strings.TrimSpace(ctx.Flag("model")))
				if err != nil {
					return err
				}
				paths = discovered
			}
			request, err := calibrationRequestFrom(root, paths, label, ctx.Flag("model"), ctx.Flag("resource-role"), sourceFlag(ctx))
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
			renderCalibration(ctx.Stdout(), result)
			if result.Refusal != "" {
				return fmt.Errorf("no calibration report: %s", result.Refusal)
			}
			return nil
		},
	}
}

// calibrationRequestFrom unions the gold files into one request. A run named in
// two files counts once (the first label wins), so overlapping files never
// weigh a run twice.
func calibrationRequestFrom(root goldRoot, paths []string, label, model, role, source string) (benchmarkRequest, error) {
	request := benchmarkRequest{
		Label: label, Split: "tuning", Model: strings.TrimSpace(model), ResourceRole: strings.TrimSpace(role), Source: source,
	}
	seen := map[string]bool{}
	var names []string
	for _, path := range paths {
		resolved, err := root.resolve(path)
		if err != nil {
			return benchmarkRequest{}, err
		}
		names = append(names, root.name(resolved))
		rows, err := readGoldRows(resolved)
		if err != nil {
			return benchmarkRequest{}, err
		}
		for _, row := range goldRowsFor(rows, label) {
			if !seen[row.RunID] {
				seen[row.RunID] = true
				request.Rows = append(request.Rows, row)
			}
		}
	}
	if len(request.Rows) == 0 {
		return benchmarkRequest{}, fmt.Errorf("no gold row names label %s", label)
	}
	request.GoldFile = strings.Join(names, ",")
	return request, nil
}

// measuredGoldFiles finds the gold files a label's recorded measurements name
// and that exist under the gold root.
func measuredGoldFiles(ctx cliapp.RunContext, root goldRoot, label, model string) ([]string, error) {
	body, err := ctx.Core().Request("GET", "annotations/labels", url.Values{"window": []string{"all"}}, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Labels []struct {
			ID   string `json:"id"`
			Gate struct {
				Measurements []measurement `json:"measurements"`
			} `json:"gate"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	var paths []string
	seen := map[string]bool{}
	for _, candidate := range response.Labels {
		if candidate.ID != label {
			continue
		}
		for _, entry := range candidate.Gate.Measurements {
			if entry.GoldFile == "" || seen[entry.GoldFile] || (model != "" && entry.Model != model) {
				continue
			}
			if resolved, err := root.resolve(entry.GoldFile); err == nil && fileExists(resolved) {
				seen[entry.GoldFile] = true
				paths = append(paths, entry.GoldFile)
			}
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no readable gold file is recorded for label %s; pass --gold <file>", label)
	}
	return paths, nil
}

func renderCalibration(out io.Writer, result benchmarkResult) {
	var buffer bytes.Buffer
	candidate := "no stored answer"
	if result.Model != "" {
		candidate = result.Model
		if result.ResourceRole != "" {
			candidate += " @ " + result.ResourceRole
		}
	}
	fmt.Fprintf(&buffer, "Calibration %s · %s · %s · %d gold row(s) scored of %d (%d gold positive)\n",
		result.Label, candidate, emptyAs(result.Source, "production"), result.Gold, result.Labelled, result.Positives)
	if result.Refusal != "" {
		fmt.Fprintf(&buffer, "  refused: %s\n", result.Refusal)
		for _, group := range result.Groups {
			fmt.Fprintf(&buffer, "    %s @ %s: %d row(s)\n", emptyAs(group.Model, "(no model)"), emptyAs(group.ResourceRole, "(no role)"), group.Rows)
		}
		_, _ = out.Write(buffer.Bytes())
		return
	}
	if result.Gold == 0 {
		fmt.Fprintln(&buffer, "  nothing could be scored: no gold row has a usable stored answer for the current definition.")
		_, _ = out.Write(buffer.Bytes())
		return
	}
	writeReliability(&buffer, "stored vectors (T=1.0000)", result.RawReliability, result.RawECE, result.RawBrier)
	fitted := "fitted"
	if !result.Fitted {
		fitted = "not fitted: the rows cannot constrain it"
	}
	writeReliability(&buffer, fmt.Sprintf("calibrated (T=%.4f, %s)", result.Temperature, fitted), result.Reliability, result.ECE, result.Brier)
	if result.StoredTemperature > 0 {
		fmt.Fprintf(&buffer, "  temperature the pack applies to this model now: %.4f\n", result.StoredTemperature)
	}
	_, _ = out.Write(buffer.Bytes())
}

func writeReliability(buffer *bytes.Buffer, title string, bins []reliabilityBin, ece, brier float64) {
	fmt.Fprintf(buffer, "  %s · ECE %.4f · Brier %.4f\n", title, ece, brier)
	fmt.Fprintf(buffer, "    %-11s %6s %15s %14s\n", "bin", "rows", "mean predicted", "observed rate")
	for _, bin := range bins {
		if bin.Count == 0 {
			fmt.Fprintf(buffer, "    %.1f - %.1f  %6d %15s %14s\n", bin.Lo, bin.Hi, 0, "-", "-")
			continue
		}
		fmt.Fprintf(buffer, "    %.1f - %.1f  %6d %15.3f %14.3f\n", bin.Lo, bin.Hi, bin.Count, bin.MeanPredicted, bin.ObservedRate)
	}
}
