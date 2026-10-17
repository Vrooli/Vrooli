package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkRepairServiceSelectedRoots records the migration/doctor cost for
// selected managed entries. The current setup path never invokes this broad
// operation after a complete ledger record; the benchmark exists for the
// explicit repair budget and future threshold evidence.
func BenchmarkRepairServiceSelectedRoots(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("entries-%d", count), func(b *testing.B) {
			root := b.TempDir()
			for i := 0; i < count; i++ {
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("entry-%06d", i)), []byte("x"), 0o600); err != nil {
					b.Fatal(err)
				}
			}
			service := RepairService{ResolveRoot: func(string) (string, error) { return root, nil }}
			uid, gid := RepairIdentity()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := service.Repair(context.Background(), RepairRequest{Scope: RepairScope{RootClass: "cache"}, ExpectedUID: uid, ExpectedGID: gid, MaxEntries: uint64(count + 1)}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
