package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/cliinstall"
)

const (
	mndMainNumberValue2 = 2
	mndMainNumberValue3 = 3
)

func main() {
	if len(os.Args) != mndMainNumberValue3 {
		fmt.Fprintln(os.Stderr, "usage: vrooli-atomic-install SOURCE DEST")
		os.Exit(mndMainNumberValue2)
	}
	if err := cliinstall.AtomicInstall(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
