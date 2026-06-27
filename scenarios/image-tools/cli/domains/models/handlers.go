package models

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
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

// explain mirrors ModelsService.ExplainResolution: the read-only `--explain`
// surface that prints which model/technique would run for an operation
// (native-vs-derived, backend tier, safety weight) without executing.
func (h *handlers) explain(ctx cliapp.RunContext) error {
	op := ctx.Positional("operation")
	resp, err := h.client.ExplainResolution(context.Background(), connect.NewRequest(&modelsv1.ExplainResolutionRequest{
		Operation: op,
		ModelId:   strings.TrimSpace(ctx.Flag("model")),
		AllowByok: ctx.BoolFlag("byok"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("explain resolution for %q", op), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Resolution == nil {
		return fmt.Errorf("server returned no resolution")
	}
	r := resp.Msg.Resolution
	results := []string{
		fmt.Sprintf("model: %s (%s)", r.ModelId, r.ModelName),
		"support: " + r.Support,
	}
	if r.Technique != "" {
		results = append(results, "technique: "+r.Technique)
	}
	if r.PipelineClass != "" {
		results = append(results, "pipeline_class: "+r.PipelineClass)
	}
	if r.Tier != "" {
		results = append(results, "tier: "+r.Tier)
	}
	results = append(results,
		fmt.Sprintf("gpu_viable: %t", r.GpuViable),
		"safety_weight: "+r.Weight,
	)
	if r.Caveat != "" {
		results = append(results, "caveat: "+r.Caveat)
	}
	for _, w := range r.Warnings {
		results = append(results, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s would run %q as a %s op.", r.ModelId, op, r.Support)},
		ResultsHeading: "Resolution",
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

func (h *handlers) doctor(ctx cliapp.RunContext) error {
	resp, err := h.client.DoctorCatalog(context.Background(), connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("doctor model catalog", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no doctor response")
	}
	results := make([]string, 0, len(resp.Msg.Findings))
	for _, f := range resp.Msg.Findings {
		results = append(results, formatFinding(f))
	}
	if len(results) == 0 {
		results = append(results, "no findings")
	}
	status := "Catalog doctor passed."
	if !resp.Msg.Ok {
		status = fmt.Sprintf("Catalog doctor found %d issue(s).", len(resp.Msg.Findings))
	}
	if err := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{status},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			"Enabled weight-backed seed models need direct `source.assets[]` entries with `kind` and `min_bytes`.",
			"`models list` — inspect effective enabled state",
		},
	}); err != nil {
		return err
	}
	if !resp.Msg.Ok {
		return fmt.Errorf("model catalog doctor failed")
	}
	return nil
}

func (h *handlers) install(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("install model %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no install response")
	}
	if resp.Msg.AlreadyInstalled {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Model %s is already installed.", id)},
			Changes: []string{"no download job submitted"},
			NextCommand: []string{
				fmt.Sprintf("`models get %s` — confirm the install state", id),
			},
		})
	}

	submitted := fmt.Sprintf("Submitted download for %s as job %s (~%ds, ~%dMB).",
		id, resp.Msg.JobId, resp.Msg.EtaSeconds, resp.Msg.SizeMbApprox)

	if !ctx.BoolFlag("wait") || resp.Msg.JobId == "" {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result: []string{submitted},
			NextCommand: []string{
				fmt.Sprintf("`jobs wait %s` — block once until the download finishes", resp.Msg.JobId),
				fmt.Sprintf("`models install %s --wait` — submit and block in one step", id),
			},
		})
	}

	job, werr := h.waitJob(resp.Msg.JobId)
	if werr != nil {
		return werr
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Install of %s %s (job %s).", id, stateName(job.GetState()), job.GetId())},
		Changes: []string{submitted},
		NextCommand: []string{
			fmt.Sprintf("`models get %s` — confirm the install state", id),
		},
	})
}

