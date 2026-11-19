package convert

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

func TestEnhancedSystemReportToProtoIncludesOwnerResources(t *testing.T) {
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	report := &models.EnhancedSystemReport{
		OwnerResources: models.OwnerResourceReport{
			WindowStart:           start,
			WindowEnd:             start.Add(7 * 24 * time.Hour),
			SampleCoverageStart:   start.Add(time.Second),
			SampleCoverageEnd:     start.Add(7*24*time.Hour - time.Second),
			Complete:              true,
			AttributionComplete:   false,
			ReturnedProcessGroups: 2,
			Owners: []models.OwnerResourceSummary{{
				Owner:                    "scenario-a",
				ProcessGroups:            1,
				Samples:                  20,
				CoreHours:                1.5,
				MeanRSSGiBProcessSum:     2.25,
				MeanSwapGiBProcessSum:    0.5,
				PeakSingleProcessVRAMGiB: 1.25,
			}},
		},
	}

	got := EnhancedSystemReportToProto(report).GetOwnerResources()
	if got == nil || !got.GetComplete() || got.GetAttributionComplete() || got.GetReturnedProcessGroups() != 2 {
		t.Fatalf("owner-resource report completeness = %+v", got)
	}
	if !got.GetWindowStart().AsTime().Equal(start) || len(got.GetOwners()) != 1 {
		t.Fatalf("owner-resource window/rows = %+v", got)
	}
	if !got.GetSampleCoverageStart().AsTime().Equal(start.Add(time.Second)) || !got.GetSampleCoverageEnd().AsTime().Equal(start.Add(7*24*time.Hour-time.Second)) {
		t.Fatalf("owner-resource sample coverage = [%s, %s]", got.GetSampleCoverageStart().AsTime(), got.GetSampleCoverageEnd().AsTime())
	}
	owner := got.GetOwners()[0]
	if owner.GetOwner() != "scenario-a" || owner.GetCoreHours() != 1.5 || owner.GetMeanRssGibProcessSum() != 2.25 || owner.GetMeanSwapGibProcessSum() != .5 || owner.GetPeakSingleProcessVramGib() != 1.25 {
		t.Fatalf("owner-resource summary = %+v", owner)
	}
}
