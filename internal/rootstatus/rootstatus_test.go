package rootstatus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
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
	progress := rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64), Bundle: strings.Repeat("c", 64), Mode: "prepared", Complete: false}
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
