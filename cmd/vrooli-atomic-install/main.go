package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/cliinstall"
)

const (
	usageExitCode         = 2
	expectedArgumentCount = 3
)

func main() {
	if len(os.Args) != expectedArgumentCount {
		fmt.Fprintln(os.Stderr, "usage: vrooli-atomic-install SOURCE DEST")
		os.Exit(usageExitCode)
	}
	if err := cliinstall.AtomicInstall(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
