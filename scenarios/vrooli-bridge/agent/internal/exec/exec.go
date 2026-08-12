// Package exec is the node-agent's job runner (OT-P0-004 node side / OT-P0-007).
// It receives a typed channel.JobPush, builds an argv — never a shell string —
// and executes the node's local `vrooli` CLI AS THE NON-PRIVILEGED RUNNER,
// streaming status/log/exit RunEvents back to the control plane. There is no
// code path here that constructs or runs `sh -c`: BuildArgv produces a
// []string{bin, tokens...} that CommandRunner hands directly to os/exec, and any
// token carrying a shell metacharacter is rejected before execution. Privileged
// provisioning is a structurally separate helper (Phase 4), never this runner.
package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// shellMetachars are characters with shell meaning. A typed job never reaches a
// shell, so any token containing one is rejected as defence in depth — it can
// only be an attempt to smuggle a shell construct through a typed field. This
// mirrors the control-plane dispatch allowlist (api internal/dispatch); the two
// are intentionally duplicated because the agent is a separate Go module that
// cannot import the api package.
const shellMetachars = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~"

// rejectExitCode is reported when a job is rejected before execution (bad verb,
// unsafe token). startFailureExitCode is reported when the command cannot start.
const (
	rejectExitCode       = 126
	startFailureExitCode = 127
)

// EventReporter delivers one RunEvent back to the control plane. The channel
// package wires the production implementation (a signed RunsService
// ReportRunEvent call); tests substitute a collector.
type EventReporter interface {
	Report(ctx context.Context, ev *sharedv1.RunEvent) error
}

// ArtifactUploader is the authenticated node-to-control-plane seam for small
// produced evidence files. It is deliberately separate from EventReporter so
// run lifecycle events remain replayable and artifact bytes have their own
// bounded request contract.
type ArtifactUploader interface {
	Upload(ctx context.Context, runID, name, mediaType string, data []byte) (artifactRef string, err error)
}

// CommandRunner executes an argv in dir, streaming combined stdout/stderr to
// onLog line-by-line, and returns the process exit code. Production wires
// osCommandRunner (command.go); tests substitute a fake. It NEVER receives a
// shell string — only a pre-split argv.
type CommandRunner interface {
	Run(ctx context.Context, argv []string, dir string, onLog func(string)) (exitCode int, err error)
}

// Runner executes typed jobs and reports their progress.
type Runner struct {
	bin         string
	workDir     string
	command     CommandRunner
	reporter    EventReporter
	uploader    ArtifactUploader
	artifactDir string
	now         func() time.Time
}

// Option customises a Runner.
type Option func(*Runner)

// WithClock overrides the time source (tests).
func WithClock(now func() time.Time) Option { return func(r *Runner) { r.now = now } }

// WithCommandRunner overrides the command execution seam (tests).
func WithCommandRunner(c CommandRunner) Option { return func(r *Runner) { r.command = c } }

// WithArtifactUploader wires the node-authenticated produced-artifact sink.
func WithArtifactUploader(u ArtifactUploader) Option { return func(r *Runner) { r.uploader = u } }

// WithArtifactDir selects the private directory used for typed output files.
func WithArtifactDir(dir string) Option { return func(r *Runner) { r.artifactDir = dir } }

