package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// SudoState is the outcome of the optional passwordless-sudo provisioning done
// at first touch. It is a small typed vocabulary the orchestrator surfaces in an
// op step detail and later phases branch on.
type SudoState string

const (
	// SudoStateProvisioned — the scoped passwordless-sudo drop-in was written and
	// verified (or already present with identical content) using the owner
	// password over the now-key-authenticated connection.
	SudoStateProvisioned SudoState = "provisioned"
	// SudoStateAlreadyPasswordless — the user could already run sudo without a
	// password (a pre-existing NOPASSWD config, or our own drop-in from a prior
	// run). Nothing was written and no password was needed.
	SudoStateAlreadyPasswordless SudoState = "already-passwordless" //gitleaks:allow // status label, not credential material
	// SudoStateDeclined — the caller did not request sudo provisioning.
	SudoStateDeclined SudoState = "declined"
	// SudoStatePasswordUnavailable — sudo is not yet passwordless and no owner
	// password was available to provision it (e.g. a re-run where the key already
	// authorizes but the drop-in was never installed). NOT a failure — the caller
	// can re-run with the password to complete it.
	SudoStatePasswordUnavailable SudoState = "password-unavailable"
	// SudoStateFailed — provisioning was attempted with a password but the write
	// or the `visudo` validation failed; no drop-in is left behind.
	SudoStateFailed SudoState = "failed"
)

// sudoersDropInName is the basename of the bridge's sudoers drop-in. The scoped
// name keeps it isolated from the distro's own drop-ins and makes a re-run's
// byte-compare unambiguous.
const sudoersDropInName = "vrooli-bridge"

// sudoersDirEnv lets a test redirect the sudoers.d directory the remote provision
// script writes to (mirroring BRIDGE_SSH_STATE_DIR's role for the key material):
// an in-process sshd test can point it at a temp dir and exercise the real
// write / visudo-verify / rollback / idempotence path without touching the host's
// /etc. In production it is unset — and `sudo`'s env_reset strips it regardless —
// so the script always resolves the canonical /etc/sudoers.d.
const sudoersDirEnv = "VROOLI_BRIDGE_SUDOERS_DIR"

// sudoUserPattern bounds the SSH user embedded verbatim into the sudoers line, so
// a hostile or malformed username can never smuggle shell/sudoers syntax into the
// drop-in. Standard POSIX/macOS account names are a strict subset of this.
var sudoUserPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ProvisionSudoRequest is the input to a sudo-provisioning attempt over a
// key-authenticated connection. Password is the transient owner credential,
// borrowed for the single `sudo -S` write; the provisioner writes it to the
// remote stdin only (never argv/logs) and zeroes its own local copy — the caller
// (FirstTouch) owns the buffer's lifetime and zeroes the original.
type ProvisionSudoRequest struct {
	Host           string
	Port           int
	User           string
	KeyPath        string
	KnownHostsFile string
	Password       []byte
}

// ProvisionSudoResult reports the provisioning outcome. It carries no credential
// material.
type ProvisionSudoResult struct {
	State SudoState
}

// SudoProvisioner installs a scoped passwordless-sudo drop-in for the service
// user over an already-key-authenticated connection. Production dials the real
// host; unit tests inject a fake (mirroring the KeyCopier seam) so FirstTouch's
// state/credential handling is testable without a host.
type SudoProvisioner interface {
	Provision(ctx context.Context, req ProvisionSudoRequest) ProvisionSudoResult
}

// streamFunc is the remote-exec-with-stdin seam the production provisioner drives
// (Service.RunStreaming). Naming it lets the provisioner be constructed over the
// service's streaming capability while staying trivially unit-testable.
type streamFunc func(ctx context.Context, cfg Config, command string, opts StreamOptions) (Result, error)

// execSudoProvisioner is the production SudoProvisioner. It runs the sudo
// pre-check and the drop-in write over the bridge key connection, handing the
// owner password to `sudo -S` on stdin only.
type execSudoProvisioner struct {
	stream streamFunc
}

var _ SudoProvisioner = (*execSudoProvisioner)(nil)

