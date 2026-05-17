//go:build linux

package runtimesupervisor

import (
	"strings"
	"testing"
)

func TestSystemdUserUnitContentCarriesSourceRoot(t *testing.T) {
	content := systemdUserUnitContent("/opt/vrooli/bin/vrooli", "/home/tester", "/srv/vrooli")
	for _, want := range []string{
		`Environment=HOME="/home/tester"`,
		`Environment=VROOLI_SOURCE_ROOT="/srv/vrooli"`,
		`WorkingDirectory=/srv/vrooli`,
		`ExecStart="/opt/vrooli/bin/vrooli" --no-stale-check runtime supervisor run`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("unit content missing %q:\n%s", want, content)
		}
	}
}
