package drills

import (
	"context"
	"testing"
	"time"

	d "data-backup-manager/internal/drills"

	"connectrpc.com/connect"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
)

type fakeService struct {
	preview d.Preview
	drill   d.Drill
	list    []d.Drill
	page    d.DrillPage
	err     error
}

func TestRecoveryDrillMapsNotificationDeliveryState(t *testing.T) {
	sentAt := time.Date(2026, 11, 13, 17, 50, 0, 0, time.UTC)
	got := toProto(d.Drill{ID: "drill-1", Status: d.StatusFailed, NotificationState: d.NotificationSent, NotificationSentAt: sentAt})
	if got.NotificationState != "sent" || got.NotificationSentAt == nil || !got.NotificationSentAt.AsTime().Equal(sentAt) {
		t.Fatalf("notification receipt mapping = %+v", got)
	}
}

func (f *fakeService) Preview(context.Context, string, string, string) (d.Preview, error) {
	return f.preview, f.err
}

func (f *fakeService) Run(context.Context, string, string, string, string, bool) (d.Drill, error) {
	return f.drill, f.err
}
func (f *fakeService) RunMigrationSnapshot(context.Context, string, string, string, string, string) (d.Drill, error) {
	return f.drill, f.err
}
func (f *fakeService) Get(context.Context, string) (d.Drill, error)            { return f.drill, f.err }
func (f *fakeService) Retire(context.Context, string, string) (d.Drill, error) { return f.drill, f.err }
func (f *fakeService) List(context.Context, string, string, int) ([]d.Drill, error) {
	return f.list, f.err
}
func (f *fakeService) ListPage(context.Context, string, string, int, string) (d.DrillPage, error) {
	if f.page.Drills != nil || f.page.NextPageToken != "" {
		return f.page, f.err
	}
	return d.DrillPage{Drills: f.list}, f.err
}
func (f *fakeService) RunDue(context.Context) error    { return f.err }
func (f *fakeService) Reconcile(context.Context) error { return f.err }
func (f *fakeService) Shutdown(context.Context) error  { return f.err }

func TestRecoveryDrillsServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := &fakeService{
		preview: d.Preview{Eligible: true, PlanID: "p1", TargetID: "t1", DestinationID: "d1", SnapshotID: "s1", Warnings: []string{"manual"}},
		drill:   d.Drill{ID: "drill-1", PlanID: "p1", TargetID: "t1", DestinationID: "d1", SnapshotID: "s1", Status: d.StatusVerified},
		page:    d.DrillPage{Drills: []d.Drill{{ID: "drill-1", Status: d.StatusVerified}, {ID: "drill-2", Status: d.StatusFailed}}, NextPageToken: "next"},
	}
	h := NewConnectHandler(Deps{Service: svc})

	preview, err := h.PreviewDrill(ctx, connect.NewRequest(&drillsv1.PreviewDrillRequest{PlanId: "p1"}))
	if err != nil || !preview.Msg.Eligible || preview.Msg.SnapshotId != "s1" {
		t.Fatalf("preview = %#v, err=%v", preview, err)
	}
	run, err := h.RunDrill(ctx, connect.NewRequest(&drillsv1.RunDrillRequest{PlanId: "p1"}))
	if err != nil || run.Msg.Drill.Status != drillsv1.DrillStatus_DRILL_STATUS_VERIFIED {
		t.Fatalf("run = %#v, err=%v", run, err)
	}
	got, err := h.GetDrill(ctx, connect.NewRequest(&drillsv1.GetDrillRequest{Id: "drill-1"}))
	if err != nil || got.Msg.Drill.Id != "drill-1" {
		t.Fatalf("get = %#v, err=%v", got, err)
	}
	retiredAt := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	svc.drill.RetiredAt = retiredAt
	svc.drill.RetirementReason = "orphaned plan reference"
	retired, err := h.RetireDrill(ctx, connect.NewRequest(&drillsv1.RetireDrillRequest{Id: "drill-1", Reason: "orphaned plan reference"}))
	if err != nil || retired.Msg.Drill.Status != drillsv1.DrillStatus_DRILL_STATUS_VERIFIED || retired.Msg.Drill.RetiredAt == nil || retired.Msg.Drill.RetirementReason != "orphaned plan reference" {
		t.Fatalf("retire = %#v, err=%v", retired, err)
	}
	listed, err := h.ListDrills(ctx, connect.NewRequest(&drillsv1.ListDrillsRequest{}))
	if err != nil || len(listed.Msg.Drills) != 2 || listed.Msg.NextPageToken != "next" {
		t.Fatalf("list = %#v, err=%v", listed, err)
	}
}

func TestRecoveryDrillsServiceErrorsAreTyped(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{err: d.ErrInvalid{Field: "plan_id", Reason: "required"}}})
	_, err := h.PreviewDrill(context.Background(), connect.NewRequest(&drillsv1.PreviewDrillRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid argument", connect.CodeOf(err))
	}
}
