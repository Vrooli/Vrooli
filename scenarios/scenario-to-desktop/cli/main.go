package main

import (
	"fmt"
	"os"

	"scenario-to-desktop/cli/pipeline"
)

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		// Don't print errors that were already printed by the command handler
		if !pipeline.IsAlreadyPrinted(err) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
