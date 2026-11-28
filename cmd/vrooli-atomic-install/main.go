package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	usageExitCode = 2
	usageText     = "usage: vrooli-atomic-install --verify-runtime-paths BIN_DIR SOURCE_POINTER | --install-project-bundle SOURCE_DIR BIN_DIR SOURCE_POINTER SOURCE_ROOT | --install-bundle-owned BUNDLE MANIFEST SOURCE_ROOT | --publish-source-owned BUNDLE MANIFEST SOURCE_ROOT | --abort-bundle BIN_DIR | --resume-bundle BIN_DIR | SOURCE DEST [EXPECTED_CURRENT_SHA256] | --companions SOURCE_DIR BIN_DIR ROOT_SHA256 | --prepare-bundle SOURCE_DIR OUTPUT_DIR | --prepare-hub-bundle SOURCE_DIR OUTPUT_DIR POINTER | --verify-bundle SOURCE_DIR MANIFEST_SHA256 | --prepared-companions SOURCE_DIR BIN_DIR ROOT_SHA256 MANIFEST_SHA256 | --publish-source BUNDLE BIN_DIR MANIFEST_SHA256 SOURCE_ROOT POINTER | --snapshot-lock-exec ROOT -- COMMAND... | --snapshot-lease-exec ROOT SNAPSHOT -- COMMAND... | --try-snapshot-lease-exec ROOT SNAPSHOT -- COMMAND... | --prune-install-snapshots ROOT | --prune-hub-snapshots ROOT CURRENT_POINTER LEGACY_POINTER -- RETIRE_COMMAND [ARGS...]"
)

type invocationMode uint8

const (
	modeInvalid invocationMode = iota
	modeSnapshotRoot
	modeAbortBundle
	modeResumeBundle
	modeCompanions
	modePrepareBundle
	modePrepareHubBundle
	modeVerifyBundle
	modePreparedCompanions
	modeInstallBundle
	modePublishSource
	modeVerifyRuntimePaths
	modeInstallProjectBundle
	modeInstallBundleOwned
	modePublishSourceOwned
	modeSnapshotLockExec
	modeSnapshotLeaseExec
	modeTrySnapshotLeaseExec
	modePruneInstallSnapshots
	modePruneHubSnapshots
	modeAtomicInstall
	modeAtomicInstallExpected
)

type invocation struct {
	mode invocationMode
	args []string
}

func parseInvocation(args []string) (invocation, error) {
	if len(args) < 2 {
		return invocation{}, errors.New("invalid invocation")
	}
	mode := modeInvalid
	switch args[1] {
	case "--snapshot-root":
		if len(args) == 2 {
			mode = modeSnapshotRoot
		}
	case "--abort-bundle":
		if len(args) == 3 {
			mode = modeAbortBundle
		}
	case "--resume-bundle":
		if len(args) == 3 {
			mode = modeResumeBundle
		}
	case "--companions":
		if len(args) == 5 {
			mode = modeCompanions
		}
	case "--prepare-bundle":
		if len(args) == 4 {
			mode = modePrepareBundle
		}
	case "--prepare-hub-bundle":
		if len(args) == 5 {
			mode = modePrepareHubBundle
		}
	case "--verify-bundle":
		if len(args) == 4 {
			mode = modeVerifyBundle
		}
	case "--prepared-companions":
		if len(args) == 6 {
			mode = modePreparedCompanions
		}
	case "--install-bundle":
		if len(args) == 5 || len(args) == 6 {
			mode = modeInstallBundle
		}
	case "--publish-source":
		if len(args) == 7 {
			mode = modePublishSource
		}
	case "--verify-runtime-paths":
		if len(args) == 4 {
			mode = modeVerifyRuntimePaths
		}
	case "--install-project-bundle":
		if len(args) == 6 {
			mode = modeInstallProjectBundle
		}
	case "--install-bundle-owned":
		if len(args) == 5 {
			mode = modeInstallBundleOwned
		}
	case "--publish-source-owned":
		if len(args) == 5 {
			mode = modePublishSourceOwned
		}
	case "--snapshot-lock-exec":
		if len(args) >= 5 && args[3] == "--" {
			mode = modeSnapshotLockExec
		}
	case "--snapshot-lease-exec":
		if len(args) >= 6 && args[4] == "--" {
			mode = modeSnapshotLeaseExec
		}
	case "--try-snapshot-lease-exec":
		if len(args) >= 6 && args[4] == "--" {
			mode = modeTrySnapshotLeaseExec
		}
	case "--prune-install-snapshots":
		if len(args) == 3 {
			mode = modePruneInstallSnapshots
		}
	case "--prune-hub-snapshots":
		if len(args) >= 7 && args[5] == "--" {
			mode = modePruneHubSnapshots
		}
	}

	// Keep the original positional forms, including their historical treatment
	// of unrecognized tokens in argument positions, for installer compatibility.
	if mode == modeInvalid {
		switch len(args) {
		case 3:
			mode = modeAtomicInstall
		case 4:
			mode = modeAtomicInstallExpected
		}
	}
	if mode == modeInvalid {
		return invocation{}, errors.New("invalid invocation")
	}
	return invocation{mode: mode, args: args}, nil
}

