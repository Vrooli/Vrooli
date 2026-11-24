package destinationreadiness_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
)

func TestLocalPreparerCreatesSubdirAndVerifiesWritable(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vrooli-backups")
	preparer := destinationreadiness.NewLocalPreparer()

	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action:     destinationreadiness.ActionCreateSubdir,
		Location:   root,
		TargetPath: target,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("target subdir was not created: info=%v err=%v", info, err)
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary probe should be removed, entries=%+v err=%v", entries, err)
	}
}

func TestLocalPreparerPreservesExistingFileAtLegacyProbeName(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vrooli-backups")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	probePath := filepath.Join(target, ".vrooli-write-probe")
	const sentinel = "operator-owned data"
	if err := os.WriteFile(probePath, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}

	err := destinationreadiness.NewLocalPreparer().Execute(context.Background(), destinationreadiness.Plan{
		Action: destinationreadiness.ActionCreateSubdir, Location: root, TargetPath: target,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got, err := os.ReadFile(probePath)
	if err != nil || string(got) != sentinel {
		t.Fatalf("pre-existing file = %q, err=%v; want the original contents preserved", got, err)
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(probePath) {
		t.Fatalf("target entries = %+v, err=%v; want only the original file", entries, err)
	}
}

func TestLocalPreparerRejectsProbeSuffixOutsideTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vrooli-backups")
	preparer := &destinationreadiness.LocalPreparer{ProbeSuffix: "../outside"}
	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action: destinationreadiness.ActionCreateSubdir, Location: root, TargetPath: target,
	})
	var invalid destinationreadiness.ErrInvalidReadiness
	if !errors.As(err, &invalid) || invalid.Field != "probe_suffix" {
		t.Fatalf("Execute error = %v, want probe_suffix validation", err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid probe suffix created target: err=%v", err)
	}
}

func TestLocalPreparerRefusesUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name   string
		action destinationreadiness.PreparationAction
	}{
		{name: "destructive action", action: destinationreadiness.ActionFormat},
		{name: "mount root target", action: destinationreadiness.ActionCreateSubdir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := destinationreadiness.NewLocalPreparer().Execute(context.Background(), destinationreadiness.Plan{
				Action: tc.action, Location: root, TargetPath: root,
			})
			if !refused(err) {
				t.Fatalf("expected refusal, got %v", err)
			}
		})
	}
}

func TestLocalPreparerCleansProbeWhenWriteFails(t *testing.T) {
	var mkdirCalled bool
	var writeCalled bool
	var removeCalled bool
	root := t.TempDir()
	target := filepath.Join(root, "vrooli-backups")
	preparer := &destinationreadiness.LocalPreparer{
		MkdirAll: func(path string, mode os.FileMode) error {
			mkdirCalled = true
			return os.MkdirAll(path, mode)
		},
		WriteFile: func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true
			return errors.New("probe failed")
		},
		Remove: func(path string) error {
			removeCalled = true
			return os.Remove(path)
		},
	}

	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action:     destinationreadiness.ActionCreateSubdir,
		Location:   root,
		TargetPath: target,
	})
	if err == nil {
		t.Fatal("expected probe write error")
	}
	entries, readErr := os.ReadDir(target)
	if !mkdirCalled || !writeCalled || !removeCalled || readErr != nil || len(entries) != 0 {
		t.Fatalf("probe failure cleanup: mkdir=%v write=%v remove=%v entries=%+v readErr=%v", mkdirCalled, writeCalled, removeCalled, entries, readErr)
	}
}
