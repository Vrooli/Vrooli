package clock

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubAll(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origHTTPHead := HTTPHeadFn
	origNow := NowFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		HTTPHeadFn = origHTTPHead
		NowFn = origNow
	}
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "clock",
		Handler: "clock",
	})
}

func mockAccurateClock() {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	NowFn = func() time.Time { return now }
	HTTPHeadFn = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Date": []string{now.Format(http.TimeFormat)}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
}

func mockInaccurateClock() {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	remote := now.Add(-10 * time.Minute) // 10 min drift exceeds 5 min tolerance
	NowFn = func() time.Time { return now }
	HTTPHeadFn = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Date": []string{remote.Format(http.TimeFormat)}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "clock" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{
		Name:   "clock",
		Kind:   hostreqspec.KindSafeguard,
		Manual: true,
	})
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectClockAccurate(t *testing.T) {
	restore := stubAll(t)
	defer restore()
	mockAccurateClock()

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{
		Name: "clock",
		Kind: hostreqspec.KindSafeguard,
	})
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectClockInaccurate(t *testing.T) {
	restore := stubAll(t)
	defer restore()
	mockInaccurateClock()

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{
		Name: "clock",
		Kind: hostreqspec.KindSafeguard,
	})
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	if status.SupportClass != hostreqkit.SupportSupported {
		t.Fatalf("SupportClass = %q (Linux should be supported)", status.SupportClass)
	}
}

func TestInspectNoSourcesReachable(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	HTTPHeadFn = func(url string) (*http.Response, error) {
		return nil, os.ErrDeadlineExceeded
	}

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{
		Name: "clock",
		Kind: hostreqspec.KindSafeguard,
	})
	if !status.Applied {
		t.Fatalf("expected Applied = true when no sources reachable, notes: %v", status.Notes)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "could not verify") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'could not verify' note, got: %v", status.Notes)
	}
}

func TestInspectNonLinuxInaccurateUnsupported(t *testing.T) {
	restore := stubAll(t)
	defer restore()
	mockInaccurateClock()

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, hostreqspec.ResolvedRequirement{
		Name: "clock",
		Kind: hostreqspec.KindSafeguard,
	})
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestApplyUnsupportedReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "darwin"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportUnsupported,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyAlreadyAppliedSkips(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyDryRun(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyHwclockSuccess(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "hwclock" || name == "sudo" {
			return "/usr/sbin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return nil
	}

	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	NowFn = func() time.Time { return now }
	HTTPHeadFn = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Date": []string{now.Format(http.TimeFormat)}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "hwclock") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hwclock in notes, got: %v", status.Notes)
	}
}

func TestApplyFallbackToNtpdate(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	// hwclock not available, ntpdate is
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "ntpdate" || name == "sudo" {
			return "/usr/sbin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return nil
	}

	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	NowFn = func() time.Time { return now }
	HTTPHeadFn = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Date": []string{now.Format(http.TimeFormat)}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "ntpdate") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ntpdate in notes, got: %v", status.Notes)
	}
}

func TestApplyAllMethodsFail(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return fmt.Errorf("command failed")
	}

	// Clock stays inaccurate after all sync attempts
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	remote := now.Add(-10 * time.Minute)
	NowFn = func() time.Time { return now }
	HTTPHeadFn = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Date": []string{remote.Format(http.TimeFormat)}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}
