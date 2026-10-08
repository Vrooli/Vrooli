package collectors

import (
	"os"
	"testing"
)

func TestParsePSI(t *testing.T) {
	got := parsePSI("some avg10=1.25 avg60=0.50 avg300=0.10 total=12\nfull avg10=0.25 total=4\n")
	if got["some"]["avg10"] != 1.25 || got["full"]["total"] != 4 {
		t.Fatalf("parsePSI = %#v", got)
	}
}

func TestParseVMStatFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/vmstat_linux.txt")
	if err != nil {
		t.Fatal(err)
	}
	values := parseVMStat(string(raw))
	for key, want := range map[string]uint64{"pswpin": 100, "pgmajfault": 300, "allocstall_normal": 800, "oom_kill": 9} {
		if values[key] != want {
			t.Fatalf("%s = %d, want %d", key, values[key], want)
		}
	}
}
