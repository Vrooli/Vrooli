package platform

import (
	"encoding/binary"
	"testing"
)

// These wire offsets come from packed-4 XNU headers, independently of Go
// layout. The kernel emits six ordered 8-byte-aligned records per PCB.
func darwinPCBFixture(family uint32, pid uint32, port uint16) []byte {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data, 24)
	binary.LittleEndian.PutUint32(data[4:], 1)
	binary.LittleEndian.PutUint64(data[8:], 7)
	binary.LittleEndian.PutUint64(data[16:], 11)
	pcb := make([]byte, 104)
	binary.LittleEndian.PutUint32(pcb, 104)
	binary.LittleEndian.PutUint32(pcb[4:], 0x10)
	binary.BigEndian.PutUint16(pcb[18:], port)
	binary.LittleEndian.PutUint64(pcb[28:], 7)
	pcb[44] = 1
	copy(pcb[76:], []byte{127, 0, 0, 1})
	if family == 30 {
		pcb[44] = 2
		for i := 64; i < 80; i++ {
			pcb[i] = 0
		}
		pcb[79] = 1
	}
	data = append(data, pcb...)
	socket := make([]byte, 104)
	binary.LittleEndian.PutUint32(socket, 104)
	binary.LittleEndian.PutUint32(socket[4:], 1)
	binary.LittleEndian.PutUint16(socket[16:], 1)
	binary.LittleEndian.PutUint32(socket[36:], 6)
	binary.LittleEndian.PutUint32(socket[40:], family)
	binary.LittleEndian.PutUint32(socket[68:], pid)
	data = append(data, socket...)
	for _, kind := range []uint32{2, 4, 8} {
		record := make([]byte, 16)
		binary.LittleEndian.PutUint32(record, 16)
		binary.LittleEndian.PutUint32(record[4:], kind)
		data = append(data, record...)
	}
	tcp := make([]byte, 40)
	binary.LittleEndian.PutUint32(tcp, 40)
	binary.LittleEndian.PutUint32(tcp[4:], 0x20)
	binary.LittleEndian.PutUint32(tcp[36:], 1)
	data = append(data, tcp...)
	return append(data, data[:24]...)
}

func TestDarwinPCBGlobalListenerIdentity(t *testing.T) {
	for _, family := range []uint32{2, 30} {
		data := darwinPCBFixture(family, 322, 6742)
		got, err := parseDarwinTCPPCBList(data, 6742)
		if err != nil || len(got) != 1 || got[0].PID != 322 || got[0].Port != 6742 {
			t.Fatalf("global snapshot: %v %v", got, err)
		}
		expected := "127.0.0.1"
		if family == 30 {
			expected = "::1"
		}
		if got[0].Address != expected {
			t.Fatalf("address lost: %v", got)
		}
		got, err = parseDarwinTCPPCBList(data, 6743)
		if err != nil || len(got) != 0 {
			t.Fatalf("different port: %v %v", got, err)
		}
	}
}

func TestDarwinPCBGlobalRejectsIncompleteChangedAndUnknownRecords(t *testing.T) {
	cases := map[string]func([]byte) []byte{
		"truncated":          func(b []byte) []byte { return b[:len(b)-1] },
		"missing trailer":    func(b []byte) []byte { return b[:len(b)-24] },
		"missing group":      func(b []byte) []byte { return append(b[:24], b[len(b)-24:]...) },
		"changed generation": func(b []byte) []byte { b[len(b)-16]++; return b },
		"unknown record":     func(b []byte) []byte { b[28] = 0x80; return b },
		"unknown protocol":   func(b []byte) []byte { b[24+104+36] = 17; return b },
		"unknown owner":      func(b []byte) []byte { binary.LittleEndian.PutUint32(b[24+104+68:], 0); return b },
		"overflow record":    func(b []byte) []byte { binary.LittleEndian.PutUint32(b[24:], 0xffffffff); return b },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDarwinTCPPCBList(mutate(darwinPCBFixture(2, 322, 6742)), 6742); err == nil {
				t.Fatal("incomplete observation accepted")
			}
		})
	}
}