// NewRunner constructs a Runner. bin is the local vrooli CLI (default "vrooli"),
// workDir the directory jobs run in, reporter the event sink.
func NewRunner(bin, workDir string, reporter EventReporter, opts ...Option) *Runner {
	r := &Runner{
		bin:         strings.TrimSpace(bin),
		workDir:     workDir,
		reporter:    reporter,
		command:     osCommandRunner{},
		artifactDir: filepath.Join(os.TempDir(), "vrooli-bridge-artifacts"),
		now:         time.Now,
	}
	if r.bin == "" {
		r.bin = "vrooli"
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// BuildArgv translates a typed job into the argv the runner executes:
//
//	[bin] + <verb tokens split on whitespace> + [scenario] + [args...]
//
// e.g. JobPush{verb:"scenario test", scenario:"web-search"} → ["vrooli",
// "scenario", "test", "web-search"]. It is pure and exported so the
// no-shell-path property is directly testable. It returns an error (not an
// argv) when the verb is empty or ANY token carries a shell metacharacter — so
// a malicious typed field can never reach execution.
func BuildArgv(bin string, job *channelv1.JobPush) ([]string, error) {
	if job == nil {
		return nil, fmt.Errorf("nil job")
	}
	verb := strings.TrimSpace(job.GetVerb())
	if verb == "" {
		return nil, fmt.Errorf("job verb is required")
	}

	verbTokens := strings.Fields(verb)
	argv := make([]string, 0, len(verbTokens)+len(job.GetArgs())+2)
	argv = append(argv, strings.TrimSpace(bin))
	argv = append(argv, verbTokens...)
	if sc := strings.TrimSpace(job.GetScenario()); sc != "" {
		argv = append(argv, sc)
	}
	argv = append(argv, job.GetArgs()...)

	// Validate every token EXCEPT the binary (argv[0]) for shell metacharacters.
	// The binary is operator-configured, not job-supplied.
	for _, tok := range argv[1:] {
		if i := strings.IndexAny(tok, shellMetachars); i >= 0 {
			return nil, fmt.Errorf("unsafe token %q: contains shell metacharacter %q", tok, string(tok[i]))
		}
	}
	return argv, nil
}

// Execute runs the job and streams its lifecycle back as RunEvents: a STATUS
// "running" (or a rejection STATUS), LOG chunks, and a terminal EXIT. It never
// returns the run's failure as a Go error — a non-zero exit is reported via the
// EXIT event, not the return value; the return error is reserved for a reporter
// transport failure the caller may log.
func (r *Runner) Execute(ctx context.Context, job *channelv1.JobPush) error {
	var seq uint64
	emit := func(ev *sharedv1.RunEvent) error {
		seq++
		ev.RunId = job.GetRunId()
		ev.Sequence = seq
		ev.EmittedAt = timestamppb.New(r.now().UTC())
		return r.reporter.Report(ctx, ev)
	}

	argv, outputs, err := r.buildArgvWithOutputs(job)
	if err != nil {
		_ = emit(statusEvent("rejected: " + err.Error()))
		return emit(exitEvent(rejectExitCode))
	}

	if err := emit(statusEvent("running")); err != nil {
		return err
	}

	runCtx := ctx
	if job.GetTimeoutSeconds() > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(job.GetTimeoutSeconds())*time.Second)
		defer cancel()
	}

	exitCode, runErr := r.command.Run(runCtx, argv, r.workDir, func(chunk string) {
		_ = emit(logEvent(chunk))
	})
	if runErr != nil && exitCode == 0 {
		// The process could not start (or was killed without yielding a code).
		_ = emit(statusEvent("error: " + runErr.Error()))
		exitCode = startFailureExitCode
	}
	if exitCode == 0 {
		for _, output := range outputs {
			data, readErr := os.ReadFile(output.path)
			if readErr != nil {
				_ = emit(statusEvent("artifact upload failed: " + readErr.Error()))
				return emit(exitEvent(startFailureExitCode))
			}
			if output.maxBytes > 0 && int64(len(data)) > output.maxBytes {
				_ = emit(statusEvent(fmt.Sprintf("artifact %q exceeds its byte limit", output.name)))
				return emit(exitEvent(rejectExitCode))
			}
			if r.uploader == nil {
				_ = emit(statusEvent("artifact upload failed: no uploader configured"))
				return emit(exitEvent(startFailureExitCode))
			}
			ref, uploadErr := r.uploader.Upload(runCtx, job.GetRunId(), output.name, output.mediaType, data)
			if uploadErr != nil {
				_ = emit(statusEvent("artifact upload failed: " + uploadErr.Error()))
				return emit(exitEvent(startFailureExitCode))
			}
			if err := emit(&sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_ARTIFACT_REF, ArtifactRef: ref}); err != nil {
				return err
			}
		}
	}
	return emit(exitEvent(exitCode))
}

type preparedOutput struct {
	name      string
	mediaType string
	maxBytes  int64
	path      string
}

func (r *Runner) buildArgvWithOutputs(job *channelv1.JobPush) ([]string, []preparedOutput, error) {
	argv, err := BuildArgv(r.bin, job)
	if err != nil {
		return nil, nil, err
	}
	outputs := make([]preparedOutput, 0, len(job.GetOutputs()))
	if len(job.GetOutputs()) > 0 {
		if strings.TrimSpace(job.GetRunId()) == "" {
			return nil, nil, fmt.Errorf("artifact output requires a run id")
		}
		if err := os.MkdirAll(r.artifactDir, 0o700); err != nil {
			return nil, nil, fmt.Errorf("create artifact directory: %w", err)
		}
		for _, spec := range job.GetOutputs() {
			name := strings.TrimSpace(spec.GetName())
			flag := strings.TrimSpace(spec.GetOutputFlag())
			if name == "" || flag == "" || strings.ContainsAny(name, `/\\`) || strings.ContainsAny(flag, " \t\r\n") {
				return nil, nil, fmt.Errorf("invalid typed artifact output")
			}
			path := filepath.Join(r.artifactDir, job.GetRunId()+"-"+name)
			argv = append(argv, flag, path)
			outputs = append(outputs, preparedOutput{name: name, mediaType: strings.TrimSpace(spec.GetMediaType()), maxBytes: spec.GetMaxBytes(), path: path})
		}
	}
	return argv, outputs, nil
}

func statusEvent(status string) *sharedv1.RunEvent {
	return &sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS, Status: status}
}

func logEvent(chunk string) *sharedv1.RunEvent {
	return &sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_LOG, LogChunk: chunk}
}

func exitEvent(code int) *sharedv1.RunEvent {
	return &sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT, ExitCode: int32(code)} // #nosec G115 -- runner exit codes are bounded OS process statuses.
}
