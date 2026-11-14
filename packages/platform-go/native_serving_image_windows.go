//go:build windows

package platform

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openCurrentNativeSystemSupervisorImage(ctx context.Context) (*os.File, error) {
	before, err := nativeSystemSupervisorPID(ctx)
	if err != nil {
		return nil, err
	}
	if before != os.Getpid() || len(os.Args) != 3 || os.Args[1] != NativeSystemSupervisorVerb {
		return nil, ErrOperationIdentity
	}
	if _, err = readNativeSystemServiceBrokerDescriptor(os.Args[2]); err != nil {
		return nil, err
	}
	d, p, err := windowsReadBrokerDescriptor(os.Args[2])
	if err != nil {
		return nil, err
	}
	source, err := os.Open(p.ImagePath)
	if err != nil {
		return nil, err
	}
	image, err := lockedWindowsImage(source)
	source.Close()
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) { image.Close(); return nil, err }
	if err = verifyWindowsMappedImageHandle(windows.CurrentProcess(), image); err != nil {
		return fail(err)
	}
	if err = verifyNativeSystemBrokerImage(image, d.Options.ExecutableSHA256); err != nil {
		return fail(err)
	}
	if _, err = image.Seek(0, 0); err != nil {
		return fail(err)
	}
	after, err := nativeSystemSupervisorPID(ctx)
	if err != nil {
		return fail(err)
	}
	if after != before {
		return fail(ErrOperationIdentity)
	}
	return image, nil
}

func windowsServingRootAlive(ctx context.Context, ref ServiceRef) error {
	o, err := InspectOwnedService(ctx, ref)
	if err != nil {
		return err
	}
	if !o.Ref.Equal(OperationRef(ref)) || o.State != OperationRunning || !o.RootAlive || !o.Populated {
		return ErrOperationIdentity
	}
	return nil
}

// Pin the process object before interpreting its inherited handle table. A PID
// reused during observation cannot redirect DuplicateHandle to another process.
func windowsServingProcess(pid int, incarnation string) (windows.Handle, error) {
	process, err := windows.OpenProcess(windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	fail := func(err error) (windows.Handle, error) { windows.CloseHandle(process); return 0, err }
	var c, x, k, u windows.Filetime
	if err = windows.GetProcessTimes(process, &c, &x, &k, &u); err != nil {
		return fail(err)
	}
	if incarnation != fmt.Sprintf("windows:%d:%d", pid, c.Nanoseconds()) {
		return fail(ErrOperationIdentity)
	}
	if err = windowsServingProcessAlive(process); err != nil {
		return fail(err)
	}
	return process, nil
}
func windowsServingProcessAlive(process windows.Handle) error {
	state, err := windows.WaitForSingleObject(process, 0)
	if err != nil {
		return err
	}
	if state != uint32(windows.WAIT_TIMEOUT) {
		return ErrOperationIdentity
	}
	return nil
}
func windowsServingBirthImage(process windows.Handle, handle uint64, imageID string) (*os.File, error) {
	if handle == 0 || imageID == "" || uint64(uintptr(handle)) != handle {
		return nil, ErrOperationIdentity
	}
	var name [32768]uint16
	size := uint32(len(name))
	if err := windows.QueryFullProcessImageName(process, 0, &name[0], &size); err != nil {
		return nil, err
	}
	var duplicate windows.Handle
	if err := windows.DuplicateHandle(process, windows.Handle(handle), windows.CurrentProcess(), &duplicate, windows.GENERIC_READ, false, 0); err != nil {
		return nil, err
	}
	image := os.NewFile(uintptr(duplicate), windows.UTF16ToString(name[:size]))
	fail := func(err error) (*os.File, error) { image.Close(); return nil, err }
	actual, err := windowsImageID(image)
	if err != nil {
		return fail(err)
	}
	if actual != imageID {
		return fail(ErrOperationIdentity)
	}
	if err = verifyWindowsMappedImageHandle(process, image); err != nil {
		return fail(err)
	}
	if err = windowsServingProcessAlive(process); err != nil {
		return fail(err)
	}
	return image, nil
}
func openOwnedServiceRootImage(ctx context.Context, ref ServiceRef) (*os.File, error) {
	if ref.Native == nil || ref.Native.Platform != "windows" {
		return nil, ErrOperationIdentity
	}
	if err := windowsServingRootAlive(ctx, ref); err != nil {
		return nil, err
	}
	record, err := readWindowsOwnedRecord(ref.Intent)
	if err != nil {
		return nil, err
	}
	if record.PID != ref.RootPID || record.Start != ref.RootStartTime || record.Incarnation != ref.Native.RootIncarnation {
		return nil, ErrOperationIdentity
	}
	process, err := windowsServingProcess(ref.RootPID, record.Incarnation)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(process)
	image, err := windowsServingBirthImage(process, record.ImageHandle, record.ImageID)
	if err != nil {
		return nil, err
	}
	if err = windowsServingRootAlive(ctx, ref); err == nil {
		err = windowsServingProcessAlive(process)
	}
	if err != nil {
		image.Close()
		return nil, err
	}
	return image, nil
}
func openOwnedServiceChildImage(ctx context.Context, ref ServiceRef, id OwnedImageChildIdentity) (*os.File, error) {
	if id.ImageHandle == 0 {
		return nil, ErrOperationIdentity
	}
	if err := windowsServingRootAlive(ctx, ref); err != nil {
		return nil, err
	}
	process, err := windowsServingProcess(id.PID, id.Incarnation)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(process)
	image, err := windowsServingBirthImage(process, id.ImageHandle, id.ImageID)
	if err != nil {
		return nil, err
	}
	if err = VerifyOwnedImageChild(ctx, ref, id, image); err == nil {
		err = windowsServingProcessAlive(process)
	}
	if err != nil {
		image.Close()
		return nil, err
	}
	return image, nil
}
