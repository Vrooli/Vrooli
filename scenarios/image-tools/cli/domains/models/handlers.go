package models

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client modelsconnect.ModelsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: modelsconnect.NewModelsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	op := strings.TrimSpace(ctx.Flag("operation"))
	resp, err := h.client.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{Operation: op}))
	if err != nil {
		return cliapp.WrapAPIError("list models", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no models response")
	}
	results := make([]string, 0, len(resp.Msg.Models))
	for _, m := range resp.Msg.Models {
		results = append(results, formatModel(m))
	}
	summary := fmt.Sprintf("Found %d model(s).", len(resp.Msg.Models))
	if op != "" {
		summary = fmt.Sprintf("Found %d model(s) for %q.", len(resp.Msg.Models), op)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Models",
		Results:        results,
		RetrievalHints: []string{
			"`models get <id>` — show one model in detail",
			"`models select <operation>` — preview which model would run on this host",
			"`models enable <id>` / `models enable <id> --disable` — toggle a model",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetModel(context.Background(), connect.NewRequest(&modelsv1.GetModelRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get model %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Model == nil {
		return fmt.Errorf("server returned no model")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Model %s.", resp.Msg.Model.Id)},
		ResultsHeading: "Model",
		Results:        []string{formatModelDetail(resp.Msg.Model)},
	})
}

func (h *handlers) operations(ctx cliapp.RunContext) error {
	resp, err := h.client.ListOperations(context.Background(), connect.NewRequest(&modelsv1.ListOperationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list operations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no operations response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d operation(s).", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        resp.Msg.Operations,
	})
}

func (h *handlers) selectModel(ctx cliapp.RunContext) error {
	op := ctx.Positional("operation")
	resp, err := h.client.SelectModel(context.Background(), connect.NewRequest(&modelsv1.SelectModelRequest{
		Operation:  op,
		OverrideId: strings.TrimSpace(ctx.Flag("override")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("select model for %q", op), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Model == nil {
		return fmt.Errorf("server returned no selection")
	}
	results := []string{
		formatModel(resp.Msg.Model),
		"reason: " + resp.Msg.Reason,
		fmt.Sprintf("gpu_viable: %t", resp.Msg.GpuViable),
	}
	for _, w := range resp.Msg.Warnings {
		results = append(results, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Selected %s for %q.", resp.Msg.Model.Id, op)},
		ResultsHeading: "Selection",
		Results:        results,
	})
}

func (h *handlers) setEnabled(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	enabled := !ctx.BoolFlag("disable")
	resp, err := h.client.SetModelEnabled(context.Background(), connect.NewRequest(&modelsv1.SetModelEnabledRequest{
		Id:      id,
		Enabled: enabled,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set model %q enabled=%t", id, enabled), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Model == nil {
		return fmt.Errorf("server returned no model")
	}
	verb := "Enabled"
	if !enabled {
		verb = "Disabled"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s model %s.", verb, resp.Msg.Model.Id)},
		Changes: []string{formatModel(resp.Msg.Model)},
		NextCommand: []string{
			fmt.Sprintf("`models get %s` — confirm the new state", resp.Msg.Model.Id),
		},
	})
}

func (h *handlers) blocklist(ctx cliapp.RunContext) error {
	resp, err := h.client.ListBlocklist(context.Background(), connect.NewRequest(&modelsv1.ListBlocklistRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list blocklist", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no blocklist response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, b := range resp.Msg.Entries {
		results = append(results, fmt.Sprintf("%s — %s (%s)", b.Id, b.Reason, b.License))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d blocklisted model(s).", len(resp.Msg.Entries))},
		ResultsHeading: "Blocklist",
		Results:        results,
	})
}

func formatModel(m *modelsv1.Model) string {
	if m == nil {
		return "(nil)"
	}
	state := "disabled"
	if m.Enabled {
		state = "enabled"
	}
	return fmt.Sprintf("%s — %s [tier=%s backend=%s ops=%s %s]",
		m.Id, m.Name, m.Tier, m.Backend, strings.Join(m.Operations, ","), state)
}

func formatModelDetail(m *modelsv1.Model) string {
	if m == nil {
		return "(nil)"
	}
	lines := []string{formatModel(m)}
	if m.Hardware != nil {
		lines = append(lines, fmt.Sprintf("  hardware: cpu_capable=%t gpu_required=%t min_vram_gb=%d min_ram_gb=%d",
			m.Hardware.CpuCapable, m.Hardware.GpuRequired, m.Hardware.MinVramGb, m.Hardware.MinRamGb))
	}
	if l := m.CapabilityLabels; l != nil {
		lines = append(lines, fmt.Sprintf("  license: %s (commercial_use=%s) nsfw_capable=%t",
			l.License, commercialUseLabel(l.CommercialUse), l.NsfwCapable))
	}
	if len(m.DefaultFor) > 0 {
		lines = append(lines, "  default_for: "+strings.Join(m.DefaultFor, ","))
	}
	return strings.Join(lines, "\n")
}

func commercialUseLabel(c modelsv1.CommercialUse) string {
	switch c {
	case modelsv1.CommercialUse_COMMERCIAL_USE_YES:
		return "yes"
	case modelsv1.CommercialUse_COMMERCIAL_USE_NO:
		return "no"
	case modelsv1.CommercialUse_COMMERCIAL_USE_CONDITIONAL:
		return "conditional"
	default:
		return "unspecified"
	}
}
