package convert

import (
	"time"

	maintenancepb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// DatabaseStatsToProto converts repository DB stats to the API message.
func DatabaseStatsToProto(s repository.DatabaseStats) *maintenancepb.DatabaseStats {
	return &maintenancepb.DatabaseStats{
		PageSize:      s.PageSize,
		PageCount:     s.PageCount,
		FreelistCount: s.FreelistCount,
		SizeBytes:     s.SizeBytes,
		MetricRows:    s.MetricRows,
	}
}

// RetentionEstimateToProto converts a retention estimate to the API message.
func RetentionEstimateToProto(e repository.RetentionEstimate) *maintenancepb.RetentionEstimate {
	return &maintenancepb.RetentionEstimate{
		RowCount:       e.RowCount,
		PayloadBytes:   e.PayloadBytes,
		OldestAffected: formatRFC3339(e.OldestAffected),
		NewestAffected: formatRFC3339(e.NewestAffected),
		Cutoff:         formatRFC3339(e.Cutoff),
	}
}

// RetentionResultToProto converts a retention result to the API message.
func RetentionResultToProto(r repository.RetentionResult) *maintenancepb.RetentionResult {
	return &maintenancepb.RetentionResult{
		DeletedRows: r.DeletedRows,
		Cutoff:      formatRFC3339(r.Cutoff),
	}
}
