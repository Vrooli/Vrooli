package decisionpage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Job states.
const (
	// StateQueued: the job waits for a free writer slot.
	StateQueued = "queued"
	// StateWorking: a writer run is on it.
	StateWorking = "working"
	// StateReady: page.html passed the check and can be sent.
	StateReady = "ready"
	// StateFailed: no page will come from this job; Failure says why. A
	// failure of class unavailable or launch_refused is retried after
	// RetryAfter: the writer is not there yet, which says nothing of the page.
	StateFailed = "failed"
)

// Failure classes: the typed outcomes a sender branches on.
const (
	FailInvalid     = "invalid_request" // the request cannot be written
	FailRefused     = "launch_refused"  // no authority to start the run (no service credential, or Agent Manager refused it); retried later
	FailUnavailable = "unavailable"     // Agent Manager or the Hub could not be reached; retried later
	FailRun         = "run_failed"      // both runs ended without a page
	FailLint        = "lint_failed"     // the second page still failed the Hub's check
)

// MaxAttempts is how many writer runs one job gets: the first, and one more
// with the first one's findings.
const MaxAttempts = 2

// Failure is why a job will not produce a page.
type Failure struct {
	Class  string    `json:"class"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}

func (f *Failure) Error() string { return f.Class + ": " + f.Detail }

// Finding is one result of the Hub's packet check.
type Finding struct {
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

// Attempt is one writer run.
type Attempt struct {
	N         int       `json:"n"`
	RunID     string    `json:"run_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Model     string    `json:"model,omitempty"`
	RunStatus string    `json:"run_status,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitzero"`
	// Problem is why the attempt produced no usable page, besides findings.
	Problem string `json:"problem,omitempty"`
	// Findings are the page findings that failed it.
	Findings []Finding `json:"findings,omitempty"`
}

