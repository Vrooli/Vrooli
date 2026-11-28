//go:build ignore

// Run with: go test seed.go seed_test.go
package main

import (
	"path/filepath"
	"testing"
)

func TestSeedRequiresExactInstalledPrivateBacking(t *testing.T) {
	lease := "fc3c4af6-ef0c-44db-8bc9-1ebd744b8ff2"
	root := filepath.Join(t.TempDir(), "test-genie-playbooks-"+lease)
	for _, test := range []struct {
		name, enabled, root, scenario, lease string
		want                                 bool
	}{
		{"exact paired owner", "1", root, "/fixture/agent-manager", lease, true},
		{"different lease", "1", root, "/fixture/agent-manager", "ca05ea5d-4cf2-4715-a402-31400bf0a72b", false},
		{"old prefix alone", "1", root + "-extra", "/fixture/agent-manager", lease, false},
		{"previous unbound scenario prefix", "1", filepath.Join(t.TempDir(), "test-genie-playbooks-agent-manager-"+lease), "/fixture/agent-manager", lease, false},
		{"missing lease", "1", root, "/fixture/agent-manager", "", false},
		{"missing paired invocation", "", root, "/fixture/agent-manager", lease, false},
		{"relative storage", "1", "test-genie-playbooks-" + lease, "/fixture/agent-manager", lease, false},
		{"missing scenario", "1", root, "", lease, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validSeedIdentity(test.enabled, test.root, test.scenario, test.lease); got != test.want {
				t.Fatalf("private seed identity accepted=%v, want %v", got, test.want)
			}
		})
	}
}
