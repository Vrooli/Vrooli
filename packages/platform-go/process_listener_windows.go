//go:build windows

package platform

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

func nativeProcessTCPListeners(pid int) ([]ProcessTCPListener, error) {
	if pid < 0 || uint64(pid) > 1<<32-1 {
		return nil, fmt.Errorf("platform: invalid pid %d", pid)
	}
	var result []ProcessTCPListener
	for _, family := range []uint32{windows.AF_INET, windows.AF_INET6} {
		var size uint32
		status, _, _ := getExtendedTCPTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(family), 3, 0) // TCP_TABLE_OWNER_PID_LISTENER
		if status != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			return nil, windows.Errno(status)
		}
		if size < 4 || size > 16<<20 {
			return nil, fmt.Errorf("platform: TCP owner table exceeds bounds")
		}
		var data []byte
		// The live table may grow between sizing and read. Bounded retries never
		// turn an incomplete table into an empty ownership observation.
		for attempt := 0; attempt < 3; attempt++ {
			if size < 4 || size > 16<<20 {
				return nil, fmt.Errorf("platform: TCP owner table exceeds bounds")
			}
			data = make([]byte, size)
			status, _, _ = getExtendedTCPTable.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(family), 3, 0)
			if status != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
				break
			}
		}
		if status != 0 {
			return nil, windows.Errno(status)
		}
		if int(size) > len(data) || size < 4 {
			return nil, fmt.Errorf("platform: truncated TCP table")
		}
		endpoints, err := parseWindowsTCPOwnerTable(data[:size], family == windows.AF_INET6, uint32(pid))
		if err != nil {
			return nil, err
		}
		result = append(result, endpoints...)
	}
	return result, nil
}

func nativeTCPListeners(port uint16) ([]ProcessTCPListener, error) {
	listeners, err := nativeProcessTCPListeners(0)
	if err != nil {
		return nil, err
	}
	var result []ProcessTCPListener
	for _, listener := range listeners {
		if listener.Port == port {
			result = append(result, listener)
		}
	}
	return result, nil
}
