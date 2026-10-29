package statusline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	settings Settings
	samples  string
}

func newFixture(t *testing.T, settingsJSON string) fixture {
	t.Helper()
	home := t.TempDir()
	f := fixture{
		settings: DefaultSettings(filepath.Join(home, ".claude", ".vrooli-hooks"), filepath.Join(home, ".claude", "settings.json")),
		samples:  filepath.Join(home, ".vrooli", "state", "vrooli", "claude-code", "quota-samples.ndjson"),
	}
	if settingsJSON != "" {
		if err := os.MkdirAll(filepath.Dir(f.settings.SettingsPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f.settings.SettingsPath, []byte(settingsJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f fixture) top(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	top, err := readSettings(f.settings.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	return top
}

func statusInput(five, seven float64) string {
	return `{"session_id":"s-1","model":{"id":"claude-opus","display_name":"Opus"},"rate_limits":{"five_hour":{"used_percentage":` + jsonNum(five) + `,"resets_at":1738425600},"seven_day":{"used_percentage":` + jsonNum(seven) + `,"resets_at":1738857600}}}`
}

func jsonNum(v float64) string { b, _ := json.Marshal(v); return string(b) }

func readSamples(t *testing.T, path string) []Sample {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var out []Sample
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var s Sample
		if err := json.Unmarshal(scanner.Bytes(), &s); err != nil {
			t.Fatalf("sample line %q: %v", scanner.Text(), err)
		}
		out = append(out, s)
	}
	return out
}

func TestInstallIsIdempotentPreservesSettingsAndUninstallRestoresPrevious(t *testing.T) {
	f := newFixture(t, `{"permissions":{"deny":["Bash(rm -rf /)"]},"statusLine":{"type":"command","command":"~/bin/my-line.sh","padding":1}}`)
	const command = "/opt/resource-claude-code statusline run"

	first, err := f.settings.Install(command)
	if err != nil || first.Status != "installed" || first.Chained != "~/bin/my-line.sh" {
		t.Fatalf("first install = %+v, %v", first, err)
	}
	second, err := f.settings.Install(command)
	if err != nil || second.Status != "unchanged" || second.Chained != "~/bin/my-line.sh" {
		t.Fatalf("second install = %+v, %v (a reinstall must not record itself as the previous command)", second, err)
	}
	top := f.top(t)
	if statusCommand(top["statusLine"]) != command {
		t.Fatalf("statusLine = %s", top["statusLine"])
	}
	if !strings.Contains(string(top["permissions"]), "rm -rf") {
		t.Fatalf("unrelated settings were not preserved: %s", top["permissions"])
	}

	removed, err := f.settings.Uninstall()
	if err != nil || removed.Status != "removed" {
		t.Fatalf("uninstall = %+v, %v", removed, err)
	}
	top = f.top(t)
	var restored map[string]any
	_ = json.Unmarshal(top["statusLine"], &restored)
	if restored["command"] != "~/bin/my-line.sh" || restored["padding"] != float64(1) {
		t.Fatalf("previous statusLine not restored verbatim: %s", top["statusLine"])
	}
	if _, err := os.Stat(f.settings.StatePath); !os.IsNotExist(err) {
		t.Fatalf("state file left behind: %v", err)
	}
	again, err := f.settings.Uninstall()
	if err != nil || again.Status != "absent" {
		t.Fatalf("second uninstall = %+v, %v", again, err)
	}
}

func TestUninstallWithoutPreviousRemovesKeyAndLeavesOperatorChangesAlone(t *testing.T) {
	f := newFixture(t, "")
	if _, err := f.settings.Install("rcc statusline run"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.settings.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.top(t)["statusLine"]; ok {
		t.Fatal("statusLine should be removed when nothing was displaced")
	}

	if _, err := f.settings.Install("rcc statusline run"); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(f.settings.SettingsPath, map[string]any{"statusLine": map[string]any{"type": "command", "command": "operator-line"}}); err != nil {
		t.Fatal(err)
	}
	if result, err := f.settings.Uninstall(); err != nil || result.Status != "absent" || statusCommand(f.top(t)["statusLine"]) != "operator-line" {
		t.Fatalf("uninstall touched an operator-owned statusLine: %+v %v", result, err)
	}
}

func TestFeedRecordsChangedReadingsWithHeartbeatAndPrintsStatus(t *testing.T) {
	f := newFixture(t, "")
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	feed := Feed{SamplesPath: f.samples, StatePath: f.settings.StatePath, Now: func() time.Time { return now }}
	run := func(input string) string {
		var out bytes.Buffer
		if err := feed.Run(strings.NewReader(input), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}

	if out := run(statusInput(23.5, 41.2)); !strings.Contains(out, "5h 24%") || !strings.Contains(out, "7d 41%") || !strings.HasPrefix(out, "Opus") {
		t.Fatalf("status line = %q", out)
	}
	now = now.Add(time.Minute)
	run(statusInput(23.5, 41.2)) // unchanged within heartbeat: not recorded
	now = now.Add(time.Minute)
	run(statusInput(24, 41.2)) // changed: recorded
	now = now.Add(Heartbeat)
	run(statusInput(24, 41.2))                                  // unchanged past heartbeat: recorded
	run(`{"session_id":"s-1","model":{"display_name":"Opus"}}`) // no rate limits (API key user): nothing recorded

	samples := readSamples(t, f.samples)
	if len(samples) != 3 {
		t.Fatalf("samples = %d, want 3: %+v", len(samples), samples)
	}
	got := samples[1]
	if got.SessionID != "s-1" || got.Model != "claude-opus" || *got.FiveHour.UsedPercentage != 24 || got.FiveHour.ResetsAt != 1738425600 || *got.SevenDay.UsedPercentage != 41.2 {
		t.Fatalf("sample = %+v", got)
	}
}

func TestFeedChainsPreviousCommandWithSameInput(t *testing.T) {
	f := newFixture(t, `{"statusLine":{"type":"command","command":"sed 's/^/mine:/'"}}`)
	if _, err := f.settings.Install("rcc statusline run"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	input := statusInput(10, 20)
	if err := (Feed{SamplesPath: f.samples, StatePath: f.settings.StatePath}).Run(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "mine:"+input {
		t.Fatalf("chained output = %q", out.String())
	}
	if len(readSamples(t, f.samples)) != 1 {
		t.Fatal("chaining must still record the sample")
	}
}

func TestFeedRotatesAtSizeBound(t *testing.T) {
	f := newFixture(t, "")
	if err := os.MkdirAll(filepath.Dir(f.samples), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.samples, bytes.Repeat([]byte("x"), MaxSampleBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (Feed{SamplesPath: f.samples, StatePath: f.settings.StatePath}).Run(strings.NewReader(statusInput(1, 2)), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(f.samples + ".1"); err != nil || info.Size() != MaxSampleBytes {
		t.Fatalf("rotated generation missing: %v", err)
	}
	if len(readSamples(t, f.samples)) != 1 {
		t.Fatal("new generation should hold just the new sample")
	}
}
