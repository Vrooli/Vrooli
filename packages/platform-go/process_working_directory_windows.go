//go:build windows

package platform

import (
	"errors"
	"golang.org/x/sys/windows"
	"unsafe"
)

func readWindowsDirectoryParameters(h windows.Handle) (windows.RTL_USER_PROCESS_PARAMETERS, error) {
	var params windows.RTL_USER_PROCESS_PARAMETERS
	var basic windows.PROCESS_BASIC_INFORMATION
	if err := windows.NtQueryInformationProcess(h, windows.ProcessBasicInformation, unsafe.Pointer(&basic), uint32(unsafe.Sizeof(basic)), nil); err != nil {
		return params, err
	}
	var peb windows.PEB
	if err := readWindowsMemory(h, uintptr(unsafe.Pointer(basic.PebBaseAddress)), unsafe.Pointer(&peb), unsafe.Sizeof(peb)); err != nil {
		return params, err
	}
	err := readWindowsMemory(h, uintptr(unsafe.Pointer(peb.ProcessParameters)), unsafe.Pointer(&params), unsafe.Sizeof(params))
	return params, err
}
func processWorkingDirectoryMatches(pid int, expected string) (bool, error) {
	h, err := openProcessForObservation(pid, windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ|windows.PROCESS_DUP_HANDLE)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(h)
	var target, native, self, selfNative uint16
	if err = windows.IsWow64Process2(h, &target, &native); err != nil {
		return false, err
	}
	if err = windows.IsWow64Process2(windows.CurrentProcess(), &self, &selfNative); err != nil {
		return false, err
	}
	if target != self || native != selfNative {
		return false, ErrUnsupported
	}
	params, err := readWindowsDirectoryParameters(h)
	if err != nil {
		return false, err
	}
	if params.CurrentDirectory.Handle == 0 {
		return false, errors.New("platform: process has no observable current-directory handle")
	}
	var directory windows.Handle
	if err = windows.DuplicateHandle(h, params.CurrentDirectory.Handle, windows.CurrentProcess(), &directory, windows.FILE_READ_ATTRIBUTES, false, 0); err != nil {
		return false, err
	}
	defer windows.CloseHandle(directory)
	name, err := windows.UTF16PtrFromString(expected)
	if err != nil {
		return false, err
	}
	want, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(want)
	var actualInfo, wantInfo windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(directory, &actualInfo); err != nil {
		return false, err
	}
	if err = windows.GetFileInformationByHandle(want, &wantInfo); err != nil {
		return false, err
	}
	after, err := readWindowsDirectoryParameters(h)
	if err != nil {
		return false, err
	}
	if params.CurrentDirectory.Handle != after.CurrentDirectory.Handle {
		return false, errors.New("platform: process directory changed during observation")
	}
	return actualInfo.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 && wantInfo.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 && actualInfo.VolumeSerialNumber == wantInfo.VolumeSerialNumber && actualInfo.FileIndexHigh == wantInfo.FileIndexHigh && actualInfo.FileIndexLow == wantInfo.FileIndexLow, nil
}
