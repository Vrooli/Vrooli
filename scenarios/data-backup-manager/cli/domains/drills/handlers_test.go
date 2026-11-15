package drills

import (
	"strings"
	"testing"
	"time"

	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFormatShowsDurableFailureNotificationState(t *testing.T) {
	sentAt := time.Date(2026, 11, 13, 17, 50, 0, 0, time.UTC)
	got := format(&drillsv1.RecoveryDrill{
		Id: "drill-1", Status: drillsv1.DrillStatus_DRILL_STATUS_FAILED,
		NotificationState: "sent", NotificationSentAt: timestamppb.New(sentAt),
	})
	if !strings.Contains(got, "notification=sent") || !strings.Contains(got, "notification-sent=2026-11-13T17:50:00Z") {
		t.Fatalf("format omitted notification receipt: %s", got)
	}
}

func TestFormatShowsRetirementReceipt(t *testing.T) {
	retiredAt := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	got := format(&drillsv1.RecoveryDrill{Id: "drill-1", Status: drillsv1.DrillStatus_DRILL_STATUS_VERIFIED, RetiredAt: timestamppb.New(retiredAt), RetirementReason: "orphaned plan reference"})
	if !strings.Contains(got, "status=verified") || !strings.Contains(got, "retired=2026-11-14T09:00:00Z") || !strings.Contains(got, `retirement-reason="orphaned plan reference"`) {
		t.Fatalf("format omitted retirement receipt or original result: %s", got)
	}
}
