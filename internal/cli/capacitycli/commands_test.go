package capacitycli

import "testing"

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
