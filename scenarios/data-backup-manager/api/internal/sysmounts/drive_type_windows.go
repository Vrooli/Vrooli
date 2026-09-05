//go:build windows

package sysmounts

import "golang.org/x/sys/windows"

func platformDriveType(root string) uint32 {
	ptr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return driveTypeUnknown
	}
	return windows.GetDriveType(ptr)
}
