package main

import (
	"os"

	"github.com/vrooli/cli-core/entrypoint"
)

func main() {
	entrypoint.RunAndExit(NewApp, os.Args[1:], os.Stderr, os.Exit)
}
