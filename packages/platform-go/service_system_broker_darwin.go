//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func darwinNativeSystemBrokerRoots() nativeSystemBrokerRoots {
	return nativeSystemBrokerRoots{Install: "/Library/Application Support/Vrooli/RuntimeSupervisor", State: "/var/db/vrooli/runtime-supervisor", Unit: "/Library/LaunchDaemons/com.vrooli.runtime-supervisor.plist"}
}
func darwinSystemBrokerPrivileged() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("platform: native system broker setup requires root")
	}
	return nil
}
func darwinSystemBrokerCheck(path string, directory bool) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	native, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return ErrOperationIdentity
	}
	target := ""
	if st.Mode()&os.ModeSymlink != 0 {
		target, err = os.Readlink(path)
		if err != nil {
			return err
		}
	}
	if err = validateDarwinBrokerMetadata(path, st.Mode(), native.Uid, uint64(native.Nlink), directory, target); err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return darwinSystemBrokerCheck("/private/var", true)
	}
	return darwinSystemBrokerACL(path, st)
}
func darwinSystemBrokerOperator(o NativeSystemServiceBrokerOptions) error {
	if o.OperatorUID == 0 || o.OperatorGID == 0 {
		return ErrOperationIdentity
	}
	u, err := user.LookupId(strconv.FormatUint(uint64(o.OperatorUID), 10))
	if err != nil {
		return err
	}
	if u.Username != o.OperatorUsername || u.HomeDir != o.OperatorHome || u.Gid != strconv.FormatUint(uint64(o.OperatorGID), 10) {
		return ErrOperationIdentity
	}
	return nil
}
func darwinSystemBrokerProcess(ctx context.Context, p nativeSystemBrokerPlan, pid int) error {
	d, err := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if err != nil {
		return err
	}
	stamp, err := darwinObservationStamp(pid)
	if err != nil {
		return err
	}
	before, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return err
	}
	if before == nil || before.Eproc.Ucred.Uid != d.Options.OperatorUID || before.Eproc.Pcred.P_ruid != d.Options.OperatorUID || before.Eproc.Pcred.P_svuid != d.Options.OperatorUID || before.Proc.P_stat == darwinProcessZombie {
		return ErrOperationIdentity
	}
	mapped, err := darwinOpenProcessExecutable(pid)
	if err != nil {
		return err
	}
	defer mapped.Close()
	if err = verifyNativeSystemBrokerImage(mapped, p.SHA256); err != nil {
		return err
	}
	actual, err := mapped.Stat()
	if err != nil {
		return err
	}
	sealed, err := os.Stat(p.ImagePath)
	if err != nil {
		return err
	}
	if !os.SameFile(actual, sealed) {
		return ErrOperationIdentity
	}
	args, err := readProcArgs(pid)
	if err != nil {
		return err
	}
	if args.environmentErr != nil {
		return args.environmentErr
	}
	if len(args.Args) != 3 || args.Args[0] != p.ImagePath || args.Args[1] != NativeSystemSupervisorVerb || args.Args[2] != p.DescriptorPath || args.Env["HOME"] != p.StateRoot || args.Env["PATH"] != "/usr/bin:/bin:/usr/sbin:/sbin" || args.Env["VROOLI_SOURCE_ROOT"] != "" {
		return ErrOperationIdentity
	}
	cwd, err := darwinWorkingDir(pid)
	if err != nil {
		return err
	}
	currentDirectory, err := os.Stat(cwd)
	if err != nil {
		return err
	}
	machineDirectory, err := os.Stat(p.StateRoot)
	if err != nil {
		return err
	}
	if !os.SameFile(currentDirectory, machineDirectory) {
		return ErrOperationIdentity
	}
	for key, value := range args.Env {
		if value != "" && strings.HasPrefix(key, "DYLD_") {
			return ErrOperationIdentity
		}
	}
	after, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return err
	}
	if after == nil || before.Proc.P_starttime != after.Proc.P_starttime || after.Eproc.Ucred.Uid != d.Options.OperatorUID || after.Eproc.Pcred.P_ruid != d.Options.OperatorUID || after.Eproc.Pcred.P_svuid != d.Options.OperatorUID {
		return ErrOperationIdentity
	}
	end, err := darwinObservationStamp(pid)
	if err != nil || stamp != end {
		return ErrOperationIdentity
	}
	return ctx.Err()
}
func nativeDarwinSystemBrokerEngine() darwinSystemBrokerEngine {
	return darwinSystemBrokerEngine{roots: darwinNativeSystemBrokerRoots(), privileged: darwinSystemBrokerPrivileged, check: darwinSystemBrokerCheck, operator: darwinSystemBrokerOperator, closure: func(file *os.File) error { return validateDarwinBrokerMachO(file, runtime.GOARCH) }, preflight: func() error {
		file, err := darwinOpenProcessExecutable(os.Getpid())
		if err != nil {
			return err
		}
		return file.Close()
	}, runtimeIdentity: func(o NativeSystemServiceBrokerOptions) error {
		if os.Getuid() != int(o.OperatorUID) || os.Geteuid() != int(o.OperatorUID) || os.Getgid() != int(o.OperatorGID) || os.Getegid() != int(o.OperatorGID) {
			return ErrOperationIdentity
		}
		return nil
	}, query: func(ctx context.Context, expected string) (darwinJobSnapshot, bool, error) {
		if err := ctx.Err(); err != nil {
			return darwinJobSnapshot{}, false, err
		}
		return darwinManagerSnapshot(true, "com.vrooli.runtime-supervisor", expected)
	}, run: func(ctx context.Context, args ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, "/bin/launchctl", args...)
		c.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "LC_ALL=C"}
		return c.CombinedOutput()
	}, validate: func(a RenderedArtifact) Verdict {
		return validateDarwinSystemBrokerArtifact(a, darwinNativeSystemBrokerRoots().State)
	}, process: darwinSystemBrokerProcess}
}
func installDarwinSystemBroker(o ServiceInstallOptions) (ServiceInstallResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return nativeDarwinSystemBrokerEngine().install(ctx, o)
}
func uninstallDarwinSystemBroker(o ServiceInstallOptions) (ServiceInstallResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return nativeDarwinSystemBrokerEngine().uninstall(ctx, o)
}
func startDarwinSystemBroker(o ServiceInstallOptions) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return nativeDarwinSystemBrokerEngine().start(ctx, o)
}
func readNativeSystemServiceBrokerDescriptor(path string) (NativeSystemServiceBrokerOptions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return nativeDarwinSystemBrokerEngine().runtime(ctx, path, os.Getpid())
}
func runNativeSystemSupervisor(ctx context.Context, o NativeSystemServiceBrokerOptions, run func(context.Context) error) error {
	roots := darwinNativeSystemBrokerRoots()
	path := nativeSystemBrokerDescriptorPath(filepath.Join(roots.Install, "versions", o.ExecutableSHA256, "vrooli"), o)
	verified, err := readNativeSystemServiceBrokerDescriptor(path)
	if err != nil {
		return err
	}
	if verified != o {
		return ErrOperationIdentity
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return run(ctx)
}

// Privileged validation never resolves tools through the caller's PATH or
// stages inputs under its TMPDIR. The renderer/XML rules remain shared.
func validateDarwinSystemBrokerArtifact(a RenderedArtifact, stateRoot string) Verdict {
	const tool = "/usr/bin/plutil"
	if err := darwinSystemBrokerCheck(tool, false); err != nil {
		return Verdict{State: VerdictUnavailable, Validator: tool, Output: err.Error()}
	}
	if err := darwinSystemBrokerCheck(stateRoot, true); err != nil {
		return Verdict{State: VerdictRejected, Validator: tool, Output: err.Error()}
	}
	for _, file := range a.Files {
		if err := xmlWellFormed(file.Content); err != nil {
			return Verdict{State: VerdictRejected, Validator: tool, Output: err.Error()}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), validatorTimeout)
	defer cancel()
	for _, file := range a.Files {
		temp, err := os.CreateTemp(stateRoot, ".plist-verify-*")
		if err != nil {
			return Verdict{State: VerdictUnavailable, Validator: tool, Output: err.Error()}
		}
		path := temp.Name()
		defer os.Remove(path)
		if _, err = temp.WriteString(file.Content); err == nil {
			err = temp.Sync()
		}
		ce := temp.Close()
		if err != nil {
			return Verdict{State: VerdictUnavailable, Validator: tool, Output: err.Error()}
		}
		if ce != nil {
			return Verdict{State: VerdictUnavailable, Validator: tool, Output: ce.Error()}
		}
		cmd := exec.CommandContext(ctx, tool, "-lint", path)
		cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "LC_ALL=C"}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return Verdict{State: VerdictRejected, Validator: tool, Output: fmt.Sprintf("%v: %s", err, out)}
		}
	}
	return Verdict{State: VerdictAccepted, Validator: tool}
}

func nativeSystemSupervisorPID(ctx context.Context) (int, error) {
	return nativeDarwinSystemBrokerEngine().supervisorPID(ctx)
}
