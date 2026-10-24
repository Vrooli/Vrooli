package recommend

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// testEnv isolates the process-wide fallback pin path so recommendation tests
// exercise their supplied host snapshot rather than an operator's persisted
// recommendation pin choice in the developer's runtime home.
func testEnv(t *testing.T, values map[string]string) func(string) string {
	t.Helper()
	pinPath := filepath.Join(t.TempDir(), "whisper-model.pin")
	return func(key string) string {
		if key == EnvPinFile {
			return pinPath
		}
		return values[key]
	}
}

type fakeHostProbe struct {
	snapshot hostinventory.Snapshot
	err      error
}

func (f fakeHostProbe) Collect(context.Context) (hostinventory.Snapshot, error) {
	return f.snapshot, f.err
}

func TestCmd_JSON_FrozenSchema(t *testing.T) {
	stdout := &bytes.Buffer{}
	h := &Handlers{
		Probe: fakeHostProbe{snapshot: hostinventory.Snapshot{
			OS: "linux", Arch: "amd64", CPU: hostinventory.CPU{Cores: 16}, Memory: hostinventory.Memory{TotalBytes: 32 << 30},
			GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 3070", VRAMBytes: 8 << 30}},
		}},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		GetEnv: testEnv(t, nil),
	}
	if err := h.Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var got jsonResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Model != "small" {
		t.Errorf("model=%s want small", got.Model)
	}
	if got.BudgetPct != 50 {
		t.Errorf("budget=%d", got.BudgetPct)
	}
	if got.Host.OS != "linux" || len(got.Host.GPUs) != 1 || got.Reason == "" {
		t.Errorf("host shape: %#v", got.Host)
	}
}

func TestCmd_HumanDefault(t *testing.T) {
	stdout := &bytes.Buffer{}
	h := &Handlers{
		Probe:  fakeHostProbe{snapshot: hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 2}, Memory: hostinventory.Memory{TotalBytes: 2 << 30}}},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		GetEnv: testEnv(t, nil),
	}
	if err := h.Run(nil); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "tiny\n" {
		t.Errorf("got %q", stdout.String())
	}
}

func TestCmd_HumanExplain(t *testing.T) {
	stdout := &bytes.Buffer{}
	h := &Handlers{
		Probe:  fakeHostProbe{snapshot: hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 2}, Memory: hostinventory.Memory{TotalBytes: 2 << 30}}},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		GetEnv: testEnv(t, nil),
	}
	if err := h.Run([]string{"--explain"}); err != nil {
		t.Fatal(err)
	}
	s := stdout.String()
	for _, want := range []string{"tiny\n", "reason:", "host:"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %q", want, s)
		}
	}
}

func TestResolveBudgetPct(t *testing.T) {
	cases := map[string]int{
		"":    DefaultBudgetPct,
		"-1":  DefaultBudgetPct,
		"0":   DefaultBudgetPct,
		"101": DefaultBudgetPct,
		"abc": DefaultBudgetPct,
		"25":  25,
		"100": 100,
	}
	for v, want := range cases {
		got := resolveBudgetPct(func(string) string { return v })
		if got != want {
			t.Errorf("input=%q got=%d want=%d", v, got, want)
		}
	}
}

func TestCmd_BudgetFlagOverridesEnv(t *testing.T) {
	stdout := &bytes.Buffer{}
	h := &Handlers{
		Probe: fakeHostProbe{snapshot: hostinventory.Snapshot{
			CPU: hostinventory.CPU{Cores: 32}, Memory: hostinventory.Memory{TotalBytes: 64 << 30},
			GPUs: []hostinventory.GPU{{Name: "RTX 4090", VRAMBytes: 24 << 30}},
		}},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
		GetEnv: testEnv(t, map[string]string{envBudgetVar: "50"}),
	}
	if err := h.Run([]string{"--budget-pct", "10", "--json"}); err != nil {
		t.Fatal(err)
	}
	var got jsonResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.BudgetPct != 10 || got.Model != "small" {
		t.Errorf("got=%#v", got)
	}
}
