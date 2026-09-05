//go:build darwin

package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runAsInvokingUserInSession(ctx context.Context, name string, args []string, options IdentityCommandOptions) error {
	return runIdentityCommand(ctx, name, args, nil, options)
}

func runAsInvokingUserInSessionWithInput(ctx context.Context, name string, args []string, input []byte, options IdentityCommandOptions) error {
	return runIdentityCommand(ctx, name, args, input, options)
}

func runIdentityCommand(ctx context.Context, name string, args []string, input []byte, options IdentityCommandOptions) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("platform: identity command is empty")
	}
	uid := strconv.Itoa(os.Getuid())
	command := "launchctl"
	argv := append([]string{"asuser", uid, name}, args...)
	if os.Geteuid() == 0 {
		operatorUID := strings.TrimSpace(os.Getenv("SUDO_UID"))
		if operatorUID == "" {
			// A real root login is already the invoking identity. There is no
			// lower-privileged launchd session to enter.
			command = name
			argv = args
			goto execute
		}
		if _, err := strconv.Atoi(operatorUID); err != nil {
			return fmt.Errorf("platform: resolve invoking-user uid: %w", err)
		}
		uid = operatorUID
		argv = append([]string{"-u", strings.TrimSpace(os.Getenv("SUDO_USER")), "-H", "--", "launchctl", "asuser", uid, name}, args...)
		command = "sudo"
	}

execute:
	cmd := exec.CommandContext(ctx, command, argv...)
	cmd.Dir = options.Dir
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	} else {
		cmd.Stdin = options.Stdin
	}
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	return cmd.Run()
}
