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
	"vrooli-bridge/internal/onboarding"
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

func (d *sshDriver) VerifyKey(ctx context.Context, conn Conn) (Conn, error) {
	res := d.svc.TestConnection(ctx, ssh.TestConnectionRequest{
		Host:    conn.Host,
		Port:    conn.Port,
		User:    conn.User,
		KeyPath: conn.KeyPath,
	})
	if !res.OK {
		detail := res.Message
		if h := strings.TrimSpace(res.Hint); h != "" && h != res.Message {
			detail += ": " + h
		}
		return Conn{}, fmt.Errorf("final key-only SSH verification failed: %s", detail)
	}
	verified := conn
	if res.Fingerprint != "" {
		verified.HostKeyFingerprint = res.Fingerprint
	}
	return verified, nil
}

func (d *sshDriver) PushScript(ctx context.Context, conn Conn, target NodePlatform) (string, error) {
	cfg := d.config(conn)
	scriptPath := d.scriptPath
	if target.OS == "windows" {
		scriptPath = filepath.Join(filepath.Dir(d.scriptPath), "bootstrap.ps1")
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("platform bootstrap script %q is unavailable: %w", scriptPath, err)
	}
	remotePath, err := remoteScriptPath(target.OS)
	if err != nil {
		return "", err
	}
	if err := d.scpRunner.Copy(ctx, cfg, scriptPath, remotePath, ssh.DefaultSCPOptions()); err != nil {
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
	o := shortRunOptions()
	o.CommandTimeout, o.MaxOutputBytes = 20*time.Second, 8*1024
	return o
}

// shortRunOptions allows the SSH client to reuse the bridge-owned control
// socket for the many bounded setup probes. Long-lived streaming operations use
// syncRunOptions/bootstrapRunOptions instead because a background master can
// delay EOF for those streams.
func shortRunOptions() ssh.RunOptions {
	o := ssh.DefaultRunOptions()
	o.CommandTimeout = 30 * time.Second
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
	remoteCmd := buildSyncRemoteCommandForPlatform(p.DestDir, p.Platform.OS)

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
	if resolvedDest == "" || !validRemotePath(resolvedDest, p.Platform.OS) {
		return SyncResult{}, fmt.Errorf("remote sync did not report a destination directory")
	}
	return SyncResult{BytesTransferred: counter.n, ResolvedDestDir: resolvedDest}, nil
}

// DetectPlatform asks the node for its kernel/architecture after first touch and
// normalises the result to the Go target names consumed by the shared builder.
func (d *sshDriver) DetectPlatform(ctx context.Context, conn Conn) (NodePlatform, error) {
	cfg := d.config(conn)
	lines, _, firstErr := d.runPlatformProbe(ctx, cfg, platformProbeCommand())
	if target, ok, probeErr := platformFromProbeLines(lines); probeErr != nil {
		return NodePlatform{}, probeErr
	} else if ok {
		return target, nil
	}

	// Windows OpenSSH commonly uses cmd.exe or Windows PowerShell as its
	// default shell. Run the PowerShell fallback as a separate command instead
	// of relying on shell-specific `||` syntax.
	lines, _, fallbackErr := d.runPlatformProbe(ctx, cfg, windowsPlatformProbeCommand())
	if target, ok, probeErr := platformFromProbeLines(lines); probeErr != nil {
		return NodePlatform{}, probeErr
	} else if ok {
		return target, nil
	}
	if fallbackErr != nil {
		if firstErr != nil {
			return NodePlatform{}, fmt.Errorf("node platform probes failed: posix: %v; windows: %w", firstErr, fallbackErr)
		}
		return NodePlatform{}, fallbackErr
	}
	return NodePlatform{}, fmt.Errorf("node platform probe returned no platform marker")
}

func (d *sshDriver) runPlatformProbe(ctx context.Context, cfg ssh.ConnectionConfig, command string) ([]string, ssh.Result, error) {
	var lines []string
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{
		Run: shortRunOptions(),
		OnStdoutLine: func(line string) {
			lines = append(lines, strings.TrimSpace(line))
		},
	})
	return lines, res, err
}

