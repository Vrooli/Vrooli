package onboard

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
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
		Host: p.Host, Port: p.Port, User: p.User, Password: p.Password, KeyName: p.KeyName,
		ProvisionSudo: p.ProvisionSudo,
	})
	if err != nil {
		return Conn{}, err
	}
	if !res.OK {
		detail := res.Message
		// Surface the raw underlying cause (never credential material) so the op step
		// detail is actionable instead of the generic category alone.
		if h := strings.TrimSpace(res.Hint); h != "" && h != res.Message {
			detail = res.Message + ": " + h
		}
		return Conn{}, fmt.Errorf("passwordless SSH not established: %s", detail)
	}
	return Conn{Host: res.Host, Port: res.Port, User: res.User, KeyPath: res.KeyPath, ClientKeyRef: "ssh-key://" + filepath.Base(res.KeyPath), ClientKeyFingerprint: res.Fingerprint, HostKeyFingerprint: res.HostKeyFingerprint, SudoState: string(res.SudoState)}, nil
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

// ProbeEndpoint performs the authoritative remote admission test. curl's
// failure codes preserve DNS/TCP/TLS/timeout distinctions without attempting
// to parse pairing prose; the only successful result is an actual /health HTTP
// response from the exact endpoint.
func (d *sshDriver) ProbeEndpoint(ctx context.Context, conn Conn, endpoint string) (AdmissionResult, error) {
	cfg := d.config(conn)
	const marker = "VBADMISSION="
	command, commandErr := admissionProbeCommand(endpoint, marker)
	if commandErr != nil {
		return AdmissionResult{Endpoint: endpoint, Category: AdmissionEndpointInvalid, Detail: commandErr.Error(), Retryable: false}, commandErr
	}
	var line string
	started := time.Now()
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{Run: admissionRunOptions(), OnStdoutLine: func(v string) {
		if strings.HasPrefix(v, marker) {
			line = strings.TrimPrefix(v, marker)
		}
	}})
	result := AdmissionResult{Endpoint: endpoint, Duration: time.Since(started), Retryable: true}
	if err != nil || res.ExitCode != 0 || line == "" {
		result.Category, result.Detail = AdmissionControlPlaneUnreachable, "remote admission probe could not complete"
		if err != nil {
			result.Detail += ": " + err.Error()
		}
		return result, err
	}
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 {
		result.Category, result.Detail = AdmissionControlPlaneUnreachable, "remote admission probe returned malformed evidence"
		return result, nil
	}
	result.SourceIP, result.Detail = strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
	switch parts[0] {
	case "passed":
		result.Category, result.Retryable = AdmissionPassed, false
	case "dns":
		result.Category = AdmissionNameUnresolvable
	case "http":
		result.Category = AdmissionControlPlaneUnhealthy
	default:
		result.Category = AdmissionControlPlaneUnreachable
	}
	return result, nil
}

// admissionProbeCommand deliberately derives src from the candidate's route to
// the Bridge endpoint. SSH_CONNECTION instead describes the Bridge SSH client,
// which is the wrong address for an inbound UFW rule on the Bridge host.
func admissionProbeCommand(endpoint, marker string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" {
		return "", fmt.Errorf("invalid candidate-admission endpoint %q", endpoint)
	}
	command := "host=" + shellQuote(u.Hostname()) + "; " +
		"src=''; if command -v ip >/dev/null 2>&1; then src=$(ip route get \"$host\" 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i == \"src\") {print $(i+1); exit}}'); " +
		"elif command -v route >/dev/null 2>&1 && command -v ipconfig >/dev/null 2>&1; then iface=$(route -n get \"$host\" 2>/dev/null | awk '/interface:/{print $2; exit}'); [ -z \"$iface\" ] || src=$(ipconfig getifaddr \"$iface\" 2>/dev/null); fi; " +
		"url=" + shellQuote(strings.TrimRight(endpoint, "/")+"/health") + "; " +
		"if ! command -v curl >/dev/null 2>&1; then printf '" + marker + "unreachable|%s|curl-unavailable\\n' \"$src\"; exit 0; fi; " +
		"curl -fsS --connect-timeout 5 --max-time 12 -o /dev/null \"$url\"; rc=$?; " +
		"case $rc in 0) cat=passed;;6) cat=dns;;7) cat=tcp;;28) cat=timeout;;35|51|60) cat=tls;;22) cat=http;;*) cat=unreachable;; esac; " +
		"printf '" + marker + "%s|%s|curl-exit-%s\\n' \"$cat\" \"$src\" \"$rc\""
	return command, nil
}

func admissionRunOptions() ssh.RunOptions {
	o := ssh.DefaultRunOptions()
	o.ControlMaster, o.CommandTimeout, o.MaxOutputBytes = false, 20*time.Second, 8*1024
	return o
}

