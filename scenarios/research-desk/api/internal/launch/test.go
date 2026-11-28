package launch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"research-desk/internal/home"
)

// TestStance and TestKernel are the launch test's texts: harmless, and a
// kernel as long as a real one (about 600 characters) that completes at once.
const (
	TestStance = "research-desk launch test. This throwaway session only checks that the research desk can start a supervisor through Web Console: there is no research to do, and you must not read files, run tools or start agents. The next message sets a goal that you complete at once.\n\nReply \"ready\" and wait for it."
	TestKernel = "research-desk launch test: this goal only checks that a launch reaches you intact. Do not read files, run tools, start agents or do any other work. Mark this goal complete at once with your goal tool, then reply with the single word done and end your turn. This objective is deliberately about as long as a real research kernel, close to six hundred characters, so the operator learns whether a goal of that length arrives whole through the /goal command and is stored as the objective. Nothing else is needed from you in this session; the desk stops it right after. It tests the desk alone."
)

// TestReport is the outcome of one launch test, kept in the desk state folder.
type TestReport struct {
	At                time.Time         `json:"at"`
	SessionID         string            `json:"session_id"`
	CodexHome         string            `json:"codex_home"`
	Steps             []home.LaunchStep `json:"steps"`
	KernelChars       int               `json:"kernel_chars"`
	StatusAfterGoal   string            `json:"status_after_goal"`
	StatusAfterPause  string            `json:"status_after_pause"`
	StatusAfterResume string            `json:"status_after_resume"`
	StatusFinal       string            `json:"status_final"`
	Stopped           bool              `json:"stopped"`
	FailedStep        string            `json:"failed_step,omitempty"`
	FailedReason      string            `json:"failed_reason,omitempty"`
	ReportPath        string            `json:"-"`
	Running           bool              `json:"-"`
}

const (
	testClaim   = "launch-test"
	testsFolder = "launch-tests"
)

// TestLaunch starts the whole launch path on one throwaway session in the
// background, then pauses, resumes and stops it; LatestTest reads the
// outcome. The session row stays, labelled as a test.
func (s *Service) TestLaunch(ctx context.Context) (TestReport, error) {
	if err := s.Writer.Held(); err != nil {
		return TestReport{}, err
	}
	limits, err := s.Limits()
	if err != nil {
		return TestReport{}, err
	}
	if !s.claim(testClaim) {
		return TestReport{}, fmt.Errorf("%w: a launch test is already running; read it with GetLaunchTest", home.ErrRefused)
	}
	now := s.Clock.Now().UTC()
	t := &tester{s: s, name: testsFolder + "/" + now.Format("20060102T150405Z") + ".json", report: TestReport{At: now, KernelChars: utf8.RuneCountInString(TestKernel)}}
	if err := t.save(); err != nil {
		s.release(testClaim)
		return TestReport{}, err
	}
	first := t.report
	first.Running = true
	// The test outlives the request that started it; GetLaunchTest reads it.
	background := context.WithoutCancel(ctx)
	go func() {
		defer s.release(testClaim)
		t.run(background, Settings{Model: limits.Defaults.SupervisorModel, Effort: limits.Defaults.SupervisorEffort})
	}()
	return first, nil
}

// LatestTest returns the newest launch test report; ReportPath is empty when
// none has run.
func (s *Service) LatestTest() (TestReport, error) {
	entries, err := os.ReadDir(filepath.Join(s.Writer.StateDir(), testsFolder))
	if errors.Is(err, fs.ErrNotExist) {
		return TestReport{}, nil
	}
	if err != nil {
		return TestReport{}, err
	}
	var newest string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") && e.Name() > newest {
			newest = e.Name()
		}
	}
	if newest == "" {
		return TestReport{}, nil
	}
	var report TestReport
	name := testsFolder + "/" + newest
	if _, err := s.Writer.ReadState(name, &report); err != nil {
		return TestReport{}, err
	}
	report.ReportPath = filepath.Join(s.Writer.StateDir(), testsFolder, newest)
	report.Running = s.isRunning(testClaim)
	return report, nil
}

type tester struct {
	s      *Service
	name   string
	report TestReport
}

func (t *tester) save() error {
	path, err := t.s.Writer.WriteState(t.name, t.report)
	t.report.ReportPath = path
	return err
}

func (t *tester) saveOrLog() {
	if err := t.save(); err != nil {
		t.s.Logger.Printf("launch test: report not written: %v", err)
	}
}

func (t *tester) fail(step, reason string) {
	t.report.FailedStep, t.report.FailedReason = step, reason
	t.saveOrLog()
	t.s.Logger.Printf("launch test: failed at %s: %s", step, reason)
}

func (t *tester) run(ctx context.Context, settings Settings) {
	s, report := t.s, &t.report
	stamp := strings.TrimSuffix(strings.TrimPrefix(t.name, testsFolder+"/"), ".json")
	spec := SessionSpec{IdempotencyKey: "research-desk-launch-test/" + stamp, Label: "research-desk launch test " + stamp + " (stopped; keep)"}
	var target Target
	err := s.Launcher.Run(ctx, spec, Texts{Stance: TestStance, Kernel: TestKernel}, settings, func(step, sessionID, threadID string) error {
		target.Session = sessionID
		if threadID != "" {
			target.Thread = threadID
		}
		report.SessionID, report.CodexHome = sessionID, s.Reader.CodexHome(sessionID)
		report.Steps = append(report.Steps, home.LaunchStep{Step: step, At: s.Clock.Now().UTC()})
		t.saveOrLog()
		return nil
	})
	if err != nil {
		if report.SessionID != "" {
			// The window may hold a half-typed message: end the session's
			// processes without typing.
			_, serr := s.Launcher.Stop(ctx, target, false)
			report.Stopped = serr == nil
		}
		t.fail(failure(err))
		return
	}
	patience := s.Launcher.patience()
	report.StatusAfterGoal = known(s.Launcher.GoalStatus(ctx, target))
	t.saveOrLog()
	// The kernel's own turn completes the goal; a control waits for it.
	if err := s.Launcher.AwaitReady(ctx, target, patience.Goal); err != nil {
		t.fail(ActPause, err.Error())
		return
	}
	if report.StatusAfterPause, err = s.Launcher.Pause(ctx, target); err != nil {
		t.fail(ActPause, err.Error())
		return
	}
	if err := s.Launcher.AwaitReady(ctx, target, patience.Goal); err != nil {
		t.fail(ActResume, err.Error())
		return
	}
	if report.StatusAfterResume, err = s.Launcher.Resume(ctx, target); err != nil {
		t.fail(ActResume, err.Error())
		return
	}
	t.saveOrLog()
	report.StatusFinal = known(s.Launcher.AwaitStatus(ctx, target, func(s string) bool { return s == "complete" }, patience.Goal))
	if err := s.Launcher.AwaitReady(ctx, target, patience.Goal); err != nil {
		t.fail(ActStop, err.Error())
		return
	}
	if _, err := s.Launcher.Stop(ctx, target, true); err != nil {
		t.fail(ActStop, err.Error())
		return
	}
	report.Stopped = true
	t.saveOrLog()
}

// known is a goal status for the report: an unreadable one is "unknown"
// with the reason, never blank.
func known(status string, err error) string {
	if err != nil {
		return "unknown: " + err.Error()
	}
	return status
}
