package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogExport(args []string) error {
	fs := flag.NewFlagSet("backlog export", flag.ContinueOnError)
	kinds := fs.String("kinds", "", "Comma-separated kinds to include (default: all)")
	statuses := fs.String("status", "", "Comma-separated statuses to include (default: non-archived)")
	names := fs.String("names", "", "Comma-separated kind/name pairs for specific items")
	priorityMax := fs.Int("priority-max", 0, "Only items with priority <= this value")
	tags := fs.String("tags", "", "Comma-separated tags (any match)")
	noPRD := fs.Bool("no-prd", false, "Exclude PRD content (smaller file)")
	includeArchived := fs.Bool("include-archived", false, "Include archived records")
	outPath := fs.String("out", "", "Output file path (default: backlog-export-YYYY-MM-DD.md)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	payload := map[string]any{}
	if strings.TrimSpace(*kinds) != "" {
		payload["kinds"] = strings.Split(*kinds, ",")
	}
	if strings.TrimSpace(*statuses) != "" {
		payload["statuses"] = strings.Split(*statuses, ",")
	}
	if strings.TrimSpace(*names) != "" {
		payload["names"] = strings.Split(*names, ",")
	}
	if *priorityMax > 0 {
		payload["priorityMax"] = *priorityMax
	}
	if strings.TrimSpace(*tags) != "" {
		payload["tags"] = strings.Split(*tags, ",")
	}
	if *noPRD {
		includePrd := false
		payload["includePrd"] = includePrd
	}
	if *includeArchived {
		payload["includeArchived"] = true
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/export", nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}

	if *jsonOut {
		// In JSON mode, output metadata about the export
		result := map[string]any{
			"size_bytes": len(body),
			"format":     "markdown",
		}
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}

	output := strings.TrimSpace(*outPath)
	if output == "" {
		output = "backlog-export-" + time.Now().Format("2006-01-02") + ".md"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("prepare output directory: %w", err)
	}
	if err := os.WriteFile(output, body, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	printSection("Result")
	fmt.Printf("  Exported backlog to %s (%d bytes)\n", output, len(body))
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "import", "--file", output),
		cliCommand("backlog", "import", "--file", output, "--apply"),
	})
	return nil
}

type exportCountExclusion struct {
	Filter string `json:"filter"`
	Rule   string `json:"rule"`
	Count  int    `json:"count"`
}

type exportCountMetadata struct {
	PreFilterTotal int                    `json:"pre_filter_total"`
	ItemsCount     int                    `json:"items_count"`
	Excluded       []exportCountExclusion `json:"excluded"`
}

type backlogCountReconciliation struct {
	RecordSurface struct {
		Basis            string                 `json:"basis"`
		OverviewTotal    int                    `json:"overview_total"`
		ExportPreFilter  int                    `json:"export_pre_filter_total"`
		ExportItems      int                    `json:"export_items_count"`
		ExportExcluded   []exportCountExclusion `json:"export_excluded"`
		TotalsAgree      bool                   `json:"totals_agree"`
		ArithmeticCloses bool                   `json:"arithmetic_closes"`
	} `json:"record_surface"`
	EventSurface struct {
		Basis                string `json:"basis"`
		EventCount           int64  `json:"event_count"`
		CreatedLast7Days     int    `json:"created_last_7_days"`
		CompletedLast7Days   int    `json:"completed_last_7_days"`
		DashboardBacklogSize int    `json:"dashboard_backlog_size"`
		CompletedAllTime     int    `json:"completed_all_time"`
	} `json:"event_surface"`
}