// SyncTree ships the control plane's working tree to the node by piping a tar
// archive of p.Files (relative to p.RepoDir) into a remote staging directory.
// The staging directory is atomically swapped into the requested destination,
// so deleted local files cannot survive from an earlier working-tree shipment.
// Filenames with spaces/newlines survive because the tar format encodes names —
// nothing is shell-word-split. The tar is streamed (never buffered whole in
// memory) and its byte count is measured for the step detail.
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

// DetectPlatform asks the node for its kernel/architecture after first touch and
// normalises the result to the Go target names consumed by the shared builder.
func (d *sshDriver) DetectPlatform(ctx context.Context, conn Conn) (NodePlatform, error) {
	cfg := d.config(conn)
	var lines []string
	res, err := d.svc.RunStreaming(ctx, cfg, `printf 'VBPLATFORM=%s/%s\n' "$(uname -s)" "$(uname -m)"`, ssh.StreamOptions{
		Run: syncRunOptions(),
		OnStdoutLine: func(line string) {
			lines = append(lines, strings.TrimSpace(line))
		},
	})
	if err != nil {
		return NodePlatform{}, err
	}
	if res.ExitCode != 0 {
		return NodePlatform{}, fmt.Errorf("node platform probe failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	for _, line := range lines {
		raw, ok := strings.CutPrefix(line, "VBPLATFORM=")
		if !ok {
			continue
		}
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) != 2 {
			break
		}
		target := NodePlatform{OS: normaliseNodeOS(parts[0]), Arch: normaliseNodeArch(parts[1])}
		if !supportedBridgeTarget(target) {
			return NodePlatform{}, fmt.Errorf("unsupported bridge node platform %s/%s", parts[0], parts[1])
		}
		return target, nil
	}
	return NodePlatform{}, fmt.Errorf("node platform probe returned no platform marker")
}

func normaliseNodeOS(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normaliseNodeArch(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// PushArtifacts copies all three executables and their sidecars into one unique,
// private node-side directory. A final remote chmod/touch makes the executables
// runnable and sidecars no older than their binaries for freshness validation.
func (d *sshDriver) PushArtifacts(ctx context.Context, p ArtifactPushParams) (RemoteArtifacts, error) {
	files := []string{
		p.Artifacts.Vrooli, p.Artifacts.VrooliSidecar,
		p.Artifacts.BridgeCLI, p.Artifacts.BridgeSidecar,
		p.Artifacts.Agent, p.Artifacts.AgentSidecar,
	}
	for _, path := range files {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			if err == nil {
				err = fmt.Errorf("path is a directory")
			}
			return RemoteArtifacts{}, fmt.Errorf("prebuilt artifact %s is unavailable: %w", path, err)
		}
	}
	remoteDirName, err := remoteArtifactDirName()
	if err != nil {
		return RemoteArtifacts{}, err
	}
	cfg := d.config(p.Conn)
	remoteDir, err := d.prepareRemoteArtifactDir(ctx, cfg, remoteDirName)
	if err != nil {
		return RemoteArtifacts{}, fmt.Errorf("prepare remote artifact directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = d.runRemoteCommand(ctx, cfg, "rm -rf "+shellQuote(remoteDir))
		}
	}()
	for _, local := range files {
		remote := remoteDir + "/" + filepath.Base(local)
		if err := d.scpRunner.Copy(ctx, cfg, local, remote, ssh.DefaultSCPOptions()); err != nil {
			return RemoteArtifacts{}, fmt.Errorf("copy %s: %w", filepath.Base(local), err)
		}
	}
	remote := RemoteArtifacts{
		Vrooli:    remoteDir + "/" + filepath.Base(p.Artifacts.Vrooli),
		BridgeCLI: remoteDir + "/" + filepath.Base(p.Artifacts.BridgeCLI),
		Agent:     remoteDir + "/" + filepath.Base(p.Artifacts.Agent),
	}
	finalise := "chmod 700 " + shellQuote(remote.Vrooli) + " " + shellQuote(remote.BridgeCLI) + " " + shellQuote(remote.Agent) +
		"; touch " + shellQuote(remote.Vrooli+".fp") + " " + shellQuote(remote.BridgeCLI+".fp") + " " + shellQuote(remote.Agent+".fp")
	if err := d.runRemoteCommand(ctx, cfg, finalise); err != nil {
		return RemoteArtifacts{}, fmt.Errorf("finalise remote artifacts: %w", err)
	}
	cleanup = false
	return remote, nil
}

const artifactDirMarker = "VBARTIFACTDIR="

func (d *sshDriver) prepareRemoteArtifactDir(ctx context.Context, cfg ssh.Config, name string) (string, error) {
	var resolved string
	command := `dest="$HOME/.local/lib/vrooli-bridge/bootstrap/` + name + `"; umask 077; mkdir -p "$dest"; printf '` + artifactDirMarker + `%s\n' "$dest"`
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{
		Run: syncRunOptions(),
		OnStdoutLine: func(line string) {
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), artifactDirMarker); ok {
				resolved = strings.TrimSpace(value)
			}
		},
	})
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("remote directory creation failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	if resolved == "" || !strings.HasPrefix(resolved, "/") {
		return "", fmt.Errorf("node did not report an absolute artifact directory")
	}
	return resolved, nil
}

