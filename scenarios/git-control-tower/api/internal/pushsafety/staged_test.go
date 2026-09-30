package pushsafety

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type stagedCommands struct {
	raw                    string
	size                   int64
	reads                  int
	changed, fail, missing bool
}

func (f *stagedCommands) Run(_ context.Context, _ string, input []byte, args ...string) ([]byte, error) {
	if f.fail {
		return nil, errors.New("unavailable")
	}
	switch args[0] {
	case "diff":
		f.reads++
		if f.changed && f.reads > 1 {
			return nil, nil
		}
		return []byte(f.raw), nil
	case "cat-file":
		if f.missing {
			return []byte(blobOID + " missing"), nil
		}
		var out string
		for _, oid := range strings.Fields(string(input)) {
			out += fmt.Sprintf("%s blob %d\n", oid, f.size)
		}
		return []byte(out), nil
	default:
		panic("unexpected command")
	}
}

func TestStagedSafety(t *testing.T) {
	raw := ":100644 100644 " + baseOID + " " + blobOID + " M\x00weird\tname\n.bin\x00"
	for _, tc := range []struct {
		name                   string
		size                   int64
		changed, fail, missing bool
		complete, blocked      bool
		files                  int
	}{
		{"oversize", GitHubLimitBytes + 1, false, false, false, true, true, 1},
		{"exact limit", GitHubLimitBytes, false, false, false, true, false, 1},
		{"small staged pointer", 130, false, false, false, true, false, 0},
		{"index changed", GitHubLimitBytes + 1, true, false, false, false, false, 0},
		{"command failure", 0, false, true, false, false, false, 0},
		{"missing object", 0, false, false, true, false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &stagedCommands{raw: raw, size: tc.size, changed: tc.changed, fail: tc.fail, missing: tc.missing}
			r := Report{Limit: GitHubLimitBytes, Fingerprint: "unchanged"}
			InspectStaged(context.Background(), f, "unused", &r)
			if r.StagedComplete != tc.complete || len(r.StagedFiles) != tc.files || r.Fingerprint != "unchanged" {
				t.Fatalf("unexpected report %+v", r)
			}
			if tc.files > 0 && (r.StagedFiles[0].Blocked != tc.blocked || r.StagedFiles[0].Paths[0] != "weird\tname\n.bin") {
				t.Fatalf("bad file %+v", r.StagedFiles[0])
			}
		})
	}
}

func TestStagedSafetyDeletionUnknownHostAndMalformedIndex(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		complete  bool
		files     int
	}{
		{"deleted blob is not staged content", ":100644 000000 " + blobOID + " " + strings.Repeat("0", 40) + " D\x00removed\x00", true, 0},
		{"unknown host retains warning", ":000000 100644 " + strings.Repeat("0", 40) + " " + blobOID + " A\x00new\x00", true, 1},
		{"missing path is incomplete", ":100644 100644 " + baseOID + " " + blobOID + " M\x00", false, 0},
		{"unmerged index is incomplete", ":100644 100644 " + baseOID + " " + blobOID + " U\x00conflict\x00", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Report{}
			InspectStaged(context.Background(), &stagedCommands{raw: tc.raw, size: GitHubLimitBytes + 1}, "unused", &r)
			if r.StagedComplete != tc.complete || len(r.StagedFiles) != tc.files {
				t.Fatalf("unexpected %+v", r)
			}
			for _, f := range r.StagedFiles {
				if f.Blocked {
					t.Fatal("unknown host invented a blocking policy")
				}
			}
		})
	}
}
