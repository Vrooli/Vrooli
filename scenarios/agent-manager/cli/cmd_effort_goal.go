// This file implements the goal-home commands the orchestrator uses
// (large-effort-orchestration §2–§3): effort lint and effort handoff set.
package main

import (
	"bytes"
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

// loadGoalHome reads the only runtime format. Legacy parsing is import-only.
func loadGoalHome(dir string) (*goalhome.Home, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a goal-home directory", dir)
	}
	if !goalHomeIsV2(dir) {
		return nil, fmt.Errorf("CHARTER.md is required; import the legacy home with agent-manager effort migrate %s --apply", dir)
	}
	return goalhome.LoadV2(os.DirFS(dir))
}

// goalHomeIsV2 reports whether dir holds a Phase-2 v2 home, detected by a
// CHARTER.md. Legacy homes must pass the explicit importer before runtime use.
func goalHomeIsV2(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, goalhome.CharterFile))
	return err == nil
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

// v1RetiredFiles are the v1 goal-home files moved into archive/v1/ when a home
// is migrated. Per-home bespoke files are left untouched.
var v1RetiredFiles = []string{
	goalhome.GoalFile, goalhome.QueueFile, goalhome.WorkaroundsFile,
	"INSTALLS.md", "TARGETS.md", "ACCEPTANCE.md",
}

// effortMigrate is `agent-manager effort migrate <goal home> [--apply]`. It
// converts a v1 home to the v2 file set (CHARTER/STATUS/LOG), self-verifying
// that the board facts are unchanged before writing. Without --apply it is a
// dry run that only reports equivalence. It is idempotent: a home that already
// has CHARTER.md is left unchanged.
func (a *App) effortMigrate(args []string) error {
	fs := flag.NewFlagSet("effort migrate", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	apply := fs.Bool("apply", false, "Write the v2 files and archive the v1 files (default: dry run)")
	dir, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 {
		return errors.New("usage: agent-manager effort migrate <goal home> [--apply] [--json]")
	}
	result := struct {
		Home       string `json:"home"`
		Status     string `json:"status"`
		Equivalent bool   `json:"equivalent"`
		Divergence string `json:"divergence,omitempty"`
		Applied    bool   `json:"applied"`
	}{Home: dir}

	if goalHomeIsV2(dir) {
		if _, err := os.Stat(filepath.Join(dir, migrationFile)); err == nil {
			if !*apply {
				result.Status = "interrupted"
				return printMigrate(result, *jsonOut, "interrupted migration; rerun with --apply to finish")
			}
			if err := finishMigration(dir); err != nil {
				return err
			}
			result.Status, result.Equivalent, result.Applied = "migrated", true, true
			return printMigrate(result, *jsonOut, "recovered interrupted migration")
		}
		home, err := goalhome.LoadV2(os.DirFS(dir))
		if err != nil {
			return err
		}
		if len(home.Missing) != 0 {
			return fmt.Errorf("incomplete v2 home: missing %s", strings.Join(home.Missing, ", "))
		}
		result.Status, result.Equivalent = "already-v2", true
		return printMigrate(result, *jsonOut, "already v2 (CHARTER.md present); nothing to do")
	}
	if _, err := os.Stat(filepath.Join(dir, migrationFile)); err == nil {
		if !*apply {
			return errors.New("interrupted migration; rerun with --apply to finish")
		}
		if err := finishMigration(dir); err != nil {
			return err
		}
		result.Status, result.Equivalent, result.Applied = "migrated", true, true
		return printMigrate(result, *jsonOut, "recovered interrupted migration")
	}

	v1, err := goalhome.LoadLegacy(os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("goal home %s: %w", dir, err)
	}
	goalRaw, err := os.ReadFile(filepath.Join(dir, goalhome.GoalFile))
	if err != nil {
		return fmt.Errorf("goal home %s: %w", dir, err)
	}
	queueRaw, _ := os.ReadFile(filepath.Join(dir, goalhome.QueueFile))
	workaroundsRaw, _ := os.ReadFile(filepath.Join(dir, goalhome.WorkaroundsFile))

	conv := goalhome.ConvertHome(v1, goalRaw, queueRaw, workaroundsRaw)
	result.Equivalent, result.Divergence = conv.Equivalent, conv.Divergence
	if !conv.Equivalent {
		result.Status = "diverged"
		_ = printMigrate(result, *jsonOut, "NOT migrated — board facts would diverge: "+conv.Divergence)
		return exitCodeError{code: exitRefused}
	}
	if !*apply {
		result.Status = "dry-run-ok"
		return printMigrate(result, *jsonOut, "would migrate (board facts verified equivalent); rerun with --apply")
	}
	if err := applyMigration(dir, conv); err != nil {
		return err
	}
	result.Status, result.Applied = "migrated", true
	return printMigrate(result, *jsonOut, "migrated to CHARTER/STATUS/LOG; v1 files moved to archive/v1/")
}

// applyMigration writes the v2 files and moves the retired v1 files into
// archive/v1/. FEEDBACK.md stays in place.
func applyMigration(dir string, conv goalhome.ConvertResult) error {
	// Retained authored briefs stay usable under epochs/. Runtime acceptance is
	// read only from LOG; these files do not restore a legacy runtime reader.
	charter := append([]byte(nil), conv.Charter...)
	for _, name := range []string{"TARGETS.md", "ACCEPTANCE.md", "INSTALLS.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			charter = append(charter, []byte("\nRetained operative "+name+": [archive/v1/"+name+"](archive/v1/"+name+"). Its obligations remain in force.\n")...)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	outputs := map[string][]byte{goalhome.CharterFile: charter, goalhome.StatusFile: conv.Status, goalhome.LogFile: conv.Log}
	if _, err := os.Stat(filepath.Join(dir, goalhome.FeedbackFile)); errors.Is(err, os.ErrNotExist) {
		outputs[goalhome.FeedbackFile] = []byte("# Feedback\n\n## Open\n\n## Resolved\n")
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, migrationFile)); errors.Is(err, os.ErrNotExist) {
		data, err := json.Marshal(outputs)
		if err != nil {
			return err
		}
		if err := publishMigrationFile(dir, migrationFile, data, 0o600); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return finishMigration(dir)
}

const migrationFile = ".migration-v2.json"

// The persisted output snapshot is the recovery authority. Publish CHARTER last;
// retries validate existing bytes before completing the archive, never overwrite edits.
func finishMigration(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, migrationFile))
	if err != nil {
		return err
	}
	var outputs map[string][]byte
	if err := json.Unmarshal(data, &outputs); err != nil {
		return err
	}
	names := []string{goalhome.StatusFile, goalhome.LogFile, goalhome.CharterFile}
	if _, ok := outputs[goalhome.FeedbackFile]; ok {
		names = append(names, goalhome.FeedbackFile)
	}
	for _, name := range names {
		body, ok := outputs[name]
		if !ok {
			return fmt.Errorf("migration snapshot missing %s", name)
		}
		current, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			if !bytes.Equal(current, body) {
				return fmt.Errorf("migration paused: %s changed since publication; preserve and reconcile the edit", name)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := publishMigrationFile(dir, name, body, 0o644); err != nil {
			return err
		}
	}
	archive := filepath.Join(dir, "archive", "v1")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		return err
	}
	for _, name := range v1RetiredFiles {
		src := filepath.Join(dir, name)
		if _, err := os.Stat(src); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		dst := filepath.Join(archive, name)
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("archive collision: %s already exists; preserve and reconcile it", dst)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}
	return os.Remove(filepath.Join(dir, migrationFile))
}

