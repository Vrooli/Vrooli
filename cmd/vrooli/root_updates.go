package main

import (
	"context"
	"io"

	"github.com/vrooli/vrooli/internal/cli/runtimehandlers"
)

func rootUpdateCommand(args []string) bool {
	return len(args) >= 2 && args[0] == "runtime" && args[1] == "update"
}

func runRootUpdateReadCommand(args []string, stdout, stderr io.Writer) (bool, int) {
	return runtimehandlers.RunRootUpdateReadCommand(args, stdout, stderr)
}

func runRootUpdateEffectCommand(args []string, stdout, stderr io.Writer) (bool, int) {
	return runtimehandlers.RunRootUpdateEffectCommand(context.Background(), args, stdout, stderr, resolveSourceRootFn)
}
