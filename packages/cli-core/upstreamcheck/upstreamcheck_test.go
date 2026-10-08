package upstreamcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestExtractSemver(t *testing.T) {
	cases := map[string]string{
		"1.15.7":                "1.15.7",
		"v1.17.9":               "1.17.9",
		"codex-cli 0.131.0":     "0.131.0",
		"2.1.153 (Claude Code)": "2.1.153",
		"no version here":       "",
	}
	for in, want := range cases {
		if got := ExtractSemver(in); got != want {
			t.Errorf("ExtractSemver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		installed, latest string
		want              Status
	}{
		{"1.15.7", "1.17.9", StatusBehind},
		{"1.17.9", "1.17.9", StatusUpToDate},
		{"1.18.0", "1.17.9", StatusAhead},
		{"0.131.0", "0.131.0", StatusUpToDate},
		{"", "1.17.9", StatusUnknown},
		{"1.17.9", "", StatusUnknown},
		{"2.1.9", "2.1.153", StatusBehind}, // numeric, not lexical
	}
	for _, c := range cases {
		if got := Compare(c.installed, c.latest); got != c.want {
			t.Errorf("Compare(%q,%q) = %q, want %q", c.installed, c.latest, got, c.want)
		}
	}
}

func newTestHandlers(installed, latest string, installedErr, latestErr error) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	h := &Handlers{
		Cfg: Config{
			DisplayName:   "opencode",
			InstalledCmd:  []string{"opencode", "--version"},
			PinnedVersion: "1.17.9",
			SourceKind:    SourceGitHub,
			SourceID:      "sst/opencode",
		},
		InstalledRunner: func(ctx context.Context, args []string) (string, error) {
			return installed, installedErr
		},
		LatestFetcher: func(ctx context.Context, kind SourceKind, id string) (string, error) {
			return latest, latestErr
		},
		Stdout: &out,
		Stderr: &errOut,
	}
	return h, &out, &errOut
}

func TestCheckJSONBehind(t *testing.T) {
	h, out, _ := newTestHandlers("1.15.7", "1.17.9", nil, nil)
	if err := h.Check([]string{"--json"}); err != nil {
		t.Fatalf("Check: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}
	if rep.Installed != "1.15.7" || rep.Latest != "1.17.9" || rep.Status != StatusBehind {
		t.Errorf("unexpected report: %+v", rep)
	}
	if rep.Source.Kind != "github" || rep.Source.ID != "sst/opencode" {
		t.Errorf("unexpected source: %+v", rep.Source)
	}
}

func TestCheckNetworkFailureIsUnknownNotError(t *testing.T) {
	h, out, _ := newTestHandlers("1.15.7", "", nil, errors.New("network down"))
	if err := h.Check([]string{"--json"}); err != nil {
		t.Fatalf("Check must not hard-fail on network error, got: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rep.Status != StatusUnknown {
		t.Errorf("status = %q, want unknown", rep.Status)
	}
	if !strings.Contains(rep.Note, "upstream lookup failed") {
		t.Errorf("note = %q, want upstream-lookup mention", rep.Note)
	}
}

func TestCheckTextOutput(t *testing.T) {
	h, out, _ := newTestHandlers("1.15.7", "1.17.9", nil, nil)
	if err := h.Check(nil); err != nil {
		t.Fatalf("Check: %v", err)
	}
	s := out.String()
	for _, want := range []string{"installed: 1.15.7", "latest:    1.17.9", "status:    behind"} {
		if !strings.Contains(s, want) {
			t.Errorf("text output missing %q\n%s", want, s)
		}
	}
}
