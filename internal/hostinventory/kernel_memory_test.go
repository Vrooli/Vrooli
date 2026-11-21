package hostinventory

import (
	"fmt"
	"strings"
	"testing"
)

func TestKernelMemoryCountsAllCachesAndBoundsLargestProjection(t *testing.T) {
	var input strings.Builder
	input.WriteString("slabinfo - version: 2.1\n# cache counters\n")
	var total uint64
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&input, ":%07d 1 2 64 64 1 : tunables 0 0 0 : slabdata %d %d 0\n", i, i, i)
		total += uint64(i) * 4096
	}
	report, err := ParseLinuxSlabinfo(strings.NewReader(input.String()), 4096)
	if err != nil || report.CacheCount != 40 || len(report.Caches) != 32 || report.AllocatedBytes != total {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if report.Caches[0].Name != ":0000040" || report.Caches[31].AllocatedBytes != 9*4096 {
		t.Fatalf("largest-cache projection is incorrect: %+v", report.Caches)
	}
}

func TestKernelMemoryRejectsMalformedOrOverflowingEvidence(t *testing.T) {
	for _, body := range []string{
		"slabinfo - version: 1.0\n",
		"slabinfo - version: 2.1\n",
		"slabinfo - version: 2.1\nbad row\n",
		"slabinfo - version: 2.1\n../secret 1 2 64 64 1 : tunables 0 0 0 : slabdata 1 2 0\n",
		"slabinfo - version: 2.1\ncache 1 2 64 64 18446744073709551615 : tunables 0 0 0 : slabdata 1 2 0\n",
	} {
		if _, err := ParseLinuxSlabinfo(strings.NewReader(body), 4096); err == nil {
			t.Fatalf("accepted invalid evidence: %q", body)
		}
	}
}
