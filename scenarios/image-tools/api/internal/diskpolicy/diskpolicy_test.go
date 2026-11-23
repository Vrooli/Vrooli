package diskpolicy

import (
	"strings"
	"testing"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

func env(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestFloorIsTheHostMinimumUnlessAnOperatorRaisesIt(t *testing.T) {
	cases := map[string]struct {
		value string
		want  int64
		note  string
	}{
		"default":       {"", HostMinimumFloorBytes, "host minimum"},
		"raised":        {"200GiB", 200 << 30, "operator raised"},
		"below minimum": {"50GiB", HostMinimumFloorBytes, "ignored"},
		"equal":         {"120GiB", HostMinimumFloorBytes, "ignored"},
		"not a size":    {"lots", HostMinimumFloorBytes, "not a size"},
	}
	for name, tc := range cases {
		f := Resolve(env(map[string]string{EnvFloor: tc.value}))
		if f.Bytes != tc.want || !strings.Contains(strings.Join(f.Reasons, " "), tc.note) {
			t.Fatalf("%s: %+v", name, f)
		}
	}
	if Resolve(nil).Bytes != HostMinimumFloorBytes {
		t.Fatal("a nil environment reads as the default")
	}
}

// The ledger would raise a lower request to the host minimum, so the policy must
// never ask for a private number below it (that would only mislead diagnostics).
func TestRequestNeverAsksBelowTheHostMinimumForBulkWrites(t *testing.T) {
	req := Resolve(nil).Request("owner", "job", []sharedcapacity.DiskJobPath{{Path: "/x", PeakBytes: 5 << 30}})
	if req.Class != "" || req.FloorBytes != HostMinimumFloorBytes {
		t.Fatalf("bulk write: %+v", req)
	}
}

func TestSmallWritesUseTheLedgersSmallWriteClass(t *testing.T) {
	small := []sharedcapacity.DiskJobPath{{Path: "/x", PeakBytes: 32 << 20}, {Path: "/y", PeakBytes: 32 << 20}}
	req := Resolve(nil).Request("owner", "job", small)
	if req.Class != sharedcapacity.DiskJobClassMetadata {
		t.Fatalf("a 64 MiB total fits the small-write class: %+v", req)
	}
	if SmallWrite([]sharedcapacity.DiskJobPath{{PeakBytes: 64<<20 + 1}}) || SmallWrite([]sharedcapacity.DiskJobPath{{PeakBytes: 0}}) || SmallWrite(nil) {
		t.Fatal("anything above the bound, empty or unsized is an ordinary write")
	}
}

func TestDescribeReportsDiskSizeAndHeadroom(t *testing.T) {
	st, err := Describe(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.DiskTotalBytes <= 0 || st.FreeBytes <= 0 || st.FloorBytes != HostMinimumFloorBytes || st.HeadroomBytes != st.FreeBytes-st.FloorBytes || st.FloorPercentOfDisk <= 0 || len(st.FloorReasons) == 0 {
		t.Fatalf("%+v", st)
	}
}
