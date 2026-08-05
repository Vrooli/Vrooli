package drills

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	d "data-backup-manager/internal/drills"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
)

type fakeService struct {
	preview d.Preview
	drill   d.Drill
	list    []d.Drill
	err     error
}

func (f *fakeService) Preview(context.Context, string, string, string) (d.Preview, error) {
	return f.preview, f.err
}

func (f *fakeService) Run(context.Context, string, string, string, string, bool) (d.Drill, error) {
	return f.drill, f.err
}
func (f *fakeService) Get(context.Context, string) (d.Drill, error) { return f.drill, f.err }
func (f *fakeService) List(context.Context, string, string, int) ([]d.Drill, error) {
	return f.list, f.err
}
func (f *fakeService) RunDue(context.Context) error    { return f.err }
func (f *fakeService) Reconcile(context.Context) error { return f.err }
func (f *fakeService) Shutdown(context.Context) error  { return f.err }

func TestRecoveryDrillsServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := &fakeService{
		preview: d.Preview{Eligible: true, PlanID: "p1", TargetID: "t1", DestinationID: "d1", SnapshotID: "s1", Warnings: []string{"manual"}},
		drill:   d.Drill{ID: "drill-1", PlanID: "p1", TargetID: "t1", DestinationID: "d1", SnapshotID: "s1", Status: d.StatusVerified},
		list:    []d.Drill{{ID: "drill-1", Status: d.StatusVerified}, {ID: "drill-2", Status: d.StatusFailed}},
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
	listed, err := h.ListDrills(ctx, connect.NewRequest(&drillsv1.ListDrillsRequest{}))
	if err != nil || len(listed.Msg.Drills) != 2 {
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
