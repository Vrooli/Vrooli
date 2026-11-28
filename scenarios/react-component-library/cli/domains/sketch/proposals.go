package sketch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

func (h *handlers) proposalInput(ctx cliapp.OperationContext) (*sharedv1.ProposeSketchRequest, error) {
	target, err := parseTarget(ctx.Positional("target"))
	if err != nil {
		return nil, err
	}
	limit, err := strconv.Atoi(ctx.Flag("candidate-limit"))
	if err != nil || limit < 1 || limit > 3 {
		return nil, fmt.Errorf("candidate-limit must be between one and three")
	}
	clean := func(values []string) []string {
		out := []string{}
		for _, v := range values {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	users, tasks := clean(ctx.FlagValues("user")), clean(ctx.FlagValues("task"))
	intent := strings.TrimSpace(ctx.Flag("intent"))
	if intent == "" || len(users) == 0 || len(tasks) == 0 {
		return nil, fmt.Errorf("intent, at least one user and at least one task are required")
	}
	viewports := clean(ctx.FlagValues("viewport"))
	if len(viewports) == 0 {
		viewports = []string{"phone", "desktop"}
	}
	for _, v := range viewports {
		if v != "phone" && v != "tablet" && v != "desktop" {
			return nil, fmt.Errorf("unsupported viewport %q", v)
		}
	}
	hash, err := h.currentHash(target)
	if err != nil {
		return nil, err
	}
	preserveRoutes, preserveBehavior := !ctx.BoolFlag("allow-route-changes"), !ctx.BoolFlag("allow-behavior-changes")
	// limit is bounded to [1, 3] above, so this protobuf-width conversion cannot overflow.
	return &sharedv1.ProposeSketchRequest{Target: target, ExpectedContentHash: hash, CandidateLimit: int32(limit), // #nosec G109 -- limit was validated to [1, 3] above.
		Intent: &sharedv1.DesignIntent{
			Intent: intent, Users: users, PrimaryTasks: tasks, Target: ctx.Flag("platform"), Kit: ctx.Flag("kit"), Viewports: viewports,
			Constraints: &sharedv1.DesignConstraints{DesignSource: strings.TrimSpace(ctx.Flag("design-source")), PreserveRoutes: &preserveRoutes, PreserveBusinessBehavior: &preserveBehavior},
		}}, nil
}

func proposalLines(result *sharedv1.ProposeSketchResponse) []string {
	lines := []string{fmt.Sprintf("Alternatives: %d; mode: %s", len(result.Candidates), result.RetrievalMode)}
	for i, c := range result.Candidates {
		ref := c.GetSketch().GetTemplate()
		lines = append(lines, fmt.Sprintf("%d. %s · %s@%s", i+1, c.Title, ref.GetAsset(), ref.GetVersion()))
		for _, obligation := range c.Obligations {
			lines = append(lines, "   "+obligation)
		}
	}
	return append(lines, result.Diagnostics...)
}

func (h *handlers) proposeCall(ctx cliapp.OperationContext) (*sharedv1.ProposeSketchResponse, error) {
	request, err := h.proposalInput(ctx)
	if err != nil {
		return nil, err
	}
	response, err := h.assistClient.ProposeSketch(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("propose design", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) inferCall(ctx cliapp.OperationContext) (*sketchv1.SketchInferenceOperation, error) {
	key := ctx.Flag("key")
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return nil, fmt.Errorf("key must contain 1 to 200 bytes")
	}
	request, err := h.proposalInput(ctx)
	if err != nil {
		return nil, err
	}
	response, err := h.inferenceClient.InferSketch(context.Background(), connect.NewRequest(&sketchv1.InferSketchRequest{Proposal: request, IdempotencyKey: key}))
	if err != nil {
		return nil, fmt.Errorf("%w; recover with sketch inference --key using the same key before considering another attempt", cliapp.WrapAPIError("infer design", err, nil))
	}
	return response.Msg, nil
}

func (h *handlers) inferenceCall(ctx cliapp.OperationContext) (*sketchv1.SketchInferenceOperation, error) {
	id, key := ctx.Flag("id"), ctx.Flag("key")
	if (id == "") == (key == "") {
		return nil, fmt.Errorf("supply exactly one of --id or --key")
	}
	response, err := h.assistClient.GetSketchInference(context.Background(), connect.NewRequest(&sketchv1.GetSketchInferenceRequest{Id: id, IdempotencyKey: key}))
	if err != nil {
		return nil, cliapp.WrapAPIError("recover design inference", err, nil)
	}
	return response.Msg, nil
}

func proposeHuman(ctx cliapp.RunContext, result *sharedv1.ProposeSketchResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: proposalLines(result)})
}

func inferenceHuman(ctx cliapp.RunContext, result *sketchv1.SketchInferenceOperation) error {
	lines := []string{fmt.Sprintf("Inference: %s; state: %s", result.Id, result.State)}
	if result.Detail != "" {
		lines = append(lines, result.Detail)
	}
	if result.Proposal != nil {
		lines = append(lines, proposalLines(result.Proposal)...)
	}
	return ctx.RenderList(cliapp.ListReport{Summary: lines})
}
