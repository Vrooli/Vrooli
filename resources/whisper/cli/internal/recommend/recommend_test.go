package recommend

import (
	"testing"

	"resource-whisper/cli/internal/hwprobe"
)

func TestPick_Table(t *testing.T) {
	gb := uint64(1 << 30)
	cases := []struct {
		name      string
		caps      hwprobe.HostCapabilities
		budget    int
		wantModel Model
	}{
		{
			name: "RTX 4090 24GB / 64GB / 32c",
			caps: hwprobe.HostCapabilities{
				CPUCores: 32, TotalRAMBytes: 64 * gb,
				GPUs: []hwprobe.GPU{{Name: "NVIDIA RTX 4090", VRAMBytes: 24 * gb}},
			},
			budget: 50, wantModel: ModelLargeV3,
		},
		{
			name: "RTX 3070 8GB / 32GB / 16c",
			caps: hwprobe.HostCapabilities{
				CPUCores: 16, TotalRAMBytes: 32 * gb,
				GPUs: []hwprobe.GPU{{Name: "NVIDIA RTX 3070", VRAMBytes: 8 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
		{
			name: "GTX 1650 4GB / 16GB / 8c",
			caps: hwprobe.HostCapabilities{
				CPUCores: 8, TotalRAMBytes: 16 * gb,
				GPUs: []hwprobe.GPU{{Name: "NVIDIA GTX 1650", VRAMBytes: 4 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
		{
			name:      "No GPU, 32GB / 16c → medium",
			caps:      hwprobe.HostCapabilities{CPUCores: 16, TotalRAMBytes: 32 * gb},
			budget:    50,
			wantModel: ModelMedium,
		},
		{
			name:      "No GPU, 16GB / 8c → small",
			caps:      hwprobe.HostCapabilities{CPUCores: 8, TotalRAMBytes: 16 * gb},
			budget:    50,
			wantModel: ModelSmall,
		},
		{
			name:      "No GPU, 8GB / 4c → base",
			caps:      hwprobe.HostCapabilities{CPUCores: 4, TotalRAMBytes: 8 * gb},
			budget:    50,
			wantModel: ModelBase,
		},
		{
			name:      "No GPU, 2GB / 2c → tiny",
			caps:      hwprobe.HostCapabilities{CPUCores: 2, TotalRAMBytes: 2 * gb},
			budget:    50,
			wantModel: ModelTiny,
		},
		{
			name: "RTX 4090, budget 10% → small (proves budget knob)",
			caps: hwprobe.HostCapabilities{
				CPUCores: 32, TotalRAMBytes: 64 * gb,
				GPUs: []hwprobe.GPU{{Name: "NVIDIA RTX 4090", VRAMBytes: 24 * gb}},
			},
			budget: 10, wantModel: ModelSmall,
		},
		{
			name: "Apple Silicon M2, 16GB unified",
			caps: hwprobe.HostCapabilities{
				OS: "darwin", Arch: "arm64", CPUCores: 8, TotalRAMBytes: 16 * gb,
				GPUs: []hwprobe.GPU{{Name: "Apple GPU (unified)", VRAMBytes: 8 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason, err := Pick(tc.caps, tc.budget)
			if err != nil {
				t.Fatal(err)
			}
			if reason == "" {
				t.Errorf("empty reason")
			}
			if got != tc.wantModel {
				t.Errorf("got=%s want=%s reason=%s", got, tc.wantModel, reason)
			}
		})
	}
}

func TestPick_BadBudget(t *testing.T) {
	if _, _, err := Pick(hwprobe.HostCapabilities{}, 0); err == nil {
		t.Error("0: want error")
	}
	if _, _, err := Pick(hwprobe.HostCapabilities{}, 101); err == nil {
		t.Error("101: want error")
	}
}
