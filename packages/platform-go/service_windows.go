//go:build windows

package platform

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func nativeTask(options NativeServiceOptions, action string) ([]byte, error) {
	args := []string{action, "/TN", options.Name}
	if action == "/Create" {
		args = append(args, "/XML", options.Path, "/F")
	} else if action == "/Query" {
		args = append(args, "/FO", "CSV", "/NH")
	}
	return exec.Command("schtasks", args...).CombinedOutput()
}

func installNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Name == "" || options.Path == "" {
		return NativeServiceResult{}, fmt.Errorf("platform: task name and path are required")
	}
	if err := os.WriteFile(options.Path, []byte(options.Content), 0o600); err != nil {
		return NativeServiceResult{}, err
	}
	if output, err := nativeTask(options, "/Create"); err != nil {
		return NativeServiceResult{}, fmt.Errorf("platform: create task: %w: %s", err, output)
	}
	_, _ = nativeTask(options, "/Run")
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "machine", Running: true, Enabled: true}, nil
}

func uninstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	output, err := nativeTask(options, "/Delete")
	if err != nil && !strings.Contains(strings.ToLower(string(output)), "does not exist") {
		return NativeServiceResult{}, err
	}
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "machine"}, nil
}

func startNativeService(options NativeServiceOptions) error {
	_, err := nativeTask(options, "/Run")
	return err
}

func stopNativeService(options NativeServiceOptions) error {
	_, err := nativeTask(options, "/End")
	return err
}

func restartNativeService(options NativeServiceOptions) error {
	_ = stopNativeService(options)
	return startNativeService(options)
}

// nativeServiceStatus asks the backend the unit kind selects: a daemon is a
// Service Control Manager service when one is registered under the name
// (the runtime supervisor), and otherwise a boot-triggered task (the
// autoheal loop); oneshots and timers are always tasks. Asking schtasks
// about an SCM service reports "does not exist" for a running daemon.
func nativeServiceStatus(options NativeServiceOptions) (NativeServiceResult, error) {
	if options.Kind == KindDaemon {
		if result, found := scmServiceStatus(options); found {
			return result, nil
		}
	}
	output, err := nativeTask(options, "/Query")
	state := parseSchtasksState(string(output), err)
	return NativeServiceResult{Name: options.Name, Path: options.Path, Scope: "machine", Running: state == ServiceStateRunning, Enabled: err == nil, State: state, Evidence: ServiceEvidence{Source: "schtasks /Query /FO CSV", RawState: strings.TrimSpace(string(output))}}, nil
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

func parseSchtasksState(raw string, commandErr error) ServiceState {
	// `/FO LIST` exposes a numeric STATE code before the localized display
	// label. Prefer that code so a non-English host cannot turn a running task
	// into an unknown state merely because its renderer says something other
	// than "Running".
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) == 0 {
			continue
		}
		code := strings.TrimSpace(fields[0])
		switch code {
		case "4":
			return ServiceStateRunning
		case "1", "2", "3", "5":
			return ServiceStateStopped
		}
	}

	reader := csv.NewReader(strings.NewReader(raw))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 3 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(record[2])) {
		case "running":
			return ServiceStateRunning
		case "ready", "disabled":
			return ServiceStateStopped
		}
	}
	if commandErr != nil && strings.Contains(strings.ToLower(raw), "does not exist") {
		return ServiceStateStopped
	}
	return ServiceStateUnknown
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
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service install requires explicit broker support")
	}
	executable := options.Executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable: %w", err)
		}
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable path: %w", err)
	}
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	// The SCM takes the definition's command line directly; there is no
	// rendered artifact to validate, so the verdict records the SCM accepting
	// the registration.
	definition, err := RuntimeSupervisorDefinition("windows", RuntimeSupervisorOptions{Home: home, Executable: executable, SourceRoot: options.SourceRoot, LogPath: options.LogPath})
	if err != nil {
		return ServiceInstallResult{}, err
	}
	machine, err := mgr.Connect()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: connect to Windows SCM: %w", err)
	}
	defer machine.Disconnect()

	service, err := machine.CreateService(runtimeSupervisorService, executable, mgr.Config{
		DisplayName:    "Vrooli Runtime Supervisor",
		StartType:      mgr.StartAutomatic,
		ErrorControl:   mgr.ErrorNormal,
		ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
		Description:    definition.Description,
		BinaryPathName: WindowsServiceCommandLine(definition),
	})
	if err != nil {
		if !errors.Is(err, windows.ERROR_SERVICE_EXISTS) {
			return ServiceInstallResult{}, fmt.Errorf("platform: create Windows service: %w", err)
		}
		service, err = machine.OpenService(runtimeSupervisorService)
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: open existing Windows service: %w", err)
		}
	}
	defer service.Close()
	if err := service.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return ServiceInstallResult{}, fmt.Errorf("platform: start Windows service: %w", err)
	}
	// Start() returning nil only means the SCM accepted the start request.
	// Query the settled state so an install cannot report success for a
	// service that never reached Running.
	verdict := Verdict{State: VerdictAccepted, Validator: "scm-create-service"}
	if state := awaitWindowsRunning(service); state != svc.Running {
		return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: false, Verdict: verdict},
			fmt.Errorf("platform: installed %s but its state is %d, not running", runtimeSupervisorService, state)
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: true, Verdict: verdict}, nil
}

// awaitWindowsRunning polls the SCM until the service settles, reporting the
// last observation so a start-then-exit is not mistaken for a start.
func awaitWindowsRunning(service *mgr.Service) svc.State {
	var state svc.State
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(250 * time.Millisecond)
		}
		status, err := service.Query()
		if err != nil {
			continue
		}
		state = status.State
	}
	return state
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service removal requires explicit broker support")
	}
	machine, err := mgr.Connect()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: connect to Windows SCM: %w", err)
	}
	defer machine.Disconnect()
	service, err := machine.OpenService(runtimeSupervisorService)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: false}, nil
		}
		return ServiceInstallResult{}, fmt.Errorf("platform: open Windows service: %w", err)
	}
	defer service.Close()
	if _, err := service.Control(svc.Stop); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return ServiceInstallResult{}, fmt.Errorf("platform: stop Windows service: %w", err)
	}
	if err := service.Delete(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return ServiceInstallResult{}, fmt.Errorf("platform: delete Windows service: %w", err)
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: false}, nil
}

func startInstalledService(ServiceInstallOptions) (bool, error) {
	machine, err := mgr.Connect()
	if err != nil {
		return false, fmt.Errorf("platform: connect to Windows SCM: %w", err)
	}
	defer machine.Disconnect()
	service, err := machine.OpenService(runtimeSupervisorService)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return false, nil
		}
		return false, fmt.Errorf("platform: open Windows service: %w", err)
	}
	defer service.Close()
	if err := service.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return false, fmt.Errorf("platform: start Windows service: %w", err)
	}
	if state := awaitWindowsRunning(service); state != svc.Running {
		return false, fmt.Errorf("platform: started %s but its state is %d, not running", runtimeSupervisorService, state)
	}
	return true, nil
}

func supportsService(user bool) bool {
	if !user {
		return false
	}
	machine, err := mgr.Connect()
	if err != nil {
		return false
	}
	_ = machine.Disconnect()
	return true
}

func serviceStartHint() string { return "sc start VrooliRuntimeSupervisor" }
