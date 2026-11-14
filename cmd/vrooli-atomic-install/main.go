package main

import (
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
)

const (
	usageExitCode         = 2
	expectedArgumentCount = 3
)

func main() { os.Exit(run(os.Args, os.Stderr)) }
func run(args []string, stderr io.Writer) int {
	if len(args) != expectedArgumentCount && len(args) != expectedArgumentCount+1 {
		fmt.Fprintln(stderr, "usage: vrooli-atomic-install SOURCE DEST [EXPECTED_CURRENT_SHA256]")
		return usageExitCode
	}
	// A retained maintenance installer may run after source loss. Source-backed
	// go-run invocations retain their existing discovery path; a stamped image
	// must pin and validate its own compiled authority before any mutation.
	if buildinfo.RuntimeHomeSnapshotBase64 != "" {
		if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
			fmt.Fprintln(stderr, "compiled runtime-home authority unavailable:", err)
			return 1
		}
	}
	var err error
	if len(args) == expectedArgumentCount+1 {
		err = cliinstall.AtomicInstallExpected(args[1], args[2], args[3])
	} else {
		err = cliinstall.AtomicInstall(args[1], args[2])
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
