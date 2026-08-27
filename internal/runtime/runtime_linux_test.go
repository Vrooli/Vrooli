//go:build linux

package runtime

import (
	"context"
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestCurrentLinuxSignalsComeFromHostInventory(t *testing.T) {
	facts := currentPlatformFacts()
	host := Current()
	if host.SupportsSystemd != facts.SupportsSystemd {
		t.Fatalf("SupportsSystemd = %t, host facts = %t", host.SupportsSystemd, facts.SupportsSystemd)
	}
	if host.SupportsSysctl != facts.SupportsSysctl {
		t.Fatalf("SupportsSysctl = %t, host facts = %t", host.SupportsSysctl, facts.SupportsSysctl)
	}
}

func TestLinuxSystemdUsesThreeSignals(t *testing.T) {
	collector := hostinventory.Collector{
		Commands: &shelltest.Fake{Paths: map[string]string{"systemctl": "systemctl"}},
		Files: fakePlatformFiles{
			"/proc/1/comm":              []byte("systemd\n"),
			"/run/systemd/system/.keep": []byte(""),
		},
		GOOS: "linux",
	}
	facts, err := collector.CollectPlatformFacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !facts.SupportsSystemd {
		t.Fatal("PID 1 systemd signal should enable systemd support")
	}
}

type fakePlatformFiles map[string][]byte

func (f fakePlatformFiles) ReadFile(path string) ([]byte, error) {
	data, ok := f[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}