func main() { os.Exit(run(os.Args, os.Stderr)) }
func run(args []string, stderr io.Writer) int {
	call, parseErr := parseInvocation(args)
	if parseErr != nil {
		fmt.Fprintln(stderr, usageText)
		return usageExitCode
	}
	mode := call.mode
	args = call.args
	// A retained maintenance installer may run after source loss. Source-backed
	// go-run invocations retain their existing discovery path; a stamped image
	// must pin and validate its own compiled authority before any mutation.
	if buildinfo.RuntimeHomeSnapshotBase64 != "" {
		if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
			fmt.Fprintln(stderr, "compiled runtime-home authority unavailable:", err)
			return 1
		}
	}
	var err error
	code := 0
	switch mode {
	case modeSnapshotLockExec:
		code = runSnapshotCommand(stderr, func() error {
			return cliinstall.WithSnapshotLockFile(args[2], func(lock *os.File) error { return runCommand(args[4:], lock, args[2], "") })
		})
	case modeSnapshotLeaseExec:
		code = runSnapshotCommand(stderr, func() error {
			return cliinstall.WithSnapshotLeaseFile(args[2], args[3], func(lock *os.File) error { return runCommand(args[5:], lock, args[2], args[3]) })
		})
	case modeTrySnapshotLeaseExec:
		code = runSnapshotCommand(stderr, func() error {
			_, leaseErr := cliinstall.TryWithSnapshotLeaseFile(args[2], args[3], func(lock *os.File) error { return runCommand(args[5:], lock, args[2], args[3]) })
			return leaseErr
		})
	case modePruneInstallSnapshots:
		err = cliinstall.PruneInstallSnapshots(args[2], time.Now())
	case modePruneHubSnapshots:
		err = cliinstall.PruneHubSnapshotsFromPointers(args[2], args[3], args[4], time.Now(), func(snapshot string, lease *os.File) error {
			command := append(append([]string(nil), args[6:]...), "runtime", "update", "retire-snapshot", snapshot)
			return runQuietCommand(command, lease, args[2], snapshot)
		})
	case modeSnapshotRoot:
		var root string
		root, err = ensureSnapshotRoot()
		if err == nil {
			fmt.Fprintln(os.Stdout, root)
		}
	case modeInstallProjectBundle:
		err = installProjectBundle(stderr, args[2], args[3], args[4], args[5], runPreparedProjectBundleInstall)
	case modeVerifyRuntimePaths:
		_, _, err = verifiedRuntimePaths(args[2], args[3])
	case modeInstallBundleOwned, modePublishSourceOwned:
		bin, pointer, pathErr := ownedRuntimePaths()
		if pathErr != nil {
			err = pathErr
		} else if mode == modeInstallBundleOwned {
			if _, err = config.EnsureOwnedDir(bin); err == nil {
				err = cliinstall.InstallPreparedBundleWithSource(args[2], bin, args[3], args[4])
			}
		} else {
			err = cliinstall.PublishPreparedBundleSource(args[2], bin, args[3], args[4], pointer)
		}
	case modeAbortBundle:
		err = cliinstall.AbortPreparedBundle(args[2])
	case modeResumeBundle:
		err = cliinstall.ResumePreparedBundle(args[2])
	case modePublishSource:
		err = cliinstall.PublishPreparedBundleSource(args[2], args[3], args[4], args[5], args[6])
	case modeInstallBundle:
		if len(args) == 6 {
			err = cliinstall.InstallPreparedBundleWithSource(args[2], args[3], args[4], args[5])
		} else {
			err = cliinstall.InstallPreparedBundle(args[2], args[3], args[4])
		}
	case modePrepareHubBundle:
		err = cliinstall.PrepareHubBundle(args[2], args[3], args[4])
	case modePrepareBundle:
		err = cliinstall.PrepareBundle(args[2], args[3])
	case modeVerifyBundle:
		err = cliinstall.VerifyPreparedBundle(args[2], args[3])
	case modePreparedCompanions:
		err = cliinstall.InstallPreparedCompanions(args[2], args[3], args[4], args[5])
	case modeCompanions:
		err = cliinstall.InstallCompanions(args[2], args[3], args[4])
	case modeAtomicInstallExpected:
		err = cliinstall.AtomicInstallExpected(args[1], args[2], args[3])
	case modeAtomicInstall:
		err = cliinstall.AtomicInstall(args[1], args[2])
	}
	if code != 0 {
		return code
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runSnapshotCommand(stderr io.Writer, fn func() error) int {
	if err := fn(); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code >= 0 {
				return code
			}
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runCommand(args []string, lockFile *os.File, leaseRoot, leaseSnapshot string) error {
	return runCommandWithIO(args, lockFile, leaseRoot, leaseSnapshot, os.Stdin, os.Stdout, os.Stderr)
}

func runQuietCommand(args []string, lockFile *os.File, leaseRoot, leaseSnapshot string) error {
	return runCommandWithIO(args, lockFile, leaseRoot, leaseSnapshot, nil, nil, nil)
}

func runCommandWithIO(args []string, lockFile *os.File, leaseRoot, leaseSnapshot string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("snapshot lock command is required")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	signals := make(chan os.Signal, 4)
	signal.Notify(signals, managedCommandSignals()...)
	defer signal.Stop(signals)
	waitCommand, signalCommand, cleanupCommand, err := startManagedCommand(cmd, lockFile, leaseRoot, leaseSnapshot)
	if err != nil {
		return err
	}
	defer cleanupCommand()
	wait := make(chan error, 1)
	go func() { wait <- waitCommand() }()
	var interrupted os.Signal
	for {
		select {
		case err := <-wait:
			if waitErr := waitManagedCommand(cmd); waitErr != nil {
				return waitErr
			}
			if interrupted != nil {
				return fmt.Errorf("snapshot command interrupted by %s", interrupted)
			}
			return err
		case sig := <-signals:
			if interrupted == nil {
				interrupted = sig
			}
			// Keep the lock owner alive until the child process group has fully
			// exited. A shell trap may return before a child that ignores TERM.
			_ = signalCommand(sig)
		}
	}
}

// ownedRuntimePaths uses the control plane's invoking-user and runtime-home
// authority, including when sudo changes HOME. It does not change identity.
func ownedRuntimePaths() (string, string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return "", "", err
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return "", "", err
	}
	return bin, repocontract.SourceRootPointerPath(home), nil
}

func verifiedRuntimePaths(binOverride, pointerOverride string) (string, string, error) {
	bin, pointer, err := ownedRuntimePaths()
	if err != nil {
		return "", "", err
	}
	if (binOverride != "" && filepath.Clean(binOverride) != filepath.Clean(bin)) || (pointerOverride != "" && filepath.Clean(pointerOverride) != filepath.Clean(pointer)) {
		return "", "", fmt.Errorf("destination overrides must match the invoking user runtime bin and source pointer")
	}
	return bin, pointer, nil
}

func ensureSnapshotRoot() (string, error) {
	root, err := config.VrooliPath("state", "root-rebaseline", "snapshots")
	if err != nil {
		return "", err
	}
	if _, err := config.EnsureOwnedDirMode(root, 0700); err != nil {
		return "", err
	}
	lock, err := config.OpenOwnedFile(filepath.Join(root, "snapshots.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	if err := lock.Close(); err != nil {
		return "", err
	}
	return root, nil
}

type projectBundleInstaller func(lock *os.File, snapshotRoot, snapshot, binDir, manifestSHA, sourceRoot string) error

func installProjectBundle(stderr io.Writer, sourceDir, binOverride, pointerOverride, sourceRoot string, install projectBundleInstaller) error {
	binDir, _, err := verifiedRuntimePaths(binOverride, pointerOverride)
	if err != nil {
		return err
	}
	snapshotRoot, err := ensureSnapshotRoot()
	if err != nil {
		return err
	}
	if err := cliinstall.PruneInstallSnapshots(snapshotRoot, time.Now()); err != nil {
		return err
	}
	var snapshot string
	defer func() {
		if snapshot == "" {
			return
		}
		if err := cliinstall.WithSnapshotLock(snapshotRoot, func() error { return os.RemoveAll(snapshot) }); err != nil {
			fmt.Fprintf(stderr, "Install completed or failed, but temporary bundle cleanup failed; it is safe to prune later: %v\n", err)
		}
	}()
	if err := cliinstall.WithSnapshotLockFile(snapshotRoot, func(*os.File) error {
		path, err := os.MkdirTemp(snapshotRoot, "install-bundle.")
		if err != nil {
			return err
		}
		snapshot = path
		if err := os.Remove(path); err != nil {
			return err
		}
		return cliinstall.PrepareBundle(sourceDir, snapshot)
	}); err != nil {
		return err
	}
	manifest, err := os.ReadFile(filepath.Join(snapshot, "SHA256SUMS"))
	if err != nil {
		return err
	}
	manifestDigest := sha256.Sum256(manifest)
	manifestSHA := hex.EncodeToString(manifestDigest[:])
	if err := cliinstall.VerifyPreparedBundle(snapshot, manifestSHA); err != nil {
		return err
	}
	if err := cliinstall.WithSnapshotLeaseFile(snapshotRoot, snapshot, func(lock *os.File) error {
		return install(lock, snapshotRoot, snapshot, binDir, manifestSHA, sourceRoot)
	}); err != nil {
		return err
	}
	return nil
}

func runPreparedProjectBundleInstall(lock *os.File, snapshotRoot, snapshot, binDir, manifestSHA, sourceRoot string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return runCommand([]string{executable, "--install-bundle", snapshot, binDir, manifestSHA, sourceRoot}, lock, snapshotRoot, snapshot)
}