// Provision establishes passwordless sudo for req.User. It first probes whether
// sudo is already passwordless (no password needed); if not, and a password is
// available, it writes+verifies the drop-in with a single `sudo -S` invocation.
func (p *execSudoProvisioner) Provision(ctx context.Context, req ProvisionSudoRequest) ProvisionSudoResult {
	if !sudoUserPattern.MatchString(req.User) {
		slog.Warn("ssh.sudo_provision", "host", req.Host, "state", SudoStateFailed, "reason", "unsupported username")
		return ProvisionSudoResult{State: SudoStateFailed}
	}

	cfg := NewConfig(req.Host, req.Port, req.User, req.KeyPath, req.KnownHostsFile)

	// 1. Already-passwordless probe (no password). `sudo -n true` exits 0 iff the
	//    user can run a command without being prompted — true after a prior run's
	//    drop-in, or on a host with a pre-existing NOPASSWD rule. Either way there
	//    is nothing to write.
	if res, err := p.stream(ctx, cfg, "sudo -n true", StreamOptions{Run: p.runOptions(15 * time.Second)}); err == nil && res.ExitCode == 0 {
		slog.Info("ssh.sudo_provision", "host", req.Host, "state", SudoStateAlreadyPasswordless)
		return ProvisionSudoResult{State: SudoStateAlreadyPasswordless}
	}

	// 2. Not yet passwordless. Without the owner password we cannot elevate — this
	//    is the re-run-without-password case: report it distinctly, not as a
	//    failure, so the caller can re-run with the password to finish the job.
	if len(req.Password) == 0 {
		slog.Info("ssh.sudo_provision", "host", req.Host, "state", SudoStatePasswordUnavailable)
		return ProvisionSudoResult{State: SudoStatePasswordUnavailable}
	}

	// 3. Elevate once with the password on stdin and write+verify the drop-in.
	//    stdin = password + newline; `sudo -S` consumes the first line, so the
	//    secret rides the channel and never appears in argv or a log line. The
	//    local newline-appended copy is zeroed the instant the exec returns.
	stdin := make([]byte, 0, len(req.Password)+1)
	stdin = append(stdin, req.Password...)
	stdin = append(stdin, '\n')

	command := "sudo -S -p '' sh -c " + quoteSingle(p.provisionScript(req.User))
	res, err := p.stream(ctx, cfg, command, StreamOptions{Run: p.runOptions(30 * time.Second), Stdin: stdin})
	zeroBytes(stdin)

	if err != nil || res.ExitCode != 0 {
		slog.Warn("ssh.sudo_provision", "host", req.Host, "state", SudoStateFailed, "exit_code", res.ExitCode)
		return ProvisionSudoResult{State: SudoStateFailed}
	}
	slog.Info("ssh.sudo_provision", "host", req.Host, "state", SudoStateProvisioned)
	return ProvisionSudoResult{State: SudoStateProvisioned}
}

// runOptions returns the run options for a short, single sudo command over the
// bridge key: offer only that key (IdentitiesOnly), pin the bridge known_hosts
// (set on the Config), and no ControlMaster (a lone short exec gains nothing from
// multiplexing and must not wait on a persisted master).
func (p *execSudoProvisioner) runOptions(timeout time.Duration) RunOptions {
	return RunOptions{
		ConnectTimeout: 15 * time.Second,
		StrictHostKey:  true,
		IdentitiesOnly: true,
		MaxOutputBytes: 64 * 1024,
		CommandTimeout: timeout,
	}
}

// provisionScript builds the remote shell script (run under `sudo -S sh -c`) that
// installs the drop-in idempotently and safely:
//
//   - byte-compare the existing drop-in first so an unchanged re-run is a true
//     no-op (no rewrite, no mtime churn);
//   - write to a sibling temp file, chmod 0440, and validate THAT file with
//     `visudo -c` BEFORE moving it into place, so a syntactically bad file is
//     never momentarily live in sudoers.d (a stronger guarantee than write-then-
//     rollback: a broken file there can wedge sudo for every caller);
//   - only an atomically-moved, validated file ever appears at the final path; on
//     validation failure the temp is removed and no drop-in is left behind.
//
// The grant is deliberately NOT command-scoped (`NOPASSWD: ALL`, not
// `NOPASSWD: /path/to/cmd`): the elevated payload is `make setup` (and the apt
// prereq install and `loginctl enable-linger` it spawns), an open-ended tree of
// subcommands with no stable, enumerable argv to scope to. A command-scoped rule
// would either break setup or be trivially bypassable, so the honest contract is
// full passwordless sudo for the dedicated service principal.
func (p *execSudoProvisioner) provisionScript(user string) string {
	content := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", user)
	// Single-quote the user-derived content for safe embedding in the script; the
	// script itself is single-quoted again by the caller (quoteSingle).
	return "set -eu; " +
		"dir=\"${" + sudoersDirEnv + ":-/etc/sudoers.d}\"; " +
		"file=\"$dir/" + sudoersDropInName + "\"; " +
		"content=" + quoteSingle(content) + "; " +
		"mkdir -p \"$dir\"; " +
		"if [ -f \"$file\" ] && [ \"$(cat \"$file\")\" = \"$content\" ]; then exit 0; fi; " +
		"umask 077; " +
		"tmp=\"$file.vrooli-tmp.$$\"; " +
		"printf '%s\\n' \"$content\" > \"$tmp\"; " +
		"chmod 0440 \"$tmp\"; " +
		"if visudo -cf \"$tmp\" >/dev/null 2>&1; then mv \"$tmp\" \"$file\"; else rm -f \"$tmp\"; exit 3; fi"
}
