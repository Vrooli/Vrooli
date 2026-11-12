package platform

import (
	"encoding/binary"
	"fmt"
	"net"
)

func parseWindowsTCPOwnerTable(data []byte, ipv6 bool, pid uint32) ([]ProcessTCPListener, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("platform: truncated TCP owner count")
	}
	rowSize := 24
	if ipv6 {
		rowSize = 56
	}
	count := uint64(binary.LittleEndian.Uint32(data))
	if count > uint64((len(data)-4)/rowSize) {
		return nil, fmt.Errorf("platform: truncated TCP owner rows")
	}
	var result []ProcessTCPListener
	for i := 0; i < int(count); i++ {
		row := data[4+i*rowSize : 4+(i+1)*rowSize]
		stateOff, pidOff, portOff := 0, 20, 8
		if ipv6 {
			stateOff, pidOff, portOff = 48, 52, 20
		}
		if binary.LittleEndian.Uint32(row[stateOff:]) != 2 {
			return nil, fmt.Errorf("platform: unexpected non-listener in TCP listener table")
		}
		if pid != 0 && binary.LittleEndian.Uint32(row[pidOff:]) != pid {
			continue
		}
		var address string
		if ipv6 {
			address = net.IP(row[:16]).String()
			scope := binary.LittleEndian.Uint32(row[16:])
			if scope != 0 {
				address = fmt.Sprintf("%s%%%d", address, scope)
			}
		} else {
			address = net.IP(row[4:8]).String()
		}
		result = append(result, ProcessTCPListener{Address: address, Port: binary.BigEndian.Uint16(row[portOff:]), PID: int(binary.LittleEndian.Uint32(row[pidOff:]))})
	}
	return result, nil
}
