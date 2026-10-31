package main

import (
	"fmt"
)

// cmdRun dispatches one run subcommand through runCommands, the single table
// the discoverable command tree is built from.
func (a *App) cmdRun(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		return nil
	}
	for _, command := range a.runCommands() {
		if command.Name == args[0] && command.Run != nil {
			return command.Run(args[1:])
		}
	}
	return fmt.Errorf("unknown run subcommand: %s\n\nRun 'agent-manager run help' for usage", args[0])
}
