package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/cliinstall"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: vrooli-atomic-install SOURCE DEST")
		os.Exit(2)
	}
	if err := cliinstall.AtomicInstall(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
