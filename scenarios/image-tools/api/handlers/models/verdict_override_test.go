package models

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"image-tools/internal/capabilities"
	internalmeasures "image-tools/internal/measures"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/verdicts"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// newVerdictHandler is a handler with the verdict service, job traces and op
// pins wired, over the real seed.
func newVerdictHandler(t *testing.T, host capabilities.Host) (*connectHandler, *verdicts.Service, *sql.DB) {
	t.Helper()
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatal(err)
	}
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(internalmodels.Schema), apidb.SchemaProviderFunc(internalmeasures.Schema), apidb.SchemaProviderFunc(verdicts.Schema)); err != nil {
		t.Fatal(err)
	}
	svc, err := verdicts.NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	return NewConnectHandler(Deps{
		Registry:        reg,
		Store:           internalmodels.NewStore(d),
		Verdicts:        svc,
		OpDefaults:      internalmodels.NewOpDefaultStore(d),
		Probe:           capabilities.FakeProbe{Host: host},
		Backends:        testBackendRegistry(t),
		RightsEvaluator: allowRightsForHandlerTest,
		Now:             func() time.Time { return time.Date(2026, 11, 22, 9, 0, 0, 0, time.UTC) },
	}), svc, d
}

func verdictFor(m *modelsv1.Model, op string) *modelsv1.Verdict {
	for _, v := range m.GetVerdicts() {
		if v.GetOperation() == op {
			return v
		}
	}
	return nil
}

// TestEveryModelCarriesAVerdictForEveryOperation proves the wire contract: a
// verdict per operation in the model's own order, a roll-up, evidence pointers on
// every graded verdict, and "untested" (never missing) for a model with none.
func TestEveryModelCarriesAVerdictForEveryOperation(t *testing.T) {
	h, _, _ := newVerdictHandler(t, cpuOnlyHost)
	resp, err := h.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*modelsv1.Model{}
	for _, m := range resp.Msg.Models {
		byID[m.Id] = m
		if len(m.Verdicts) != len(m.Operations) {
			t.Fatalf("%s: %d verdicts for %d operations", m.Id, len(m.Verdicts), len(m.Operations))
		}
		for i, v := range m.Verdicts {
			if v.Operation != m.Operations[i] {
				t.Fatalf("%s: verdict %d is for %q, want %q", m.Id, i, v.Operation, m.Operations[i])
			}
			if v.Grade == modelsv1.VerdictGrade_VERDICT_GRADE_UNSPECIFIED || strings.TrimSpace(v.Reason) == "" {
				t.Fatalf("%s/%s: verdict %+v has no grade or reason", m.Id, v.Operation, v)
			}
			if v.Grade != modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED && len(v.Evidence) == 0 {
				t.Fatalf("%s/%s: graded %s with no evidence pointer", m.Id, v.Operation, v.Grade)
			}
		}
		if m.Verdict == nil {
			t.Fatalf("%s has no roll-up verdict", m.Id)
		}
	}
	for id, want := range map[string]modelsv1.VerdictGrade{
		"flux-2-klein-4b":         modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED,
		"restoreformer-plus-plus": modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED,
		"instruct-pix2pix":        modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED,
		"sd-1.5":                  modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED,
		"codeformer":              modelsv1.VerdictGrade_VERDICT_GRADE_WORKS,
		"tesseract":               modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED,
	} {
		if got := byID[id].GetVerdict().GetGrade(); got != want {
			t.Errorf("%s roll-up = %s (%s), want %s", id, got, byID[id].GetVerdict().GetReason(), want)
		}
	}
}

// TestOperatorRatingsMoveTheVerdictsEveryHandlerServes closes the loop through
// the handler: ratings written for a model's jobs change the grade that
// ListModels, GetModel, SelectModel and the picker all report.
func TestOperatorRatingsMoveTheVerdictsEveryHandlerServes(t *testing.T) {
	ctx := context.Background()
	h, svc, d := newVerdictHandler(t, gpuTestHost)
	// Three succeeded depth jobs by V2 Small, rated 1 star: the curated "works"
	// is withdrawn by the operator.
	for i, id := range []string{"j1", "j2", "j3"} {
		if _, err := d.ExecContext(ctx,
			`INSERT INTO job_trace (job_id, operation, model_id, state, recorded_at) VALUES (?,?,?,?,?)`,
			id, "depth_map", "depth-anything-v2-small", "succeeded", time.Date(2026, 11, 22, 0, 0, i, 0, time.UTC).Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Ratings.Rate(ctx, id, 1, "blobby", "operator"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: "depth-anything-v2-small"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := verdictFor(got.Msg.Model, "depth_map"); v.Grade != modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED || !strings.Contains(v.Reason, "1.0 of 5") {
		t.Fatalf("GetModel verdict = %+v", v)
	}
	picker, err := h.ListOperationModels(ctx, connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "depth_map"}))
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, c := range picker.Msg.Candidates {
		if c.Model.Id == "depth-anything-v2-small" {
			seen = true
			if c.Verdict.GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED {
				t.Fatalf("picker verdict = %+v", c.Verdict)
			}
		}
		if c.Verdict == nil || c.Verdict.Operation != "depth_map" {
			t.Fatalf("picker candidate %s verdict = %+v, want one for depth_map", c.Model.Id, c.Verdict)
		}
	}
	if !seen {
		t.Fatal("depth-anything-v2-small missing from the picker")
	}
	sel, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: "depth_map", OverrideId: "depth-anything-v2-small"}))
	if err != nil {
		t.Fatal(err)
	}
	if sel.Msg.Verdict.GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED {
		t.Fatalf("SelectModel verdict = %+v", sel.Msg.Verdict)
	}
	// ExplainResolution carries the verdict of the model it resolves (naturalize
	// has a builtin provider in the test registry and no evidence yet).
	explain, err := h.ExplainResolution(ctx, connect.NewRequest(&modelsv1.ExplainResolutionRequest{Operation: "naturalize"}))
	if err != nil {
		t.Fatal(err)
	}
	if v := explain.Msg.Resolution.Verdict; v.GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED || v.GetOperation() != "naturalize" {
		t.Fatalf("ExplainResolution verdict = %+v", v)
	}
}

