package requirements

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	reqservice "test-genie/internal/requirements"
	reqtypes "test-genie/internal/requirements/types"
)

func runSync(args []string) error {
	fs := flag.NewFlagSet("requirements sync", flag.ContinueOnError)
	dirFlag, scenarioFlag := parseCommonFlags(fs)
	commands := multiStringFlag(fs, "command", "Test command to record (repeatable)")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	svc := reqservice.NewService()
	name := scenarioNameFromDir(dir, *scenarioFlag)
	input := reqservice.SyncInput{
		ScenarioName:   name,
		ScenarioDir:    dir,
		CommandHistory: *commands,
	}

	report, err := svc.Sync(context.Background(), input)
	if err != nil {
		return fmt.Errorf("requirements sync failed: %w", err)
	}

	fmt.Printf("✅ Requirements synced for '%s'\n", name)
	if report != nil {
		fmt.Printf("   Operational targets: %d/%d complete · Requirements: %d/%d complete\n",
			report.OT.Complete, report.OT.Total,
			report.Summary.ByDeclaredStatus[reqtypes.StatusComplete], report.Summary.Total)
	}
	return nil
}

// multiStringFlag collects repeated string flags into a slice.
func multiStringFlag(fs *flag.FlagSet, name, usage string) *[]string {
	var values []string
	fs.Func(name, usage, func(v string) error {
		values = append(values, strings.TrimSpace(v))
		return nil
	})
	return &values
}