func publishMigrationFile(dir, name string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(dir, ".migration-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Link is exclusive: a concurrent publication or operator file is never replaced.
	return os.Link(f.Name(), filepath.Join(dir, name))
}

func printMigrate(result any, jsonOut bool, text string) error {
	if jsonOut {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
		return nil
	}
	fmt.Println(text)
	return nil
}

func printLintReport(dir string, report *goalhome.LintReport) {
	r := report.ResumeSet
	goalLabel, queueLabel := goalhome.GoalFile, goalhome.QueueFile
	if report.V2 {
		goalLabel, queueLabel = goalhome.CharterFile, goalhome.StatusFile
	}
	parts := fmt.Sprintf("%s %d, %s %d, open FEEDBACK %d", goalLabel, r.Goal, queueLabel, r.Queue, r.OpenFeedback)
	if !report.V2 {
		parts += fmt.Sprintf(", open WORKAROUNDS %d", r.OpenWorkarounds)
	}
	fmt.Printf("Goal home %s: resume set %d of %d bytes (%s)\n", dir, r.Total, r.Budget, parts)
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
// It replaces the single ## Handoff section of the sections file (STATUS.md for
// the canonical runtime home) atomically, writes nothing when the text is
// unchanged, and refuses (exit 4) a handoff over 4 KB, a file with more than one
// handoff, or a sections file that changed while it was being written.
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
	path := filepath.Join(dir, goalhome.StatusFile)
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
