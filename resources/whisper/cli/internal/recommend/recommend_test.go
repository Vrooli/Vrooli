package recommend

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestPick_Table(t *testing.T) {
	gb := uint64(1 << 30)
	cases := []struct {
		name      string
		caps      hostinventory.Snapshot
		budget    int
		wantModel Model
	}{
		{
			name: "RTX 4090 24GB / 64GB / 32c",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4090", VRAMBytes: 24 * gb}},
			},
			budget: 50, wantModel: ModelLargeV3,
		},
		{
			name: "RTX 3070 8GB / 32GB / 16c",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 16}, Memory: hostinventory.Memory{TotalBytes: 32 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 3070", VRAMBytes: 8 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
		{
			name: "GTX 1650 4GB / 16GB / 8c",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 8}, Memory: hostinventory.Memory{TotalBytes: 16 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA GTX 1650", VRAMBytes: 4 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
		{
			name:      "No GPU, 32GB / 16c → medium",
			caps:      hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 16}, Memory: hostinventory.Memory{TotalBytes: 32 * gb}},
			budget:    50,
			wantModel: ModelMedium,
		},
		{
			name:      "No GPU, 16GB / 8c → small",
			caps:      hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 8}, Memory: hostinventory.Memory{TotalBytes: 16 * gb}},
			budget:    50,
			wantModel: ModelSmall,
		},
		{
			name:      "No GPU, 8GB / 4c → base",
			caps:      hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 4}, Memory: hostinventory.Memory{TotalBytes: 8 * gb}},
			budget:    50,
			wantModel: ModelBase,
		},
		{
			name:      "No GPU, 2GB / 2c → tiny",
			caps:      hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 2}, Memory: hostinventory.Memory{TotalBytes: 2 * gb}},
			budget:    50,
			wantModel: ModelTiny,
		},
		{
			name: "RTX 4090, budget 10% → small (proves budget knob)",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4090", VRAMBytes: 24 * gb}},
			},
			budget: 10, wantModel: ModelSmall,
		},
		{
			name: "Apple Silicon M2, 16GB unified",
			caps: hostinventory.Snapshot{
				OS: "darwin", Arch: "arm64", CPU: hostinventory.CPU{Cores: 8}, Memory: hostinventory.Memory{TotalBytes: 16 * gb},
				GPUs: []hostinventory.GPU{{Name: "Apple GPU (unified)", VRAMBytes: 8 * gb}},
			},
			budget: 50, wantModel: ModelSmall,
		},
		{
			// Contention-aware (plan §3): a 16GB card with 13GB already held by
			// co-tenant model servers has only 3GB free → budget 50% = 1.5GB →
			// base. Sizing off TOTAL would have wrongly picked medium/large.
			name: "RTX 16GB, 13GB held by co-tenants → base (contention-aware)",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4070 Ti SUPER", VRAMBytes: 16 * gb, VRAMUsedBytes: 13 * gb}},
			},
			budget: 50, wantModel: ModelBase,
		},
		{
			// Same card, idle (nothing resident) → 16GB free → budget 50% = 8GB
			// → medium. Proves the pick rises with available headroom.
			name: "RTX 16GB, idle → medium",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4070 Ti SUPER", VRAMBytes: 16 * gb, VRAMUsedBytes: 0}},
			},
			budget: 50, wantModel: ModelMedium,
		},
		{
			// Card fully consumed (used >= total) → 0 free → falls through to the
			// CPU/RAM path. 64GB RAM / 32c → medium (CPU tier cap).
			name: "GPU fully consumed → CPU path",
			caps: hostinventory.Snapshot{
				CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 * gb},
				GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4070 Ti SUPER", VRAMBytes: 16 * gb, VRAMUsedBytes: 16 * gb}},
			},
			budget: 50, wantModel: ModelMedium,
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
	if _, _, err := Pick(hostinventory.Snapshot{}, 0); err == nil {
		t.Error("0: want error")
	}
	if _, _, err := Pick(hostinventory.Snapshot{}, 101); err == nil {
		t.Error("101: want error")
	}
}
