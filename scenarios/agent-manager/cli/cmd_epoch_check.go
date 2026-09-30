// This file implements the generic epoch step-back check (large-effort-orchestration §4.4).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// exitStepBack is the distinct status the epoch check returns when a
// step-back trigger fires, so a worker or orchestrator script can branch on it.
const exitStepBack = 3

// epochFile is the parsed brief, directives and slice log of one epoch.
type epochFile struct {
	Fields     map[string]string
	Directives []epochDirective
	Log        []sliceLine
	Accepted   string
	Malformed  []string
}

type epochDirective struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// sliceLine is one work unit:
// <ISO time> | <what changed> | exit metric=<v> | net runtime lines=<±n> | net test lines=<±n> | ack=<text naming D<n> ids, or ->
type sliceLine struct {
	At           time.Time `json:"at"`
	What         string    `json:"what"`
	Metric       string    `json:"metric"`
	RuntimeLines int       `json:"net_runtime_lines"`
	TestLines    int       `json:"net_test_lines"`
	Acks         []string  `json:"acks"`
}

// epochThresholds are the step-back trigger settings; defaults follow the skill.
type epochThresholds struct {
	FlatlineK      int
	OverrunFactor  float64
	RefactorGrowth int
	FeatureFactor  float64
	WallClock      time.Duration
	SpendThreshold float64
}

type epochTrigger struct {
	Name   string `json:"name"`
	Fired  bool   `json:"fired"`
	Review bool   `json:"review_only,omitempty"`
	Detail string `json:"detail"`
}

type epochReport struct {
	File              string         `json:"file"`
	Kind              string         `json:"kind"`
	WorkUnits         int            `json:"work_units"`
	Estimate          int            `json:"estimate"`
	MetricHistory     []string       `json:"metric_history"`
	NetRuntimeLines   int            `json:"net_runtime_lines"`
	NetTestLines      int            `json:"net_test_lines"`
	OpenDirectives    []string       `json:"open_directives"`
	WeightedTokens    *float64       `json:"weighted_tokens,omitempty"`
	Accepted          string         `json:"accepted,omitempty"`
	Triggers          []epochTrigger `json:"triggers"`
	StepBack          []string       `json:"step_back"`
	Review            []string       `json:"review"`
	MalformedLogLines []string       `json:"malformed_log_lines,omitempty"`
	Woken             []string       `json:"woken,omitempty"`
}

var (
	epochFieldPattern     = regexp.MustCompile(`^- ([A-Za-z][A-Za-z ]*): (.*)$`)
	epochDirectivePattern = regexp.MustCompile(`^- (D\d+) (\S+) (.+)$`)
	epochLeadingInt       = regexp.MustCompile(`^[+-]?\d+`)
	epochDirectiveID      = regexp.MustCompile(`\bD\d+\b`)
)

// parseEpochFile reads the format defined in large-effort-orchestration §4.2.
func parseEpochFile(r io.Reader) (*epochFile, error) {
	file := &epochFile{Fields: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			continue
		}
		if trimmed == "" {
			continue
		}
		switch {
		case section == "":
			if m := epochFieldPattern.FindStringSubmatch(trimmed); m != nil {
				file.Fields[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}
		case section == "directives":
			if m := epochDirectivePattern.FindStringSubmatch(trimmed); m != nil {
				file.Directives = append(file.Directives, epochDirective{ID: m[1], Text: m[3]})
			}
		case strings.HasPrefix(section, "slice log"):
			if strings.HasPrefix(trimmed, "ACCEPTED ") {
				file.Accepted = trimmed
				continue
			}
			entry, err := parseSliceLine(trimmed)
			if err != nil {
				file.Malformed = append(file.Malformed, trimmed)
				continue
			}
			file.Log = append(file.Log, entry)
		}
	}
	return file, scanner.Err()
}

// parseSliceLine reads "<ISO time> | <what changed> | key=value | ...". The
// time, the change and an exit metric are required. Line counts default to
// zero, ack is free text whose directive IDs (D<n>) are kept, unknown
// key=value fields are ignored, and extra free text joins the change.
func parseSliceLine(line string) (sliceLine, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		return sliceLine{}, fmt.Errorf("want <time> | <what changed> | exit metric=<value>")
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
	if err != nil {
		return sliceLine{}, err
	}
	entry := sliceLine{At: at, What: strings.TrimSpace(parts[1])}
	hasMetric := false
	for _, part := range parts[2:] {
		part = strings.TrimSpace(part)
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			if part != "" {
				entry.What += " | " + part
			}
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "exit metric", "metric":
			entry.Metric, hasMetric = value, true
		case "net runtime lines", "runtime lines":
			entry.RuntimeLines = leadingNumber(value)
		case "net test lines", "test lines":
			entry.TestLines = leadingNumber(value)
		case "ack", "acks":
			entry.Acks = epochDirectiveID.FindAllString(value, -1)
		}
	}
	if !hasMetric {
		return sliceLine{}, fmt.Errorf("want exit metric=<value>")
	}
	return entry, nil
}

