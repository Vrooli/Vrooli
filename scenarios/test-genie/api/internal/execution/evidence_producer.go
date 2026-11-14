package execution

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// EvidenceProducerCommand is copied from a validated descriptor by the
// validation broker. It is never decoded from caller argv or environment.
type EvidenceProducerCommand struct {
	Provider, Name                   string
	Argv                             []string
	WorkingDirectory, OutputRoot     string
	Timeout                          time.Duration
	MaximumOutputBytes               int64
	MutatesLifecycle                 bool
	DescriptorDigest, SourceIdentity string
}

type boundedLog struct {
	mu             sync.Mutex
	w              io.Writer
	limit, written int64
	truncated      bool
}

func (w *boundedLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	original := len(p)
	remaining := w.limit - w.written
	if remaining > 0 {
		part := p
		if int64(len(part)) > remaining {
			part = part[:remaining]
		}
		n, err := w.w.Write(part)
		w.written += int64(n)
		if err != nil {
			return n, err
		}
		if n != len(part) {
			return n, io.ErrShortWrite
		}
	}
	if int64(original) > remaining {
		w.truncated = true
	}
	return original, nil
}

type producerLogMetadata struct {
	Provider            string    `json:"provider"`
	Producer            string    `json:"producer"`
	RunID               string    `json:"runId"`
	IntentDigest        string    `json:"intentDigest"`
	DescriptorDigest    string    `json:"descriptorDigest"`
	SourceIdentity      string    `json:"sourceIdentity"`
	TimeoutMilliseconds int64     `json:"timeoutMilliseconds"`
	BytesWritten        int64     `json:"bytesWritten"`
	Truncated           bool      `json:"truncated"`
	ExitCode            int       `json:"exitCode"`
	CompletedAt         time.Time `json:"completedAt"`
}

func (s *SuiteExecutionService) runEvidenceProducer(ctx context.Context, input SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error) {
	command := input.EvidenceProducer
	if command == nil || len(command.Argv) == 0 || command.Timeout <= 0 || command.MaximumOutputBytes <= 0 {
		return nil, errors.New("pinned evidence producer command is incomplete")
	}
	if runtime.GOOS != "linux" {
		return nil, errors.New("evidence producer containment is supported only on Linux")
	}
	if err := preflightBubblewrap(ctx); err != nil {
		return nil, fmt.Errorf("evidence producer containment unavailable: %w", err)
	}
	if input.ArtifactRoot == "" || input.Request.RunID == "" {
		return nil, errors.New("evidence producer run artifact identity is missing")
	}
	started := time.Now().UTC()
	artifactDir := sharedartifacts.RunDir(input.ArtifactRoot, input.Request.RunID)
	logDir := filepath.Join(artifactDir, "evidence-producer")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create producer artifact directory: %w", err)
	}
	logPath := filepath.Join(logDir, "output.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	logLimit := command.MaximumOutputBytes
	if logLimit > 1<<20 {
		logLimit = 1 << 20
	}
	bounded := &boundedLog{w: logFile, limit: logLimit}
	working, err := filepath.EvalSymlinks(command.WorkingDirectory)
	if err != nil || working != command.WorkingDirectory {
		return nil, errors.Join(errors.New("producer working directory changed after admission"), logFile.Close())
	}
	ctx, cancel := context.WithTimeout(ctx, command.Timeout)
	defer cancel()
	cmd, ready, release, finishGate, closePipes, stdoutDone, gateErr := containedProducerCommand(ctx, working, command.MaximumOutputBytes, command.Argv, bounded)
	if gateErr != nil {
		return nil, errors.Join(gateErr, logFile.Close())
	}
	cmd.Stderr = bounded
	if err = cmd.Start(); err == nil {
		var outputHandle *os.File
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case gate := <-ready:
			gateErr = gate.err
			if gateErr != nil {
				err = gateErr
			} else {
				childPID, pidErr := containedHostChildPID(cmd.Process.Pid)
				if pidErr != nil {
					err = pidErr
				} else {
					outputHandle, err = os.Open(fmt.Sprintf("/proc/%d/root/dev/shm/tg-output", childPID))
				}
				if err == nil {
					err = release()
				}
			}
		}
		if err != nil {
			_ = cmd.Process.Kill()
		}
		finishGate()
		// StdoutPipe must drain before Wait closes it. Cancellation kills the
		// namespace owner, so failure paths also reach EOF before reaping.
		stdoutErr := <-stdoutDone
		if stdoutErr != nil {
			_ = cmd.Process.Kill()
		}
		err = errors.Join(err, stdoutErr, cmd.Wait())
		if outputHandle != nil {
			if err == nil {
				err = publishProducerOutput(outputHandle, filepath.Join(logDir, "output"), command.MaximumOutputBytes)
			}
			err = errors.Join(err, outputHandle.Close())
		}
	} else {
		finishGate()
		closePipes()
		<-stdoutDone
	}
	closeErr := logFile.Close()
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	meta := producerLogMetadata{Provider: command.Provider, Producer: command.Name, RunID: input.Request.RunID, IntentDigest: input.Request.EvidenceProductionIntentDigest, DescriptorDigest: command.DescriptorDigest, SourceIdentity: command.SourceIdentity, TimeoutMilliseconds: command.Timeout.Milliseconds(), BytesWritten: bounded.written, Truncated: bounded.truncated, ExitCode: exitCode, CompletedAt: time.Now().UTC()}
	metaBytes, metaErr := json.Marshal(meta)
	if metaErr == nil {
		metaErr = os.WriteFile(filepath.Join(logDir, "metadata.json"), metaBytes, 0o600)
	}
	err = errors.Join(err, closeErr, metaErr)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("producer command stopped: %w", ctx.Err())
		}
		s.recordTerminalOutcome(context.Background(), input, started, classifyTerminalError(ctx, err))
		return nil, err
	}
	completed := time.Now().UTC()
	result := &orchestrator.SuiteExecutionResult{RunID: input.Request.RunID, ScenarioName: input.Request.ScenarioName, ArtifactDir: artifactDir, RequestedAt: input.Request.RequestedAt, StartedAt: started, CompletedAt: completed, Success: true, Verdict: "PRODUCER_COMPLETED", FailureReason: "", RequestedPhases: []string{}, PlannedPhases: []string{}}
	if s.executions == nil {
		return nil, errors.New("suite execution repository is not configured")
	}
	if err := s.executions.Create(ctx, &SuiteExecutionRecord{RunID: result.RunID, ScenarioName: result.ScenarioName, Success: true, RequestedAt: result.RequestedAt, StartedAt: started, CompletedAt: completed, Phases: []phases.ExecutionResult{}, PreparationStages: []orchestrator.PreparationStage{}}); err != nil {
		return nil, err
	}
	return result, nil
}

