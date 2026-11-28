package cliutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type peerRecordFixture struct {
	Name    string         `json:"name"`
	Target  string         `json:"target"`
	PortKey string         `json:"port_key"`
	Owner   string         `json:"owner"`
	Record  map[string]any `json:"record"`
	Want    string         `json:"want"`
}

func TestSharedPeerRecordConformanceFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "api-core", "discovery", "testdata", "peer-record-conformance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []peerRecordFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	peers := filepath.Join(home, "peers")
	if err := os.MkdirAll(peers, 0o700); err != nil {
		t.Fatal(err)
	}
	previous := peerOwnerPIDAliveFn
	t.Cleanup(func() { peerOwnerPIDAliveFn = previous })
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			peerOwnerPIDAliveFn = func(pid int) bool { return fixture.Owner == "alive" || fixture.Owner == "eperm" }
			encoded, err := json.Marshal(fixture.Record)
			if err != nil {
				t.Fatal(err)
			}
			encoded = []byte(strings.ReplaceAll(string(encoded), `"$PID"`, `1`))
			scenario, variant := SplitInstance(fixture.Target)
			slug := InstanceSlug(scenario, normalizeVariant(variant))
			if err := os.WriteFile(filepath.Join(peers, slug+".json"), encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			out := lookupPeerRecord(fixture.Target, fixture.PortKey)
			if fixture.Want == "" {
				if out.Resolved() {
					t.Fatalf("resolved port %s; want rejection", out.Port)
				}
				return
			}
			if !out.Resolved() || out.Port != fixture.Want {
				t.Fatalf("outcome = %#v, want port %s", out, fixture.Want)
			}
		})
	}
}
