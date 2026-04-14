package resources_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestEnabledResourcesUseGoNativeCLIContract(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}

	var service struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled bool `json:"enabled"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		t.Fatalf("unmarshal service.json: %v", err)
	}

	var enabled []string
	for name, entry := range service.Dependencies.Resources {
		if entry.Enabled {
			enabled = append(enabled, name)
		}
	}
	sort.Strings(enabled)

	for _, name := range enabled {
		t.Run(name, func(t *testing.T) {
			for _, rel := range []string{
				filepath.Join("resources", name, "cli", "go.mod"),
				filepath.Join("resources", name, "cli", "go.sum"),
				filepath.Join("resources", name, "cli", "main.go"),
				filepath.Join("resources", name, "cli", "install.sh"),
				filepath.Join("resources", name, "cli", "install.ps1"),
			} {
				if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
					t.Fatalf("expected %s: %v", rel, err)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "resources", name, "cli.sh")); !os.IsNotExist(err) {
				t.Fatalf("expected root-level cli.sh removal for %s, stat err=%v", name, err)
			}
			installPath := filepath.Join(root, "resources", name, "lib", "install.sh")
			data, err := os.ReadFile(installPath)
			if err == nil {
				body := string(data)
				for _, forbidden := range []string{
					"install_resource_cli",
					"uninstall_resource_cli",
					".vrooli/resource-registry",
				} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("%s still references deprecated shell-era resource CLI plumbing %q", installPath, forbidden)
					}
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("read %s: %v", installPath, err)
			}
		})
	}
}

func TestResourceShellInstallHelpersAreRemoved(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	if _, err := os.Stat(filepath.Join(root, "scripts", "lib", "resources")); !os.IsNotExist(err) {
		t.Fatalf("expected scripts/lib/resources removal, stat err=%v", err)
	}
}
