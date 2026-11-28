package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/vroolicli/sessionlease"
	"github.com/vrooli/vrooli/internal/recovery"
	codingagentshims "github.com/vrooli/vrooli/internal/safeguards/coding-agent-shims"
)

func writeCodingAgentRecoveryCapability(args []string, stdout io.Writer) (bool, int) {
	if len(args) != 1 || args[0] != "--coding-agent-recovery-capability" {
		return false, 0
	}
	_, err := fmt.Fprintln(stdout, `{"protocol":1}`)
	if err != nil {
		return true, 1
	}
	return true, 0
}

// The retained root image executes this one admitted job before ordinary CLI
// initialization. Its descendants stay in the native job; launching a sibling
// agent scope here would escape the job's deadline and loss reconciliation.
func runCodingAgentRecoveryWorker(args []string, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "--coding-agent-recovery-execute" {
		return false, 0
	}
	if len(args) != 3 {
		fmt.Fprintln(stderr, "invalid native recovery worker invocation")
		return true, 2
	}
	if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
		fmt.Fprintln(stderr, "native recovery authority unavailable")
		return true, 1
	}
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil || path != args[2] {
		fmt.Fprintln(stderr, "native recovery ledger authority mismatch")
		return true, 1
	}
	b, err := recovery.NewCodingAgentBroker(path, nil)
	if err != nil {
		fmt.Fprintln(stderr, "native recovery ledger unavailable")
		return true, 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), recovery.CodingAgentRecoveryTimeout)
	defer cancel()
	e, err := b.ClaimNativeExecution(ctx, args[1])
	if err != nil {
		fmt.Fprintln(stderr, "native recovery claim refused")
		return true, 1
	}
	// This early entrypoint bypasses ordinary CLI initialization. Reuse its
	// managed-tool PATH recovery before resolving a real native runner.
	recoverUserToolPath()
	recorder := cliutil.DefaultEditorLeaseRecorder
	if recorder == nil {
		sessionlease.Register()
		recorder = cliutil.DefaultEditorLeaseRecorder
	}
	cliutil.RegisterEditorLeaseRecorder(recoveryLeaseRecorder{broker: b, execution: e, delegate: recorder})
	err = b.RunNativeExecution(ctx, e, func(ctx context.Context, runner string, req recovery.CodingAgentRequest) error {
		_, err := cliutil.CodingAgentBinary(runner)
		if err != nil {
			return recovery.ErrCodingAgentUnavailable
		}
		prompt := recovery.CodingAgentPrompt(req)
		_, err = cliutil.LaunchCodingAgentResult(ctx, cliutil.AgentLaunchRequest{Agent: runner, Args: recovery.CodingAgentArguments(runner, prompt), WorkingDir: req.WorkingDir, APIBase: e.AttachBase, Claims: []string{filepath.Join(req.WorkingDir, req.Claim)}, Stdout: io.Discard, Stderr: io.Discard,
			LookPathInEnvironment: func(binary string, environment []string) (string, error) {
				shim, err := codingagentshims.LauncherPath()
				if err != nil {
					return "", err
				}
				path, err := cliutil.ResolveAgentBinaryExcludingInEnvironment(binary, shim, environment)
				if err != nil {
					return "", recovery.ErrCodingAgentUnavailable
				}
				return path, nil
			},
			RunChild: func(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
				if err := b.GuardNativeChild(ctx, e.ID); err != nil {
					return err
				}
				cmd := exec.CommandContext(ctx, path, args...)
				cmd.Dir = req.WorkingDir
				cmd.Env = env
				cmd.Stdin = stdin
				cmd.Stdout = stdout
				cmd.Stderr = stderr
				cmd.WaitDelay = time.Second
				// KillMode=control-group and RuntimeMaxSec on the independent job
				// own descendant exclusion, including after this wrapper is lost.
				if err := cmd.Run(); err != nil {
					return errors.New("coding-agent child did not complete successfully")
				}
				return nil
			}})
		if errors.Is(err, recovery.ErrCodingAgentUnavailable) {
			return recovery.ErrCodingAgentUnavailable
		}
		if err != nil {
			return errors.New("coding-agent launch outcome unconfirmed")
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(stderr, "native recovery execution did not finish")
		return true, 1
	}
	return true, 0
}

type recoveryLeaseRecorder struct {
	broker    *recovery.CodingAgentBroker
	execution recovery.CodingAgentExecution
	delegate  cliutil.EditorLeaseRecorder
}

func (r recoveryLeaseRecorder) Create(ctx context.Context, record cliutil.EditorLeaseRecord) error {
	if err := r.broker.BindNativeLease(ctx, r.execution.ID, record.SessionID, false); err != nil {
		return err
	}
	record.Scope = r.execution.Ref.Scope.String()
	record.PID = r.execution.Ref.RootPID
	record.ContainmentMethod = "owned-operation"
	if err := r.delegate.Create(ctx, record); err != nil {
		return err
	}
	return r.broker.BindNativeLease(ctx, r.execution.ID, record.SessionID, true)
}
func (r recoveryLeaseRecorder) Heartbeat(ctx context.Context, id string) error {
	return r.delegate.Heartbeat(ctx, id)
}
func (r recoveryLeaseRecorder) Stop(ctx context.Context, id, reason string) error {
	return r.delegate.Stop(ctx, id, reason)
}
func (r recoveryLeaseRecorder) Overlaps(ctx context.Context, claims []string) ([]cliutil.ClaimHolder, error) {
	return r.delegate.Overlaps(ctx, claims)
}
