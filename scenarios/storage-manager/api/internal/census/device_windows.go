//go:build windows

package census

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type hostDeviceProbe struct{}

func NewDeviceProbe() DeviceProbe { return hostDeviceProbe{} }

func (hostDeviceProbe) Probe(root string) (DeviceInfo, error) {
	path, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return DeviceInfo{}, err
	}
	var free, total, freeTotal uint64
	if err := windows.GetDiskFreeSpaceEx(path, &free, &total, &freeTotal); err != nil {
		return DeviceInfo{}, fmt.Errorf("GetDiskFreeSpaceEx: %w", err)
	}
	return DeviceInfo{TotalBytes: int64(total), AvailableBytes: int64(free), Privilege: "least-privilege"}, nil
}
