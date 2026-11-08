package hostinventory

import (
	"os"
	"path/filepath"
	"strings"
)

func ReadKernelMemory() (KernelMemoryReport, error) {
	f, err := os.Open("/proc/slabinfo")
	if err != nil {
		return KernelMemoryReport{}, err
	}
	defer f.Close()
	report, err := ParseLinuxSlabinfo(f, uint64(os.Getpagesize()))
	if err != nil {
		return report, err
	}
	for i := range report.Caches {
		value, err := os.ReadFile(filepath.Join("/sys/kernel/slab", report.Caches[i].Name, "reclaim_account"))
		if err == nil && (strings.TrimSpace(string(value)) == "0" || strings.TrimSpace(string(value)) == "1") {
			reclaimable := strings.TrimSpace(string(value)) == "1"
			report.Caches[i].Reclaimable = &reclaimable
		}
	}
	return report, nil
}