// Result is a job's state as a sender reads it.
type Result struct {
	State    string    `json:"state"`
	Dir      string    `json:"dir"`
	Page     string    `json:"page,omitempty"`
	HTML     string    `json:"-"`
	SHA256   string    `json:"sha256,omitempty"`
	Failure  *Failure  `json:"failure,omitempty"`
	Attempts []Attempt `json:"attempts,omitempty"`
	// Advice is what the Hub's check says about the cover (question, big
	// picture, recommendation); the writer cannot change it, the sender can.
	Advice    []Finding `json:"advice,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Terminal reports whether the job will not change without a new request.
func (r Result) Terminal() bool {
	return r.State == StateReady || (r.State == StateFailed && !retried(r.Failure))
}

// RunRef is the launcher's view of one run.
type RunRef struct {
	ID     string
	TaskID string
	Status string
	Model  string
	// Ended is true once the run will not change; Succeeded when it ended well.
	Ended, Succeeded bool
	Error            string
}

// LaunchSpec is one writer run to start.
type LaunchSpec struct {
	Title          string
	Description    string
	Prompt         string
	Tag            string
	IdempotencyKey string
}

// Launcher starts and reads writer runs (Agent Manager in production).
type Launcher interface {
	Launch(ctx context.Context, spec LaunchSpec) (RunRef, error)
	Run(ctx context.Context, id string) (RunRef, error)
	Stop(ctx context.Context, id string) error
}

// Checker runs the Hub's packet check on a page with its decision.
type Checker interface {
	Check(ctx context.Context, r Request, html []byte) ([]Finding, error)
}

// ErrRefused marks a launch the run owner refused (identity, grant, policy);
// ErrUnavailable one that could not be attempted.
var (
	ErrRefused     = errors.New("refused")
	ErrUnavailable = errors.New("unavailable")
)

// Writer runs decision-page jobs.
type Writer struct {
	// Root holds the job folders: Root/<sender>/<subject>/<digest>/.
	Root     string
	Launcher Launcher
	Checker  Checker
	// SkillDir is the notification-hub skill folder the brief points to.
	SkillDir string
	// Command is the CLI the brief tells the run to check and photograph
	// its page with ("notification-hub").
	Command string
	Now     func() time.Time
	// AttemptTimeout bounds one run; a run past it is stopped.
	AttemptTimeout time.Duration
	// MaxActive bounds the writer runs working at once across all senders.
	MaxActive int
	// RetryAfter is how long an unavailable job waits before trying again.
	RetryAfter time.Duration
}

// Defaults for a zero Writer field.
const (
	DefaultAttemptTimeout = 25 * time.Minute
	DefaultMaxActive      = 2
	DefaultRetryAfter     = 30 * time.Minute
	DefaultCommand        = "notification-hub"
)

func (w *Writer) now() time.Time {
	if w.Now != nil {
		return w.Now().UTC()
	}
	return time.Now().UTC()
}

func (w *Writer) attemptTimeout() time.Duration {
	if w.AttemptTimeout > 0 {
		return w.AttemptTimeout
	}
	return DefaultAttemptTimeout
}

func (w *Writer) maxActive() int {
	if w.MaxActive > 0 {
		return w.MaxActive
	}
	return DefaultMaxActive
}

func (w *Writer) retryAfter() time.Duration {
	if w.RetryAfter > 0 {
		return w.RetryAfter
	}
	return DefaultRetryAfter
}

func (w *Writer) command() string {
	if w.Command != "" {
		return w.Command
	}
	return DefaultCommand
}

// Dir is the job folder of a request.
func (w *Writer) Dir(r Request) string {
	return filepath.Join(w.Root, slug(r.Sender), slug(r.SubjectKey), r.Digest()[:16])
}

const (
	stateFile   = "state.json"
	requestFile = "request.json"
	lockFile    = ".lock"
)

// job is a job folder's stored state.
type job struct {
	Result
	QueuedAt time.Time `json:"queued_at"`
}

// Status reads a job without changing anything. A request never seen is
// reported queued with no attempts.
func (w *Writer) Status(r Request) (Result, error) {
	if err := r.Validate(); err != nil {
		return Result{State: StateFailed, Failure: &Failure{Class: FailInvalid, Detail: err.Error(), At: w.now()}}, nil
	}
	dir := w.Dir(r)
	j, err := readJob(dir)
	if err != nil {
		return Result{}, err
	}
	if j == nil {
		return Result{State: StateQueued, Dir: dir}, nil
	}
	return w.withPage(j.Result), nil
}

// Advance moves the job one step and returns its state. It never waits on a
// run: it starts one, reads one, or checks a finished page. Safe to call from
// several processes; one job is changed by one caller at a time.
func (w *Writer) Advance(ctx context.Context, r Request) (Result, error) {
	if err := r.Validate(); err != nil {
		return Result{State: StateFailed, Failure: &Failure{Class: FailInvalid, Detail: err.Error(), At: w.now()}}, nil
	}
	dir := w.Dir(r)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("decision-page job folder: %w", err)
	}
	unlock, err := lock(dir)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	j, err := readJob(dir)
	if err != nil {
		return Result{}, err
	}
	if j == nil {
		if err := writeJSON(filepath.Join(dir, requestFile), r); err != nil {
			return Result{}, err
		}
		j = &job{Result: Result{State: StateQueued, Dir: dir}, QueuedAt: w.now()}
	}
	j.Dir = dir
	before := j.UpdatedAt
	if err := w.step(ctx, r, j); err != nil {
		return Result{}, err
	}
	if j.UpdatedAt.IsZero() || j.UpdatedAt.Equal(before) {
		j.UpdatedAt = w.now()
	}
	if err := writeJSON(filepath.Join(dir, stateFile), j); err != nil {
		return Result{}, err
	}
	return w.withPage(j.Result), nil
}

// Write advances the job until it is terminal or wait runs out, pausing poll
// between steps. The returned result may still be working.
func (w *Writer) Write(ctx context.Context, r Request, wait, poll time.Duration) (Result, error) {
	if poll <= 0 {
		poll = 15 * time.Second
	}
	deadline := w.now().Add(wait)
	for {
		res, err := w.Advance(ctx, r)
		if err != nil || res.Terminal() || res.State == StateFailed || !w.now().Before(deadline) {
			return res, err
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return res, ctx.Err()
		case <-timer.C:
		}
	}
}

func (w *Writer) step(ctx context.Context, r Request, j *job) error {
	now := w.now()
	switch j.State {
	case StateReady:
		return nil
	case StateFailed:
		if !retried(j.Failure) || now.Sub(j.Failure.At) < w.retryAfter() {
			return nil
		}
		// The writer was not there (an outage or no authority); try again
		// from where it stopped.
		j.Failure = nil
		j.State = StateQueued
		if n := len(j.Attempts); n > 0 && j.Attempts[n-1].RunID == "" {
			j.Attempts = j.Attempts[:n-1]
		}
		if len(j.Attempts) > 0 {
			j.State = StateWorking
		}
	}
	if len(j.Attempts) == 0 {
		return w.start(ctx, r, j, nil)
	}
	last := &j.Attempts[len(j.Attempts)-1]
	if last.EndedAt.IsZero() {
		run, err := w.Launcher.Run(ctx, last.RunID)
		if err != nil {
			if errors.Is(err, ErrUnavailable) {
				j.State = StateWorking // the run goes on; read it next time
				return nil
			}
			return w.fail(j, FailUnavailable, "reading the writer run: "+err.Error())
		}
		last.RunStatus, last.Model = run.Status, firstNonEmpty(run.Model, last.Model)
		if !run.Ended {
			if now.Sub(last.StartedAt) < w.attemptTimeout() {
				j.State = StateWorking
				return nil
			}
			last.Problem = fmt.Sprintf("the run did not finish within %s and was stopped", w.attemptTimeout())
			if err := w.Launcher.Stop(ctx, last.RunID); err != nil {
				// A service principal's standing grant admits run creation only, so
				// Agent Manager may refuse the stop; say so rather than claim it.
				last.Problem = fmt.Sprintf("the run did not finish within %s and could not be stopped (%v); it may still be running", w.attemptTimeout(), err)
			}
		} else if !run.Succeeded && run.Error != "" {
			last.Problem = "the run ended " + run.Status + ": " + run.Error
		}
		last.EndedAt = now
	}
	// The last attempt ended: judge its page.
	findings, advice, problem, err := w.judge(ctx, r, j.Dir)
	if err != nil {
		return w.fail(j, FailUnavailable, "checking the page with Notification Hub: "+err.Error())
	}
	j.Advice = advice
	if problem == "" && len(findings) == 0 {
		last.Problem, last.Findings = "", nil
		j.State, j.Failure = StateReady, nil
		j.Page = filepath.Join(j.Dir, PageFile)
		return nil
	}
	if problem != "" && last.Problem == "" {
		last.Problem = problem
	} else if problem != "" {
		last.Problem += "; " + problem
	}
	last.Findings = findings
	if len(j.Attempts) >= MaxAttempts {
		if len(findings) > 0 && problem == "" {
			return w.fail(j, FailLint, summarize(findings))
		}
		return w.fail(j, FailRun, last.Problem)
	}
	if err := keepAttempt(j.Dir, last.N); err != nil {
		return err
	}
	return w.start(ctx, r, j, last)
}

// start launches the next attempt, or queues the job when the writer is full.
func (w *Writer) start(ctx context.Context, r Request, j *job, previous *Attempt) error {
	if previous == nil {
		active, err := w.active(j.Dir)
		if err != nil {
			return err
		}
		if active >= w.maxActive() {
			j.State = StateQueued
			return nil
		}
	}
	n := len(j.Attempts) + 1
	brief := renderBrief(briefInput{Dir: j.Dir, Request: r, Attempt: n, Previous: previous, SkillDir: w.SkillDir, Command: w.command()})
	if err := os.WriteFile(filepath.Join(j.Dir, BriefFile), []byte(brief), 0o644); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(j.Dir, PageFile))
	run, err := w.Launcher.Launch(ctx, LaunchSpec{
		Title:       "Decision page: " + clip(r.Question, 120),
		Description: "Write the Notification Hub decision page for " + r.Sender + " (" + r.SubjectKey + ") following " + filepath.Join(j.Dir, BriefFile),
		Prompt:      Prompt(j.Dir),
		Tag:         "decision-page-" + slug(r.SubjectKey),
		// Agent Manager remembers a key for good, so it names this job, not
		// only its facts: a job made again for the same facts is a new run.
		IdempotencyKey: fmt.Sprintf("decision-page-%s-%s-%d", filepath.Base(j.Dir), strconv.FormatInt(j.QueuedAt.UnixNano(), 36), n),
	})
	if err != nil {
		class := FailUnavailable
		if errors.Is(err, ErrRefused) {
			class = FailRefused
		}
		return w.fail(j, class, "starting the writer run: "+err.Error())
	}
	j.Attempts = append(j.Attempts, Attempt{N: n, RunID: run.ID, TaskID: run.TaskID, Model: run.Model, RunStatus: run.Status, StartedAt: w.now()})
	j.State = StateWorking
	return nil
}

// retried reports whether a failure is the writer's absence rather than the
// page's: retried after the cool-down, never a miss.
func retried(f *Failure) bool {
	return f != nil && (f.Class == FailUnavailable || f.Class == FailRefused)
}

func (w *Writer) fail(j *job, class, detail string) error {
	j.State = StateFailed
	j.Failure = &Failure{Class: class, Detail: clip(detail, 2000), At: w.now()}
	return nil
}

// judge checks the page in dir with the writer's checker.
func (w *Writer) judge(ctx context.Context, r Request, dir string) (findings, advice []Finding, problem string, err error) {
	in, err := Inspect(ctx, w.Checker, r, filepath.Join(dir, PageFile))
	return in.Findings, in.Advice, in.Problem, err
}

// Inspection is the gate's verdict on one page: Problem says why it is not a
// decision page at all, Findings are what the Hub's check says about the page
// (each must be fixed), Advice what it says about the rest of the decision.
type Inspection struct {
	Problem  string    `json:"problem,omitempty"`
	Findings []Finding `json:"findings,omitempty"`
	Advice   []Finding `json:"advice,omitempty"`
}

// Passes reports whether the page may be sent.
func (in Inspection) Passes() bool { return in.Problem == "" && len(in.Findings) == 0 }

// Inspect runs the writer's gate on a page: the local checks, then the Hub's
// packet check with the decision. The run that writes a page and the writer
// that accepts it run this same gate. An error means the Hub could not judge.
func Inspect(ctx context.Context, checker Checker, r Request, page string) (Inspection, error) {
	html, readErr := os.ReadFile(page)
	switch {
	case errors.Is(readErr, os.ErrNotExist):
		return Inspection{Problem: "the run wrote no " + PageFile}, nil
	case readErr != nil:
		return Inspection{Problem: "the page cannot be read: " + readErr.Error()}, nil
	}
	if p := pageProblem(r, html); p != "" {
		return Inspection{Problem: p}, nil
	}
	all, err := checker.Check(ctx, r, html)
	if err != nil {
		return Inspection{}, err
	}
	var in Inspection
	for _, f := range all {
		if PageFinding(f) {
			in.Findings = append(in.Findings, f)
		} else {
			in.Advice = append(in.Advice, f)
		}
	}
	return in, nil
}

// ReadRequest reads the request a job folder was made for.
func ReadRequest(dir string) (Request, error) {
	var r Request
	data, err := os.ReadFile(filepath.Join(dir, requestFile))
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(data, &r)
}

// active counts working jobs other than skip, across every sender.
func (w *Writer) active(skip string) (int, error) {
	paths, err := filepath.Glob(filepath.Join(w.Root, "*", "*", "*", stateFile))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range paths {
		dir := filepath.Dir(p)
		if dir == skip {
			continue
		}
		j, err := readJob(dir)
		if err == nil && j != nil && j.State == StateWorking && w.now().Sub(j.UpdatedAt) < 2*w.attemptTimeout() {
			n++
		}
	}
	return n, nil
}

// withPage fills the page of a ready result.
func (w *Writer) withPage(r Result) Result {
	if r.State != StateReady {
		return r
	}
	html, err := os.ReadFile(filepath.Join(r.Dir, PageFile))
	if err != nil {
		r.State = StateFailed
		r.Failure = &Failure{Class: FailRun, Detail: "the checked page is gone: " + err.Error(), At: w.now()}
		return r
	}
	r.Page, r.HTML, r.SHA256 = filepath.Join(r.Dir, PageFile), string(html), sha(html)
	return r
}

// keepAttempt moves an attempt's page, return and shots aside so the next
// run starts clean and can read what the last one made.
func keepAttempt(dir string, n int) error {
	keep := filepath.Join(dir, attemptDir(n))
	if err := os.MkdirAll(keep, 0o755); err != nil {
		return err
	}
	for _, name := range []string{PageFile, ReturnFile, BriefFile, ShotsDir} {
		src := filepath.Join(dir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		_ = os.RemoveAll(filepath.Join(keep, name))
		if err := os.Rename(src, filepath.Join(keep, name)); err != nil {
			return err
		}
	}
	return nil
}

func attemptDir(n int) string { return fmt.Sprintf("attempt-%d", n) }

func readJob(dir string) (*job, error) {
	data, err := os.ReadFile(filepath.Join(dir, stateFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var j job
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("decision-page job %s: %w", dir, err)
	}
	j.Dir = dir
	return &j, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func lock(dir string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(dir, lockFile), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}

func summarize(fs []Finding) string {
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		parts = append(parts, f.Code+": "+f.Message)
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func clip(s string, n int) string {
	s = oneLine(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
