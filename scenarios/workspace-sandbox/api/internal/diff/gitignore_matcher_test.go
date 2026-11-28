package diff

import (
	"context"
	"testing"
)

func TestGitIgnoreMatcher_DropsOnlyIgnoredUntrackedPaths(t *testing.T) {
	runner := NewMockCommandRunner()
	runner.AddResponse("git -C /repo rev-parse --is-inside-work-tree", CommandResult{Stdout: "true\n"})
	runner.AddResponse("git -C /repo check-ignore -z --", CommandResult{Stdout: "ignored/cache.bin\x00.vrooli/service.json\x00"})
	runner.AddResponse("git -C /repo ls-files -z --", CommandResult{Stdout: ".vrooli/service.json\x00"})
	matcher := NewGitIgnoreMatcher("/repo", runner)
	ignored, err := matcher.Ignored(context.Background(), []string{"ignored/cache.bin", ".vrooli/service.json", "main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ignored["ignored/cache.bin"]; !ok {
		t.Fatalf("gitignored untracked path was not dropped: %#v", ignored)
	}
	if _, ok := ignored[".vrooli/service.json"]; ok {
		t.Fatalf("tracked dot path was dropped: %#v", ignored)
	}
}

func TestGitIgnoreMatcher_PreservesSpacesInPaths(t *testing.T) {
	runner := NewMockCommandRunner()
	runner.AddResponse("git -C /repo rev-parse --is-inside-work-tree", CommandResult{Stdout: "true\n"})
	runner.AddResponse("git -C /repo check-ignore -z --", CommandResult{Stdout: "build cache/output.bin\x00"})
	runner.AddResponse("git -C /repo ls-files -z --", CommandResult{})
	ignored, err := NewGitIgnoreMatcher("/repo", runner).Ignored(context.Background(), []string{"build cache/output.bin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ignored["build cache/output.bin"]; !ok {
		t.Fatalf("space-containing ignored path missing: %#v", ignored)
	}
}

func TestGitIgnoreMatcher_FailsOpenWhenGitUnavailable(t *testing.T) {
	runner := NewMockCommandRunner()
	runner.AddResponse("git -C /repo rev-parse --is-inside-work-tree", CommandResult{Err: context.DeadlineExceeded, ExitCode: 1})
	ignored, err := NewGitIgnoreMatcher("/repo", runner).Ignored(context.Background(), []string{"important.go"})
	if err != nil || len(ignored) != 0 {
		t.Fatalf("ignored=%#v err=%v; want fail-open empty result", ignored, err)
	}
}
