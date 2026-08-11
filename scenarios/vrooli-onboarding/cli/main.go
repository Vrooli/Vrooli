package main

import "os"

func main() {
	os.Exit(runExitCode(os.Args[1:]))
}

func runMain(args []string) error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	return app.Run(args)
}

func runExitCode(args []string) int {
	if err := runMain(args); err != nil {
		if coded, ok := err.(interface{ ExitCode() int }); ok {
			return coded.ExitCode()
		}
		return 1
	}
	return 0
}
