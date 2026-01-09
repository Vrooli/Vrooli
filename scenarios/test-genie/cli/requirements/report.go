package requirements

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	reqservice "test-genie/internal/requirements"
	"test-genie/internal/requirements/reporting"
)

func runReport(args []string) error {
	fs := flag.NewFlagSet("requirements report", flag.ContinueOnError)
	format := fs.String("format", "json", "Report format: json|markdown|trace|summary")
	includePending := fs.Bool("include-pending", false, "Include pending requirements")
	outputPath := fs.String("output", "", "Write output to file instead of stdout")
	includeValidations := fs.Bool("include-validations", true, "Include validation details")
	phase := fs.String("phase", "", "Filter validations to a specific phase (phase-inspect)")
	dirFlag, _ := parseCommonFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	service := reqservice.NewService()
	opts := reporting.DefaultOptions()
	opts.IncludePending = *includePending
	opts.IncludeValidations = *includeValidations
	if *phase != "" {
		opts.Phase = *phase
	}

	switch normalizeFormat(*format) {
	case "json":
		opts.Format = reporting.FormatJSON
	case "markdown":
		opts.Format = reporting.FormatMarkdown
	case "trace":
		opts.Format = reporting.FormatTrace
	case "summary":
		opts.Format = reporting.FormatSummary
	default:
		return fmt.Errorf("unsupported format: %s", *format)
	}

	ctx := context.Background()
	var out *os.File
	if *outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		f, err := os.Create(*outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	if err := service.Report(ctx, dir, opts, out); err != nil {
		return fmt.Errorf("report failed: %w", err)
	}

	if *outputPath != "" {
		rel := *outputPath
		if abs, _ := filepathAbs(*outputPath); abs != "" {
			if relPath, err := filepathRel(dir, abs); err == nil {
				rel = relPath
			}
		}
		fmt.Fprintf(os.Stderr, "[requirements] wrote report: %s\n", rel)
	}
	return nil
}

func normalizeFormat(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return "json"
	case "md", "markdown":
		return "markdown"
	case "trace":
		return "trace"
	case "summary":
		return "summary"
	default:
		return s
	}
}

// filepathRel is extracted for testability.
var filepathRel = func(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}
