package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/adguard-home/cli/internal/adguardcmd"
)

func main() {
	app, err := adguardcmd.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