func (a *App) cmdBacklogReconcileCounts(args []string) error {
	fs := flag.NewFlagSet("backlog reconcile-counts", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	overviewBody, err := a.core.Get("/overview", nil)
	if err != nil {
		return err
	}
	var overview OverviewResponse
	if err := json.Unmarshal(overviewBody, &overview); err != nil {
		return fmt.Errorf("parse overview counts: %w", err)
	}
	exportBody, err := a.core.Request("POST", "/backlog/export", nil, json.RawMessage(`{}`))
	if err != nil {
		return err
	}
	metadata, err := parseExportCountMetadata(exportBody)
	if err != nil {
		return err
	}
	statsBody, err := a.fetchStats("")
	if err != nil {
		return err
	}
	var stats StatsResponse
	if err := json.Unmarshal(statsBody, &stats); err != nil {
		return fmt.Errorf("parse event-derived stats: %w", err)
	}

	var report backlogCountReconciliation
	report.RecordSurface.Basis = "current backlog records; overview includes archived, export applies the disclosed exclusion ledger"
	report.RecordSurface.OverviewTotal = overview.Summary.TotalItems
	report.RecordSurface.ExportPreFilter = metadata.PreFilterTotal
	report.RecordSurface.ExportItems = metadata.ItemsCount
	report.RecordSurface.ExportExcluded = metadata.Excluded
	report.RecordSurface.TotalsAgree = overview.Summary.TotalItems == metadata.PreFilterTotal
	excluded := 0
	for _, item := range metadata.Excluded {
		excluded += item.Count
	}
	report.RecordSurface.ArithmeticCloses = metadata.PreFilterTotal-excluded == metadata.ItemsCount
	report.EventSurface.Basis = "append-only lifecycle events; throughput windows are flows, not current-record totals"
	report.EventSurface.EventCount = stats.EventCount
	report.EventSurface.CreatedLast7Days = stats.Throughput.CreatedLast7Days
	report.EventSurface.CompletedLast7Days = stats.Throughput.CompletedLast7Days
	report.EventSurface.DashboardBacklogSize = stats.Dashboard.TotalBacklogSize
	report.EventSurface.CompletedAllTime = stats.Dashboard.TotalCompletedAllTime

	if *jsonOut {
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}
	printSection("Record-derived counts")
	fmt.Printf("  Basis: %s\n", report.RecordSurface.Basis)
	fmt.Printf("  Overview total: %d | Export pre-filter: %d | Exported: %d\n", report.RecordSurface.OverviewTotal, report.RecordSurface.ExportPreFilter, report.RecordSurface.ExportItems)
	for _, item := range report.RecordSurface.ExportExcluded {
		fmt.Printf("  Excluded by %s: %d (%s)\n", item.Filter, item.Count, item.Rule)
	}
	fmt.Printf("  Totals agree: %t | Arithmetic closes: %t\n", report.RecordSurface.TotalsAgree, report.RecordSurface.ArithmeticCloses)
	printSection("Event-derived counts")
	fmt.Printf("  Basis: %s\n", report.EventSurface.Basis)
	fmt.Printf("  Events: %d | Created 7d: %d | Completed 7d: %d\n", report.EventSurface.EventCount, report.EventSurface.CreatedLast7Days, report.EventSurface.CompletedLast7Days)
	return nil
}

func parseExportCountMetadata(body []byte) (exportCountMetadata, error) {
	var metadata exportCountMetadata
	var current *exportCountExclusion
	inFrontmatter := false
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(raw)
		if line == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		parseInt := func(prefix string) (int, error) {
			return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		}
		switch {
		case strings.HasPrefix(line, "pre_filter_total:"):
			value, parseErr := parseInt("pre_filter_total:")
			if parseErr != nil {
				return metadata, fmt.Errorf("parse export pre-filter total: %w", parseErr)
			}
			metadata.PreFilterTotal = value
		case strings.HasPrefix(line, "items_count:"):
			value, parseErr := parseInt("items_count:")
			if parseErr != nil {
				return metadata, fmt.Errorf("parse export item count: %w", parseErr)
			}
			metadata.ItemsCount = value
		case strings.HasPrefix(line, "- filter:"):
			metadata.Excluded = append(metadata.Excluded, exportCountExclusion{Filter: strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "- filter:")), `"`)})
			current = &metadata.Excluded[len(metadata.Excluded)-1]
		case current != nil && strings.HasPrefix(line, "rule:"):
			current.Rule = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "rule:")), `"`)
		case current != nil && strings.HasPrefix(line, "count:"):
			value, parseErr := parseInt("count:")
			if parseErr != nil {
				return metadata, fmt.Errorf("parse export exclusion count: %w", parseErr)
			}
			current.Count = value
		}
	}
	if len(metadata.Excluded) == 0 {
		return metadata, fmt.Errorf("export frontmatter does not disclose exclusions")
	}
	return metadata, nil
}

func (a *App) cmdBacklogImport(args []string) error {
	fs := flag.NewFlagSet("backlog import", flag.ContinueOnError)
	fileFlag := fs.String("file", "", "Import file path")
	apply := fs.Bool("apply", false, "Apply changes (default: dry-run only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *fileFlag); err != nil {
		return fmt.Errorf("usage: backlog import --file FILE [--apply] [--json]\n\n%s", err)
	}

	filePath := strings.TrimSpace(*fileFlag)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}
	if *apply {
		if err := writer.WriteField("apply", "true"); err != nil {
			return fmt.Errorf("write apply field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	body, err := a.requestMultipart("POST", "/backlog/import", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ImportBacklogResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	if response.DryRun {
		fmt.Println("  Mode: dry-run (use --apply to apply changes)")
	} else {
		fmt.Println("  Mode: applied")
	}
	fmt.Printf("  %s\n", response.Summary)

	if len(response.Changes) > 0 {
		printSection("Changes")
		for _, change := range response.Changes {
			fmt.Printf("  [%s] %s\n", change.Action, change.Item)
			for _, detail := range change.Details {
				fmt.Printf("    - %s\n", detail)
			}
		}
	}

	if len(response.Errors) > 0 {
		printSection("Errors")
		for _, e := range response.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if response.DryRun {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "import", "--file", filePath, "--apply"),
		})
	} else {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
		})
	}
	return nil
}