func platformFromProbeLines(lines []string) (NodePlatform, bool, error) {
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
			return NodePlatform{}, false, fmt.Errorf("unsupported bridge node platform %s/%s", parts[0], parts[1])
		}
		return target, true, nil
	}
	return NodePlatform{}, false, nil
}

// platformProbeCommand is the POSIX fast path. Windows OpenSSH hosts use the
// separate PowerShell fallback below because their default shell may be cmd.exe
// or a Windows PowerShell version without `||` syntax.
func platformProbeCommand() string {
	return `printf 'VBPLATFORM=%s/%s\n' "$(uname -s)" "$(uname -m)"`
}

// windowsPlatformProbeCommand uses RuntimeInformation rather than
// PROCESSOR_ARCHITECTURE, which can expose a 32-bit process view on a 64-bit
// Windows host.
func windowsPlatformProbeCommand() string {
	return `powershell.exe -NoProfile -NonInteractive -Command "$a=[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant(); if($a -eq 'x64'){$a='amd64'}; Write-Output ('VBPLATFORM=windows/'+$a)"`
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
// private node-side directory. The bundle crosses SSH as one tar stream, so a
// slow or lossy connection pays one setup cost rather than six SCP handshakes.
// A final remote chmod/touch makes the executables runnable and sidecars no
// older than their binaries for freshness validation.
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
	remoteDir, err := d.prepareRemoteArtifactDir(ctx, cfg, remoteDirName, p.Artifacts.Target.OS)
	if err != nil {
		return RemoteArtifacts{}, fmt.Errorf("prepare remote artifact directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = d.runRemoteCommand(ctx, cfg, removeRemoteDirectoryCommand(remoteDir, p.Artifacts.Target.OS))
		}
	}()
	pr, pw := io.Pipe()
	counter := &countingReader{r: pr}
	go func() {
		pw.CloseWithError(writeArtifactTarStream(pw, files))
	}()
	extractCommand := "tar -x -C " + shellQuote(remoteDir)
	if p.Artifacts.Target.OS == "windows" {
		extractCommand = windowsTarExtractCommand(remoteDir)
	}
	res, err := d.svc.RunStreaming(ctx, cfg, extractCommand, ssh.StreamOptions{
		Run:         syncRunOptions(),
		StdinReader: counter,
	})
	if err != nil {
		return RemoteArtifacts{}, err
	}
	if res.ExitCode != 0 {
		return RemoteArtifacts{}, fmt.Errorf("remote artifact extract failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	remote := RemoteArtifacts{
		Vrooli:    remoteArtifactPath(remoteDir, filepath.Base(p.Artifacts.Vrooli), p.Artifacts.Target.OS),
		BridgeCLI: remoteArtifactPath(remoteDir, filepath.Base(p.Artifacts.BridgeCLI), p.Artifacts.Target.OS),
		Agent:     remoteArtifactPath(remoteDir, filepath.Base(p.Artifacts.Agent), p.Artifacts.Target.OS),
	}
	finalise := "chmod 700 " + shellQuote(remote.Vrooli) + " " + shellQuote(remote.BridgeCLI) + " " + shellQuote(remote.Agent) +
		"; touch " + shellQuote(remote.Vrooli+".fp") + " " + shellQuote(remote.BridgeCLI+".fp") + " " + shellQuote(remote.Agent+".fp")
	if p.Artifacts.Target.OS == "windows" {
		finalise = windowsArtifactFinaliseCommand(remote)
	}
	if err := d.runRemoteCommand(ctx, cfg, finalise); err != nil {
		return RemoteArtifacts{}, fmt.Errorf("finalise remote artifacts: %w", err)
	}
	cleanup = false
	return remote, nil
}

func writeArtifactTarStream(w io.Writer, files []string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode := int64(0o644)
		if info.Mode()&0o111 != 0 {
			mode = 0o700
		}
		header := &tar.Header{Name: filepath.Base(path), Mode: mode, Size: info.Size(), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

const artifactDirMarker = "VBARTIFACTDIR="

func (d *sshDriver) prepareRemoteArtifactDir(ctx context.Context, cfg ssh.ConnectionConfig, name, targetOS string) (string, error) {
	var resolved string
	command := `dest="$HOME/.local/lib/vrooli-bridge/bootstrap/` + name + `"; umask 077; mkdir -p "$dest"; printf '` + artifactDirMarker + `%s\n' "$dest"`
	if targetOS == "windows" {
		command = `powershell.exe -NoProfile -NonInteractive -Command "$dest=Join-Path $env:LOCALAPPDATA 'Vrooli\\Bridge\\bootstrap\\` + name + `'; New-Item -ItemType Directory -Force -Path $dest | Out-Null; [Console]::WriteLine('` + artifactDirMarker + `'+$dest)"`
	}
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{
		Run: shortRunOptions(),
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
	if resolved == "" || !validRemotePath(resolved, targetOS) {
		return "", fmt.Errorf("node did not report an absolute artifact directory")
	}
	return resolved, nil
}

func remoteArtifactPath(dir, base, targetOS string) string {
	if targetOS == "windows" {
		return strings.TrimRight(dir, `\\/`) + `\` + base
	}
	return dir + "/" + base
}

func removeRemoteDirectoryCommand(dir, targetOS string) string {
	if targetOS == "windows" {
		return `powershell.exe -NoProfile -NonInteractive -Command "if(Test-Path -LiteralPath '` + windowsPowerShellLiteral(dir) + `'){Remove-Item -Recurse -Force -LiteralPath '` + windowsPowerShellLiteral(dir) + `'}"`
	}
	return "rm -rf " + shellQuote(dir)
}

func windowsTarExtractCommand(dir string) string {
	quoted := windowsPowerShellLiteral(dir)
	return `powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference='Stop'; tar.exe -xf - -C '` + quoted + `'; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}"`
}

func windowsArtifactFinaliseCommand(remote RemoteArtifacts) string {
	paths := []string{remote.Vrooli, remote.BridgeCLI, remote.Agent}
	for i := range paths {
		paths[i] = "'" + windowsPowerShellLiteral(paths[i]) + "'"
	}
	return `powershell.exe -NoProfile -NonInteractive -Command "$paths=@(` + strings.Join(paths, ",") + `); foreach($path in $paths){if(-not(Test-Path -LiteralPath $path)){throw 'received artifact missing'}}"`
}

func (d *sshDriver) runRemoteCommand(ctx context.Context, cfg ssh.ConnectionConfig, command string) error {
	res, err := d.svc.RunStreaming(ctx, cfg, command, ssh.StreamOptions{Run: shortRunOptions()})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("remote command failed (exit %d): %s", res.ExitCode, res.Stderr)
	}
	return nil
}

// Run implements the transport-neutral onboarding bridge contract. It is kept
// off SSHDriver so existing bootstrap fakes remain valid; production drivers
// expose it through a narrow type assertion when a setup profile requests the
// declarative onboarding surface.
func (d *sshDriver) Run(ctx context.Context, target onboarding.Target, command string) (onboarding.Result, error) {
	conn := Conn{Host: target.Host, Port: target.Port, User: target.User, KeyPath: target.Key}
	var stdout strings.Builder
	result, err := d.svc.RunStreaming(ctx, d.config(conn), command, ssh.StreamOptions{
		Run: syncRunOptions(),
		OnStdoutLine: func(line string) {
			stdout.WriteString(line)
			stdout.WriteByte('\n')
		},
	})
	return onboarding.Result{ExitCode: result.ExitCode, Stdout: stdout.String(), Stderr: result.Stderr}, err
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

func buildSyncRemoteCommandForPlatform(destDir, targetOS string) string {
	if targetOS != "windows" {
		return buildSyncRemoteCommand(destDir)
	}
	dest := strings.TrimSpace(destDir)
	if dest == "" {
		dest = "$env:USERPROFILE\\vrooli"
	}
	if strings.HasPrefix(dest, "$env:") {
		return `powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference='Stop'; $dest=` + dest + `; $parent=Split-Path -Parent $dest; $base=Split-Path -Leaf $dest; $stage=Join-Path $parent ('.'+$base+'.bridge-sync-'+[guid]::NewGuid().ToString('N')); $backup=Join-Path $parent ('.'+$base+'.bridge-old-'+[guid]::NewGuid().ToString('N')); New-Item -ItemType Directory -Force -Path $stage | Out-Null; [Console]::WriteLine('` + syncDestMarker + `'+$dest); tar.exe -xf - -C $stage; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}; if(Test-Path -LiteralPath $dest){Move-Item -Force -LiteralPath $dest -Destination $backup}; Move-Item -Force -LiteralPath $stage -Destination $dest; if(Test-Path -LiteralPath $backup){Remove-Item -Recurse -Force -LiteralPath $backup}"`
	}
	return `powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference='Stop'; $dest='` + windowsPowerShellLiteral(dest) + `'; $parent=Split-Path -Parent $dest; $base=Split-Path -Leaf $dest; $stage=Join-Path $parent ('.'+$base+'.bridge-sync-'+[guid]::NewGuid().ToString('N')); $backup=Join-Path $parent ('.'+$base+'.bridge-old-'+[guid]::NewGuid().ToString('N')); New-Item -ItemType Directory -Force -Path $stage | Out-Null; [Console]::WriteLine('` + syncDestMarker + `'+$dest); tar.exe -xf - -C $stage; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}; if(Test-Path -LiteralPath $dest){Move-Item -Force -LiteralPath $dest -Destination $backup}; Move-Item -Force -LiteralPath $stage -Destination $dest; if(Test-Path -LiteralPath $backup){Remove-Item -Recurse -Force -LiteralPath $backup}"`
}

func validRemotePath(value, targetOS string) bool {
	if targetOS == "windows" {
		return len(value) >= 3 && ((value[1] == ':' && (value[2] == '\\' || value[2] == '/')) || strings.HasPrefix(value, `\\\\`))
	}
	return strings.HasPrefix(value, "/")
}

func windowsPowerShellLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
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

	// The pairing code rides stdin (env-only, never argv/logs). POSIX hosts use
	// a shell prelude to export it; the PowerShell bootstrap reads the first
	// stdin line itself so no Windows shell variable expansion is involved.
	remoteCmd := ""
	if p.Platform.OS == "windows" {
		remoteCmd = "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File " +
			windowsPowerShellQuote(p.RemotePath) + " " + windowsPowerShellArgs(p.Args)
	} else {
		remoteCmd = "IFS= read -r __vb_code; export BRIDGE_PAIRING_CODE=\"$__vb_code\"; unset __vb_code; exec bash " +
			shellQuote(p.RemotePath) + " " + quoteArgs(p.Args)
	}

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

// config builds the ssh.ConnectionConfig for a resolved Conn, pinned to the bridge-owned
// known_hosts the first touch populated.
func (d *sshDriver) config(conn Conn) ssh.ConnectionConfig {
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
func remoteScriptPath(targetOS string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate remote script suffix: %w", err)
	}
	if targetOS == "windows" {
		// A relative SCP destination lands in the Windows OpenSSH user's home,
		// which is also the working directory for the subsequent PowerShell
		// command. Avoid POSIX /tmp assumptions and drive-letter parsing here.
		return "vrooli-bridge-bootstrap-" + hex.EncodeToString(b[:]) + ".ps1", nil
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

func windowsPowerShellQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func windowsPowerShellArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = windowsPowerShellQuote(arg)
	}
	return strings.Join(quoted, " ")
}
