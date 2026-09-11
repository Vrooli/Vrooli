// Package sshadapter is the shared bounded SSH transport for reach. It is a
// policy-equivalent adapter, not a private connection plane: it accepts only
// validated argv, joins it with one quoting rule, runs the target's vrooli
// binary from the bound workdir, and classifies failures into reach kinds. It
// is selected only by an explicit "ssh" transport binding.
//
// The adapter owns exactly three fixed transport-internal commands besides
// the verb runner: the platform probe (`uname -s && uname -m`), the parent
// directory creation and digest check around an artifact copy, and the mode
// change after it. They are the transport's own quoting policy applied to
// validated paths, never a caller-supplied shell string.
package sshadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/reach"
)

// ConfigResolver turns a target binding into the SSH connection to use.
// Cloud never reads key material here; the resolver is expected to obtain
// it from the credential binding declared for the deployment.
type ConfigResolver func(ctx context.Context, target identity.TargetRef) (ConnectionConfig, error)

// Adapter runs reach commands over the SSH runner and places artifacts over
// the SCP runner.
type Adapter struct {
	Runner Runner
	SCP    SCPRunner
	Config ConfigResolver
	// DefaultTimeout bounds a command without its own timeout.
	DefaultTimeout time.Duration
	// Dial opens the client an interactive session rides on; nil selects
	// the production dialer (bound key + local agent, TOFU host keys).
	Dial Dialer
}

var _ reach.Reach = (*Adapter)(nil)

// RemoteCommand renders the single remote string for an argument vector.
// Every element is single-quoted by one policy function; the vrooli path
// and workdir come from the binding's locator. Exported so tests and the
// preview renderer share the exact rule.
func RemoteCommand(workdir string, argv []string) string {
	return shellutil.VrooliCommand(workdir, quoteArgv(argv))
}

// ObservationCommand renders the remote string for a host observation
// program: the program and its arguments, each single-quoted, with no
// workdir change and no vrooli path. The same quoting rule as RemoteCommand
// applies, so a validated argument can never become shell syntax.
func ObservationCommand(argv []string) string {
	return quoteArgv(argv)
}

func quoteArgv(argv []string) string {
	quoted := make([]string, 0, len(argv))
	for _, a := range argv {
		quoted = append(quoted, shellutil.QuoteSingle(a))
	}
	return strings.Join(quoted, " ")
}

// platformProbe is the fixed command the transport uses to learn the target
// platform before any native binary exists on it.
const platformProbe = "uname -s && uname -m"

func (a *Adapter) unavailable(target identity.TargetRef, detail string, err error) *reach.Error {
	return &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: detail, Err: err}
}

func (a *Adapter) connect(ctx context.Context, target identity.TargetRef) (ConnectionConfig, error) {
	if a == nil || a.Runner == nil || a.Config == nil {
		return ConnectionConfig{}, a.unavailable(target, "ssh adapter not configured", nil)
	}
	cfg, err := a.Config(ctx, target)
	if err != nil {
		return ConnectionConfig{}, a.unavailable(target, "ssh connection cannot be resolved", err)
	}
	return cfg, nil
}

// run executes one remote string and classifies transport failures.
func (a *Adapter) run(ctx context.Context, target identity.TargetRef, cfg ConnectionConfig, remote string, opts RunOptions) (reach.Result, error) {
	res, runErr := a.Runner.Run(ctx, cfg, remote, opts)
	out := reach.Result{ExitCode: res.ExitCode, Stdout: res.Stdout, Stderr: res.Stderr, Transport: identity.TransportSSH}
	if runErr == nil {
		return out, nil
	}
	if errors.Is(runErr, context.DeadlineExceeded) {
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Detail: "command timed out", Err: runErr}
	}
	// Classification is by exit code only (D8): 255 is ssh's own failure
	// (unreachable, refused, rejected key, changed host key) and 127 is the
	// remote shell reporting the program absent. Message text is never read.
	switch res.ExitCode {
	case 255:
		return out, &reach.Error{Kind: reach.KindTargetOffline, Transport: identity.TransportSSH, Target: target.Key(), Detail: "ssh connection failed: " + firstLine(res.Stderr), Err: runErr}
	case 127:
		return out, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportSSH, Target: target.Key(), Detail: "program absent on target", Err: runErr}
	}
	// A non-zero exit from the verb itself is the target's typed answer, not
	// a transport failure; the caller decodes stdout.
	if res.ExitCode > 0 {
		return out, nil
	}
	return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Err: runErr}
}

// Exec implements reach.Reach.
func (a *Adapter) Exec(ctx context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if err := reach.ValidateCommand(cmd); err != nil {
		return reach.Result{}, err
	}
	cfg, err := a.connect(ctx, target)
	if err != nil {
		return reach.Result{}, err
	}
	argv := cmd.Argv()
	remote := RemoteCommand(target.Locator.Workdir, argv[1:])
	if cmd.IsObservation() && cmd.Observation == nil {
		remote = ObservationCommand(argv)
	}
	opts := DefaultRunOptions()
	if cmd.Timeout > 0 {
		opts.CommandTimeout = cmd.Timeout
	} else if a.DefaultTimeout > 0 {
		opts.CommandTimeout = a.DefaultTimeout
	}
	if len(cmd.Stdin) > 0 {
		opts.Stdin = cmd.Stdin
	}
	return a.run(ctx, target, cfg, remote, opts)
}

