//go:build !windows && !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package census

import "fmt"

type hostDeviceProbe struct{}

func NewDeviceProbe() DeviceProbe { return hostDeviceProbe{} }

func (hostDeviceProbe) Probe(_ string) (DeviceInfo, error) {
	return DeviceInfo{}, fmt.Errorf("device totals are unavailable on this platform")
}
