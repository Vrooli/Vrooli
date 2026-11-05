//go:build linux

package collectors

import (
	"context"
	"testing"
)

func TestSocketInodeFromLink(t *testing.T) {
	cases := []struct {
		link  string
		want  uint64
		valid bool
	}{
		{"socket:[12345]", 12345, true},
		{"socket:[0]", 0, true},
		{"pipe:[12345]", 0, false},
		{"/dev/null", 0, false},
		{"socket:[abc]", 0, false},
		{"socket:[12345", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := socketInodeFromLink(tc.link)
		if ok != tc.valid || (tc.valid && got != tc.want) {
			t.Errorf("socketInodeFromLink(%q) = (%d, %v), want (%d, %v)", tc.link, got, ok, tc.want, tc.valid)
		}
	}
}

// Attribution must never claim more coverage than it achieved: /proc/<pid>/fd is
// unreadable for other users' processes, so Attributed is normally below Total.
func TestAttributeSocketOwnersReportsCoverageNotPopulation(t *testing.T) {
	result := attributeSocketOwners(context.Background(), 0, 5)
	if !result.Supported {
		t.Skipf("attribution unsupported in this environment: %s", result.Reason)
	}
	if result.Attributed < 0 {
		t.Fatalf("negative attributed count %d", result.Attributed)
	}
	sum := 0
	for _, owner := range result.Owners {
		if owner.Count <= 0 {
			t.Errorf("owner %+v has a non-positive count", owner)
		}
		sum += owner.Count
	}
	if sum > result.Attributed {
		t.Fatalf("top owners sum to %d, exceeding attributed total %d", sum, result.Attributed)
	}
}

func TestReadForkRateOnLinux(t *testing.T) {
	reading := readForkRate()
	if !reading.supported {
		t.Fatalf("fork counter unsupported on linux: %s", reading.reason)
	}
	if reading.total == 0 {
		t.Fatal("fork counter is zero; /proc/stat parsing is wrong")
	}
	if reading.provenance == "" {
		t.Fatal("reading has no provenance")
	}
}
