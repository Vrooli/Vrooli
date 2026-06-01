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
	if _, err := os.Stat(filepath.Join(target, ".vrooli-write-probe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("probe file should be removed, got err=%v", err)
	}
}

func TestLocalPreparerRefusesDestructiveActions(t *testing.T) {
	root := t.TempDir()
	preparer := destinationreadiness.NewLocalPreparer()

	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action:     destinationreadiness.ActionFormat,
		Location:   root,
		TargetPath: root,
	})
	if !refused(err) {
		t.Fatalf("expected destructive action refusal, got %v", err)
	}
}

func TestLocalPreparerRefusesMountRootTarget(t *testing.T) {
	root := t.TempDir()
	preparer := destinationreadiness.NewLocalPreparer()

	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action:     destinationreadiness.ActionCreateSubdir,
		Location:   root,
		TargetPath: root,
	})
	if !refused(err) {
		t.Fatalf("expected mount-root target refusal, got %v", err)
	}
}

func TestLocalPreparerDoesNotUseRealFilesystemWhenProbeFails(t *testing.T) {
	var mkdirCalled bool
	var writeCalled bool
	preparer := &destinationreadiness.LocalPreparer{
		MkdirAll: func(_ string, _ os.FileMode) error {
			mkdirCalled = true
			return nil
		},
		WriteFile: func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true
			return errors.New("probe failed")
		},
		Remove: func(_ string) error {
			t.Fatal("remove must not run after a failed probe write")
			return nil
		},
	}

	err := preparer.Execute(context.Background(), destinationreadiness.Plan{
		Action:     destinationreadiness.ActionCreateSubdir,
		Location:   "/media/user/USB",
		TargetPath: "/media/user/USB/vrooli-backups",
	})
	if err == nil {
		t.Fatal("expected probe write error")
	}
	if !mkdirCalled || !writeCalled {
		t.Fatalf("expected mkdir/write calls, mkdir=%v write=%v", mkdirCalled, writeCalled)
	}
}
