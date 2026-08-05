//go:build aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris

package census

import (
	"os"
	"syscall"
)

type hostDeviceProbe struct{}

func NewDeviceProbe() DeviceProbe { return hostDeviceProbe{} }

func (hostDeviceProbe) Probe(root string) (DeviceInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return DeviceInfo{}, err
	}
	blockSize := int64(stat.Bsize)
	if blockSize <= 0 {
		return DeviceInfo{}, syscall.EINVAL
	}
	privilege := "least-privilege"
	if os.Geteuid() == 0 {
		privilege = "privileged"
	}
	return DeviceInfo{TotalBytes: int64(stat.Blocks) * blockSize, AvailableBytes: int64(stat.Bavail) * blockSize, Privilege: privilege}, nil
}
