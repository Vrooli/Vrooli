// This file implements the goal-home commands the orchestrator uses
// (large-effort-orchestration §2–§3): effort lint and effort handoff set.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
)

// loadGoalHome reads a goal-home directory.
func loadGoalHome(dir string) (*goalhome.Home, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a goal-home directory", dir)
	}
	return goalhome.Load(os.DirFS(dir))
}

// goalHomeArg splits the leading goal-home path from the flags.
func goalHomeArg(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

// effortLint is `agent-manager effort lint <goal home>`. It exits 4 when a
// blocking rule fails; it never runs inside epoch-check, so a goal-home rule
// never fails a worker.
func (a *App) effortLint(args []string) error {
	fs := flag.NewFlagSet("effort lint", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	dir, args := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: agent-manager effort lint <goal home> [--json]")
	}
	home, err := loadGoalHome(dir)
	if err != nil {
		return err
	}
	report := goalhome.Lint(home)
	if *jsonOut {
		data, err := json.Marshal(struct {
			Home string `json:"home"`
			*goalhome.LintReport
		}{dir, report})
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
	} else {
		printLintReport(dir, report)
	}
	if report.Blocking {
		return exitCodeError{code: exitRefused}
	}
	return nil
}

func printLintReport(dir string, report *goalhome.LintReport) {
	r := report.ResumeSet
	fmt.Printf("Goal home %s: resume set %d of %d bytes (GOAL %d, QUEUE %d, open FEEDBACK %d, open WORKAROUNDS %d)\n", dir, r.Total, r.Budget, r.Goal, r.Queue, r.OpenFeedback, r.OpenWorkarounds)
	for _, finding := range report.Findings {
		label := "WARN"
		if finding.Blocking {
			label = "FAIL"
		}
		location := finding.File
		if location != "" && finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, finding.Line)
		}
		if location != "" {
			location = " " + location
		}
		fmt.Printf("%s %s%s: %s\n", label, finding.Code, location, finding.Detail)
	}
	for _, rule := range report.Forecast.Rules {
		state := "not fired"
		if rule.Fired {
			state = "FIRED"
		}
		fmt.Printf("  %-14s %-9s %s\n", rule.Name, state, rule.Detail)
	}
	fmt.Println(report.Summary())
}

// effortHandoff is `agent-manager effort handoff set <goal home> (--file F | --stdin)`.
// It replaces the single ## Handoff section of QUEUE.md atomically, writes
// nothing when the text is unchanged, and refuses (exit 4) a handoff over
// 4 KB, a queue with more than one handoff, or a QUEUE.md that changed while
// it was being written.
func (a *App) effortHandoff(args []string) error {
	const usage = "usage: agent-manager effort handoff set <goal home> (--file <markdown> | --stdin) [--json]"
	if len(args) == 0 || args[0] != "set" {
		return errors.New(usage)
	}
	fs := flag.NewFlagSet("effort handoff set", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	file := fs.String("file", "", "Markdown file holding the new handoff text")
	stdin := fs.Bool("stdin", false, "Read the new handoff text from standard input")
	dir, rest := goalHomeArg(args[1:])
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 || (*file == "") == !*stdin {
		return errors.New(usage)
	}
	var body []byte
	var err error
	if *stdin {
		body, err = io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	} else {
		body, err = os.ReadFile(*file)
	}
	if err != nil {
		return err
	}
	path := filepath.Join(dir, goalhome.QueueFile)
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("goal home %s: %w", dir, err)
	}
	result := struct {
		Queue   string `json:"queue"`
		Changed bool   `json:"changed"`
		Bytes   int    `json:"bytes"`
		Refused string `json:"refused,omitempty"`
		Detail  string `json:"detail,omitempty"`
	}{Queue: path}
	updated, changed, err := goalhome.SetHandoff(original, string(body))
	if err == nil && changed {
		err = goalhome.WriteIfUnchanged(path, original, updated)
	}
	var refused *goalhome.HandoffError
	switch {
	case errors.As(err, &refused):
		result.Refused, result.Detail = refused.Code, refused.Detail
	case errors.Is(err, goalhome.ErrConcurrentChange):
		result.Refused, result.Detail = "handoff-concurrent-change", err.Error()
	case err != nil:
		return err
	default:
		result.Changed, result.Bytes = changed, len(updated)
	}
	if *jsonOut {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
	} else if result.Refused != "" {
		fmt.Printf("REFUSED %s: %s\n", result.Refused, result.Detail)
	} else if result.Changed {
		fmt.Printf("Handoff replaced in %s\n", path)
	} else {
		fmt.Printf("Handoff unchanged; %s not written\n", path)
	}
	if result.Refused != "" {
		return exitCodeError{code: exitRefused}
	}
	return nil
}
