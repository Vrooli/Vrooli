package collectors

import (
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestKernelMemoryValuesPreserveMeasuredZeroAndAvailability(t *testing.T) {
	zero, slab := uint64(0), uint64(18000)
	snapshot := hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 64000, SlabBytes: &slab, SlabReclaimableBytes: &zero}}
	for _, tc := range []struct {
		name, platform, want string
		err                  error
	}{
		{"linux", "linux", "measured", nil},
		{"darwin", "darwin", "unsupported", nil},
		{"windows", "windows", "unsupported", nil},
		{"read failure", "linux", "failed", errors.New("permission denied")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := kernelMemoryValues(snapshot, tc.err, tc.platform)
			if got["kernel_slab_bytes_status"] != tc.want {
				t.Fatalf("state=%v, want %s", got, tc.want)
			}
			if tc.want == "measured" {
				if value, present := got["kernel_reclaimable_bytes"]; !present || value != float64(0) {
					t.Errorf("measured zero = %v present=%v", value, present)
				}
				if got["kernel_unreclaimable_bytes_status"] != "unsupported" {
					t.Errorf("missing field = %v", got)
				}
			} else if _, present := got["kernel_slab_bytes"]; present {
				t.Errorf("unavailable state carries measured bytes: %v", got)
			}
		})
	}
}
