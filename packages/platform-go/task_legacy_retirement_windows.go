//go:build windows

package platform

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Initial administrative migration may retire the old, exact root-sealed SCM
// registration after staging its replacement. It never adopts an arbitrary
// service sharing our name or treats an old descriptor as the new schema.
func retireWindowsLegacySupervisor() error {
	m, e := mgr.Connect()
	if e != nil {
		return e
	}
	defer m.Disconnect()
	service, e := m.OpenService(runtimeSupervisorService)
	if errors.Is(e, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return nil
	}
	if e != nil {
		return e
	}
	defer service.Close()
	config, e := service.Config()
	if e != nil {
		return e
	}
	if config.ServiceType != windows.SERVICE_WIN32_OWN_PROCESS || !strings.EqualFold(config.ServiceStartName, "LocalSystem") {
		return ErrOperationIdentity
	}
	args, e := windows.DecomposeCommandLine(config.BinaryPathName)
	if e != nil || len(args) != 3 || args[1] != NativeSystemSupervisorVerb {
		return ErrOperationIdentity
	}
	roots, e := nativeWindowsSystemBrokerRoots()
	if e != nil {
		return e
	}
	relative, e := filepath.Rel(filepath.Join(roots.Install, "versions"), args[0])
	if e != nil {
		return e
	}
	parts := strings.Split(relative, string(filepath.Separator))
	if len(parts) != 2 || !validNativeBrokerDigest(parts[0]) || parts[1] != "vrooli.exe" || filepath.Dir(args[2]) != filepath.Dir(args[0]) {
		return ErrOperationIdentity
	}
	base := filepath.Base(args[2])
	if !strings.HasPrefix(base, "broker-") || !strings.HasSuffix(base, ".json") || !validNativeBrokerDigest(strings.TrimSuffix(strings.TrimPrefix(base, "broker-"), ".json")) {
		return ErrOperationIdentity
	}
	held, e := windowsEnsureExistingBrokerParents(filepath.Dir(args[0]))
	if e != nil {
		return e
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	h, e := windowsOpenBrokerPath(args[0], false, windows.GENERIC_READ)
	if e != nil {
		return e
	}
	image := os.NewFile(uintptr(h), args[0])
	defer image.Close()
	if e = verifyNativeSystemBrokerImage(image, parts[0]); e != nil {
		return e
	}
	h, e = windowsOpenBrokerPath(args[2], false, windows.GENERIC_READ)
	if e != nil {
		return e
	}
	windows.CloseHandle(h)
	// Stop by the proven canonical service object, retaining its exact process
	// handle until termination so SCM STOPPED alone is not root absence.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	state, e := service.Query()
	if e != nil {
		return e
	}
	var process windows.Handle
	if state.ProcessId != 0 {
		process, e = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ|windows.SYNCHRONIZE, false, state.ProcessId)
		if e != nil {
			return e
		}
		defer windows.CloseHandle(process)
		if e = verifyWindowsMappedImageHandle(process, image); e != nil {
			return e
		}
	}
	if state.State != svc.Stopped && state.State != svc.StopPending {
		if _, e = service.Control(svc.Stop); e != nil && !errors.Is(e, windows.ERROR_SERVICE_NOT_ACTIVE) {
			return e
		}
	}
	for {
		state, e = service.Query()
		if e != nil {
			return e
		}
		if state.State == svc.Stopped {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	if process != 0 {
		for {
			status, e := windows.WaitForSingleObject(process, 50)
			if e != nil {
				return e
			}
			if status == windows.WAIT_OBJECT_0 {
				break
			}
			if status != uint32(windows.WAIT_TIMEOUT) {
				return ErrOperationIdentity
			}
			if e = ctx.Err(); e != nil {
				return e
			}
		}
	}
	confirmed, e := service.Config()
	if e != nil {
		return e
	}
	if confirmed.BinaryPathName != config.BinaryPathName || confirmed.ServiceStartName != config.ServiceStartName {
		return ErrOperationIdentity
	}
	return service.Delete()
}