func (d *sshDriver) runRemoteCommand(ctx context.Context, cfg ssh.Config, command string) error {
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{Run: syncRunOptions()})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("remote command failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	return nil
}

func remoteArtifactDirName() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate remote artifact suffix: %w", err)
	}
	return "artifacts-" + hex.EncodeToString(b[:]), nil
}

// buildSyncRemoteCommand renders the remote shell that resolves the destination,
// extracts into a same-parent staging directory, reports the destination (the
// VBSYNCDEST marker), and swaps the complete snapshot into place. The old
// destination is removed only after the new tree is fully extracted; this is the
// convergence guarantee that makes deleted working-tree files disappear without
// exposing a partially extracted checkout to the bootstrap.
// An explicit destDir is shell-quoted; an empty one defaults to $HOME/vrooli,
// resolved on the node (the control plane cannot know the node's home).
func buildSyncRemoteCommand(destDir string) string {
	var assign string
	if d := strings.TrimSpace(destDir); d != "" {
		assign = "dest=" + shellQuote(d)
	} else {
		assign = `dest="$HOME/vrooli"`
	}
	return assign + `; parent=$(dirname "$dest"); base=$(basename "$dest"); stage="$parent/.${base}.bridge-sync-$$"; backup="$parent/.${base}.bridge-old-$$"; rm -rf "$stage" "$backup"; mkdir -p "$stage" && printf '` + syncDestMarker + `%s\n' "$dest" && tar -xf - -C "$stage" && { [ ! -e "$dest" ] || mv "$dest" "$backup"; } && mv "$stage" "$dest" && rm -rf "$backup"`
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
	var nodeID string

	// The pairing code rides stdin (env-only, never argv/logs): the remote shell
	// reads one line into BRIDGE_PAIRING_CODE, exports it, then execs the script.
	// The script's flags carry NO secret.
	remoteCmd := "IFS= read -r __vb_code; export BRIDGE_PAIRING_CODE=\"$__vb_code\"; unset __vb_code; exec bash " +
		shellQuote(p.RemotePath) + " " + quoteArgs(p.Args)

	// stdin = code + newline. Built here and zeroed on return so the only lasting
	// copy of the secret is the caller's, which the orchestrator wipes too.
	stdin := make([]byte, 0, len(p.PairingCode)+len(p.SetupPassphrase)+2)
	stdin = append(stdin, p.PairingCode...)
	stdin = append(stdin, '\n')
	if len(p.SetupPassphrase) > 0 {
		stdin = append(stdin, p.SetupPassphrase...)
		stdin = append(stdin, '\n')
	}
	defer zeroBytes(stdin)

	res, err := d.svc.RunStreaming(ctx, cfg, remoteCmd, ssh.StreamOptions{
		Run:   bootstrapRunOptions(),
		Stdin: stdin,
		OnStdoutLine: func(line string) {
			if m, ok := parseMarker(line); ok {
				if m.Event == eventNodeID {
					nodeID = m.NodeID
				}
				onMarker(m)
			}
		},
	})
	if err != nil {
		return BootstrapResult{ExitCode: res.ExitCode, Diagnostics: diagnosticsTail(res.Stderr), NodeID: nodeID}, err
	}
	return BootstrapResult{ExitCode: res.ExitCode, Diagnostics: diagnosticsTail(res.Stderr), NodeID: nodeID}, nil
}

// diagnosticsTailMaxBytes bounds the node-side diagnostic tail carried on a
// BootstrapResult (and persisted on a failed op). The full stream can be MiB of
// build output; the operator needs the end — where the failing step's error
// lands — not the whole log. Sized to comfortably hold a `make setup` failure's
// trailing context without bloating the durable record.
const diagnosticsTailMaxBytes = 8 * 1024

// diagnosticsTail returns at most the last diagnosticsTailMaxBytes of the
// node-side stderr stream, trimmed to whole lines so the surfaced tail never
// starts mid-line. The bootstrap already bounds its stderr capture (4 MiB), so
// this is a second, display-oriented bound. An empty stream yields "".
func diagnosticsTail(stderr string) string {
	s := strings.TrimRight(stderr, "\n")
	if s == "" {
		return ""
	}
	if len(s) <= diagnosticsTailMaxBytes {
		return s
	}
	s = s[len(s)-diagnosticsTailMaxBytes:]
	// Drop a partial leading line so the tail begins at a line boundary.
	if i := strings.IndexByte(s, '\n'); i >= 0 && i+1 < len(s) {
		s = s[i+1:]
	}
	return s
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
