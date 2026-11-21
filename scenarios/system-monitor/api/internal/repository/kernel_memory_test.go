package repository

import (
	"testing"
	"time"
)

func TestKernelMemoryProjectionPreservesHistoryAbsenceAndFailure(t *testing.T) {
	when := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, want string
		values     map[string]interface{}
	}{
		{"historical absence", "not_yet_sampled", nil},
		{"measured zero", "measured", map[string]interface{}{"kernel_slab_bytes_status": "measured", "kernel_slab_bytes": float64(0)}},
		{"failed", "failed", map[string]interface{}{"kernel_slab_bytes_status": "failed", "kernel_slab_bytes_reason": "snapshot failed"}},
		{"unsupported", "unsupported", map[string]interface{}{"kernel_slab_bytes_status": "unsupported", "kernel_slab_bytes_reason": "not Linux"}},
		{"missing numeric value", "failed", map[string]interface{}{"kernel_slab_bytes_status": "measured"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := KernelMemoryFromValues(tc.values, "cycle", when).SlabBytes
			if got.Status != tc.want || got.Units != "bytes" || got.CycleID != "cycle" || !got.ObservedAt.Equal(when) {
				t.Fatalf("state = %+v", got)
			}
			if tc.want != "measured" && got.Reason == "" {
				t.Fatal("unavailable state lost its reason")
			}
		})
	}
}

func TestKernelMemoryProjectionKeepsSourceObservationTimeAfterJSONRetention(t *testing.T) {
	source := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	for _, raw := range []interface{}{source, source.Format(time.RFC3339Nano)} {
		got := KernelMemoryFromValues(map[string]interface{}{"kernel_observed_at": raw}, "cycle", source.Add(time.Minute))
		if !got.SlabBytes.ObservedAt.Equal(source) {
			t.Fatalf("source time was replaced: %+v", got.SlabBytes)
		}
	}
}
