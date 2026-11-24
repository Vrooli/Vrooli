package rootstatus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestUpdatesCheckNamesTheAction(t *testing.T) {
	incomplete := &rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64), Complete: false}
	for name, tc := range map[string]struct {
		enrolled    bool
		enrollment  coreactivation.Enrollment
		err         error
		progress    *rebaselineprogress.Progress
		progressErr error
		wantStatus  string
		wantText    string
	}{
		"enrolled":              {true, coreactivation.Enrollment{Enabled: true}, nil, nil, nil, "ok", "activate automatically"},
		"disabled":              {true, coreactivation.Enrollment{Enabled: false}, nil, nil, nil, "ok", "`make install` installs directly"},
		"not enrolled":          {false, coreactivation.Enrollment{}, nil, nil, nil, "attention", "`vrooli runtime update enroll`"},
		"paused flag":           {true, coreactivation.Enrollment{Enabled: false, RebaselinePaused: true}, nil, nil, nil, "attention", "`make rebaseline`"},
		"incomplete progress":   {true, coreactivation.Enrollment{Enabled: true}, nil, incomplete, nil, "attention", "unfinished root transition"},
		"progress unreadable":   {true, coreactivation.Enrollment{Enabled: true}, nil, nil, os.ErrPermission, "attention", "permission"},
		"enrollment unreadable": {false, coreactivation.Enrollment{}, os.ErrPermission, nil, nil, "attention", "cannot read"},
	} {
		t.Run(name, func(t *testing.T) {
			check := updatesCheck(tc.enrolled, tc.enrollment, tc.err, tc.progress, tc.progressErr)
			if check.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (%s)", check.Status, tc.wantStatus, check.Message)
			}
			if !strings.Contains(check.Message, tc.wantText) {
				t.Fatalf("message = %q, want it to contain %q", check.Message, tc.wantText)
			}
		})
	}
}

func TestChecksReportsAPausedTransitionFromTheProgressRecord(t *testing.T) {
	home := t.TempDir()
	state := filepath.Join(home, ".vrooli", "state", "root-rebaseline")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64), Bundle: strings.Repeat("c", 64), Complete: false}
	if err := rebaselineprogress.Write(filepath.Join(state, "progress.json"), progress); err != nil {
		t.Fatal(err)
	}
	checks := Checks(context.Background(), home)
	byName := map[string]Check{}
	for _, check := range checks {
		byName[check.Name] = check
	}
	if got := byName["root_rebaseline"]; got.Status != "attention" || !strings.Contains(got.Message, "`make rebaseline`") {
		t.Fatalf("root_rebaseline = %+v", got)
	}
	if got := byName["root_updates"]; got.Status != "attention" {
		t.Fatalf("root_updates = %+v", got)
	}
}

func TestChecksWithNoTransitionReportsNoPendingWork(t *testing.T) {
	home := t.TempDir()
	checks := Checks(context.Background(), home)
	for _, check := range checks {
		if check.Name == "root_rebaseline" {
			t.Fatalf("a fresh home reported a pending transition: %+v", check)
		}
	}
}

func TestStateCoverageCountsDeclaredContractOnlyAndUnmigrated(t *testing.T) {
	// The check reads the installed image, so it is exercised through its
	// counting rule rather than a fixture binary.
	digest := strings.Repeat("a", 64)
	manifest := stateformat.Manifest{Format: stateformat.ManifestFormat, Stores: []stateformat.Store{
		{Name: "a", Write: 1, Reads: []stateformat.Version{1}, Schemas: map[stateformat.Version]string{1: digest}, StatePaths: []string{"state/a.json"}},
		{Name: "b", Write: 1, Reads: []stateformat.Version{1}, Schemas: map[stateformat.Version]string{1: digest}, ContractOnly: true},
		{Name: "c", Write: 1, Reads: []stateformat.Version{1}, Schemas: map[stateformat.Version]string{1: digest}},
	}}
	coverage := manifest.Coverage()
	declared, contractOnly, unmigrated := coverage.Declared, coverage.ContractOnly, coverage.Unmigrated
	if declared != 1 || contractOnly != 1 || unmigrated != 1 {
		t.Fatalf("coverage = %d/%d/%d, want 1/1/1", declared, contractOnly, unmigrated)
	}
}

func TestChecksNamesTheRecoveryForAnUnfinishedOrdinaryBundle(t *testing.T) {
	home := t.TempDir()
	intent := buildinfo.PreparedBundleIntent{
		Version:      1,
		Manifest:     strings.Repeat("a", 64),
		Root:         strings.Repeat("b", 64),
		SameContract: true,
		SourceRoot:   filepath.Join(home, "source"),
	}
	if err := buildinfo.WritePreparedBundleIntent(home, intent); err != nil {
		t.Fatal(err)
	}
	checks := Checks(context.Background(), home)
	byName := map[string]Check{}
	for _, check := range checks {
		byName[check.Name] = check
	}
	got, ok := byName["root_bundle_recovery"]
	if !ok {
		t.Fatal("doctor did not report the incomplete ordinary bundle")
	}
	if got.Status != "attention" {
		t.Fatalf("status = %q (%s)", got.Status, got.Message)
	}
	for _, want := range []string{"--resume-bundle", "--abort-bundle", "vrooli-atomic-install"} {
		if !strings.Contains(got.Message, want) {
			t.Fatalf("message = %q, want it to name %q", got.Message, want)
		}
	}
}
