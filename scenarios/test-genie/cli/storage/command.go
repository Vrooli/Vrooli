package storage

import "fmt"

// Run dispatches storage maintenance subcommands.
func Run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "import-postgres":
		return runImportPostgres(args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie storage help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: test-genie storage <command>

Commands:
  import-postgres   One-time import from the legacy Postgres store into the embedded SQLite database

Run 'test-genie storage <command> -h' for command-specific options.`)
	return nil
}
