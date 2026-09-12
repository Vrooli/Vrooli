package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/repo-contract-go/cliinvoke"
)

func (a *App) runLoop(args []string) error {
	loopBinary, err := support.ResolveLoopBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(loopBinary, translateLoopArgs(args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func translateLoopArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--interval-seconds="):
			out = append(out, "--interval="+strings.TrimPrefix(arg, "--interval-seconds="))
		case arg == "--interval-seconds" && i+1 < len(args):
			i++
			out = append(out, "--interval", args[i])
		default:
			out = append(out, arg)
		}
	}
	return out
}

func (a *App) diagnosePort(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: vrooli-autoheal diagnose-port <port> [scenario]")
	}
	port := strings.TrimSpace(args[0])
	scenario := ""
	if len(args) > 1 {
		scenario = strings.TrimSpace(args[1])
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be an integer")
	}

	home, _ := os.UserHomeDir()
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{RuntimeHome: home})
	if err != nil {
		return err
	}
	return cliinvoke.Run(context.Background(), cliinvoke.Invocation{
		Binary: binary,
		Args:   cliinvoke.DiagnosePort(port, scenario),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}).Error()
}