// leadingNumber reads "12 work units" or "2000 lines" as 12 or 2000.
func leadingNumber(raw string) int {
	n, _ := strconv.Atoi(epochLeadingInt.FindString(strings.TrimSpace(raw)))
	return n
}

// evaluateEpoch applies the step-back triggers. weighted is nil when spend was
// not measured; the spend trigger then reports that it was not evaluated.
func evaluateEpoch(path string, file *epochFile, limits epochThresholds, now time.Time, weighted *float64) *epochReport {
	report := &epochReport{File: path, Kind: strings.ToLower(file.Fields["kind"]), WorkUnits: len(file.Log),
		Estimate: leadingNumber(file.Fields["estimate"]), Accepted: file.Accepted, WeightedTokens: weighted,
		MetricHistory: []string{}, OpenDirectives: []string{}, StepBack: []string{}, Review: []string{}, MalformedLogLines: file.Malformed}
	acked := map[string]bool{}
	for _, entry := range file.Log {
		report.MetricHistory = append(report.MetricHistory, entry.Metric)
		report.NetRuntimeLines += entry.RuntimeLines
		report.NetTestLines += entry.TestLines
		for _, id := range entry.Acks {
			acked[id] = true
		}
	}
	for _, directive := range file.Directives {
		if !acked[directive.ID] {
			report.OpenDirectives = append(report.OpenDirectives, directive.ID)
		}
	}
	add := func(trigger epochTrigger) {
		report.Triggers = append(report.Triggers, trigger)
		if trigger.Fired && trigger.Review {
			report.Review = append(report.Review, trigger.Name+": "+trigger.Detail)
		} else if trigger.Fired {
			report.StepBack = append(report.StepBack, trigger.Name+": "+trigger.Detail)
		}
	}

	flat := epochTrigger{Name: "flatline", Detail: fmt.Sprintf("needs %d log lines, have %d", limits.FlatlineK, len(file.Log))}
	if limits.FlatlineK > 0 && len(file.Log) >= limits.FlatlineK {
		tail := file.Log[len(file.Log)-limits.FlatlineK:]
		flat.Fired = true
		for _, entry := range tail[1:] {
			if entry.Metric != tail[0].Metric {
				flat.Fired = false
			}
		}
		flat.Detail = fmt.Sprintf("exit metric over the last %d lines: %s", limits.FlatlineK, map[bool]string{true: "unchanged at " + tail[0].Metric, false: "moved"}[flat.Fired])
	}
	add(flat)

	overrun := epochTrigger{Name: "overrun", Detail: "no estimate in the brief"}
	if report.Estimate > 0 {
		limit := limits.OverrunFactor * float64(report.Estimate)
		overrun.Fired = float64(len(file.Log)) > limit
		overrun.Detail = fmt.Sprintf("%d log lines against %.0f (%.1fx estimate %d)", len(file.Log), limit, limits.OverrunFactor, report.Estimate)
	}
	add(overrun)

	growth := epochTrigger{Name: "growth", Detail: "kind is not refactor or feature"}
	switch report.Kind {
	case "refactor":
		growth.Fired = report.NetRuntimeLines > limits.RefactorGrowth
		growth.Detail = fmt.Sprintf("net runtime lines %+d against +%d", report.NetRuntimeLines, limits.RefactorGrowth)
	case "feature":
		expected := leadingNumber(file.Fields["expected size"])
		if expected > 0 {
			limit := limits.FeatureFactor * float64(expected)
			growth.Fired = float64(report.NetRuntimeLines) > limit
			growth.Detail = fmt.Sprintf("net runtime lines %+d against %.0f (%.1fx expected %d)", report.NetRuntimeLines, limit, limits.FeatureFactor, expected)
		} else {
			growth.Detail = "feature epoch has no expected size"
		}
	}
	add(growth)

	spend := epochTrigger{Name: "spend", Detail: "not evaluated: set --spend-threshold after calibrating on 3 epochs"}
	if limits.SpendThreshold > 0 && weighted == nil {
		spend.Detail = "not evaluated: no worker runs to measure"
	} else if limits.SpendThreshold > 0 {
		spend.Fired = *weighted > limits.SpendThreshold
		spend.Detail = fmt.Sprintf("weighted non-cache tokens %.0f against %.0f", *weighted, limits.SpendThreshold)
	}
	add(spend)

	wall := epochTrigger{Name: "wall-clock", Review: true, Detail: "no Started time in the brief"}
	if started, err := time.Parse(time.RFC3339, file.Fields["started"]); err == nil {
		elapsed := now.Sub(started)
		wall.Fired = limits.WallClock > 0 && elapsed > limits.WallClock
		wall.Detail = fmt.Sprintf("%s since start against %s (prompts a review; never forces a stop)", elapsed.Round(time.Minute), limits.WallClock)
	}
	add(wall)
	return report
}

