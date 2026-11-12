package capacitycli

import (
	"bytes"
	"strings"
	"testing"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
)

func TestParseDiskReservationsRequest(t *testing.T) {
	got, err := ParseDiskReservationsRequest([]string{"--active", "--owner", " foreign:job ", "--filesystem", "dev:66306", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.Owner != "foreign:job" || got.Filesystem != "dev:66306" {
		t.Fatalf("parsed = %+v", got)
	}
}

func TestParseDiskListRequestCurrentObservation(t *testing.T) {
	got, err := ParseDiskListRequest([]string{"--observe", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Active || !got.Observe {
		t.Fatalf("disk-list flags = %+v", got)
	}
}

func TestParseDiskSampleDockerIdentityFlags(t *testing.T) {
	got, err := ParseDiskSampleRequest([]string{"--lease-file", "/tmp/lease.json", "--path", "/var/lib/docker/volumes/x/_data", "--bytes", "42", "--docker-container", "m5-openemr", "--docker-destination", "/var/lib/mysql", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if got.LeaseFile != "/tmp/lease.json" || got.DockerContainer != "m5-openemr" || got.DockerDestination != "/var/lib/mysql" || got.DockerUpperDir || got.Bytes != 42 {
		t.Fatalf("parsed sample=%+v", got)
	}
	upper, err := ParseDiskSampleRequest([]string{"--lease-file", "/tmp/lease.json", "--path", "/var/lib/docker/overlay2/x/diff", "--bytes", "7", "--docker-container", "m5-openemr", "--docker-upper-dir", "--json"})
	if err != nil || !upper.DockerUpperDir {
		t.Fatalf("upper-dir sample=%+v, err=%v", upper, err)
	}
	root, err := ParseDiskSampleRequest([]string{"--lease-file", "/tmp/lease.json", "--path", "/var/lib/docker", "--bytes", "0", "--docker-root-dir", "--json"})
	if err != nil || !root.DockerRootDir {
		t.Fatalf("Docker root sample=%+v, err=%v", root, err)
	}
}

func TestRenderClaimSeparatesRecordedAmountFromAdvisoryVerdict(t *testing.T) {
	var output bytes.Buffer
	err := RenderClaim(&output, "", capacityapp.ClaimOutput{
		Verdict: engine.Verdict{Kind: engine.VerdictDeny},
		Claim:   capacityapp.ClaimView{ClaimID: "claim-1", Status: engine.StatusGranted, AmountBytes: 4 << 30},
		Enforce: engine.EnforceAdvisory,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"recorded: granted", "4.00 GiB", "verdict: deny", "recommended: 0 B"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("claim output lacks %q: %s", want, &output)
		}
	}
}

func TestParseFootprintRequest(t *testing.T) {
	request, err := ParseFootprintRequest([]string{"list", "--resource", "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	if request.Action != "list" || request.Resource != "ollama" {
		t.Fatalf("request = %+v", request)
	}
	if _, err := ParseFootprintRequest([]string{"erase"}); err == nil {
		t.Fatal("invalid footprint action was accepted")
	}
}

func TestParseFitRequestSimulation(t *testing.T) {
	req, err := ParseFitRequest([]string{"--simulate-host", "vram=8GiB,backends=cuda+cpu,compute=8.9"})
	if err != nil {
		t.Fatal(err)
	}
	if req.SimulatedVRAM == nil || *req.SimulatedVRAM != 8*1024*1024*1024 {
		t.Fatalf("simulated vram = %v", req.SimulatedVRAM)
	}
	if len(req.SimulatedBackends) != 2 || req.SimulatedCompute != "8.9" {
		t.Fatalf("simulation = %#v", req)
	}
}

func TestParsePolicyRequest(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		wantAction     string
		wantKey, value string
		wantErr        bool
	}{
		{name: "get all", args: []string{"get"}, wantAction: "get"},
		{name: "get one key", args: []string{"get", "idle_yield_floor"}, wantAction: "get", wantKey: "idle_yield_floor"},
		{name: "set key value", args: []string{"set", "enforce", "advisory"}, wantAction: "set", wantKey: "enforce", value: "advisory"},
		{name: "set multi-token value", args: []string{"set", "auto_stop_allowlist", "a,", "b"}, wantAction: "set", wantKey: "auto_stop_allowlist", value: "a, b"},
		{name: "set missing value", args: []string{"set", "enforce"}, wantErr: true},
		{name: "bad action", args: []string{"frobnicate"}, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParsePolicyRequest(c.args)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Action != c.wantAction || got.Key != c.wantKey || got.Value != c.value {
				t.Errorf("got {action=%q key=%q value=%q}, want {action=%q key=%q value=%q}",
					got.Action, got.Key, got.Value, c.wantAction, c.wantKey, c.value)
			}
		})
	}
}

func TestParseClaimRequestYieldWhenIdle(t *testing.T) {
	on, err := ParseClaimRequest([]string{"--owner-id", "whisper", "--preferred", "8GiB", "--priority", "interactive", "--yield-when-idle"})
	if err != nil {
		t.Fatalf("parse with flag: %v", err)
	}
	if !on.YieldWhenIdle {
		t.Error("--yield-when-idle should set YieldWhenIdle=true")
	}
	off, err := ParseClaimRequest([]string{"--owner-id", "whisper", "--preferred", "8GiB"})
	if err != nil {
		t.Fatalf("parse without flag: %v", err)
	}
	if off.YieldWhenIdle {
		t.Error("YieldWhenIdle should default false")
	}
}

func TestParseClaimRequestNoReclaimPreservesPriority(t *testing.T) {
	r, err := ParseClaimRequest([]string{"--owner-id", "scene:segment", "--preferred", "2GiB", "--priority", "interactive", "--no-reclaim"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.NoReclaim || r.PriorityTier != "interactive" {
		t.Fatalf("independent admission policy lost: %+v", r)
	}
	r, err = ParseClaimRequest([]string{"--owner-id", "legacy", "--preferred", "2GiB"})
	if err != nil || r.NoReclaim {
		t.Fatalf("legacy reclaim default changed: %+v, %v", r, err)
	}
}

func TestCPUClaimRenderingUsesMillicoresInsteadOfBytes(t *testing.T) {
	for _, amount := range []int64{4, 4000} {
		c := capacityapp.ClaimView{ClaimID: "cpu", ResourceKind: engine.ResourceKindCPU, AmountBytes: amount, ObservedBytes: amount, ObservedPeakBytes: amount}
		want := "4000 mCPU (4.000 cores)"
		if amount == 4 {
			want = "4 mCPU (0.004 cores)"
		}
		renders := map[string]func(*bytes.Buffer) error{
			"admission": func(b *bytes.Buffer) error {
				return RenderClaim(b, "", capacityapp.ClaimOutput{Claim: c, Verdict: engine.Verdict{GrantedBytes: amount}})
			},
			"view": func(b *bytes.Buffer) error { return RenderClaimView(b, "", c) },
			"list": func(b *bytes.Buffer) error {
				return RenderList(b, "", capacityapp.ListOutput{Claims: []capacityapp.ClaimView{c}})
			},
		}
		for name, render := range renders {
			t.Run(name+want, func(t *testing.T) {
				var b bytes.Buffer
				if err := render(&b); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(b.String(), want) || strings.Contains(b.String(), " B") || strings.Contains(b.String(), "KiB") {
					t.Fatalf("CPU quantity must show its actual unit: %s", &b)
				}
			})
		}
	}
}
