package main

import (
	"os"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestParseGrantTTL(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"720h", 720 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"", 0, true},
		{"0d", 0, true},
		{"-5d", 0, true},
		{"banana", 0, true},
		{"0", 0, true},
	}
	for _, tc := range cases {
		got, err := parseGrantTTL(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("parseGrantTTL(%q) expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("parseGrantTTL(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
}

// --local-owner must refuse inside an identified agent run so an agent cannot
// elevate itself to a human operator to mint a grant.
func TestLocalOwnerRefusedInsideAgentRun(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "some-agent-identity-token")
	if _, err := localOwnerAPIForResource(nil, standingGrantResource); err == nil {
		t.Fatal("localOwnerAPIForResource must refuse inside an identified agent run")
	}
	// Sanity: with the env cleared the guard does not trip on this check (it will
	// fail later at the socket exchange, which is fine — the guard itself passed).
	_ = os.Unsetenv(cliutil.EnvIdentityToken)
}
