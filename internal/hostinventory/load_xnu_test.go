package hostinventory

import (
	"encoding/binary"
	"testing"
)

func TestDarwinLoadUsesFixedPointABI(t *testing.T) {
	raw := make([]byte, 24)
	for i, value := range []uint32{1250, 2500, 3750} {
		binary.LittleEndian.PutUint32(raw[i*4:], value)
	}
	// Padding is not part of the third load value or the scale.
	binary.LittleEndian.PutUint32(raw[12:], 0xdeadbeef)
	binary.LittleEndian.PutUint64(raw[16:], 1000)
	got, err := decodeDarwinLoad(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Load1 != 1.25 || got.Load5 != 2.5 || got.Load15 != 3.75 {
		t.Fatalf("load: %+v", got)
	}
	if got.RunningProcs != 0 {
		t.Fatal("XNU load does not contain runnable-process count")
	}
}

func TestDarwinLoadRejectsMalformedABI(t *testing.T) {
	for _, size := range []int{0, 12, 16, 20, 23, 25, 32} {
		if _, err := decodeDarwinLoad(make([]byte, size)); err == nil {
			t.Errorf("accepted %d bytes", size)
		}
	}
	for _, scale := range []uint64{0, 1 << 63, ^uint64(0)} {
		raw := make([]byte, 24)
		binary.LittleEndian.PutUint64(raw[16:], scale)
		if _, err := decodeDarwinLoad(raw); err == nil {
			t.Errorf("accepted scale %d", scale)
		}
	}
}

func TestDarwinLoadProcessFieldAbsenceIsExplicit(t *testing.T) {
	snap := Snapshot{ProbeStatuses: map[string]string{"load": "ok"}}
	markDarwinLoadProcessFields(&snap)
	for _, field := range []string{"running_procs", "total_procs", "last_pid"} {
		if snap.ProbeStatuses["load."+field] != "unsupported" {
			t.Fatalf("%s absence was lost: %+v", field, snap.ProbeStatuses)
		}
	}
	if snap.ProbeStatuses["load"] != "ok" {
		t.Fatal("load measurement status changed")
	}
}
