package scenariocli

import (
	"strings"
	"testing"
)

// TestStartResolvesInstanceFlagAndSuffix proves the `--instance` flag and the
// `name@variant` suffix both resolve through the shared ParseInstanceKey parser
// to the same canonical slug, and that a flag/suffix disagreement is rejected.
func TestStartResolvesInstanceFlagAndSuffix(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{name: "bare live", args: []string{"alpha"}, want: []string{"alpha"}},
		{name: "instance flag", args: []string{"alpha", "--instance", "shadow"}, want: []string{"alpha@shadow"}},
		{name: "suffix", args: []string{"alpha@shadow"}, want: []string{"alpha@shadow"}},
		{name: "flag equals suffix", args: []string{"alpha@shadow", "--instance", "shadow"}, want: []string{"alpha@shadow"}},
		{name: "explicit live flag normalizes to bare", args: []string{"alpha", "--instance", "live"}, want: []string{"alpha"}},
		{name: "multiple names share flag", args: []string{"alpha", "beta", "--instance", "shadow"}, want: []string{"alpha@shadow", "beta@shadow"}},
		{name: "flag disagrees with suffix", args: []string{"alpha@shadow", "--instance", "live"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := ParseStartRequest(false, tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for args %v", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStartRequest(%v): %v", tc.args, err)
			}
			if strings.Join(req.Names, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("names = %v, want %v", req.Names, tc.want)
			}
		})
	}
}

func TestRestartResolvesInstance(t *testing.T) {
	req, err := ParseRestartRequest(false, []string{"alpha", "--instance", "shadow"})
	if err != nil {
		t.Fatalf("ParseRestartRequest: %v", err)
	}
	if req.Name != "alpha@shadow" {
		t.Fatalf("name = %q, want alpha@shadow", req.Name)
	}
}

func TestStopResolvesInstance(t *testing.T) {
	req, err := ParseStopRequest(false, []string{"alpha@shadow"})
	if err != nil {
		t.Fatalf("ParseStopRequest suffix: %v", err)
	}
	if req.Name != "alpha@shadow" {
		t.Fatalf("name = %q, want alpha@shadow", req.Name)
	}
	flagReq, err := ParseStopRequest(false, []string{"alpha", "--instance", "shadow"})
	if err != nil {
		t.Fatalf("ParseStopRequest flag: %v", err)
	}
	if flagReq.Name != "alpha@shadow" {
		t.Fatalf("flag name = %q, want alpha@shadow", flagReq.Name)
	}
	if _, err := ParseStopRequest(false, []string{"alpha@shadow", "--instance", "live"}); err == nil {
		t.Fatal("expected disagreement error")
	}
}

func TestStatusResolvesInstance(t *testing.T) {
	req, err := ParseStatusRequest(false, []string{"alpha@shadow"})
	if err != nil {
		t.Fatalf("ParseStatusRequest: %v", err)
	}
	if req.Name != "alpha@shadow" {
		t.Fatalf("name = %q, want alpha@shadow", req.Name)
	}
	// status with no name lists every scenario — instance resolution must stay opt-in.
	listReq, err := ParseStatusRequest(false, nil)
	if err != nil {
		t.Fatalf("ParseStatusRequest list: %v", err)
	}
	if listReq.Name != "" {
		t.Fatalf("list name = %q, want empty", listReq.Name)
	}
	if _, err := ParseStatusRequest(false, []string{"--instance", "shadow"}); err == nil {
		t.Fatal("expected error for --instance without a scenario name")
	}
}

func TestPortResolvesInstance(t *testing.T) {
	req, err := ParsePortRequest(false, []string{"alpha@shadow", "API_PORT"})
	if err != nil {
		t.Fatalf("ParsePortRequest suffix: %v", err)
	}
	if req.ScenarioName != "alpha@shadow" || req.PortName != "API_PORT" {
		t.Fatalf("port req = %+v, want alpha@shadow/API_PORT", req)
	}
	flagReq, err := ParsePortRequest(false, []string{"alpha", "API_PORT", "--instance", "shadow"})
	if err != nil {
		t.Fatalf("ParsePortRequest flag: %v", err)
	}
	if flagReq.ScenarioName != "alpha@shadow" {
		t.Fatalf("flag scenario = %q, want alpha@shadow", flagReq.ScenarioName)
	}
}
