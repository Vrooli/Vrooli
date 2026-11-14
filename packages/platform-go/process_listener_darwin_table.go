package platform

import (
	"encoding/binary"
	"fmt"
	"net"
)

// The public sysctl's extensible records have length/kind prefixes and packed-4
// fixed-width fields. These prefix offsets follow Apple's XNU in_pcb.h,
// socketvar.h and tcp_var.h. Never reinterpret a host Go struct as that ABI.
// Fields appended to a record do not change these prefixes.
func parseDarwinTCPPCBList(data []byte, port uint16) ([]ProcessTCPListener, error) {
	const genSize = 24
	bad := func() ([]ProcessTCPListener, error) {
		return nil, fmt.Errorf("platform: incomplete, changed or unrecognized Darwin TCP PCB snapshot")
	}
	if len(data) < genSize || binary.LittleEndian.Uint32(data) != genSize {
		return bad()
	}
	count := binary.LittleEndian.Uint32(data[4:])
	if len(data) == genSize {
		if count == 0 {
			return []ProcessTCPListener{}, nil
		}
		return bad()
	}
	var result []ProcessTCPListener
	offset, groups := genSize, uint32(0)
	// XNU emits one complete ordered group per PCB. Each record is rounded up
	// to 8 bytes independently; an unknown/missing group is not absence evidence.
	order := []uint32{0x10, 0x1, 0x2, 0x4, 0x8, 0x20}
	for offset < len(data) {
		if len(data)-offset == genSize && binary.LittleEndian.Uint32(data[offset:]) == genSize {
			trailer := data[offset:]
			if binary.LittleEndian.Uint32(trailer[4:]) != count || binary.LittleEndian.Uint64(trailer[8:]) != binary.LittleEndian.Uint64(data[8:]) || binary.LittleEndian.Uint64(trailer[16:]) != binary.LittleEndian.Uint64(data[16:]) || groups != count {
				return bad()
			}
			return result, nil
		}
		var pcb, socket, tcp []byte
		for _, kind := range order {
			if len(data)-offset < 8 {
				return bad()
			}
			n := uint64(binary.LittleEndian.Uint32(data[offset:]))
			rounded := (n + 7) &^ uint64(7)
			if n < 8 || rounded > uint64(len(data)-offset) || binary.LittleEndian.Uint32(data[offset+4:]) != kind {
				return bad()
			}
			record := data[offset : offset+int(n)]
			switch kind {
			case 0x10:
				pcb = record
			case 1:
				socket = record
			case 0x20:
				tcp = record
			}
			offset += int(rounded)
		}
		groups++
		if len(pcb) < 104 || len(socket) < 76 || len(tcp) < 40 {
			return bad()
		}
		// Ignore PCB instances created after the snapshot's initial generation.
		if binary.LittleEndian.Uint64(pcb[28:]) > binary.LittleEndian.Uint64(data[8:]) {
			return bad()
		}
		state := binary.LittleEndian.Uint32(tcp[36:])
		if state > 11 {
			return bad()
		}
		if state != 1 {
			continue
		} // TCPS_LISTEN
		if binary.LittleEndian.Uint32(socket[36:]) != 6 || binary.LittleEndian.Uint16(socket[16:]) != 1 {
			return bad()
		} // TCP stream
		localPort := binary.BigEndian.Uint16(pcb[18:])
		if localPort != port {
			continue
		}
		pid := int(int32(binary.LittleEndian.Uint32(socket[68:])))
		effectivePID := int(int32(binary.LittleEndian.Uint32(socket[72:])))
		// Delegated sockets belong to their recorded effective process. These IDs
		// describe kernel last/effective ownership, not every duplicated FD holder.
		if effectivePID != 0 {
			pid = effectivePID
		}
		if pid <= 0 {
			return bad()
		}
		family := binary.LittleEndian.Uint32(socket[40:])
		flags := pcb[44]
		switch family {
		case 2:
			if flags&1 == 0 {
				return bad()
			}
			result = append(result, ProcessTCPListener{Address: net.IP(pcb[76:80]).String(), Port: localPort, PID: pid})
		case 30:
			if flags&2 == 0 {
				return bad()
			}
			result = append(result, ProcessTCPListener{Address: net.IP(pcb[64:80]).String(), Port: localPort, PID: pid})
		default:
			return bad()
		}
	}
	return bad() // a nonempty snapshot must have its generation trailer
}