type producerReady struct {
	err error
}

func containedHostChildPID(ownerPID int) (int, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", ownerPID, ownerPID))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) != 1 {
		return 0, fmt.Errorf("expected exactly one contained producer child after readiness, got %d", len(fields))
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil || pid <= 0 {
		return 0, errors.New("contained producer child pid is invalid")
	}
	ownerNS, err := os.Stat(fmt.Sprintf("/proc/%d/ns/pid", ownerPID))
	if err != nil {
		return 0, fmt.Errorf("verify containment owner pid namespace: %w", err)
	}
	childNS, err := os.Stat(fmt.Sprintf("/proc/%d/ns/pid", pid))
	if err != nil {
		return 0, fmt.Errorf("verify producer child pid namespace: %w", err)
	}
	if os.SameFile(ownerNS, childNS) {
		return 0, errors.New("producer child is not in a verified private pid namespace")
	}
	return pid, nil
}

func containedProducerCommand(ctx context.Context, working string, outputLimit int64, argv []string, output io.Writer) (*exec.Cmd, <-chan producerReady, func() error, func(), func(), <-chan error, error) {
	ready := make(chan producerReady, 1)
	stdoutGate := make(chan struct{})
	scratchMount, outputMount := "/dev/shm/tg-scratch", "/dev/shm/tg-output"
	path := trustedProducerPath()
	moduleCache := trustedGoModuleCache()
	args := []string{"--die-with-parent", "--unshare-pid", "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc", "--dir", scratchMount, "--size", "536870912", "--tmpfs", scratchMount, "--dir", outputMount, "--size", strconv.FormatInt(outputLimit, 10), "--tmpfs", outputMount, "--clearenv", "--setenv", "PATH", path, "--setenv", "HOME", scratchMount, "--setenv", "TMPDIR", scratchMount, "--setenv", "GOCACHE", scratchMount + "/go-build", "--setenv", "GOPROXY", "off", "--setenv", "GOTOOLCHAIN", "local"}
	if moduleCache != "" {
		args = append(args, "--setenv", "GOMODCACHE", moduleCache)
	}
	args = append(args, "--chdir", working, "--", "/bin/sh", "-c", "printf 'TG_OWNER_READY\\n'; IFS= read -r gate; [ \"$gate\" = go ] || exit 125; exec \"$@\"", "tg-owner-gate")
	args = append(args, argv...)
	cmd := exec.CommandContext(ctx, "bwrap", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, nil, nil, nil, nil, nil, err
	}
	stdoutDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(stdout)
		line, e := reader.ReadString('\n')
		if e == nil && line != "TG_OWNER_READY\n" {
			e = errors.New("producer readiness handshake invalid")
		}
		ready <- producerReady{err: e}
		<-stdoutGate
		if e == nil {
			_, e = io.Copy(output, reader)
		}
		stdoutDone <- e
	}()
	release := func() error {
		_, e := io.WriteString(stdin, "go\n")
		if e == nil {
			e = stdin.Close()
		}
		close(stdoutGate)
		return e
	}
	finish := func() {
		_ = stdin.Close()
		select {
		case <-stdoutGate:
		default:
			close(stdoutGate)
		}
	}
	closePipes := func() { _ = stdin.Close(); _ = stdout.Close() }
	return cmd, ready, release, finish, closePipes, stdoutDone, nil
}

