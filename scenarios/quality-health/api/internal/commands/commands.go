package commands

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"quality-health/internal/surfaces"
)

type Result struct {
	Name             string
	Command          string
	WorkingDirectory string
	Status           string
	ExitCode         int
	StdoutExcerpt    string
	StderrExcerpt    string
	TimeoutSeconds   int
	FailureReason    string
}

type Executor interface {
	Run(ctx context.Context, name string, args []string, dir string, timeout time.Duration) Result
}

type LocalExecutor struct{}

func (LocalExecutor) Run(ctx context.Context, name string, args []string, dir string, timeout time.Duration) Result {
	cmdText := strings.Join(append([]string{name}, args...), " ")
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	res := Result{
		Name:             name,
		Command:          cmdText,
		WorkingDirectory: dir,
		Status:           "passed",
		ExitCode:         0,
		StdoutExcerpt:    excerpt(string(out)),
		TimeoutSeconds:   int(timeout.Seconds()),
	}
	if err != nil {
		res.Status = "failed"
		res.ExitCode = 1
		res.FailureReason = err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			res.StderrExcerpt = excerpt(string(ee.Stderr))
		}
		if cctx.Err() == context.DeadlineExceeded {
			res.Status = "timeout"
			res.FailureReason = "command timed out"
		}
	}
	return res
}

func Resolve(inv surfaces.Inventory) []struct {
	Name string
	Args []string
	Dir  string
} {
	var out []struct {
		Name string
		Args []string
		Dir  string
	}
	for _, s := range inv.Surfaces {
		switch s.Language {
		case "typescript", "javascript":
			out = append(out,
				struct {
					Name string
					Args []string
					Dir  string
				}{"pnpm", []string{"run", "lint"}, s.RootPath},
				struct {
					Name string
					Args []string
					Dir  string
				}{"pnpm", []string{"run", "type-check"}, s.RootPath},
			)
		case "go":
			out = append(out, struct {
				Name string
				Args []string
				Dir  string
			}{"golangci-lint", []string{"run", "./..."}, s.RootPath})
		}
	}
	if inv.RootPath != "" {
		out = append(out, struct {
			Name string
			Args []string
			Dir  string
		}{"make", []string{"lint"}, inv.RootPath})
	}
	return out
}

func RunAll(ctx context.Context, execer Executor, inv surfaces.Inventory) []Result {
	if execer == nil {
		execer = LocalExecutor{}
	}
	var out []Result
	for _, cmd := range Resolve(inv) {
		if cmd.Dir == "" || !filepath.IsAbs(cmd.Dir) {
			continue
		}
		out = append(out, execer.Run(ctx, cmd.Name, cmd.Args, cmd.Dir, 2*time.Minute))
	}
	return out
}

func excerpt(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 1200 {
		return s
	}
	return s[:1200]
}
