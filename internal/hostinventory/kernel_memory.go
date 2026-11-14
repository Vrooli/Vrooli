package hostinventory

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

// KernelMemoryReport contains allocation metadata, never kernel addresses or
// arbitrary file contents. Only a bounded set of the largest caches is returned.
type KernelMemoryReport struct {
	Source         string      `json:"source"`
	AllocatedBytes uint64      `json:"allocated_bytes"`
	CacheCount     int         `json:"cache_count"`
	Caches         []SlabCache `json:"caches"`
}

type SlabCache struct {
	Name           string `json:"name"`
	ActiveObjects  uint64 `json:"active_objects"`
	TotalObjects   uint64 `json:"total_objects"`
	ObjectBytes    uint64 `json:"object_bytes"`
	AllocatedBytes uint64 `json:"allocated_bytes"`
	Reclaimable    *bool  `json:"reclaimable,omitempty"`
}

// ParseLinuxSlabinfo interprets the kernel's v2.1 slab accounting format. The
// caller supplies the host page size; allocated bytes include full slab pages.
func ParseLinuxSlabinfo(reader io.Reader, pageSize uint64) (KernelMemoryReport, error) {
	report := KernelMemoryReport{Source: "linux:/proc/slabinfo"}
	if pageSize == 0 {
		return report, fmt.Errorf("host page size is required")
	}
	limited := &io.LimitedReader{R: reader, N: (8 << 20) + 1}
	scanner := bufio.NewScanner(limited)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "slabinfo - version: 2.1" {
		return report, fmt.Errorf("unsupported slabinfo format")
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 16 || fields[6] != ":" || fields[7] != "tunables" || fields[11] != ":" || fields[12] != "slabdata" {
			return report, fmt.Errorf("invalid slabinfo counters")
		}
		name := fields[0]
		if len(name) > 128 || strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			return report, fmt.Errorf("invalid slab cache name")
		}
		values := make([]uint64, 0, 6)
		for _, field := range append(fields[1:6:6], fields[14]) {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return report, fmt.Errorf("invalid slab counter: %w", err)
			}
			values = append(values, value)
		}
		pages, count := values[4], values[5]
		if pages > math.MaxUint64/pageSize || (pages*pageSize != 0 && count > math.MaxUint64/(pages*pageSize)) {
			return report, fmt.Errorf("slab allocation overflow")
		}
		allocated := pages * pageSize * count
		if allocated > math.MaxUint64-report.AllocatedBytes {
			return report, fmt.Errorf("slab total overflow")
		}
		report.AllocatedBytes += allocated
		report.Caches = append(report.Caches, SlabCache{Name: name, ActiveObjects: values[0], TotalObjects: values[1], ObjectBytes: values[2], AllocatedBytes: allocated})
	}
	if err := scanner.Err(); err != nil {
		return report, err
	}
	if limited.N == 0 {
		return report, fmt.Errorf("slabinfo exceeds the diagnostic size limit")
	}
	if len(report.Caches) == 0 {
		return report, fmt.Errorf("slab cache counters are unavailable")
	}
	report.CacheCount = len(report.Caches)
	sort.Slice(report.Caches, func(i, j int) bool {
		if report.Caches[i].AllocatedBytes == report.Caches[j].AllocatedBytes {
			return report.Caches[i].Name < report.Caches[j].Name
		}
		return report.Caches[i].AllocatedBytes > report.Caches[j].AllocatedBytes
	})
	if len(report.Caches) > 32 {
		report.Caches = report.Caches[:32]
	}
	return report, nil
}
