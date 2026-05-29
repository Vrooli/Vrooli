package migration

import (
	"architecture-cartographer/internal/migration"

	migrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// statusToProto maps the domain FindingStatus to the wire enum.
func statusToProto(s migration.FindingStatus) migrationv1.TrackedFindingStatus {
	switch s {
	case migration.StatusDetected:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_DETECTED
	case migration.StatusAssigned:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_ASSIGNED
	case migration.StatusSplit:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_SPLIT
	case migration.StatusResolved:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_RESOLVED
	case migration.StatusValidated:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_VALIDATED
	case migration.StatusCommitted:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_COMMITTED
	case migration.StatusForceResolved:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_FORCE_RESOLVED
	default:
		return migrationv1.TrackedFindingStatus_TRACKED_FINDING_STATUS_UNSPECIFIED
	}
}

func lifecycleToProto(s migration.MigrationStatus) migrationv1.MigrationLifecycle {
	switch s {
	case migration.MigrationOpen:
		return migrationv1.MigrationLifecycle_MIGRATION_LIFECYCLE_OPEN
	case migration.MigrationClosed:
		return migrationv1.MigrationLifecycle_MIGRATION_LIFECYCLE_CLOSED
	default:
		return migrationv1.MigrationLifecycle_MIGRATION_LIFECYCLE_UNSPECIFIED
	}
}

func findingToProto(f migration.Finding) *migrationv1.TrackedFinding {
	return &migrationv1.TrackedFinding{
		StableId:       f.StableID,
		Scenario:       f.Scenario,
		Source:         f.Source,
		Code:           f.Code,
		Severity:       f.Severity,
		Locations:      f.Locations,
		Domains:        f.Domains,
		Message:        f.Message,
		Suggestion:     f.Suggestion,
		Status:         statusToProto(f.Status),
		ResolutionNote: f.ResolutionNote,
		Regressed:      f.Regressed,
		FirstSeenAt:    timestamppb.New(f.FirstSeenAt),
		UpdatedAt:      timestamppb.New(f.UpdatedAt),
	}
}

func findingsToProto(in []migration.Finding) []*migrationv1.TrackedFinding {
	out := make([]*migrationv1.TrackedFinding, 0, len(in))
	for _, f := range in {
		out = append(out, findingToProto(f))
	}
	return out
}

func migrationToProto(m migration.Migration) *migrationv1.Migration {
	return &migrationv1.Migration{
		Id:        m.ID,
		Scenario:  m.Scenario,
		Name:      m.Name,
		Status:    lifecycleToProto(m.Status),
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
	}
}

func statusProjectionToProto(st migration.Status) *migrationv1.MigrationStatus {
	return &migrationv1.MigrationStatus{
		Migration:   migrationToProto(st.Migration),
		Findings:    findingsToProto(st.Findings),
		Total:       int32(st.Total),
		Open:        int32(st.Open),
		Resolved:    int32(st.Resolved),
		Validated:   int32(st.Validated),
		Regressions: int32(st.Regressions),
		BySeverity:  intMap(st.BySeverity),
		ByStatus:    intMap(st.ByStatus),
	}
}

func intMap(in map[string]int) map[string]int32 {
	out := make(map[string]int32, len(in))
	for k, v := range in {
		out[k] = int32(v)
	}
	return out
}
