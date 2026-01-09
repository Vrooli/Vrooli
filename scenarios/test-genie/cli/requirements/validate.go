package requirements

import (
	"context"
	"flag"
	"fmt"
	"os"

	reqservice "test-genie/internal/requirements"
)

func runValidate(args []string) error {
	fs := flag.NewFlagSet("requirements validate", flag.ContinueOnError)
	quiet := fs.Bool("quiet", false, "Suppress success output (errors still print)")
	dirFlag, scenarioFlag := parseCommonFlags(fs)

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
	result, err := service.Validate(context.Background(), dir)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(result.Issues) == 0 {
		if !*quiet {
			fmt.Printf("✅ Requirements valid for '%s'\n", scenarioNameFromDir(dir, *scenarioFlag))
		}
		return nil
	}

	var errorsFound, warningsFound int
	for _, issue := range result.Issues {
		switch issue.Severity {
		case reqservice.SeverityError:
			errorsFound++
			fmt.Fprintf(os.Stderr, "❌ %s\n", issue.Error())
		case reqservice.SeverityWarning:
			warningsFound++
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", issue.Error())
		default:
			fmt.Fprintf(os.Stderr, "ℹ️  %s\n", issue.Error())
		}
	}

	if errorsFound > 0 {
		return fmt.Errorf("validation failed (%d error(s), %d warning(s))", errorsFound, warningsFound)
	}
	if !*quiet {
		fmt.Printf("✅ Requirements valid with %d warning(s)\n", warningsFound)
	}
	return nil
}
