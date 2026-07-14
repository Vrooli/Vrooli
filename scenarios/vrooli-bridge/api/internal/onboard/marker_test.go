package onboard

import "testing"

func TestParseMarker(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		wantOK     bool
		wantEvent  string
		wantStep   string
		wantDetail string
	}{
		{
			name: "non-marker line ignored", line: "some human log to stderr echoed", wantOK: false,
		},
		{
			name: "run-start no step", line: `VBOOTSTRAP v=1 event=run-start detail="vrooli-bridge node bootstrap"`,
			wantOK: true, wantEvent: "run-start", wantDetail: "vrooli-bridge node bootstrap",
		},
		{
			name: "step-start with step and detail", line: `VBOOTSTRAP v=1 event=step-start step=detect-os detail="identify platform"`,
			wantOK: true, wantEvent: "step-start", wantStep: "detect-os", wantDetail: "identify platform",
		},
		{
			name: "step-ok no detail", line: `VBOOTSTRAP v=1 event=step-ok step=clone`,
			wantOK: true, wantEvent: "step-ok", wantStep: "clone",
		},
		{
			name: "step-skip", line: `VBOOTSTRAP v=1 event=step-skip step=setup detail="sentinel present"`,
			wantOK: true, wantEvent: "step-skip", wantStep: "setup", wantDetail: "sentinel present",
		},
		{
			name: "detail with spaces and equals", line: `VBOOTSTRAP v=1 event=run-ok detail="node abc paired and online (k=v)"`,
			wantOK: true, wantEvent: "run-ok", wantDetail: "node abc paired and online (k=v)",
		},
		{
			name: "trailing CR trimmed", line: "VBOOTSTRAP v=1 event=step-fail step=setup detail=\"boom\"\r",
			wantOK: true, wantEvent: "step-fail", wantStep: "setup", wantDetail: "boom",
		},
		{
			name: "missing event is not a marker", line: `VBOOTSTRAP v=1 step=clone`,
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := parseMarker(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if m.Event != tc.wantEvent || m.Step != tc.wantStep || m.Detail != tc.wantDetail {
				t.Fatalf("got %+v, want event=%q step=%q detail=%q", m, tc.wantEvent, tc.wantStep, tc.wantDetail)
			}
		})
	}
}

func TestStepStatusForEvent(t *testing.T) {
	cases := map[string]StepStatus{
		eventStepStart: StepStatusStarted,
		eventStepOK:    StepStatusOK,
		eventStepSkip:  StepStatusSkipped,
		eventStepFail:  StepStatusFailed,
		eventRunStart:  StepStatusUnspecified,
		"nonsense":     StepStatusUnspecified,
	}
	for event, want := range cases {
		if got := stepStatusForEvent(event); got != want {
			t.Errorf("stepStatusForEvent(%q) = %v, want %v", event, got, want)
		}
	}
}

func TestExtractNodeID(t *testing.T) {
	id := "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
	cases := map[string]string{
		"paired as " + id:                   id,
		"pinned key present, node " + id:    id,
		"node " + id + " paired and online": id,
		"no id here":                        "",
		"":                                  "",
	}
	for detail, want := range cases {
		if got := extractNodeID(detail); got != want {
			t.Errorf("extractNodeID(%q) = %q, want %q", detail, got, want)
		}
	}
}
