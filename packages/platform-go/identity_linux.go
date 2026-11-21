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
	command, argv, err := identityCommandForUser(os.Geteuid(), os.Getenv, user.Lookup, name, args)
	if err != nil {
		return err
	}
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

// identityCommandForUser builds the invoking-user command without executing
// it. Keeping identity selection pure lets fixtures verify root and sudo-via-
// root routing without changing credentials or invoking sudo.
func identityCommandForUser(euid int, getenv func(string) string, lookup func(string) (*user.User, error), name string, args []string) (string, []string, error) {
	argv := append([]string(nil), args...)
	if euid != 0 {
		return name, argv, nil
	}
	operator := strings.TrimSpace(getenv("SUDO_USER"))
	if operator == "" {
		// A real root login has no lower-privileged invoking user. In that
		// case root is the operator and the command may run directly.
		operator = "root"
	}
	if operator == "root" {
		return name, argv, nil
	}
	uid := strings.TrimSpace(getenv("SUDO_UID"))
	if uid == "" {
		if account, err := lookup(operator); err == nil {
			uid = account.Uid
		}
	}
	if _, err := strconv.Atoi(uid); err != nil {
		return "", nil, fmt.Errorf("platform: resolve invoking-user uid: %w", err)
	}
	runtimeDir := "/run/user/" + uid
	argv = append([]string{"-u", operator, "-H", "--", "env", "XDG_RUNTIME_DIR=" + runtimeDir, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus", name}, argv...)
	return "sudo", argv, nil
}
