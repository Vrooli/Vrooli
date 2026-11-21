package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// annotationGateMeasure is the registry measure every annotation reader shares:
// the Usage page, `usage annotations` and the friction digest quote a rate only
// from it, so the gate that licenses a rate travels with the rate.
const annotationGateMeasure = "usage.annotation_gate"

var annotationWindows = []string{"this_week", "last_7d", "last_30d", "this_month", "last_month", "this_quarter"}

// annotationsCommand is the inferred-annotation friction lens: gated labels over
// a bounded window, each with the model its proof is for.
func annotationsCommand(fetch measureFetch) cliapp.Command {
	return cliapp.Command{
		Name:        "annotations",
		NeedsAPI:    true,
		Description: "Which inferred labels are proven, for which model, and what do they say? [--window last_7d] [--json]",
		Usage:       "agent-manager usage annotations [--window last_7d] [--json]",
		HelpText:    "Reads `usage.annotation_gate`: per label whether it is proven, partial or unproven, the model and resource role the proof is for, whether its probabilities are calibrated or raw, the hold-out precision, recall, calibration error and gold size, and a rate only over observations from a proven model. An unproven label is evidence, never a rate. Labels about what a person wrote count human-written messages only. An empty window is unknown coverage, not zero friction. `agent-manager annotations show <run-id>` prints one run's labels and evidence.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "window", Description: "Window token (default last_7d)", Values: annotationWindows},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			window := strings.TrimSpace(ctx.Flag("window"))
			if window == "" {
				window = "last_7d"
			}
			body, err := fetch(ctx, annotationGateMeasure, map[string]string{"window": window})
			if err != nil {
				return fmt.Errorf("annotation gate: %w", err)
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var result struct {
				Fields []gateRow `json:"fields"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("decode annotation gate: %w", err)
			}
			renderAnnotationGate(ctx.Stdout(), result.Fields, window)
			return nil
		},
	}
}

// gateRow is one row of usage.annotation_gate; every cell is a string on the wire.
type gateRow map[string]string

func (r gateRow) int(key string) int {
	n, _ := strconv.Atoi(r[key])
	return n
}

func (r gateRow) float(key string) float64 {
	f, _ := strconv.ParseFloat(r[key], 64)
	return f
}

// gateLabel is a label row with the model rows that belong to it.
type gateLabel struct {
	gateRow
	models []gateRow
}

// groupGate splits the measure's rows into its coverage row and its labels.
func groupGate(rows []gateRow) (coverage gateRow, labels []*gateLabel) {
	byID := map[string]*gateLabel{}
	for _, row := range rows {
		switch row["kind"] {
		case "coverage":
			coverage = row
		case "label":
			label := &gateLabel{gateRow: row}
			byID[row["label"]] = label
			labels = append(labels, label)
		case "model":
			if label := byID[row["label"]]; label != nil {
				label.models = append(label.models, row)
			}
		}
	}
	return coverage, labels
}

func modelName(row gateRow) string {
	if row["resource_role"] == "" {
		return row["model"]
	}
	return row["model"] + "@" + row["resource_role"]
}

// proof is the sentence a rate rests on: the model, how its probabilities are
// read, and what its hold-out measured.
func (l *gateLabel) proof() string {
	name := l.gateRow["gate_model"]
	if role := l.gateRow["gate_role"]; role != "" {
		name += "@" + role
	}
	how := l.gateRow["calibration"]
	if how == "calibrated" {
		how += " T=" + l.gateRow["temperature"]
	}
	line := fmt.Sprintf("%s, %s · hold-out precision %.2f recall %.2f", name, how, l.float("precision"), l.float("recall"))
	if l.gateRow["ece"] != "" && l.gateRow["calibration"] != "rule" {
		line += fmt.Sprintf(" ECE %.3f", l.float("ece"))
	}
	return line + fmt.Sprintf(" · gold %d (%d positive)", l.int("gold"), l.int("gold_positives"))
}

// why says in words what keeps a label from being proven.
func (l *gateLabel) why() string {
	switch l.gateRow["gate_state"] {
	case "tuning_only":
		return fmt.Sprintf("%s measured on its tuning rows only; no hold-out", l.gateRow["gate_model"])
	case "insufficient_gold":
		return fmt.Sprintf("hold-out gold %d/%d, needs more labelled rows", l.int("gold"), l.int("min_gold"))
	case "unproven":
		return fmt.Sprintf("hold-out precision %.2f / recall %.2f below target %.2f / %.2f", l.float("precision"), l.float("recall"), l.float("target_precision"), l.float("target_recall"))
	case "legacy_unattributed":
		return "measured before gates were per model; the model is unknown"
	default:
		return fmt.Sprintf("no hold-out measurement (target %.2f / %.2f)", l.float("target_precision"), l.float("target_recall"))
	}
}

func (l *gateLabel) unit() string {
	if unit := l.gateRow["population_unit"]; unit != "" {
		return unit + "s"
	}
	if l.gateRow["scope"] == "message" {
		return "messages"
	}
	return "runs"
}

func renderAnnotationGate(out io.Writer, rows []gateRow, window string) {
	coverage, labels := groupGate(rows)
	pack := strings.TrimSpace(coverage["pack"] + " " + coverage["pack_version"])
	if pack == "" {
		pack = "unknown"
	}
	fmt.Fprintf(out, "Inferred annotations: pack %s · %s\n", pack, window)
	fmt.Fprintf(out, "Coverage: %.0f%% (%d of %d runs observed)\n", 100*gateRow(coverage).float("coverage"), gateRow(coverage).int("observed_runs"), gateRow(coverage).int("eligible_runs"))
	if coverage["human_messages"] != "" {
		fmt.Fprintf(out, "Human-written operator text: %d runs, %d messages; workflow and delegation briefs are not counted as a person's words\n", gateRow(coverage).int("human_text_runs"), gateRow(coverage).int("human_messages"))
	}
	if stale := gateRow(coverage).int("stale_runs"); stale > 0 {
		fmt.Fprintf(out, "%d runs not yet classified for operator text; the counts above leave them out until the recompute finishes\n", stale)
	}
	shown, silent := 0, 0
	for _, label := range labels {
		if label.int("observed") == 0 && label.gateRow["state"] != "proven" {
			silent++
			continue
		}
		if shown == 0 {
			fmt.Fprintln(out, "Labels (proven: a hold-out cleared the targets for that model; a rate counts only that model's answers):")
		}
		shown++
		state := label.gateRow["state"]
		line := fmt.Sprintf("  [%s] %s · observed %d/%d %s", state, label.gateRow["label"], label.int("observed"), label.int("eligible"), label.unit())
		if state == "partial" {
			line += fmt.Sprintf(" · %d from a proven model", label.int("proven_observed"))
		}
		fmt.Fprintln(out, line)
		if label.gateRow["gate_state"] == "proven" {
			fmt.Fprintln(out, "      proof: "+label.proof())
		}
		if rate := label.gateRow["rate"]; rate != "" {
			fmt.Fprintf(out, "      rate %.1f%% of %d proven observations\n", 100*label.float("rate"), label.int("proven_observed"))
		} else {
			fmt.Fprintln(out, "      no rate: "+label.why())
		}
		if len(label.models) > 1 || (len(label.models) == 1 && label.models[0]["proven"] != "true") {
			for _, model := range label.models {
				fmt.Fprintf(out, "      %s · %d observed · %s\n", modelName(model), model.int("observed"), model["gate_state"])
			}
		}
	}
	switch {
	case len(labels) == 0:
		fmt.Fprintln(out, "No annotations in this window: coverage is unknown, not zero friction.")
	case shown == 0:
		fmt.Fprintln(out, "No label has an observation in this window; unknown, not zero friction.")
	}
	if silent > 0 {
		fmt.Fprintf(out, "  %d label(s) have no observation in this window (unknown, not zero).\n", silent)
	}
	fmt.Fprintln(out, "Hints: agent-manager annotations show <run-id>  # one run's labels and evidence")
	fmt.Fprintln(out, "       agent-manager annotations series --label <message-label>  # a mood label over time")
}

// inferredAnnotationLines renders the gated labels the friction read already
// carries. Reading them from the response keeps one authoritative source and
// avoids a second network call that could disagree with the finding list. A rate
// is quoted only for a proven gate; every other label is named without one.
func inferredAnnotationLines(section *measurepb.UsageInferredAnnotations) []string {
	header := fmt.Sprintf("Inferred annotations: coverage %.0f%% (%d of %d runs observed)",
		100*section.GetCoverage(), section.GetObservedRuns(), section.GetEligibleRuns())
	if pack := strings.TrimSpace(strings.TrimSpace(section.GetPackName()) + " " + strings.TrimSpace(section.GetPackVersion())); pack != "" {
		header += " · pack " + pack
	}
	lines := []string{header}
	if note := strings.TrimSpace(section.GetNote()); note != "" {
		lines = append(lines, "   "+note)
	}
	if len(section.GetLabels()) == 0 {
		return append(lines, "   no observations in this window; unknown, not zero friction")
	}
	var unproven []string
	quoted, silent := 0, 0
	for _, label := range section.GetLabels() {
		if !label.GetFriction() {
			// The friction lens reports friction labels only; run.type and
			// run.outcome are useful facets but not friction signals.
			continue
		}
		if label.GetObserved() == 0 && label.GetGateState() != "proven" {
			silent++
			continue
		}
		if label.Rate == nil {
			unproven = append(unproven, fmt.Sprintf("%s (%s)", label.GetId(), label.GetGateState()))
			continue
		}
		quoted++
		lines = append(lines, fmt.Sprintf("   [proven] %s · %.1f%% of %d proven observations (of %d/%d observed) · %s",
			label.GetId(), 100*label.GetRate(), label.GetProvenObserved(), label.GetObserved(), label.GetEligible(), protoProof(label)))
	}
	if len(unproven) > 0 {
		lines = append(lines, "   no rate (unproven): "+strings.Join(unproven, ", "))
	}
	if quoted == 0 && len(unproven) == 0 && silent == 0 {
		lines = append(lines, "   no friction label has an observation in this window; unknown, not zero friction")
	}
	if silent > 0 {
		lines = append(lines, fmt.Sprintf("   %d label(s) have no observation in this window (unknown, not zero).", silent))
	}
	return append(lines, "   Every model's answers per label: agent-manager usage annotations")
}

// protoProof names the model a proven label's rate rests on, how its probabilities
// are read and its calibration error, from the friction read's own fields.
func protoProof(label *measurepb.UsageInferredAnnotation) string {
	name := label.GetGateModel()
	if role := label.GetGateRole(); role != "" {
		name += "@" + role
	}
	if name == "" {
		name = "model unknown"
	}
	how := label.GetCalibration()
	if how == "calibrated" {
		how += fmt.Sprintf(" T=%g", label.GetTemperature())
	}
	proof := name + ", " + how
	if label.GetCalibration() != "rule" {
		proof += fmt.Sprintf(", ECE %.3f", label.GetEce())
	}
	return proof
}
