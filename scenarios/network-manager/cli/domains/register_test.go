package domains_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	"network-manager/cli/domains/adapters"
	"network-manager/cli/domains/devices"
	"network-manager/cli/domains/home"
	"network-manager/cli/domains/optimize"
	"network-manager/cli/domains/policy"
	"network-manager/cli/domains/privacy"
	"network-manager/cli/domains/resolver"
	"network-manager/cli/domains/snapshot"
)

type registerFunc func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error)

func TestDomainRegistrationsLoadManifestGroups(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	tests := []struct {
		name      string
		groupName string
		register  registerFunc
	}{
		{"adapters", adapters.GroupName, adapters.Register},
		{"devices", devices.GroupName, devices.Register},
		{"home", home.GroupName, home.Register},
		{"optimize", optimize.GroupName, optimize.Register},
		{"policy", policy.GroupName, policy.Register},
		{"privacy", privacy.GroupName, privacy.Register},
		{"resolver", resolver.GroupName, resolver.Register},
		{"snapshot", snapshot.GroupName, snapshot.Register},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			group, err := tc.register(&cliapp.ScenarioApp{}, manifest)
			if err != nil {
				t.Fatalf("register %s: %v", tc.groupName, err)
			}
			if group.Name != tc.groupName {
				t.Fatalf("group name = %q, want %q", group.Name, tc.groupName)
			}
			if len(group.Subcommands) == 0 {
				t.Fatalf("expected %s subcommands", tc.groupName)
			}
		})
	}
}
