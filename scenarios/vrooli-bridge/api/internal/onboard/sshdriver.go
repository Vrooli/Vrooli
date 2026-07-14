package onboard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"vrooli-bridge/internal/onboard/ssh"
)

// sshDriver is the production SSHDriver: it drives the phase-1 SSH capability
// (first touch), stages the bootstrap script with SCP, and runs it remotely
// while streaming its VBOOTSTRAP markers. It is the single place the onboard
// domain reaches the ssh package.
type sshDriver struct {
	svc        *ssh.Service
	scriptPath string
	scpRunner  ssh.SCPRunner
}

// NewSSHDriver constructs the production SSHDriver over an ssh.Service and the
// local path to the bootstrap script that gets copied to each node.
func NewSSHDriver(svc *ssh.Service, scriptPath string) SSHDriver {
	return &sshDriver{svc: svc, scriptPath: scriptPath, scpRunner: ssh.ExecSCPRunner{}}
}

var _ SSHDriver = (*sshDriver)(nil)

func (d *sshDriver) FirstTouch(ctx context.Context, p FirstTouchParams) (Conn, error) {
	res, err := d.svc.FirstTouch(ctx, ssh.FirstTouchRequest{
		Host: p.Host, Port: p.Port, User: p.User, Password: p.Password,
	})
	if err != nil {
		return Conn{}, err
	}
	if !res.OK {
		return Conn{}, fmt.Errorf("passwordless SSH not established: %s", res.Message)
	}
	return Conn{Host: res.Host, Port: res.Port, User: res.User, KeyPath: res.KeyPath}, nil
}

func (d *sshDriver) PushScript(ctx context.Context, conn Conn) (string, error) {
	cfg := d.config(conn)
	remotePath, err := remoteScriptPath()
	if err != nil {
		return "", err
	}
	if err := d.scpRunner.Copy(ctx, cfg, d.scriptPath, remotePath, ssh.DefaultSCPOptions()); err != nil {
		return "", err
	}
	return remotePath, nil
}

func (d *sshDriver) RunBootstrap(ctx context.Context, p RunParams, onMarker func(Marker)) (BootstrapResult, error) {
	cfg := d.config(p.Conn)

	// The pairing code rides stdin (env-only, never argv/logs): the remote shell
	// reads one line into BRIDGE_PAIRING_CODE, exports it, then execs the script.
	// The script's flags carry NO secret.
	remoteCmd := "IFS= read -r __vb_code; export BRIDGE_PAIRING_CODE=\"$__vb_code\"; unset __vb_code; exec bash " +
		shellQuote(p.RemotePath) + " " + quoteArgs(p.Args)

	// stdin = code + newline. Built here and zeroed on return so the only lasting
	// copy of the secret is the caller's, which the orchestrator wipes too.
	stdin := make([]byte, 0, len(p.PairingCode)+1)
	stdin = append(stdin, p.PairingCode...)
	stdin = append(stdin, '\n')
	defer zeroBytes(stdin)

	res, err := d.svc.RunStreaming(ctx, cfg, remoteCmd, ssh.StreamOptions{
		Run:   bootstrapRunOptions(),
		Stdin: stdin,
		OnStdoutLine: func(line string) {
			if m, ok := parseMarker(line); ok {
				onMarker(m)
			}
		},
	})
	if err != nil {
		return BootstrapResult{ExitCode: res.ExitCode}, err
	}
	return BootstrapResult{ExitCode: res.ExitCode}, nil
}

// config builds the ssh.Config for a resolved Conn, pinned to the bridge-owned
// known_hosts the first touch populated.
func (d *sshDriver) config(conn Conn) ssh.Config {
	return ssh.NewConfig(conn.Host, conn.Port, conn.User, conn.KeyPath, d.svc.KnownHostsPath())
}

// bootstrapRunOptions extends the default run options with a generous command
// timeout — a full bootstrap (clone + build + setup) is long-running.
//
// ControlMaster is disabled: connection multiplexing spawns a background master
// ssh process that inherits the command's stdout fd, so the streaming reader
// only sees EOF once that master exits (ControlPersist seconds later). A single
// long-lived bootstrap exec gains nothing from multiplexing and must not stall
// on the master's lifetime, so we run a plain, non-multiplexed connection.
func bootstrapRunOptions() ssh.RunOptions {
	o := ssh.DefaultRunOptions()
	o.ControlMaster = false
	o.CommandTimeout = 60 * time.Minute
	o.MaxOutputBytes = 4 * 1024 * 1024
	return o
}

// remoteScriptPath returns a unique, non-predictable /tmp path for the staged
// bootstrap script on the node.
func remoteScriptPath() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate remote script suffix: %w", err)
	}
	return "/tmp/vrooli-bridge-bootstrap-" + hex.EncodeToString(b[:]) + ".sh", nil
}

// shellQuote single-quotes a string for safe use as one remote shell argument.
// Duplicated from the ssh package's unexported helper per duplicate-before-extract.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// quoteArgs shell-quotes each arg and joins them with spaces.
func quoteArgs(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = shellQuote(a)
	}
	return strings.Join(out, " ")
}
