//go:build linux

package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
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
	argv := append([]string(nil), args...)
	command := name
	if os.Geteuid() == 0 {
		operator := strings.TrimSpace(os.Getenv("SUDO_USER"))
		if operator == "" {
			// A real root login has no lower-privileged invoking user. In
			// that case root is the operator and the command may run directly;
			// only sudo-via-root needs a session transition.
			operator = "root"
		}
		if operator == "root" {
			goto execute
		}
		uid := strings.TrimSpace(os.Getenv("SUDO_UID"))
		if uid == "" {
			if account, err := user.Lookup(operator); err == nil {
				uid = account.Uid
			}
		}
		if _, err := strconv.Atoi(uid); err != nil {
			return fmt.Errorf("platform: resolve invoking-user uid: %w", err)
		}
		runtimeDir := "/run/user/" + uid
		command = "sudo"
		argv = append([]string{"-u", operator, "-H", "--", "env", "XDG_RUNTIME_DIR=" + runtimeDir, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus", name}, argv...)
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