// Deliver copies each file with scp into its parent directory (created
// first), verifies the remote sha256 against the local bytes and applies the
// requested mode. A digest mismatch is a transport failure: the bytes on the
// target are not the bytes that were sent.
func (a *Adapter) Deliver(ctx context.Context, target identity.TargetRef, delivery reach.Delivery) (reach.DeliveryReceipt, error) {
	if err := reach.ValidateDelivery(delivery); err != nil {
		return reach.DeliveryReceipt{}, err
	}
	cfg, err := a.connect(ctx, target)
	if err != nil {
		return reach.DeliveryReceipt{}, err
	}
	if a.SCP == nil {
		return reach.DeliveryReceipt{}, a.unavailable(target, "ssh adapter has no artifact copier", nil)
	}
	receipt := reach.DeliveryReceipt{Transport: identity.TransportSSH}
	for _, file := range delivery.Files {
		localSum, err := fileSHA256(file.LocalPath)
		if err != nil {
			return receipt, &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportSSH, Target: target.Key(), Detail: "local artifact unreadable: " + file.LocalPath, Err: err}
		}
		if file.SHA256 != "" && !strings.EqualFold(file.SHA256, localSum) {
			return receipt, &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportSSH, Target: target.Key(), Detail: "local artifact digest does not match the declared sha256: " + file.LocalPath}
		}
		if _, err := a.run(ctx, target, cfg, "mkdir -p "+shellutil.QuoteSingle(path.Dir(file.RemotePath)), DefaultRunOptions()); err != nil {
			return receipt, err
		}
		if err := a.SCP.Copy(ctx, cfg, file.LocalPath, file.RemotePath, DefaultSCPOptions()); err != nil {
			return receipt, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Detail: "artifact copy failed: " + file.RemotePath, Err: err}
		}
		res, err := a.run(ctx, target, cfg, "sha256sum -- "+shellutil.QuoteSingle(file.RemotePath), DefaultRunOptions())
		if err != nil {
			return receipt, err
		}
		observed := ""
		if fields := strings.Fields(res.Stdout); len(fields) > 0 {
			observed = strings.ToLower(fields[0])
		}
		if res.ExitCode != 0 || observed != localSum {
			return receipt, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Detail: fmt.Sprintf("delivered bytes at %s hash to %q, expected %s", file.RemotePath, observed, localSum)}
		}
		if file.Mode != 0 {
			if _, err := a.run(ctx, target, cfg, fmt.Sprintf("chmod %04o %s", file.Mode&0o7777, shellutil.QuoteSingle(file.RemotePath)), DefaultRunOptions()); err != nil {
				return receipt, err
			}
		}
		receipt.Files = append(receipt.Files, reach.DeliveredFile{Role: file.Role, RemotePath: file.RemotePath, SHA256: localSum})
	}
	return receipt, nil
}

// Negotiate probes the target platform and asks for its vrooli binary.
// Absence of the binary is protocol_unsupported (the host is online and its
// platform is known, which is what delivery needs); an unreachable host is
// target_offline. SSH carries no scope vocabulary of its own: the held
// scopes are the operator's, which the authz boundary has already enforced
// before reach is entered.
func (a *Adapter) Negotiate(ctx context.Context, target identity.TargetRef) (reach.Capabilities, error) {
	caps := reach.Capabilities{Transport: identity.TransportSSH}
	cfg, err := a.connect(ctx, target)
	if err != nil {
		return caps, err
	}
	probe, err := a.run(ctx, target, cfg, platformProbe, DefaultRunOptions())
	if err != nil {
		return caps, err
	}
	caps.Online = true
	if platform, ok := ParsePlatform(probe.Stdout); ok {
		caps.Platform = platform
	}
	res, err := a.Exec(ctx, target, reach.Command{Verb: "cloud-target", Args: []string{"--help"}, Timeout: 20 * time.Second})
	if err != nil {
		return caps, err
	}
	caps.NativeCLI = res.ExitCode == 0
	caps.ProtocolVersion = "cloud-target/v1"
	if !caps.NativeCLI {
		return caps, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportSSH, Target: target.Key(), Detail: "target vrooli binary does not expose cloud-target"}
	}
	return caps, nil
}

// ParsePlatform turns `uname -s && uname -m` output into "<goos>/<goarch>".
// Only the platforms a release can carry are recognised.
func ParsePlatform(output string) (string, bool) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return "", false
	}
	kernel := strings.ToLower(strings.TrimSpace(lines[0]))
	machine := strings.ToLower(strings.TrimSpace(lines[1]))
	if kernel != "linux" {
		return "", false
	}
	switch machine {
	case "x86_64", "amd64":
		return "linux/amd64", true
	case "aarch64", "arm64":
		return "linux/arm64", true
	}
	return "", false
}

// firstLine is the first non-empty line of a stream, for a refusal detail.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p) //nolint:gosec // validated local artifact path
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
