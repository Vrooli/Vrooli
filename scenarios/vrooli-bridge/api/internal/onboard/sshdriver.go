package onboard

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vrooli-bridge/internal/onboard/ssh"
)

// syncDestMarker prefixes the single stdout line the remote sync command emits to
// report the concrete directory the tree landed in (the operator's DestDir, or the
// node-resolved default). The driver parses it back to fill ResolvedDestDir.
const syncDestMarker = "VBSYNCDEST="

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
		ProvisionSudo: p.ProvisionSudo,
	})
	if err != nil {
		return Conn{}, err
	}
	if !res.OK {
		return Conn{}, fmt.Errorf("passwordless SSH not established: %s", res.Message)
	}
	return Conn{Host: res.Host, Port: res.Port, User: res.User, KeyPath: res.KeyPath, SudoState: string(res.SudoState)}, nil
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

// SyncTree ships the control plane's working tree to the node by piping a tar
// archive of p.Files (relative to p.RepoDir) into a remote `tar -xf -`. The remote
// command resolves the destination ($HOME/vrooli by default, or the explicit
// DestDir), creates it, reports it back on stdout, then extracts. Filenames with
// spaces/newlines survive because the tar format encodes names — nothing is
// shell-word-split. The tar is streamed (never buffered whole in memory) and its
// byte count is measured for the step detail.
func (d *sshDriver) SyncTree(ctx context.Context, p SyncParams) (SyncResult, error) {
	cfg := d.config(p.Conn)
	remoteCmd := buildSyncRemoteCommand(p.DestDir)

	// Produce the tar on a background goroutine writing into a pipe; the ssh
	// command reads the pipe as its stdin. A counting reader measures what actually
	// crossed the wire.
	pr, pw := io.Pipe()
	counter := &countingReader{r: pr}
	go func() {
		pw.CloseWithError(writeTarStream(pw, p.RepoDir, p.Files))
	}()

	var resolvedDest string
	res, err := d.svc.RunStreaming(ctx, cfg, remoteCmd, ssh.StreamOptions{
		Run:         syncRunOptions(),
		StdinReader: counter,
		OnStdoutLine: func(line string) {
			if v, ok := strings.CutPrefix(line, syncDestMarker); ok {
				resolvedDest = strings.TrimSpace(v)
			}
		},
	})
	if err != nil {
		return SyncResult{}, err
	}
	if res.ExitCode != 0 {
		return SyncResult{}, fmt.Errorf("remote tar extract failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	if resolvedDest == "" {
		return SyncResult{}, fmt.Errorf("remote sync did not report a destination directory")
	}
	return SyncResult{BytesTransferred: counter.n, ResolvedDestDir: resolvedDest}, nil
}

// buildSyncRemoteCommand renders the remote shell that resolves the destination,
// creates it, reports it (the VBSYNCDEST marker), then extracts the incoming tar.
// An explicit destDir is shell-quoted; an empty one defaults to $HOME/vrooli,
// resolved on the node (the control plane cannot know the node's home).
func buildSyncRemoteCommand(destDir string) string {
	var assign string
	if d := strings.TrimSpace(destDir); d != "" {
		assign = "dest=" + shellQuote(d)
	} else {
		assign = `dest="$HOME/vrooli"`
	}
	return assign + `; mkdir -p "$dest" && printf '` + syncDestMarker + `%s\n' "$dest" && tar -xf - -C "$dest"`
}

// writeTarStream writes a tar archive of the repo-relative files (rooted at
// repoDir) to w. Regular files carry their content; symlinks are preserved as
// links (git ls-files can list them). A path that vanished mid-ship is skipped
// rather than aborting the whole onboarding.
func writeTarStream(w io.Writer, repoDir string, files []string) error {
	tw := tar.NewWriter(w)
	for _, rel := range files {
		abs := filepath.Join(repoDir, rel)
		info, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		var link string
		if info.Mode()&os.ModeSymlink != 0 {
			if link, err = os.Readlink(abs); err != nil {
				return fmt.Errorf("readlink %s: %w", rel, err)
			}
		}
		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return fmt.Errorf("tar header %s: %w", rel, err)
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header %s: %w", rel, err)
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(abs)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("open %s: %w", rel, err)
			}
			_, cErr := io.Copy(tw, f)
			_ = f.Close()
			if cErr != nil {
				return fmt.Errorf("copy %s: %w", rel, cErr)
			}
		}
	}
	return tw.Close()
}

// countingReader tallies the bytes read through it, so SyncTree can report how
// much crossed the wire.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// syncRunOptions is a non-multiplexed connection with a generous timeout — a full
// monorepo tree is many MiB and the extract writes every file.
func syncRunOptions() ssh.RunOptions {
	o := ssh.DefaultRunOptions()
	o.ControlMaster = false
	o.CommandTimeout = 30 * time.Minute
	o.MaxOutputBytes = 1024 * 1024
	return o
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
