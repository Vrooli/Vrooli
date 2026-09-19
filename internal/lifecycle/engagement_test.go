package lifecycle

import (
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

// fakeEngagementResolver is the lifecycle's test double for the engagement seam.
type fakeEngagementResolver struct {
	info    EngagementInfo
	engaged bool
	err     error
	asked   []string
}

var _ EngagementResolver = (*fakeEngagementResolver)(nil)

func (f *fakeEngagementResolver) Engagement(scenarioName string) (EngagementInfo, bool, error) {
	f.asked = append(f.asked, scenarioName)
	return f.info, f.engaged, f.err
}

func newSourceDirItem(variant string) scenario.Scenario {
	return scenario.Scenario{Slug: "demo", Path: "/repo/scenarios/demo", Variant: variant}
}

func TestEffectiveSourceDir(t *testing.T) {
	const copyDir = "/cache/vrooli/demo/baseline-x/restore-point"

	tests := []struct {
		name     string
		variant  string
		resolver EngagementResolver
		want     string
		wantErr  bool
	}{
		{
			name:     "nil resolver runs from working tree",
			variant:  "",
			resolver: nil,
			want:     "/repo/scenarios/demo",
		},
		{
			name:     "live with no engagement runs from working tree",
			variant:  "",
			resolver: &fakeEngagementResolver{engaged: false},
			want:     "/repo/scenarios/demo",
		},
		{
			name:    "live with open engagement runs from restore-point copy",
			variant: "",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: copyDir, Slug: "x"},
			},
			want: copyDir,
		},
		{
			name:    "shadow with open engagement runs from working tree",
			variant: "shadow",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: copyDir, Slug: "x"},
			},
			want: "/repo/scenarios/demo",
		},
		{
			name:     "resolver error fails closed",
			variant:  "",
			resolver: &fakeEngagementResolver{err: errors.New("floor unreadable")},
			wantErr:  true,
		},
		{
			name:    "engagement routing to copy with empty path errors",
			variant: "",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: "", Slug: "x"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Runner{Engagements: tc.resolver}
			got, err := r.effectiveSourceDir(newSourceDirItem(tc.variant))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got dir %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("effectiveSourceDir = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEffectiveSourceDirKeysByScenario confirms the resolver is consulted with
// the scenario slug (the engagement key), independent of the variant.
func TestEffectiveSourceDirKeysByScenario(t *testing.T) {
	f := &fakeEngagementResolver{engaged: false}
	r := &Runner{Engagements: f}
	if _, err := r.effectiveSourceDir(newSourceDirItem("shadow")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.asked) != 1 || f.asked[0] != "demo" {
		t.Errorf("resolver asked %v, want [demo]", f.asked)
	}
}
