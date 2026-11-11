package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Both Makefile-backed scanner commands share one execution contract. Keep
// cases parameterized by the target and result accessor so command plumbing,
// timeout and concurrency behavior cannot drift between lint and type scans.
type scannerCommandCase struct {
	name       string
	command    string
	validLines []string
	result     func(*ScanResult) *CommandRun
}

var scannerCommands = []scannerCommandCase{
	{
		name: "lint", command: "make lint",
		validLines: []string{
			`@echo "Running lint checks..."`,
			`@echo "src/file.go:10:5: unused variable [deadcode]"`,
			`@echo "src/another.go:25:10: missing return statement [missing-return]"`,
		},
		result: func(result *ScanResult) *CommandRun { return result.LintOutput },
	},
	{
		name: "type", command: "make type",
		validLines: []string{
			`@echo "Running type checks..."`,
			`@echo "src/file.ts(10,5): error TS2304: Cannot find name 'unknown'"`,
			`@echo "src/app.tsx(25,10): error TS2322: Type 'string' is not assignable to type 'number'"`,
			`@echo "src/main.go:42:5: undefined: missingFunc"`,
		},
		result: func(result *ScanResult) *CommandRun { return result.TypeOutput },
	},
}

func TestMakefileCommandExecutionValidOutput(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name, command.validLines...))
			result, err := NewLightScanner(root, 30*time.Second).Scan(context.Background())
			if err != nil {
				t.Fatalf("scan failed: %v", err)
			}
			if command.name == "lint" && !result.HasMakefile {
				t.Fatal("expected Makefile detection")
			}
			got := command.result(result)
			if got == nil {
				t.Fatal("expected command output")
			}
			if got.Command != command.command || got.Skipped || !got.Success || got.ExitCode != 0 || got.Stdout == "" {
				t.Fatalf("unexpected command result: %+v", got)
			}
			for _, want := range command.validLines[1:] {
				message := strings.Trim(strings.TrimPrefix(want, `@echo `), `"`)
				if !strings.Contains(got.Stdout, message) {
					t.Errorf("output %q does not contain %q", got.Stdout, message)
				}
			}
		})
	}
}

func TestMakefileCommandExecutionNonzeroExit(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name,
				`@echo "src/error.go:1:1: synthetic failure"`, "@exit 1"))
			result, err := NewLightScanner(root, 30*time.Second).Scan(context.Background())
			if err != nil {
				t.Fatalf("scan should complete despite target failure: %v", err)
			}
			got := command.result(result)
			if got == nil || got.Success || got.ExitCode == 0 || got.Stdout == "" {
				t.Fatalf("expected captured failure result, got %+v", got)
			}
		})
	}
}

func TestMakefileCommandExecutionInvalidCommand(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name,
				"@definitely-missing-vrooli-command --flag", `@echo "must not run"`))
			result, err := NewLightScanner(root, 30*time.Second).Scan(context.Background())
			if err != nil {
				t.Fatalf("scan should tolerate invalid target commands: %v", err)
			}
			got := command.result(result)
			if got == nil || got.Success || got.ExitCode == 0 {
				t.Fatalf("expected invalid command failure, got %+v", got)
			}
		})
	}
}

func TestMakefileCommandExecutionTimeout(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name, "@sleep 10"))
			result, err := NewLightScanner(root, 100*time.Millisecond).Scan(context.Background())
			if err != nil {
				t.Fatalf("scan should return partial metrics after command timeout: %v", err)
			}
			got := command.result(result)
			if got == nil || got.Success {
				t.Fatalf("expected failed/timed-out command result, got %+v", got)
			}
		})
	}
}

func TestMakefileCommandExecutionConcurrentSafety(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name,
				`@echo "command output for concurrent test"`, `@echo "file.go:1:1: synthetic issue"`))
			type scanResult struct {
				command *CommandRun
				err     error
			}
			const workers = 5
			done := make(chan scanResult, workers)
			for i := 0; i < workers; i++ {
				go func() {
					result, err := NewLightScanner(root, 30*time.Second).Scan(context.Background())
					if err != nil || result == nil {
						done <- scanResult{err: err}
						return
					}
					done <- scanResult{command: command.result(result)}
				}()
			}
			for i := 0; i < workers; i++ {
				got := <-done
				if got.err != nil {
					t.Errorf("scan %d: %v", i, got.err)
				} else if got.command == nil || !got.command.Success || !strings.Contains(got.command.Stdout, "command output") {
					t.Errorf("scan %d returned corrupted or missing output: %+v", i, got.command)
				}
			}
		})
	}
}

func TestMakefileCommandExecutionContextCancellation(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name, "@sleep 2"))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			result, err := NewLightScanner(root, 10*time.Second).Scan(ctx)
			if time.Since(started) > time.Second {
				t.Fatal("scan did not stop promptly after context cancellation")
			}
			if err == nil && result == nil {
				t.Fatal("canceled scan returned neither a result nor an error")
			}
			if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				t.Errorf("unexpected cancellation error: %v", err)
			}
		})
	}
}

func TestMakefileCommandExecutionEmptyOutput(t *testing.T) {
	for _, command := range scannerCommands {
		t.Run(command.name, func(t *testing.T) {
			root := writeScannerProject(t, makeTarget(command.name, "@true"))
			result, err := NewLightScanner(root, 30*time.Second).Scan(context.Background())
			if err != nil {
				t.Fatalf("scan failed: %v", err)
			}
			got := command.result(result)
			if got == nil || !got.Success || got.ExitCode != 0 {
				t.Fatalf("expected successful silent command, got %+v", got)
			}
		})
	}
}

func writeScannerProject(t *testing.T, makefile string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte(makefile), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	api := filepath.Join(root, "api")
	if err := os.MkdirAll(api, 0o755); err != nil {
		t.Fatalf("create api directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(api, "fixture.go"), []byte("package main\nfunc fixture() {}\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	return root
}

func makeTarget(name string, lines ...string) string {
	return fmt.Sprintf(".PHONY: %s\n%s:\n\t%s\n", name, name, strings.Join(lines, "\n\t"))
}
