package componentrevisions

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	devicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	"google.golang.org/protobuf/proto"
)

func TestCandidateInputIsStrictAndBounded(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       string
	}{
		{"unknown", `{"version":"v1","surprise":true}`, "unknown field"},
		{"oversized", `{"version":"` + strings.Repeat("x", maxCandidateBytes) + `"}`, "exceeds"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir() + "/candidate.json"
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := readCandidate(path, func() proto.Message { return &devicev1.DeviceCandidate{} })
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestRevisionNumbersAreBounded(t *testing.T) {
	for _, tc := range []struct {
		raw               string
		allowZero, wantOK bool
	}{{"0", true, true}, {"0", false, false}, {"4294967295", true, true}, {"4294967296", true, false}, {"-1", true, false}, {" 3 ", false, true}} {
		_, err := parseUint32(tc.raw, "revision", tc.allowZero)
		if (err == nil) != tc.wantOK {
			t.Fatalf("parseUint32(%q,allowZero=%t) error=%v", tc.raw, tc.allowZero, err)
		}
	}
	for _, raw := range []string{"0", "101", "4294967296"} {
		if _, err := boundedPageSize(raw); err == nil {
			t.Fatalf("unexpectedly accepted page-size %s", raw)
		}
	}
	if got, err := boundedPageSize(""); err != nil || got != 50 {
		t.Fatalf("default page size=%d err=%v", got, err)
	}
}

func TestAbortedSaveGuidesExplicitHeadReview(t *testing.T) {
	err := APIError("save device candidate", "devices", connect.NewError(connect.CodeAborted, fmt.Errorf("head changed")))
	if !strings.Contains(err.Error(), "devices head <id>") || !strings.Contains(err.Error(), "--expected-head") {
		t.Fatalf("error=%v", err)
	}
	other := APIError("save device candidate", "devices", connect.NewError(connect.CodeUnavailable, fmt.Errorf("offline")))
	if !strings.Contains(other.Error(), "offline") {
		t.Fatalf("unexpected wrapped API error: %v", other)
	}
}
