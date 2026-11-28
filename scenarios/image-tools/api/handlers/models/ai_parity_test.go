package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	aiH "image-tools/handlers/ai"

	internalcapability "image-tools/internal/capability"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// TestAIListDefaultsAreWhatModelsSelectReturns pins `ai list` to the same
// selection code as `models select`. The default column used to read the seed's
// static default_for, so it named instruct-pix2pix and sd-1.5 and left upscale
// blank while the live selection was flux-2-klein-4b and real-esrgan. Every
// operation must agree, including one with an operator pin, one only a model
// override enables, and operations nothing can serve.
func TestAIListDefaultsAreWhatModelsSelectReturns(t *testing.T) {
	ctx := context.Background()
	h, svc, _ := newVerdictHandler(t, gpuTestHost)
	// The live situation: upscale has no seed default and real-esrgan is enabled
	// and pinned by the operator.
	if _, err := h.SetModelEnabled(ctx, connect.NewRequest(&modelsv1.SetModelEnabledRequest{Id: "real-esrgan", Enabled: true, Actor: "test"})); err != nil {
		t.Fatal(err)
	}
	if err := h.deps.OpDefaults.Set(ctx, "upscale", "real-esrgan"); err != nil {
		t.Fatal(err)
	}

	discovery := aiH.NewConnectHandler(h.previewer(), svc, nil)
	list, err := discovery.ListAIOperations(ctx, connect.NewRequest(&aiv1.ListAIOperationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.Operations) == 0 {
		t.Fatal("ai list returned no operations")
	}
	var withDefault, without int
	for _, op := range list.Msg.Operations {
		sel, selErr := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: op.Name}))
		if selErr != nil {
			without++
			if op.DefaultModelId != "" || op.DefaultUnavailable == "" || op.DefaultVerdict != nil {
				t.Errorf("%s: models select fails (%v) but ai list shows default %q unavailable=%q verdict=%v", op.Name, selErr, op.DefaultModelId, op.DefaultUnavailable, op.DefaultVerdict)
			}
			continue
		}
		withDefault++
		if op.DefaultModelId != sel.Msg.Model.Id {
			t.Errorf("%s: ai list default %q, models select %q", op.Name, op.DefaultModelId, sel.Msg.Model.Id)
		}
		if op.DefaultUnavailable != "" {
			t.Errorf("%s: has a default but reports unavailable %q", op.Name, op.DefaultUnavailable)
		}
		expectedVerdictOp := op.Name
		if runs, _, _, ok := internalcapability.PresetTarget(op.Name); ok {
			expectedVerdictOp = runs
		}
		if op.DefaultVerdict.GetOperation() != expectedVerdictOp || op.DefaultVerdict.GetGrade() != sel.Msg.Verdict.GetGrade() {
			t.Errorf("%s: ai list verdict %v, models select verdict %v", op.Name, op.DefaultVerdict, sel.Msg.Verdict)
		}
	}
	if withDefault == 0 || without == 0 {
		t.Fatalf("the check must cover both selectable and unselectable operations: %d with a default, %d without", withDefault, without)
	}
	for _, op := range list.Msg.Operations {
		switch op.Name {
		case "upscale":
			if op.DefaultModelId != "real-esrgan" {
				t.Errorf("upscale default = %q, want the pinned real-esrgan", op.DefaultModelId)
			}
		case "edit_instruct":
			if op.DefaultModelId == "instruct-pix2pix" {
				t.Errorf("edit_instruct still shows the stale seed default instruct-pix2pix")
			}
		}
	}
}
