//go:build windows

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func installNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Name == "" || options.Path == "" {
		return NativeServiceResult{}, fmt.Errorf("platform: task name and path are required")
	}
	if err := os.WriteFile(options.Path, []byte(options.Content), 0o600); err != nil {
		return NativeServiceResult{}, err
	}
	if err := registerWindowsTask(options.Name, options.Content); err != nil {
		return NativeServiceResult{}, err
	}
	if err := windowsTaskControl(options.Name, "start"); err != nil {
		return NativeServiceResult{}, err
	}
	return nativeServiceStatus(options)
}
func uninstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	if err := stopNativeService(options); err != nil && !taskNativeErrorMissing(err) {
		return NativeServiceResult{}, err
	}
	if err := windowsDeleteTask(options.Name); err != nil && !taskNativeErrorMissing(err) {
		return NativeServiceResult{}, err
	}
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "machine", State: ServiceStateStopped}, nil
}
func startNativeService(options NativeServiceOptions) error {
	if options.Name == runtimeSupervisorService {
		return windowsOperatorTaskLifecycle("start")
	}
	return windowsTaskControl(options.Name, "start")
}
func stopNativeService(options NativeServiceOptions) error {
	if options.Name == runtimeSupervisorService {
		return windowsOperatorTaskLifecycle("stop")
	}
	return windowsTaskControl(options.Name, "stop")
}
func restartNativeService(options NativeServiceOptions) error {
	if options.Name == runtimeSupervisorService {
		return windowsOperatorTaskLifecycle("restart")
	}
	if err := stopNativeService(options); err != nil {
		return err
	}
	return startNativeService(options)
}
func windowsOperatorTaskLifecycle(verb string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return controlWindowsInstalledOperatorTask(ctx, verb)
}

func nativeServiceStatus(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Kind == KindDaemon && options.Name != runtimeSupervisorService {
		if result, found := scmServiceStatus(options); found {
			return result, nil
		}
	}
	result := NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "machine", State: ServiceStateUnknown, Evidence: ServiceEvidence{Source: "TaskScheduler.COM"}}
	var snapshot WindowsTaskSnapshot
	var err error
	if options.Name == runtimeSupervisorService {
		_, _, snapshot, err = windowsInstalledTaskPlan()
	} else {
		snapshot, err = WindowsTaskStatus(options.Name)
	}
	if err != nil {
		if taskNativeErrorMissing(err) {
			result.State = ServiceStateStopped
			return result, nil
		}
		result.Evidence.Detail = err.Error()
		return result, err
	}
	result.Enabled = snapshot.Enabled
	result.Evidence.RawState = strconv.Itoa(int(snapshot.State))
	switch snapshot.State {
	case 4:
		if options.Name == runtimeSupervisorService {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, err = nativeSystemSupervisorPID(ctx)
			cancel()
			if err != nil {
				result.Evidence.Detail = err.Error()
				return result, err
			}
		}
		result.State = ServiceStateRunning
	case windowsTaskDisabled, windowsTaskReady:
		quiet, e := windowsTaskQuiescent(snapshot)
		if e != nil {
			result.Evidence.Detail = e.Error()
			return result, e
		}
		if quiet {
			result.State = ServiceStateStopped
		}
	}
	result.Running = result.State == ServiceStateRunning
	return result, nil
}

// scmServiceStatus reports a registered Windows service, or found=false when
// no service of that name exists so the caller can fall back to a task.
func scmServiceStatus(options NativeServiceOptions) (NativeServiceResult, bool) {
	machine, err := mgr.Connect()
	if err != nil {
		return NativeServiceResult{}, false
	}
	defer machine.Disconnect()
	service, err := machine.OpenService(options.Name)
	if err != nil {
		return NativeServiceResult{}, false
	}
	defer service.Close()
	result := NativeServiceResult{Name: options.Name, Scope: "machine", State: ServiceStateUnknown, Evidence: ServiceEvidence{Source: "scm"}}
	status, err := service.Query()
	if err != nil {
		result.Evidence.Detail = err.Error()
		return result, true
	}
	result.Evidence.RawState = strconv.Itoa(int(status.State))
	switch status.State {
	case svc.Running:
		result.State = ServiceStateRunning
	case svc.Stopped:
		result.State = ServiceStateStopped
	default:
		result.State = ServiceStateUnknown
	}
	result.Running = result.State == ServiceStateRunning
	if config, err := service.Config(); err == nil {
		result.Enabled = config.StartType == mgr.StartAutomatic
	}
	return result, true
}

func nativeServiceLogs(NativeServiceOptions, int) ([]byte, error) {
	return nil, fmt.Errorf("platform: Windows scheduled tasks do not expose service logs")
}

func readHostLogs(options HostLogOptions) (HostLogResult, error) {
	command := "Get-WinEvent -FilterHashtable @{LogName='System'} | Select-Object TimeCreated,ProviderName,Id,Message,MachineName,Level,ProcessId | ConvertTo-Json -Depth 4 -Compress"
	if options.Tail > 0 {
		command = fmt.Sprintf("Get-WinEvent -FilterHashtable @{LogName='System'} | Select-Object -First %d TimeCreated,ProviderName,Id,Message,MachineName,Level,ProcessId | ConvertTo-Json -Depth 4 -Compress", options.Tail)
	}
	args := []string{"-NoProfile", "-NonInteractive", "-Command", command}
	out, err := exec.Command("powershell.exe", args...).CombinedOutput()
	return HostLogResult{Source: "Get-WinEvent", Raw: out, Entries: ParseWindowsEventJSON(out), Evidence: ServiceEvidence{Source: "Get-WinEvent", Detail: strings.TrimSpace(string(out))}}, err
}

const runtimeSupervisorService = "VrooliRuntimeSupervisor"

func installService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: per-user Windows RuntimeSupervisor is unsupported; explicit sealed boot-task setup is required: %w", ErrUnsupported)
	}
	if options.SystemBroker == nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: explicit checksum-bound SystemBroker options required")
	}
	return installWindowsSystemBroker(options)
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: interactive-user request cannot remove a machine boot task: %w", ErrUnsupported)
	}
	return uninstallWindowsSystemBroker(options)
}
func startInstalledService(options ServiceInstallOptions) (bool, error) {
	if options.User {
		return false, fmt.Errorf("platform: interactive-user request cannot start a machine boot task: %w", ErrUnsupported)
	}
	return startWindowsSystemBroker(options)
}

func supportsService(user bool) bool {
	if user {
		return false
	}
	return withTaskScheduler(func(*taskDispatch) error { return nil }) == nil
}

func serviceStartHint() string { return "vrooli runtime service start" }
