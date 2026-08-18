//go:build linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests guard the unit that production actually installs. An earlier
// version of this contract lived in internal/runtimesupervisor against a
// second, unused copy of the renderer: the test asserted the correct unit
// while installService shipped a different one, so a quoting regression that
// made the unit unloadable passed CI for days. Assert the shipping function,
// and — where systemd is present — assert it against systemd rather than
// against our reading of the documentation.

func TestSystemdUnitContentDirectiveQuoting(t *testing.T) {
	content := systemdUnitContent("/opt/vrooli/bin/vrooli", "/home/tester", "/srv/vrooli", "/home/tester/.vrooli/logs/runtime-supervisor.log")

	for _, want := range []string{
		// Quote-aware directives keep their quotes.
		`Environment=HOME="/home/tester"`,
		`Environment=VROOLI_SOURCE_ROOT="/srv/vrooli"`,
		`ExecStart="/opt/vrooli/bin/vrooli" --no-stale-check runtime supervisor run`,
		// WorkingDirectory= is read verbatim: quotes become part of the path
		// and systemd rejects the unit as "path is not absolute".
		"WorkingDirectory=/srv/vrooli\n",
		// The daemon must have somewhere to say why it died.
		"StandardOutput=append:/home/tester/.vrooli/logs/runtime-supervisor.log\n",
		"StandardError=append:/home/tester/.vrooli/logs/runtime-supervisor.log\n",
		// A control-plane daemon that exits cleanly still needs to come back.
		"Restart=always\n",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("unit content missing %q:\n%s", want, content)
		}
	}

	if strings.Contains(content, `WorkingDirectory="`) {
		t.Fatalf("WorkingDirectory must not be quoted; systemd rejects the unit:\n%s", content)
	}
}

func TestSystemdUnitContentOmitsLoggingWhenNoLogPath(t *testing.T) {
	content := systemdUnitContent("/opt/vrooli/bin/vrooli", "/home/tester", "/srv/vrooli", "")
	if strings.Contains(content, "StandardOutput=") || strings.Contains(content, "StandardError=") {
		t.Fatalf("expected journal defaults when no log path is configured:\n%s", content)
	}
}

func TestSystemdUnitContentOmitsSourceRootDirectives(t *testing.T) {
	content := systemdUnitContent("/opt/vrooli/bin/vrooli", "/home/tester", "  ", "")
	if strings.Contains(content, "WorkingDirectory=") || strings.Contains(content, "VROOLI_SOURCE_ROOT") {
		t.Fatalf("expected no source-root directives when source root is empty:\n%s", content)
	}
}

// TestSystemdUnitContentLoadsUnderSystemd is the contract that matters: systemd
// itself must accept the rendered unit. Skipped where systemd-analyze is
// unavailable (containers, CI images, non-systemd hosts) so the suite stays
// portable, but on any developer or host machine with systemd it fails loudly
// on a directive systemd will not load.
func TestSystemdUnitContentLoadsUnderSystemd(t *testing.T) {
	analyze, err := exec.LookPath("systemd-analyze")
	if err != nil {
		t.Skip("systemd-analyze is unavailable; skipping real-systemd unit verification")
	}

	// systemd-analyze checks that the executable and working directory really
	// exist, so the fixtures are real files on disk. That keeps the test
	// focused on directive syntax — the only thing this contract is about.
	// Paths with spaces are the reason the quoting is per-directive rather
	// than uniform, so they are the case most likely to regress.
	cases := map[string]struct {
		dirName        string
		withSourceRoot bool
	}{
		"plain paths":       {dirName: "vrooli", withSourceRoot: true},
		"paths with spaces": {dirName: "my vrooli install", withSourceRoot: true},
		"no source root":    {dirName: "vrooli", withSourceRoot: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), tc.dirName)
			exe := filepath.Join(base, "bin", "vrooli")
			logPath := filepath.Join(base, ".vrooli", "logs", "runtime-supervisor.log")
			for _, dir := range []string{filepath.Dir(exe), filepath.Dir(logPath)} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("create fixture dir: %v", err)
				}
			}
			if err := os.WriteFile(exe, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
				t.Fatalf("write fixture executable: %v", err)
			}
			sourceRoot := ""
			if tc.withSourceRoot {
				sourceRoot = base
			}

			// A distinct unit name keeps systemd-analyze from resolving — and
			// then reporting on — a unit already installed on this host.
			path := filepath.Join(t.TempDir(), "vrooli-runtime-supervisor-contract-test.service")
			content := systemdUnitContent(exe, base, sourceRoot, logPath)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("write unit: %v", err)
			}

			// verify exits 0 even when it reports fatal setting errors, so the
			// output is the signal. Only lines naming the file under test count.
			output, _ := exec.Command(analyze, "--user", "verify", path).CombinedOutput()
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(line, filepath.Base(path)) {
					t.Fatalf("systemd rejected the rendered unit: %s\n\nunit:\n%s", strings.TrimSpace(line), content)
				}
			}
		})
	}
}
