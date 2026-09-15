package adapters

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters/adapters_v1connect"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client adaptersconnect.AdaptersServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: adaptersconnect.NewAdaptersServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	kind := strings.TrimSpace(ctx.Flag("kind"))
	arch := strings.TrimSpace(ctx.Flag("architecture"))
	resp, err := h.client.ListAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListAdaptersRequest{
		Kind:         kind,
		Architecture: arch,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list adapters", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no adapters response")
	}
	results := make([]string, 0, len(resp.Msg.Adapters))
	for _, a := range resp.Msg.Adapters {
		results = append(results, formatAdapter(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d adapter(s).", len(resp.Msg.Adapters))},
		ResultsHeading: "Adapters",
		Results:        results,
		RetrievalHints: []string{
			"`adapters get <id>` — show one adapter in detail",
			"`adapters compatible --architecture <arch>` — adapters compatible with a base model",
			"`adapters enable <id>` / `adapters enable <id> --disable` — toggle an adapter",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetAdapter(context.Background(), connect.NewRequest(&adaptersv1.GetAdapterRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get adapter %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adapter == nil {
		return fmt.Errorf("server returned no adapter")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Adapter %s.", resp.Msg.Adapter.Id)},
		ResultsHeading: "Adapter",
		Results:        []string{formatAdapterDetail(resp.Msg.Adapter)},
	})
}

func (h *handlers) setEnabled(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	enabled := !ctx.BoolFlag("disable")
	resp, err := h.client.SetAdapterEnabled(context.Background(), connect.NewRequest(&adaptersv1.SetAdapterEnabledRequest{
		Id:      id,
		Enabled: enabled,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set adapter %q enabled=%t", id, enabled), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adapter == nil {
		return fmt.Errorf("server returned no adapter")
	}
	verb := "Enabled"
	if !enabled {
		verb = "Disabled"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s adapter %s.", verb, resp.Msg.Adapter.Id)},
		Changes: []string{formatAdapter(resp.Msg.Adapter)},
		NextCommand: []string{
			fmt.Sprintf("`adapters get %s` — confirm the new state", resp.Msg.Adapter.Id),
		},
	})
}

func (h *handlers) doctor(ctx cliapp.RunContext) error {
	resp, err := h.client.DoctorCatalog(context.Background(), connect.NewRequest(&adaptersv1.DoctorCatalogRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("doctor adapter catalog", err, nil)
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
			"Enabled adapters need a concrete `source.assets[]`/repo/local_path fetch strategy.",
			"`adapters list` — inspect effective enabled state",
		},
	}); err != nil {
		return err
	}
	if !resp.Msg.Ok {
		return fmt.Errorf("adapter catalog doctor failed")
	}
	return nil
}

func (h *handlers) compatible(ctx cliapp.RunContext) error {
	modelID := strings.TrimSpace(ctx.Flag("model"))
	arch := strings.TrimSpace(ctx.Flag("architecture"))
	resp, err := h.client.ListCompatibleAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListCompatibleAdaptersRequest{
		ModelId:      modelID,
		Architecture: arch,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list compatible adapters", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no compatible-adapters response")
	}
	results := make([]string, 0, len(resp.Msg.Adapters))
	for _, a := range resp.Msg.Adapters {
		results = append(results, formatAdapter(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d adapter(s) compatible with %s.", len(resp.Msg.Adapters), resp.Msg.Architecture)},
		ResultsHeading: "Compatible adapters",
		Results:        results,
	})
}

func (h *handlers) install(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.InstallAdapter(context.Background(), connect.NewRequest(&adaptersv1.InstallAdapterRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("install adapter %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no install response")
	}
	if resp.Msg.AlreadyInstalled {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Adapter %s is already installed.", id)},
			Changes: []string{"no download job submitted"},
			NextCommand: []string{
				fmt.Sprintf("`adapters get %s` — confirm the install state", id),
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
				fmt.Sprintf("`adapters install %s --wait` — submit and block in one step", id),
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
			fmt.Sprintf("`adapters get %s` — confirm the install state", id),
		},
	})
}

func (h *handlers) remove(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RemoveAdapter(context.Background(), connect.NewRequest(&adaptersv1.RemoveAdapterRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("remove adapter %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no remove response")
	}
	result := fmt.Sprintf("Removed weights for %s.", id)
	if !resp.Msg.Removed {
		result = fmt.Sprintf("Adapter %s had no installed weights to remove.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{result},
		NextCommand: []string{
			fmt.Sprintf("`adapters get %s` — confirm the install state", id),
		},
	})
}

func (h *handlers) inspect(ctx cliapp.RunContext) error {
	source := strings.TrimSpace(ctx.Positional("source"))
	resp, err := h.client.InspectAdapterSource(context.Background(), connect.NewRequest(&adaptersv1.InspectAdapterSourceRequest{Source: source}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("inspect %q", source), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no inspect response")
	}
	m := resp.Msg
	kind := m.GetKind()
	arch := m.GetArchitecture()
	lines := []string{
		fmt.Sprintf("source:        %s", m.GetSource()),
		fmt.Sprintf("kind:          %s (%s)", kind.GetKind(), kind.GetEvidence()),
		fmt.Sprintf("architecture:  %s (%s confidence — %s)", arch.GetArchitecture(), arch.GetConfidence(), arch.GetEvidence()),
		fmt.Sprintf("license:       %s", m.GetLicense()),
		fmt.Sprintf("nsfw:          %t", m.GetNsfw()),
		fmt.Sprintf("size:          ~%d MB", m.GetSizeBytes()>>20),
	}
	if m.GetProposed() != nil {
		lines = append(lines, fmt.Sprintf("proposed id:   %s", m.GetProposed().GetId()))
	}
	next := fmt.Sprintf("`adapters import %s --kind %s --architecture %s` — register + install", source, kind.GetKind(), arch.GetArchitecture())
	if kind.GetKind() == "" {
		next = fmt.Sprintf("`adapters import %s --kind <lora|controlnet|ip-adapter> --architecture <sd15|sdxl|flux>` — confirm kind + architecture, then install", source)
	} else if arch.GetConfidence() == "none" {
		next = fmt.Sprintf("`adapters import %s --kind %s --architecture <sd15|sdxl|flux>` — confirm the architecture, then install", source, kind.GetKind())
	}
	lines = append(lines, "", "next: "+next)
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Inspected %s.", source)},
		ResultsHeading: "Import preview",
		Results:        lines,
	})
}

func (h *handlers) importAdapter(ctx cliapp.RunContext) error {
	source := strings.TrimSpace(ctx.Positional("source"))
	resp, err := h.client.ImportAdapter(context.Background(), connect.NewRequest(&adaptersv1.ImportAdapterRequest{
		Source:                 source,
		Id:                     strings.TrimSpace(ctx.Flag("id")),
		Name:                   strings.TrimSpace(ctx.Flag("name")),
		Kind:                   strings.TrimSpace(ctx.Flag("kind")),
		Architecture:           strings.TrimSpace(ctx.Flag("architecture")),
		Preprocessor:           strings.TrimSpace(ctx.Flag("preprocessor")),
		AttestCommercialRights: ctx.BoolFlag("attest-commercial-rights"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("import %q", source), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adapter == nil {
		return fmt.Errorf("server returned no adapter")
	}
	id := resp.Msg.Adapter.GetId()
	if resp.Msg.AlreadyInstalled {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("Imported %s; weights already present.", id)},
			Changes:     []string{formatAdapterDetail(resp.Msg.Adapter)},
			NextCommand: []string{fmt.Sprintf("`adapters get %s` — confirm the entry", id)},
		})
	}
	result := fmt.Sprintf("Registered %s and submitted install job %s (~%ds).", id, resp.Msg.JobId, resp.Msg.EtaSeconds)
	next := []string{
		fmt.Sprintf("`jobs wait %s` — block once until the download finishes", resp.Msg.JobId),
		fmt.Sprintf("`adapters get %s` — confirm the registered entry + install state", id),
	}
	if resp.Msg.JobId == "" {
		result = fmt.Sprintf("Registered %s (no install job submitted).", id)
		next = []string{fmt.Sprintf("`adapters install %s` — install the weights", id)}
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     []string{formatAdapterDetail(resp.Msg.Adapter)},
		NextCommand: next,
	})
}

// waitJob blocks once on JobsService.WaitJob and returns the terminal job.
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

func formatAdapter(a *adaptersv1.Adapter) string {
	if a == nil {
		return "(nil)"
	}
	state := "disabled"
	if a.Enabled {
		state = "enabled"
	}
	ready := "not-ready"
	if a.Ready {
		ready = "ready"
	}
	tags := installLabel(a)
	if a.Custom {
		tags += " custom"
	}
	return fmt.Sprintf("%s — %s [kind=%s arch=%s weight=%s %s %s %s]",
		a.Id, a.Name, a.Kind, a.Architecture, a.Weight, state, ready, tags)
}

// installLabel renders the on-disk install state in one token.
func installLabel(a *adaptersv1.Adapter) string {
	if a.GetInstall().GetInstalled() {
		return "installed"
	}
	return "not-installed"
}

func formatAdapterDetail(a *adaptersv1.Adapter) string {
	if a == nil {
		return "(nil)"
	}
	lines := []string{formatAdapter(a)}
	if a.Preprocessor != "" {
		lines = append(lines, "  preprocessor: "+a.Preprocessor)
	}
	if sr := a.GetScaleRange(); sr != nil {
		lines = append(lines, fmt.Sprintf("  scale_range: min=%.3g max=%.3g default=%.3g", sr.GetMin(), sr.GetMax(), sr.GetDefault()))
	}
	if l := a.CapabilityLabels; l != nil {
		lines = append(lines, fmt.Sprintf("  license: %s (commercial_use=%s) nsfw_capable=%t provenance=%s",
			l.License, commercialUseLabel(l.CommercialUse), l.NsfwCapable, l.Provenance))
	}
	if a.Pending != "" {
		lines = append(lines, "  pending: "+a.Pending)
	}
	if inst := a.GetInstall(); inst.GetInstalled() {
		lines = append(lines, fmt.Sprintf("  install: installed at %s (size_bytes=%d checksum=%s installed_at=%s)",
			inst.GetPath(), inst.GetSizeBytes(), inst.GetChecksum(), inst.GetInstalledAt()))
	} else {
		lines = append(lines, "  install: not installed (run `adapters install "+a.Id+"`)")
	}
	return strings.Join(lines, "\n")
}

func commercialUseLabel(c adaptersv1.CommercialUse) string {
	switch c {
	case adaptersv1.CommercialUse_COMMERCIAL_USE_YES:
		return "yes"
	case adaptersv1.CommercialUse_COMMERCIAL_USE_NO:
		return "no"
	case adaptersv1.CommercialUse_COMMERCIAL_USE_CONDITIONAL:
		return "conditional"
	default:
		return "unspecified"
	}
}

func formatFinding(f *adaptersv1.CatalogFinding) string {
	if f == nil {
		return "(nil)"
	}
	subject := f.GetAdapterId()
	if subject == "" {
		subject = "catalog"
	}
	return fmt.Sprintf("%s %s %s — %s",
		findingSeverityLabel(f.GetSeverity()), f.GetCode(), subject, f.GetMessage())
}

func findingSeverityLabel(s adaptersv1.CatalogFindingSeverity) string {
	switch s {
	case adaptersv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_ERROR:
		return "error"
	case adaptersv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_WARNING:
		return "warning"
	default:
		return "unknown"
	}
}