func trustedGoModuleCache() string {
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func publishProducerOutput(root *os.File, destination string, limit int64) error {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	tmp := destination + ".partial"
	_ = os.RemoveAll(tmp)
	if err := os.Mkdir(tmp, 0o700); err != nil {
		return err
	}
	type item struct {
		src, dst string
		info     os.FileInfo
	}
	queue := []item{{src: "", dst: tmp}}
	var total int64
	entries := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		dir := filepath.Join("/proc/self/fd", fmt.Sprint(root.Fd()), cur.src)
		directory, err := os.Open(dir)
		if err != nil {
			_ = os.RemoveAll(tmp)
			return err
		}
		for {
			children, readErr := directory.ReadDir(1)
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				err = readErr
				break
			}
			child := children[0]
			entries++
			if entries > 4096 {
				err = errors.New("producer output exceeds 4096 entries")
				break
			}
			src := filepath.Join(cur.src, child.Name())
			dst := filepath.Join(cur.dst, child.Name())
			var info os.FileInfo
			info, err = os.Lstat(filepath.Join(dir, child.Name()))
			if err != nil {
				break
			}
			switch {
			case info.Mode().IsDir():
				if err = os.Mkdir(dst, 0o700); err == nil {
					queue = append(queue, item{src: src, dst: dst})
				}
			case info.Mode().IsRegular():
				total += info.Size()
				if total > limit {
					err = errors.New("producer output exceeds declared byte limit")
					break
				}
				var in *os.File
				in, err = os.Open(filepath.Join(dir, child.Name()))
				if err == nil {
					var out *os.File
					out, err = os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
					if err == nil {
						_, err = io.CopyN(out, in, info.Size())
						err = errors.Join(err, out.Close())
					}
					err = errors.Join(err, in.Close())
				}
			default:
				err = errors.New("producer output contains unsupported file type or link")
			}
			if err != nil {
				break
			}
		}
		closeErr := directory.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.RemoveAll(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, destination); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return nil
}

// trustedProducerPath derives runtime discovery from the service owner's
// process environment and Vrooli's installed tool directory. Request data is
// never allowed to modify this path.
func trustedProducerPath() string {
	entries := filepath.SplitList(os.Getenv("PATH"))
	if home, err := os.UserHomeDir(); err == nil {
		entries = append(entries, filepath.Join(home, ".vrooli", "bin"))
	}
	return strings.Join(entries, string(os.PathListSeparator))
}

func preflightBubblewrap(ctx context.Context) error {
	probe := exec.CommandContext(ctx, "bwrap", "--die-with-parent", "--unshare-pid", "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc", "--", "/bin/true")
	output, err := probe.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bubblewrap PID namespace probe failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// CheckEvidenceProducerContainment fails closed when the whole-tree Linux
// backend cannot create its PID namespace. Call it during broker admission.
func CheckEvidenceProducerContainment(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("evidence producer containment is supported only on Linux")
	}
	return preflightBubblewrap(ctx)
}
