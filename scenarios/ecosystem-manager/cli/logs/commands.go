// Package logs provides the CLI command for viewing logs.
package logs

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands returns the logs command group.
func Commands() cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Logs",
		Commands: []cliapp.Command{
			{
				Name:        "logs",
				Description: "View ecosystem-manager API logs",
				Run:         cmdLogs,
			},
		},
	}
}

func cmdLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	follow := fs.Bool("f", false, "Follow log output")
	lines := fs.Int("n", 50, "Number of lines to show")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	logFile := home + "/.vrooli/logs/scenarios/ecosystem-manager/vrooli.develop.ecosystem-manager.start-api.log"

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Println("Log file not found. The ecosystem-manager may not be running.")
		fmt.Println("Start it with: vrooli scenario run ecosystem-manager")
		return nil
	}

	tailArgs := []string{"-n", strconv.Itoa(*lines)}
	if *follow {
		tailArgs = append(tailArgs, "-f")
	}
	tailArgs = append(tailArgs, logFile)

	cmd := exec.Command("tail", tailArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
