package platform

import (
	"encoding/binary"
	"testing"
)

func TestWindowsTCPOwnerTablesPreserveForeignWildcardAndIPv6(t *testing.T) {
	for _, ipv6 := range []bool{false, true} {
		rowSize, stateOff, pidOff, portOff := 24, 0, 20, 8
		if ipv6 {
			rowSize, stateOff, pidOff, portOff = 56, 48, 52, 20
		}
		data := make([]byte, 4+2*rowSize)
		binary.LittleEndian.PutUint32(data, 2)
		for i, pid := range []uint32{10, 20} {
			row := data[4+i*rowSize:]
			binary.LittleEndian.PutUint32(row[stateOff:], 2)
			binary.LittleEndian.PutUint32(row[pidOff:], pid)
			binary.BigEndian.PutUint16(row[portOff:], 6742)
			if ipv6 {
				row[15] = 1
			} // ::1
		}
		all, err := parseWindowsTCPOwnerTable(data, ipv6, 0)
		if err != nil || len(all) != 2 || all[0].Port != 6742 || all[1].PID != 20 {
			t.Fatalf("all listeners: %v %v", all, err)
		}
		if !ipv6 && all[0].Address != "0.0.0.0" {
			t.Fatalf("wildcard was rewritten: %v", all)
		}
		if ipv6 && all[0].Address != "::1" {
			t.Fatalf("IPv6 lost: %v", all)
		}
		owned, err := parseWindowsTCPOwnerTable(data, ipv6, 10)
		if err != nil || len(owned) != 1 || owned[0].PID != 10 {
			t.Fatalf("owned listeners: %v %v", owned, err)
		}
	}
}

func TestWindowsTCPOwnerTablesRejectPartialOrUnknownObservations(t *testing.T) {
	for _, data := range [][]byte{nil, {1, 0, 0, 0}, make([]byte, 27)} {
		if len(data) == 27 {
			binary.LittleEndian.PutUint32(data, 1)
		}
		if _, err := parseWindowsTCPOwnerTable(data, false, 0); err == nil {
			t.Fatal("accepted truncated table")
		}
	}
	data := make([]byte, 28)
	binary.LittleEndian.PutUint32(data, 1)
	if _, err := parseWindowsTCPOwnerTable(data, false, 0); err == nil {
		t.Fatal("unknown state manufactured absence")
	}
}
