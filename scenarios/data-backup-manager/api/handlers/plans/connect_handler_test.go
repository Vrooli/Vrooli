package plans

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/plans/mocks"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"
)

// TestPlansService_Contract exercises every PlansService RPC against the
// handler backed by a fake service and asserts request→domain and
// domain→response translation including typed error codes.
func TestPlansService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("CreatePlan maps fields and returns wire plan", func(t *testing.T) {
		svc := &mocks.FakeService{CreateOut: plans.Plan{
			ID:             "plan-1",
			Name:           "nightly",
			TargetIDs:      []string{"tgt-1", "tgt-2"},
			DestinationIDs: []string{"dst-a"},
			Schedule:       "24h",
			KeepLatest:     7,
			Enabled:        true,
			CreatedAt:      time.Unix(1700000000, 0).UTC(),
			UpdatedAt:      time.Unix(1700000000, 0).UTC(),
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.CreatePlan(ctx, connect.NewRequest(&plansv1.CreatePlanRequest{
			Name:           "nightly",
			TargetIds:      []string{"tgt-1", "tgt-2"},
			DestinationIds: []string{"dst-a"},
			Schedule:       "24h",
			Retention:      &plansv1.RetentionPolicy{KeepLatest: 7},
		}))
		if err != nil {
			t.Fatalf("CreatePlan: %v", err)
		}
		if len(svc.CreateInputs) != 1 {
			t.Fatalf("service not called once: %d", len(svc.CreateInputs))
		}
		if svc.CreateInputs[0].Name != "nightly" {
			t.Fatalf("name not passed: %q", svc.CreateInputs[0].Name)
		}
		if svc.CreateInputs[0].KeepLatest != 7 {
			t.Fatalf("keep_latest not passed: %d", svc.CreateInputs[0].KeepLatest)
		}
		got := resp.Msg.Plan
		if got.Id != "plan-1" || got.Name != "nightly" {
			t.Fatalf("response plan wrong: %+v", got)
		}
		if len(got.TargetIds) != 2 || len(got.DestinationIds) != 1 {
			t.Fatalf("membership lists wrong: targets=%v dests=%v", got.TargetIds, got.DestinationIds)
		}
		if got.Retention == nil || got.Retention.KeepLatest != 7 {
			t.Fatalf("retention wrong: %+v", got.Retention)
		}
		if got.CreatedAt == nil || got.UpdatedAt == nil {
			t.Fatal("timestamps missing from response")
		}
	})

	t.Run("CreatePlan surfaces invalid-argument", func(t *testing.T) {
		svc := &mocks.FakeService{CreateErr: plans.ErrInvalidPlan{Field: "name", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.CreatePlan(ctx, connect.NewRequest(&plansv1.CreatePlanRequest{}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
		}
	})

	t.Run("GetPlan returns plan", func(t *testing.T) {
		svc := &mocks.FakeService{GetOut: plans.Plan{
			ID: "plan-1", Name: "nightly",
			TargetIDs: []string{"tgt-1"}, DestinationIDs: []string{"dst-a"},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.GetPlan(ctx, connect.NewRequest(&plansv1.GetPlanRequest{Id: "plan-1"}))
		if err != nil {
			t.Fatalf("GetPlan: %v", err)
		}
		if svc.GetID != "plan-1" {
			t.Fatalf("id not passed: %q", svc.GetID)
		}
		if resp.Msg.Plan.Id != "plan-1" {
			t.Fatalf("response id wrong: %q", resp.Msg.Plan.Id)
		}
	})

	t.Run("GetPlan surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: plans.ErrPlanNotFound{ID: "missing"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetPlan(ctx, connect.NewRequest(&plansv1.GetPlanRequest{Id: "missing"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("ListPlans maps the collection", func(t *testing.T) {
		svc := &mocks.FakeService{ListOut: []plans.Plan{
			{ID: "a", Name: "alpha", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}, Enabled: true},
			{ID: "b", Name: "beta", TargetIDs: []string{"t2"}, DestinationIDs: []string{"d2"}, Enabled: false},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListPlans(ctx, connect.NewRequest(&plansv1.ListPlansRequest{}))
		if err != nil {
			t.Fatalf("ListPlans: %v", err)
		}
		if len(resp.Msg.Plans) != 2 {
			t.Fatalf("list len = %d, want 2", len(resp.Msg.Plans))
		}
		if resp.Msg.Plans[0].Id != "a" || resp.Msg.Plans[1].Id != "b" {
			t.Fatalf("plan ids wrong: %v", resp.Msg.Plans)
		}
	})

	t.Run("UpdatePlan maps fields", func(t *testing.T) {
		svc := &mocks.FakeService{UpdateOut: plans.Plan{
			ID: "plan-1", Name: "updated", TargetIDs: []string{"tgt-x"}, DestinationIDs: []string{"dst-y"},
			Enabled: false,
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.UpdatePlan(ctx, connect.NewRequest(&plansv1.UpdatePlanRequest{
			Id: "plan-1", Name: "updated",
			TargetIds: []string{"tgt-x"}, DestinationIds: []string{"dst-y"},
			Enabled: false,
		}))
		if err != nil {
			t.Fatalf("UpdatePlan: %v", err)
		}
		if len(svc.UpdateInputs) != 1 || svc.UpdateInputs[0].ID != "plan-1" {
			t.Fatalf("update input wrong: %+v", svc.UpdateInputs)
		}
		if resp.Msg.Plan.Name != "updated" {
			t.Fatalf("response name wrong: %q", resp.Msg.Plan.Name)
		}
	})

	t.Run("UpdatePlan surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{UpdateErr: plans.ErrPlanNotFound{ID: "missing"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.UpdatePlan(ctx, connect.NewRequest(&plansv1.UpdatePlanRequest{Id: "missing", Name: "x", TargetIds: []string{"t"}, DestinationIds: []string{"d"}}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("DeletePlan returns removed flag", func(t *testing.T) {
		svc := &mocks.FakeService{DeleteOut: true}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.DeletePlan(ctx, connect.NewRequest(&plansv1.DeletePlanRequest{Id: "plan-1"}))
		if err != nil {
			t.Fatalf("DeletePlan: %v", err)
		}
		if !resp.Msg.Removed {
			t.Fatal("removed = false, want true")
		}
		if svc.DeleteID != "plan-1" {
			t.Fatalf("delete id wrong: %q", svc.DeleteID)
		}
	})
}
