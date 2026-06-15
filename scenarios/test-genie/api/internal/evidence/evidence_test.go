package evidence

import "testing"

func intPtr(n int) *int { return &n }

// signaled is a passing handshake, the common precondition for the non-handshake
// failure cases.
var signaled = Handshake{Signaled: true}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		ev          Evidence
		wantStatus  Status
		wantMessage string
	}{
		{
			name:        "clean load passes",
			ev:          Evidence{Loaded: true, Handshake: signaled},
			wantStatus:  StatusPassed,
			wantMessage: "UI loaded successfully",
		},
		{
			name:        "engine failure outranks everything",
			ev:          Evidence{Loaded: false, LoadError: "navigation timeout", Handshake: signaled, Network: []NetworkEntry{{URL: "x"}}},
			wantStatus:  StatusFailed,
			wantMessage: "navigation timeout",
		},
		{
			name:        "engine failure with no error text gets engine-neutral default",
			ev:          Evidence{Loaded: false},
			wantStatus:  StatusFailed,
			wantMessage: "browser execution failed",
		},
		{
			name:        "handshake not signaled fails even with no other problems",
			ev:          Evidence{Loaded: true, Handshake: Handshake{Signaled: false, TimedOut: true}},
			wantStatus:  StatusFailed,
			wantMessage: "Iframe bridge never signaled ready. See: docs/phases/structure/ui-smoke.md#handshake-timeout",
		},
		{
			name:        "handshake outranks network and page errors",
			ev:          Evidence{Loaded: true, Handshake: Handshake{Signaled: false}, Network: []NetworkEntry{{URL: "x"}}, PageErrors: []PageError{{Message: "boom"}}},
			wantStatus:  StatusFailed,
			wantMessage: "Iframe bridge never signaled ready. See: docs/phases/structure/ui-smoke.md#handshake-timeout",
		},
		{
			name:        "single network failure with status",
			ev:          Evidence{Loaded: true, Handshake: signaled, Network: []NetworkEntry{{URL: "http://x/api", Status: intPtr(500)}}},
			wantStatus:  StatusFailed,
			wantMessage: "HTTP 500 → http://x/api",
		},
		{
			name:        "single network failure with transport error",
			ev:          Evidence{Loaded: true, Handshake: signaled, Network: []NetworkEntry{{URL: "http://x/api", ErrorText: "net::ERR_CONNECTION_REFUSED"}}},
			wantStatus:  StatusFailed,
			wantMessage: "net::ERR_CONNECTION_REFUSED → http://x/api",
		},
		{
			name:       "network outranks page errors",
			ev:         Evidence{Loaded: true, Handshake: signaled, Network: []NetworkEntry{{URL: "x"}}, PageErrors: []PageError{{Message: "boom"}}},
			wantStatus: StatusFailed,
			// default branch (no status, no error text)
			wantMessage: "Request error → x",
		},
		{
			name:        "page error when nothing higher-precedence fails",
			ev:          Evidence{Loaded: true, Handshake: signaled, PageErrors: []PageError{{Message: "TypeError: undefined"}}},
			wantStatus:  StatusFailed,
			wantMessage: "UI exception: TypeError: undefined",
		},
		{
			name:        "console errors alone do not fail the load",
			ev:          Evidence{Loaded: true, Handshake: signaled, Console: []ConsoleEntry{{Level: "error", Message: "handled"}}},
			wantStatus:  StatusPassed,
			wantMessage: "UI loaded successfully",
		},
		{
			name:        "broken render fails the load",
			ev:          Evidence{Loaded: true, Handshake: signaled, RenderBroken: true, RenderBrokenReason: "98% of the frame is a single tone"},
			wantStatus:  StatusFailed,
			wantMessage: "rendered blank/solid color: 98% of the frame is a single tone",
		},
		{
			name:        "network failure outranks broken render",
			ev:          Evidence{Loaded: true, Handshake: signaled, Network: []NetworkEntry{{URL: "http://x/api", Status: intPtr(500)}}, RenderBroken: true, RenderBrokenReason: "blank"},
			wantStatus:  StatusFailed,
			wantMessage: "HTTP 500 → http://x/api",
		},
		{
			name:        "broken render outranks page exception",
			ev:          Evidence{Loaded: true, Handshake: signaled, RenderBroken: true, RenderBrokenReason: "blank", PageErrors: []PageError{{Message: "boom"}}},
			wantStatus:  StatusFailed,
			wantMessage: "rendered blank/solid color: blank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(tt.ev)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMessage)
			}
		})
	}
}

func TestAnalyzeCounts(t *testing.T) {
	ev := Evidence{
		Loaded:    true,
		Handshake: signaled,
		Console: []ConsoleEntry{
			{Level: "error", Message: "e1"},
			{Level: "error", Message: "e2"},
			{Level: "warn", Message: "w1"},
			{Level: "warning", Message: "w2"},
			{Level: "log", Message: "ignored"},
			{Level: "info", Message: "ignored"},
		},
		Network:    []NetworkEntry{{URL: "a"}, {URL: "b"}},
		PageErrors: []PageError{{Message: "x"}},
	}
	got := Analyze(ev)
	if got.ConsoleErrorCount != 2 {
		t.Errorf("ConsoleErrorCount = %d, want 2", got.ConsoleErrorCount)
	}
	if got.ConsoleWarningCount != 2 {
		t.Errorf("ConsoleWarningCount = %d, want 2", got.ConsoleWarningCount)
	}
	if got.NetworkFailureCount != 2 {
		t.Errorf("NetworkFailureCount = %d, want 2", got.NetworkFailureCount)
	}
	if got.PageErrorCount != 1 {
		t.Errorf("PageErrorCount = %d, want 1", got.PageErrorCount)
	}
}

func TestFormatNetworkFailuresCapsAtFive(t *testing.T) {
	var failures []NetworkEntry
	for i := 0; i < 7; i++ {
		failures = append(failures, NetworkEntry{URL: "u", Status: intPtr(500)})
	}
	got := formatNetworkFailures(failures)
	want := "Network failures (7 total): HTTP 500 → u; HTTP 500 → u; HTTP 500 → u; HTTP 500 → u; HTTP 500 → u; ... and 2 more"
	if got != want {
		t.Errorf("formatNetworkFailures =\n  %q\nwant\n  %q", got, want)
	}
}
