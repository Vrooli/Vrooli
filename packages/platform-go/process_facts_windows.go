//go:build windows

package platform

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openProcessForObservation(pid int, access uint32) (windows.Handle, error) {
	if pid <= 0 || uint64(pid) > 1<<32-1 {
		return 0, fmt.Errorf("platform: invalid pid %d", pid)
	}
	if pid == os.Getpid() {
		return windows.CurrentProcess(), nil
	}
	return windows.OpenProcess(access, false, uint32(pid))
}

// The native PEB layout is architecture dependent. Reject cross-bitness rather
// than interpreting a foreign pointer layout as the observer's layout.
func windowsProcessParameters(pid int) (windows.Handle, windows.RTL_USER_PROCESS_PARAMETERS, error) {
	var params windows.RTL_USER_PROCESS_PARAMETERS
	h, err := openProcessForObservation(pid, windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ)
	if err != nil {
		return 0, params, err
	}
	fail := func(err error) (windows.Handle, windows.RTL_USER_PROCESS_PARAMETERS, error) {
		windows.CloseHandle(h)
		return 0, params, err
	}
	var target, native, self, selfNative uint16
	if err = windows.IsWow64Process2(h, &target, &native); err != nil {
		return fail(err)
	}
	if err = windows.IsWow64Process2(windows.CurrentProcess(), &self, &selfNative); err != nil {
		return fail(err)
	}
	if target != self || native != selfNative {
		return fail(fmt.Errorf("platform: cross-architecture process metadata: %w", ErrUnsupported))
	}
	var basic windows.PROCESS_BASIC_INFORMATION
	if err = windows.NtQueryInformationProcess(h, windows.ProcessBasicInformation, unsafe.Pointer(&basic), uint32(unsafe.Sizeof(basic)), nil); err != nil {
		return fail(err)
	}
	var peb windows.PEB
	if err = readWindowsMemory(h, uintptr(unsafe.Pointer(basic.PebBaseAddress)), unsafe.Pointer(&peb), unsafe.Sizeof(peb)); err != nil {
		return fail(err)
	}
	if err = readWindowsMemory(h, uintptr(unsafe.Pointer(peb.ProcessParameters)), unsafe.Pointer(&params), unsafe.Sizeof(params)); err != nil {
		return fail(err)
	}
	return h, params, nil
}

func readWindowsMemory(h windows.Handle, address uintptr, dest unsafe.Pointer, size uintptr) error {
	if address == 0 || size == 0 || size > 1<<20 {
		return fmt.Errorf("platform: invalid bounded process memory range")
	}
	var read uintptr
	if err := windows.ReadProcessMemory(h, address, (*byte)(dest), size, &read); err != nil {
		return err
	}
	if read != size {
		return fmt.Errorf("platform: truncated process memory observation")
	}
	return nil
}

func readWindowsProcessString(pid int, selectString func(*windows.RTL_USER_PROCESS_PARAMETERS) windows.NTUnicodeString) (string, error) {
	h, params, err := windowsProcessParameters(pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	s := selectString(&params)
	if s.Length == 0 || s.Length%2 != 0 || s.Length > s.MaximumLength {
		return "", fmt.Errorf("platform: invalid process Unicode string")
	}
	buf := make([]uint16, int(s.Length)/2)
	if err := readWindowsMemory(h, uintptr(unsafe.Pointer(s.Buffer)), unsafe.Pointer(&buf[0]), uintptr(s.Length)); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf), nil
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	h, params, err := windowsProcessParameters(pid)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	n := params.EnvironmentSize
	if n < 4 || n%2 != 0 || n > 1<<20 {
		return nil, fmt.Errorf("platform: invalid bounded environment size")
	}
	buf := make([]uint16, int(n)/2)
	if err := readWindowsMemory(h, uintptr(params.Environment), unsafe.Pointer(&buf[0]), n); err != nil {
		return nil, err
	}
	return parseWindowsProcessEnvironment(buf)
}

func processArguments(pid int) ([]string, error) {
	command, err := processCommandLine(pid)
	if err != nil {
		return nil, err
	}
	args, err := windows.DecomposeCommandLine(command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 || len(args) > maxProcessArguments || len(command) > maxProcessArgumentBytes {
		return nil, fmt.Errorf("platform: invalid bounded process command line")
	}
	return args, nil
}

// QueryFullProcessImageName supplies a path, not an immutable reference to the
// mapped image. Until the image section itself can be identified, fail closed.
func openProcessExecutable(int) (*os.File, error) { return nil, ErrUnsupported }