func (h *handlers) remove(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RemoveModel(context.Background(), connect.NewRequest(&modelsv1.RemoveModelRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("remove model %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no remove response")
	}
	result := fmt.Sprintf("Removed weights for %s.", id)
	if !resp.Msg.Removed {
		result = fmt.Sprintf("Model %s had no installed weights to remove.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{result},
		NextCommand: []string{
			fmt.Sprintf("`models get %s` — confirm the install state", id),
		},
	})
}

func (h *handlers) addCustom(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	operations := ctx.FlagValues("operation")
	model := &modelsv1.Model{
		Id:         id,
		Name:       strings.TrimSpace(ctx.Flag("name")),
		Operations: operations,
		Backend:    strings.TrimSpace(ctx.Flag("backend")),
	}
	resp, err := h.client.AddCustomModel(context.Background(), connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model:       model,
		LocalPath:   strings.TrimSpace(ctx.Flag("local-path")),
		DownloadUrl: strings.TrimSpace(ctx.Flag("download-url")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("add custom model %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Model == nil {
		return fmt.Errorf("server returned no model")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Registered custom model %s.", resp.Msg.Model.Id)},
		Changes: []string{formatModelDetail(resp.Msg.Model)},
		NextCommand: []string{
			fmt.Sprintf("`models get %s` — show the registered entry", resp.Msg.Model.Id),
		},
	})
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := strings.ToLower(strings.TrimSpace(ctx.Positional("query")))
	resp, err := h.client.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("search models", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no models response")
	}
	matches := make([]*modelsv1.Model, 0, len(resp.Msg.Models))
	for _, m := range resp.Msg.Models {
		if modelMatches(m, query) {
			matches = append(matches, m)
		}
	}
	filtered := &modelsv1.ListModelsResponse{Models: matches}
	results := make([]string, 0, len(matches))
	for _, m := range matches {
		results = append(results, formatModel(m))
	}
	return cliapp.RenderProtoList(ctx, filtered, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d model(s) matching %q.", len(matches), ctx.Positional("query"))},
		ResultsHeading: "Models",
		Results:        results,
		RetrievalHints: []string{
			"`models get <id>` — show one model in detail",
		},
	})
}

// modelMatches reports whether query (already lower-cased) is a substring of the
// model's id, name, or any operation it serves.
func modelMatches(m *modelsv1.Model, query string) bool {
	if m == nil {
		return false
	}
	if query == "" {
		return true
	}
	if strings.Contains(strings.ToLower(m.Id), query) || strings.Contains(strings.ToLower(m.Name), query) {
		return true
	}
	for _, op := range m.Operations {
		if strings.Contains(strings.ToLower(op), query) {
			return true
		}
	}
	return false
}

// waitJob blocks once on JobsService.WaitJob and returns the terminal job. It
// reuses the block-once-no-polling pattern from the ai domain.
func (h *handlers) waitJob(jobID string) (*jobsv1.Job, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := jobsconnect.NewJobsServiceClient(httpClient, baseURL)
	resp, err := client.WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: jobID}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wait job", err, nil)
	}
	job := resp.Msg.GetJob()
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		return job, fmt.Errorf("install job %s %s: %s", jobID, stateName(job.GetState()), job.GetError())
	}
	return job, nil
}

func stateName(s jobsv1.JobState) string {
	switch s {
	case jobsv1.JobState_JOB_STATE_SUCCEEDED:
		return "succeeded"
	case jobsv1.JobState_JOB_STATE_FAILED:
		return "failed"
	case jobsv1.JobState_JOB_STATE_CANCELED:
		return "canceled"
	case jobsv1.JobState_JOB_STATE_RUNNING:
		return "running"
	case jobsv1.JobState_JOB_STATE_QUEUED:
		return "queued"
	default:
		return "unknown"
	}
}

func formatModel(m *modelsv1.Model) string {
	if m == nil {
		return "(nil)"
	}
	state := "disabled"
	if m.Enabled {
		state = "enabled"
	}
	tags := installLabel(m)
	if m.Custom {
		tags += " custom"
	}
	return fmt.Sprintf("%s — %s [tier=%s backend=%s ops=%s %s %s]",
		m.Id, m.Name, m.Tier, m.Backend, strings.Join(m.Operations, ","), state, tags)
}

// installLabel renders the on-disk install state in one token.
func installLabel(m *modelsv1.Model) string {
	if m.GetInstall().GetInstalled() {
		return "installed"
	}
	return "not-installed"
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
	if inst := m.GetInstall(); inst.GetInstalled() {
		lines = append(lines, fmt.Sprintf("  install: installed at %s (size_bytes=%d checksum=%s installed_at=%s)",
			inst.GetPath(), inst.GetSizeBytes(), inst.GetChecksum(), inst.GetInstalledAt()))
	} else {
		lines = append(lines, "  install: not installed (run `models install "+m.Id+"`)")
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

func formatFinding(f *modelsv1.CatalogFinding) string {
	if f == nil {
		return "(nil)"
	}
	subject := f.GetModelId()
	if subject == "" {
		subject = f.GetOperation()
	}
	if subject == "" {
		subject = "catalog"
	}
	return fmt.Sprintf("%s %s %s — %s",
		findingSeverityLabel(f.GetSeverity()), f.GetCode(), subject, f.GetMessage())
}

func findingSeverityLabel(s modelsv1.CatalogFindingSeverity) string {
	switch s {
	case modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR:
		return "error"
	case modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_WARNING:
		return "warning"
	default:
		return "unknown"
	}
}
