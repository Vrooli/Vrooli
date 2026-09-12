package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitProcess(1)
	}
}

var exitProcess = os.Exit

func run(args []string) error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	return app.Run(args)
}
