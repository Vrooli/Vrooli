package systemevents

import (
	"strings"
	"testing"
	"time"
)

func TestParseDPKGLogExtractsDriverAndKernelEvents(t *testing.T) {
	events := ParseDPKGLog([]byte(strings.Join([]string{
		"2026-05-06 06:10:23 install linux-image-6.17.0-23-generic:amd64 <none> 6.17.0-23.23~24.04.1",
		"2026-05-08 12:57:16 upgrade nvidia-driver-580-open:amd64 580.126.09-0ubuntu0.24.04.2 580.142-0ubuntu0.24.04.1",
		"2026-05-08 22:59:39 upgrade linux-firmware 20240318.git3b128b60-0ubuntu2.26 20240318.git3b128b60-0ubuntu2.27",
		"2026-05-08 23:00:00 upgrade unrelated-package 1 2",
	}, "\n")))

	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}
	if events[0].Category != "kernel" {
		t.Fatalf("first category = %q, want kernel", events[0].Category)
	}
	if events[1].Category != "driver" {
		t.Fatalf("second category = %q, want driver", events[1].Category)
	}
	if events[2].Category != "firmware" {
		t.Fatalf("third category = %q, want firmware", events[2].Category)
	}
}

func TestParseAPTHistoryGroupsRelevantPackages(t *testing.T) {
	events := ParseAPTHistory([]byte(`Start-Date: 2026-05-08  12:57:16
Commandline: apt-get install linux-modules-nvidia-580-open-6.17.0-23-generic
Install: nvidia-firmware-580-580.142:amd64 (580.142-0ubuntu0.24.04.1), linux-modules-nvidia-580-open-6.17.0-23-generic:amd64 (6.17.0-23.23~24.04.1+1)
Upgrade: nvidia-driver-580-open:amd64 (580.126.09-0ubuntu0.24.04.2, 580.142-0ubuntu0.24.04.1)
End-Date: 2026-05-08  12:57:26
`))

	if len(events) < 2 {
		t.Fatalf("event count = %d, want driver and firmware events", len(events))
	}
	categories := map[string]bool{}
	for _, event := range events {
		categories[event.Category] = true
	}
	if !categories["driver"] || !categories["firmware"] {
		t.Fatalf("categories = %#v, want driver and firmware", categories)
	}
}

func TestBuildCorrelationsUsesConservativeTemporalHints(t *testing.T) {
	base := time.Date(2026, 5, 6, 6, 10, 0, 0, time.UTC)
	events := []Event{
		{ID: 1, OccurredAt: base, Source: "dpkg-log", Category: "kernel", Title: "Package install: linux-image", Summary: "install linux-image"},
		{ID: 2, OccurredAt: base.Add(16 * time.Hour), Source: "journalctl", Category: "crash", Title: "Hardware/reset signal", Summary: "unclean reset"},
		{ID: 3, OccurredAt: base.Add(40 * time.Hour), Source: "dpkg-log", Category: "firmware", Title: "Package upgrade: linux-firmware", Summary: "upgrade linux-firmware"},
	}

	correlations := BuildCorrelations(events)
	if len(correlations) < 2 {
		t.Fatalf("correlation count = %d, want at least kernel-before-crash and firmware-after-crash", len(correlations))
	}
	for _, correlation := range correlations {
		if correlation.Confidence != "temporal" {
			t.Fatalf("confidence = %q, want temporal", correlation.Confidence)
		}
		if strings.Contains(strings.ToLower(correlation.Summary), "root cause") {
			t.Fatalf("correlation overclaims causality: %#v", correlation)
		}
	}
}