// effortEpochCheck is `agent-manager effort epoch-check <epoch-file>`.
func (a *App) effortEpochCheck(args []string) error {
	fs := flag.NewFlagSet("effort epoch-check", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	k := fs.Int("k", 10, "Flatline: log lines over which an unchanged exit metric fires")
	overrun := fs.Float64("overrun", 2, "Overrun: log lines above this multiple of the estimate fire")
	growth := fs.Int("growth", 300, "Refactor epochs: cumulative net runtime lines above this fire")
	featureFactor := fs.Float64("feature-factor", 1.5, "Feature epochs: net runtime lines above this multiple of the expected size fire")
	wallClock := fs.Duration("wall-clock", 24*time.Hour, "Elapsed time after which a review is prompted (never a stop)")
	spendThreshold := fs.Float64("spend-threshold", 0, "Weighted non-cache tokens of the epoch's workers above which spend fires (0 = not evaluated)")
	runs := fs.String("runs", "", "Comma-separated worker run IDs for the spend trigger (default: the brief's Workers field)")
	wakeKey := fs.String("wake-key", "", "On a fired step-back, wake the run parked on this key (the orchestrator's run ID)")
	var path string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("usage: agent-manager effort epoch-check <epoch-file> [--wake-key <run-id>] [--runs ids] [--spend-threshold n] [--json]")
	}
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	file, err := parseEpochFile(handle)
	handle.Close()
	if err != nil {
		return err
	}
	workerIDs := *runs
	if workerIDs == "" {
		workerIDs = file.Fields["workers"]
	}
	var weighted *float64
	if ids := strings.FieldsFunc(workerIDs, func(r rune) bool { return r == ',' || r == ' ' }); *spendThreshold > 0 && len(ids) > 0 {
		spend, err := a.weightedTokens(ids, defaultTierWeights)
		if err != nil {
			return fmt.Errorf("worker spend: %w", err)
		}
		weighted = &spend.TotalWeighted
	}
	limits := epochThresholds{FlatlineK: *k, OverrunFactor: *overrun, RefactorGrowth: *growth, FeatureFactor: *featureFactor, WallClock: *wallClock, SpendThreshold: *spendThreshold}
	report := evaluateEpoch(path, file, limits, time.Now().UTC(), weighted)
	if len(report.StepBack) > 0 && *wakeKey != "" {
		woken, err := a.services.Runs.WakeByKey(&domainpb.WakeParkedRunsRequest{Producer: "children", Key: *wakeKey, Result: "STEP_BACK " + strings.Join(report.StepBack, "; ") + " (epoch " + path + ")"})
		if err != nil {
			fmt.Fprintf(os.Stderr, "friction wake failed: %v\n", err)
		}
		report.Woken = woken
	}
	if *jsonOut {
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
	} else {
		printEpochReport(report)
	}
	if len(report.StepBack) > 0 {
		return exitCodeError{code: exitStepBack}
	}
	return nil
}

func printEpochReport(report *epochReport) {
	fmt.Printf("Epoch %s (%s): %d work units, estimate %d\n", report.File, report.Kind, report.WorkUnits, report.Estimate)
	fmt.Printf("Net lines: runtime %+d, test %+d\n", report.NetRuntimeLines, report.NetTestLines)
	fmt.Printf("Metric history: %s\n", strings.Join(report.MetricHistory, " → "))
	if len(report.OpenDirectives) > 0 {
		fmt.Printf("Open directives: %s\n", strings.Join(report.OpenDirectives, ", "))
	} else {
		fmt.Println("Open directives: none")
	}
	if report.WeightedTokens != nil {
		fmt.Printf("Weighted non-cache tokens: %.0f\n", *report.WeightedTokens)
	}
	for _, trigger := range report.Triggers {
		state := "not fired"
		if trigger.Fired {
			state = "FIRED"
		}
		fmt.Printf("  %-10s %-9s %s\n", trigger.Name, state, trigger.Detail)
	}
	for _, line := range report.MalformedLogLines {
		fmt.Printf("Malformed slice-log line (not counted): %s\n", line)
	}
	if report.Accepted != "" {
		fmt.Println(report.Accepted)
	}
	for _, reason := range report.Review {
		fmt.Printf("REVIEW %s\n", reason)
	}
	if len(report.StepBack) > 0 {
		fmt.Printf("STEP_BACK %s\n", strings.Join(report.StepBack, "; "))
		if len(report.Woken) > 0 {
			fmt.Printf("Woke %s\n", strings.Join(report.Woken, ", "))
		}
	}
}
