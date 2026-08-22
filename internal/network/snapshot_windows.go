//go:build windows

package network

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// captureTCPListenerSnapshot builds the global listening-port set from
// GetExtendedTcpTable (iphlpapi, TCP_TABLE_OWNER_PID_LISTENER) for both
// address families — zero forks, all users, owner PIDs included. x/sys does
// not wrap this API, so it is loaded via a lazy system DLL.
// Attribution is free here: GetExtendedTcpTable already returns owner PIDs,
// so opts is accepted for signature parity and never changes the work done.
func captureTCPListenerSnapshot(_ CaptureOptions) TCPListenerSnapshot {
	ports := make(map[int][]SnapshotListener)
	for _, family := range []uint32{windows.AF_INET, windows.AF_INET6} {
		entries, err := listListenerTable(family)
		if err != nil {
			// Both families are required. A snapshot missing one family would
			// still read Known:true, so every listener bound in the failed
			// family would be "known-absent" — and reconcile expires claims on
			// known-absent listeners. Degrade the whole snapshot to unknown.
			return TCPListenerSnapshot{
				Reason: fmt.Sprintf("GetExtendedTcpTable failed: %v", err),
				Tool:   "iphlpapi",
			}
		}
		for _, entry := range entries {
			if entry.port <= 0 || entry.port > 65535 {
				continue
			}
			if entry.pid > 0 && !containsListenerPID(ports[entry.port], entry.pid) {
				ports[entry.port] = append(ports[entry.port], SnapshotListener{
					PID:   entry.pid,
					Label: processImageLabel(entry.pid),
				})
			} else if _, ok := ports[entry.port]; !ok {
				ports[entry.port] = nil
			}
		}
	}
	return TCPListenerSnapshot{Known: true, Tool: "iphlpapi", Ports: ports}
}

var (
	modiphlpapi             = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtendedTcpTable = modiphlpapi.NewProc("GetExtendedTcpTable")
)

const tcpTableOwnerPidListener = 3 // TCP_TABLE_OWNER_PID_LISTENER

// mibTCPRowOwnerPid mirrors MIB_TCPROW_OWNER_PID (IPv4).
type mibTCPRowOwnerPid struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

// mibTCP6RowOwnerPid mirrors MIB_TCP6ROW_OWNER_PID (IPv6).
type mibTCP6RowOwnerPid struct {
	LocalAddr     [16]byte
	LocalScopeID  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeID uint32
	RemotePort    uint32
	State         uint32
	OwningPid     uint32
}

type listenerTableEntry struct {
	port int
	pid  int
}

func listListenerTable(family uint32) ([]listenerTableEntry, error) {
	var size uint32
	// First call sizes the buffer (ERROR_INSUFFICIENT_BUFFER expected).
	_, _, _ = procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(family), tcpTableOwnerPidListener, 0)
	if size == 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable returned zero size for family %d", family)
	}
	// The table can grow between the sizing call and the fetch — exactly when
	// scenarios are starting or stopping — and the API then fails with
	// ERROR_INSUFFICIENT_BUFFER after updating size. Retry with the updated
	// size instead of trusting a single exact-size fetch.
	var buf []byte
	for attempt := 0; ; attempt++ {
		buf = make([]byte, size)
		ret, _, _ := procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(family), tcpTableOwnerPidListener, 0)
		if ret == 0 {
			break
		}
		if syscall.Errno(ret) != windows.ERROR_INSUFFICIENT_BUFFER || attempt >= 3 {
			return nil, fmt.Errorf("GetExtendedTcpTable(family=%d) failed: code %d", family, ret)
		}
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("GetExtendedTcpTable(family=%d) returned short buffer", family)
	}
	count := binary.LittleEndian.Uint32(buf[:4])
	rows := buf[4:]
	entries := make([]listenerTableEntry, 0, count)
	switch family {
	case windows.AF_INET:
		rowSize := int(unsafe.Sizeof(mibTCPRowOwnerPid{}))
		for i := 0; i < int(count) && (i+1)*rowSize <= len(rows); i++ {
			row := (*mibTCPRowOwnerPid)(unsafe.Pointer(&rows[i*rowSize]))
			entries = append(entries, listenerTableEntry{port: ntohsPort(row.LocalPort), pid: int(row.OwningPid)})
		}
	case windows.AF_INET6:
		rowSize := int(unsafe.Sizeof(mibTCP6RowOwnerPid{}))
		for i := 0; i < int(count) && (i+1)*rowSize <= len(rows); i++ {
			row := (*mibTCP6RowOwnerPid)(unsafe.Pointer(&rows[i*rowSize]))
			entries = append(entries, listenerTableEntry{port: ntohsPort(row.LocalPort), pid: int(row.OwningPid)})
		}
	}
	return entries, nil
}

// ntohsPort converts the DWORD-encoded network-byte-order port used by the
// MIB row structs into a host-order port number.
func ntohsPort(dwPort uint32) int {
	return int(dwPort>>8&0xff | dwPort<<8&0xff00)
}

// processImageLabel best-effort resolves a PID to its executable path; an
// empty label is fine (attribution is optional enrichment).
func processImageLabel(pid int) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}
