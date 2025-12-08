package requirements

import (
	"context"
	"flag"
	"fmt"
	"strings"

	reqservice "test-genie/internal/requirements"
)

func runSync(args []string) error {
	fs := flag.NewFlagSet("requirements sync", flag.ContinueOnError)
	dirFlag, scenarioFlag := parseCommonFlags(fs)
	commands := multiStringFlag(fs, "command", "Test command to record (repeatable)")

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

	svc := reqservice.NewService()
	name := scenarioNameFromDir(dir, *scenarioFlag)
	input := reqservice.SyncInput{
		ScenarioName:   name,
		ScenarioDir:    dir,
		CommandHistory: *commands,
	}

	if err := svc.Sync(context.Background(), input); err != nil {
		return fmt.Errorf("requirements sync failed: %w", err)
	}

	fmt.Printf("✅ Requirements synced for '%s'\n", name)
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
