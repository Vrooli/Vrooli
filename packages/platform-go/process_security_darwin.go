//go:build darwin

package platform

import (
	"fmt"
	"golang.org/x/sys/unix"
	"strconv"
)

func nativeProcessSecurity(pid int) (ProcessSecurityContext, error) {
	before, err := darwinObservationStamp(pid)
	if err != nil {
		return ProcessSecurityContext{}, err
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return ProcessSecurityContext{}, err
	}
	if info == nil {
		return ProcessSecurityContext{}, fmt.Errorf("platform: process security unavailable")
	}
	after, err := darwinObservationStamp(pid)
	if err != nil || before != after {
		return ProcessSecurityContext{}, ErrOperationIdentity
	}
	uid := int(info.Eproc.Ucred.Uid)
	return ProcessSecurityContext{Principal: strconv.Itoa(uid), EffectiveUID: uid, Elevated: uid == 0}, nil
}