// TestPickerSelectedHonoursTheOperatorPin pins the picker to the same selection
// code as `models select`: with a pinned default, the picker's selected marker
// names the pinned model, as SelectModel does.
func TestPickerSelectedHonoursTheOperatorPin(t *testing.T) {
	ctx := context.Background()
	h, _, _ := newVerdictHandler(t, gpuTestHost)
	if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "real-esrgan", Enabled: true, Actor: "test"})); err != nil {
		t.Fatal(err)
	}
	if err := h.deps.OpDefaults.Set(ctx, "upscale", "real-esrgan"); err != nil {
		t.Fatal(err)
	}
	sel, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: "upscale"}))
	if err != nil {
		t.Fatal(err)
	}
	picker, err := h.ListOperationModels(ctx, connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "upscale"}))
	if err != nil {
		t.Fatal(err)
	}
	if sel.Msg.Model.Id != "real-esrgan" || picker.Msg.SelectedId != sel.Msg.Model.Id {
		t.Fatalf("select = %s, picker selected = %s, want both real-esrgan", sel.Msg.Model.Id, picker.Msg.SelectedId)
	}
}

// TestSetModelEnabledRecordsProvenance proves who/why/pinned are stored and
// served, a pin needs a reason, and clearing returns the model to the seed.
func TestSetModelEnabledRecordsProvenance(t *testing.T) {
	ctx := context.Background()
	h, _, _ := newVerdictHandler(t, cpuOnlyHost)

	if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "restoreformer-plus-plus", Enabled: false, Pinned: true})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a pin with no reason: %v", err)
	}
	resp, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{
		Id: "restoreformer-plus-plus", Enabled: false, Actor: "agent-7", Reason: "judge flagged skin artifacts",
	}))
	if err != nil {
		t.Fatal(err)
	}
	o := resp.Msg.Model.Override
	if o == nil || o.SetBy != "agent-7" || o.Reason != "judge flagged skin artifacts" || o.SetAt != "2026-11-22T09:00:00Z" || o.Pinned {
		t.Fatalf("override = %+v", o)
	}
	if resp.Msg.Model.Enabled || o.Relation != "contradicts_seed" || !o.SeedEnabled || !strings.Contains(o.Note, "until the seed's flag changes") {
		t.Fatalf("a disable of a seed-enabled model must be shown as contradicting the seed: enabled=%t %+v", resp.Msg.Model.Enabled, o)
	}

	// Same state through GetModel, and an unattributed call is recorded as "api".
	if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "codeformer", Enabled: false})); err != nil {
		t.Fatal(err)
	}
	got, _ := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: "codeformer"}))
	if got.Msg.Model.Override.GetSetBy() != "api" {
		t.Fatalf("default actor = %q", got.Msg.Model.Override.GetSetBy())
	}

	cleared, err := h.ClearModelOverride(ctx, connect.NewRequest(&modelsv1.ClearModelOverrideRequest{Id: "restoreformer-plus-plus"}))
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Msg.Model.Override != nil || !cleared.Msg.Model.Enabled {
		t.Fatalf("after clear: enabled=%t override=%+v, want the seed's enabled model with no override", cleared.Msg.Model.Enabled, cleared.Msg.Model.Override)
	}
	if _, err := h.ClearModelOverride(ctx, connect.NewRequest(&modelsv1.ClearModelOverrideRequest{Id: "restoreformer-plus-plus"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("second clear: %v", err)
	}
}

// TestDisabledModelsSayWhyInTheirVerdicts proves a switched-off seed row is never
// a bare "untested": its recorded reason is the verdict reason, a runtime reason
// reads broken, and the reason goes away when the operator enables the model.
func TestDisabledModelsSayWhyInTheirVerdicts(t *testing.T) {
	ctx := context.Background()
	h, _, _ := newVerdictHandler(t, cpuOnlyHost)
	get := func(id string) *modelsv1.Model {
		t.Helper()
		resp, err := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: id}))
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		return resp.Msg.GetModel()
	}

	broken := get("dncnn")
	if v := verdictFor(broken, "denoise"); v.GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_BROKEN || !strings.HasPrefix(v.GetReason(), "disabled: ") || !strings.Contains(v.GetReason(), ".onnx") {
		t.Fatalf("dncnn (runtime hold) verdict = %+v", v)
	}
	held := get("real-cugan")
	if v := verdictFor(held, "upscale"); v.GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED || !strings.HasPrefix(v.GetReason(), "disabled: ") {
		t.Fatalf("real-cugan (needs_install hold) verdict = %+v", v)
	}
	if ev := verdictFor(held, "upscale").GetEvidence(); len(ev) == 0 || ev[len(ev)-1].GetKind() != verdicts.KindCatalogHold {
		t.Fatalf("hold evidence = %+v", ev)
	}

	// The operator turns it on: the seed's reason no longer describes the model.
	if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "real-cugan", Enabled: true, Reason: "test"})); err != nil {
		t.Fatalf("enable real-cugan: %v", err)
	}
	if v := verdictFor(get("real-cugan"), "upscale"); strings.HasPrefix(v.GetReason(), "disabled: ") {
		t.Fatalf("enabled real-cugan still reads %q", v.GetReason())
	}
}
