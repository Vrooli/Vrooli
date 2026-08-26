package adoptions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the
// generated Connect-Go client. Mirrors cli/domains/components/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client adoptionsconnect.AdoptionsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: adoptionsconnect.NewAdoptionsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ListAdoptionsRequest{
		ComponentId: ctx.Flag("component-id"),
		Scenario:    ctx.Flag("scenario"),
	}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list response")
	}
	results := make([]string, 0, len(resp.Msg.Adoptions))
	for _, a := range resp.Msg.Adoptions {
		results = append(results, formatAdoption(a))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d adoption(s).", len(resp.Msg.Adoptions))},
		ResultsHeading: "Adoptions",
		Results:        results,
		RetrievalHints: []string{
			"`adoptions refresh` — recompute drift status",
			"`adoptions apply <component-id> <scenario> <adopted-path>` — copy a component into a scenario",
		},
	})
}

func (h *handlers) listScenarios(ctx cliapp.RunContext) error {
	resp, err := h.client.ListScenarios(context.Background(), connect.NewRequest(&adoptionsv1.ListScenariosRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list selectable scenarios", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no scenarios response")
	}
	rows := make([]string, 0, len(resp.Msg.Scenarios))
	for _, scenario := range resp.Msg.Scenarios {
		if scenario == nil {
			continue
		}
		label := scenario.DisplayName
		if label == "" || label == scenario.Name {
			label = scenario.Name
		} else {
			label = fmt.Sprintf("%s (%s)", label, scenario.Name)
		}
		rows = append(rows, label)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d selectable scenario(s).", len(rows))},
		ResultsHeading: "Scenarios",
		Results:        rows,
		RetrievalHints: []string{"`adoptions apply <component-id> <scenario> <adopted-path>` — adopt a component into a scenario"},
	})
}

func (h *handlers) listEffective(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ListEffectiveAdoptionsRequest{ComponentId: ctx.Positional("component-id")}
	if raw := ctx.Flag("limit"); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(limit)
	}
	resp, err := h.client.ListEffectiveAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list effective adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no effective-adoptions response")
	}
	rows := make([]string, 0, len(resp.Msg.Adoptions))
	for _, effective := range resp.Msg.Adoptions {
		if effective.ParentAdoption == nil {
			continue
		}
		kind := "direct"
		if effective.Mediated {
			kind = "indirect"
		}
		rows = append(rows, fmt.Sprintf("%s %s@%s via %s (%s)", kind, effective.SourceLibraryId, effective.SourceVersion, effective.ParentAdoption.Scenario, effective.ParentAdoption.Id))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d effective adoption(s).", len(rows))}, ResultsHeading: "Effective adoptions", Results: rows})
}

func (h *handlers) preflight(ctx cliapp.RunContext) error {
	resp, err := h.client.PreflightAdoption(context.Background(), connect.NewRequest(&adoptionsv1.PreflightAdoptionRequest{
		ComponentId: ctx.Positional("component-id"), Scenario: ctx.Positional("scenario"), Version: ctx.Flag("version"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preflight adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no preflight response")
	}
	rows := append([]string{}, resp.Msg.RequiredTokens...)
	rows = append(rows, resp.Msg.RequiredTokenPatterns...)
	for _, token := range resp.Msg.UnsatisfiedTokens {
		rows = append(rows, "UNSATISFIED "+token)
	}
	rows = append(rows, fmt.Sprintf("version=%s maturity=%s (floor=%s)", resp.Msg.VersionStatus, resp.Msg.MaturityRung, resp.Msg.MaturityFloor))
	rows = append(rows, resp.Msg.Warnings...)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Preflight %s@%s for %s: blocking=%t", resp.Msg.ComponentId, resp.Msg.Version, resp.Msg.Scenario, resp.Msg.Blocking)}, ResultsHeading: "Adoptability verdict", Results: rows})
}

func (h *handlers) syncTokens(ctx cliapp.RunContext) error {
	resp, err := h.client.SyncScenarioTokens(context.Background(), connect.NewRequest(&adoptionsv1.SyncScenarioTokensRequest{Scenario: ctx.Positional("scenario"), DryRun: ctx.Flag("dry-run") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError("sync scenario tokens", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Token sync for %s: added=%d changed=%t", resp.Msg.Scenario, len(resp.Msg.Added), resp.Msg.Changed)}, ResultsHeading: "Added tokens", Results: resp.Msg.Added})
}

func (h *handlers) pruneTokens(ctx cliapp.RunContext) error {
	resp, err := h.client.PruneScenarioTokens(context.Background(), connect.NewRequest(&adoptionsv1.PruneScenarioTokensRequest{Scenario: ctx.Positional("scenario"), Apply: ctx.Flag("apply") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError("prune scenario tokens", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Token prune for %s: removed=%d changed=%t", resp.Msg.Scenario, len(resp.Msg.Removed), resp.Msg.Changed)}, ResultsHeading: "Removed tokens", Results: resp.Msg.Removed})
}

func (h *handlers) link(ctx cliapp.RunContext) error {
	resp, err := h.client.LinkAdoption(context.Background(), connect.NewRequest(&adoptionsv1.LinkAdoptionRequest{
		ComponentId: ctx.Positional("component-id"), Scenario: ctx.Positional("scenario"), Version: ctx.Flag("version"),
		ImportSubpath: ctx.Flag("import-subpath"), ConfirmExisting: ctx.Flag("confirm-existing") == "true",
	}))
	if err != nil {
		return cliapp.WrapAPIError("link adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no linked adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Linked %s through %s.", resp.Msg.Adoption.Id, resp.Msg.PackagePath)}, ResultsHeading: "Adoption", Results: []string{formatAdoption(resp.Msg.Adoption)}})
}

func (h *handlers) eject(ctx cliapp.RunContext) error {
	if strings.TrimSpace(ctx.Flag("reason")) == "" {
		return fmt.Errorf("--reason is required for an ejection")
	}
	resp, err := h.client.EjectAdoption(context.Background(), connect.NewRequest(&adoptionsv1.EjectAdoptionRequest{
		ComponentId: ctx.Positional("component-id"), Scenario: ctx.Positional("scenario"), AdoptedPath: ctx.Positional("adopted-path"), Version: ctx.Flag("version"), Reason: ctx.Flag("reason"),
		ConfirmOverwrite: ctx.Flag("confirm-overwrite") == "true", OverrideValidation: ctx.Flag("override-validation") == "true", ReplaceExisting: ctx.Flag("replace-existing") == "true",
	}))
	if err != nil {
		return cliapp.WrapAPIError("eject adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no ejected adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Ejected %s to %s for reason: %s", resp.Msg.Adoption.Id, resp.Msg.WrittenPath, resp.Msg.Adoption.ForkReason)}, ResultsHeading: "Adoption", Results: []string{formatAdoption(resp.Msg.Adoption)}})
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	if ctx.FlagDeclared("set") {
		values := ctx.FlagValues("set")
		if len(values) == 0 {
			return h.applySingle(ctx)
		}
		items := make([]*adoptionsv1.BatchApplyItem, 0, len(values))
		for _, value := range values {
			parts := strings.SplitN(value, ":", 4)
			if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
				return fmt.Errorf("--set must be component-id:scenario:adopted-path[:version] (got %q)", value)
			}
			item := &adoptionsv1.BatchApplyItem{ComponentId: parts[0], Scenario: parts[1], AdoptedPath: parts[2], ConfirmOverwrite: ctx.Flag("confirm-overwrite") == "true", OverrideValidation: ctx.Flag("override-validation") == "true", ReplaceExisting: ctx.Flag("replace-existing") == "true"}
			if len(parts) == 4 {
				item.Version = parts[3]
			}
			items = append(items, item)
		}
		return h.batchApply(ctx, items)
	}
	return h.applySingle(ctx)
}

func (h *handlers) applySingle(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ApplyAdoptionRequest{
		ComponentId:        ctx.Positional("component-id"),
		Scenario:           ctx.Positional("scenario"),
		AdoptedPath:        ctx.Positional("adopted-path"),
		Version:            ctx.Flag("version"),
		ConfirmOverwrite:   ctx.Flag("confirm-overwrite") == "true",
		OverrideValidation: ctx.Flag("override-validation") == "true",
		ReplaceExisting:    ctx.Flag("replace-existing") == "true",
	}
	if ctx.FlagDeclared("include") {
		req.IncludeSuggestions = ctx.FlagValues("include")
	}
	resp, err := h.client.ApplyAdoption(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("apply adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: append([]string{
			fmt.Sprintf("Applied adoption %s to %s.", resp.Msg.Adoption.Id, resp.Msg.WrittenPath),
			fmt.Sprintf("Copied: %d; satisfied ports: %d; available suggestions: %d.", len(resp.Msg.CopiedAssets), len(resp.Msg.SatisfiedPorts), len(resp.Msg.AvailableSuggestions)),
		}, formatImportSites(resp.Msg.ImportSites)...),
		ResultsHeading: "Adoption",
		Results:        []string{formatAdoption(resp.Msg.Adoption)},
		RetrievalHints: []string{"`adoptions refresh` — compute drift status now"},
	})
}

func (h *handlers) batchApply(ctx cliapp.RunContext, items []*adoptionsv1.BatchApplyItem) error {
	resp, err := h.client.BatchApplyAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.BatchApplyAdoptionsRequest{Items: items}))
	if err != nil {
		return cliapp.WrapAPIError("batch apply adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no batch adoption response")
	}
	rows := make([]string, 0, len(resp.Msg.Results))
	for _, item := range resp.Msg.Results {
		if item != nil && item.Adoption != nil {
			rows = append(rows, formatAdoption(item.Adoption))
		}
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Batch-applied %d adoption(s); shared dependencies=%d.", len(rows), len(resp.Msg.SharedDependencies))}, ResultsHeading: "Adoptions", Results: rows})
}

func (h *handlers) batchApplyCommand(ctx cliapp.RunContext) error {
	values := ctx.FlagValues("set")
	if len(values) == 0 {
		return fmt.Errorf("batch apply requires at least one --set value")
	}
	items := make([]*adoptionsv1.BatchApplyItem, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, ":", 4)
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return fmt.Errorf("--set must be component-id:scenario:adopted-path[:version] (got %q)", value)
		}
		item := &adoptionsv1.BatchApplyItem{ComponentId: parts[0], Scenario: parts[1], AdoptedPath: parts[2], ConfirmOverwrite: ctx.Flag("confirm-overwrite") == "true", OverrideValidation: ctx.Flag("override-validation") == "true", ReplaceExisting: ctx.Flag("replace-existing") == "true"}
		if len(parts) == 4 {
			item.Version = parts[3]
		}
		items = append(items, item)
	}
	return h.batchApply(ctx, items)
}

func (h *handlers) suggest(ctx cliapp.RunContext) error {
	req := &adoptionsv1.SuggestAdoptionsRequest{Scenario: ctx.Flag("scenario"), ComponentId: ctx.Flag("component-id")}
	if raw := ctx.Flag("limit"); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(limit)
	}
	resp, err := h.client.SuggestAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("suggest adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no suggestions")
	}
	rows := make([]string, 0, len(resp.Msg.Suggestions))
	for _, suggestion := range resp.Msg.Suggestions {
		rows = append(rows, fmt.Sprintf("[%s] %s → %s: %s", suggestion.Classification, suggestion.Scenario, suggestion.LibraryId, strings.Join(suggestion.Reasons, "; ")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d adoption suggestion(s).", len(rows))}, ResultsHeading: "Suggestions", Results: rows, RetrievalHints: []string{"`adoptions apply <component-id> <scenario> <adopted-path>` — act on a suggestion"}})
}

func formatImportSites(sites []string) []string {
	if len(sites) == 0 {
		return []string{"No direct import sites found."}
	}
	return []string{fmt.Sprintf("Direct import sites (%d): %s", len(sites), strings.Join(sites, ", "))}
}

func (h *handlers) reapply(ctx cliapp.RunContext) error {
	dryRun := false
	if ctx.FlagDeclared("dry-run") {
		dryRun = ctx.Flag("dry-run") == "true"
	}
	req := &adoptionsv1.ReapplyAdoptionRequest{
		Id:                    ctx.Positional("id"),
		Version:               ctx.Flag("version"),
		ConfirmLocalOverwrite: ctx.Flag("confirm-local-overwrite") == "true",
		OverrideValidation:    ctx.Flag("override-validation") == "true",
		DryRun:                dryRun,
	}
	resp, err := h.client.ReapplyAdoption(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("reapply adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reapplied adoption %s to %s.", resp.Msg.Adoption.Id, resp.Msg.WrittenPath)},
		ResultsHeading: "Adoption",
		Results:        []string{formatAdoption(resp.Msg.Adoption)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteAdoption(context.Background(), connect.NewRequest(&adoptionsv1.DeleteAdoptionRequest{Id: id, ConfirmRemoveFiles: ctx.Flag("confirm-remove-files") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete adoption %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Deleted adoption %s; removed %d file(s).", id, len(resp.Msg.RemovedFiles))}, ResultsHeading: "Removable files", Results: resp.Msg.RemovableFiles})
}

func (h *handlers) refresh(ctx cliapp.RunContext) error {
	report := ctx.FlagDeclared("report") && ctx.Flag("report") == "true"
	apply := !report || (ctx.FlagDeclared("apply") && ctx.Flag("apply") == "true")
	req := &adoptionsv1.RefreshAdoptionsRequest{ComponentId: ctx.Flag("component-id"), ForkReport: report, Apply: apply}
	resp, err := h.client.RefreshAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("refresh adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no refresh response")
	}
	results := make([]string, 0, len(resp.Msg.Adoptions))
	for _, a := range resp.Msg.Adoptions {
		results = append(results, formatAdoption(a))
	}
	summary := fmt.Sprintf("Refreshed %d adoption(s): library current=%d behind=%d source-drifted=%d; local clean=%d modified=%d missing=%d.",
		len(resp.Msg.Adoptions), resp.Msg.LibraryCurrent, resp.Msg.LibraryBehind, resp.Msg.LibrarySourceDrifted, resp.Msg.LocalClean, resp.Msg.LocalModified, resp.Msg.LocalMissing)
	if report {
		counts := map[string]int{}
		for _, adoption := range resp.Msg.Adoptions {
			if adoption != nil {
				counts[nonEmpty(adoption.ForkStatus, "unclassified")]++
			}
		}
		mode := "dry-run"
		if apply {
			mode = "applied"
		}
		summary = fmt.Sprintf("Fork report (%s) for %d adoption(s): mechanical-translation=%d contract-preserved=%d local-addition=%d local-fork=%d declared-fork=%d unclassified=%d.",
			mode, len(resp.Msg.Adoptions), counts["mechanical-translation"], counts["contract-preserved"], counts["local-addition"], counts["local-fork"], counts["declared-fork"], counts["unclassified"])
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Adoptions",
		Results:        results,
	})
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.ReconcileAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.ReconcileAdoptionsRequest{Apply: ctx.Flag("apply") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reconcile response")
	}
	rows := make([]string, 0, len(resp.Msg.Findings))
	for _, finding := range resp.Msg.Findings {
		rows = append(rows, fmt.Sprintf("%s:%s (%s@%s): %s", finding.Scenario, finding.AdoptedPath, finding.LibraryId, finding.Version, finding.Detail))
	}
	mode := "Dry-run"
	if resp.Msg.Created > 0 && ctx.Flag("apply") == "true" {
		mode = "Applied"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%s reconciliation: scanned %d provenance file(s), already recorded %d, records to create/created %d.", mode, resp.Msg.Scanned, resp.Msg.AlreadyRecorded, resp.Msg.Created)}, ResultsHeading: "Unresolved provenance", Results: rows, RetrievalHints: []string{"`adoptions reconcile --apply true` — write the proposed missing records"}})
}

func (h *handlers) reconverge(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ReconvergeAdoptionsRequest{
		Scenario: ctx.Flag("scenario"),
		Apply:    ctx.Flag("apply") == "true",
	}
	resp, err := h.client.ReconvergeAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("reconverge adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reconverge response")
	}
	rows := make([]string, 0, len(resp.Msg.Outcomes))
	for _, o := range resp.Msg.Outcomes {
		line := fmt.Sprintf("[%s] %s ← %s %s→%s (%s)", reconvergeActionLabel(o.Action), o.Scenario, o.LibraryId, o.AdoptedVersion, nonEmpty(o.TargetVersion, "?"), o.AdoptionId)
		if o.Detail != "" {
			line += " — " + o.Detail
		}
		for _, f := range o.Files {
			line += fmt.Sprintf("\n      · %s [%s]", f.AdoptedPath, localStatusLabel(f.LocalStatus))
		}
		rows = append(rows, line)
	}
	mode := "Dry-run"
	if req.Apply {
		mode = "Applied"
	}
	summary := fmt.Sprintf("%s reconverge: scanned %d, behind %d, reapplied %d, flagged-modified %d, skipped %d, errored %d.",
		mode, resp.Msg.Scanned, resp.Msg.Behind, resp.Msg.Reapplied, resp.Msg.Flagged, resp.Msg.Skipped, resp.Msg.Errored)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Reconverge outcomes",
		Results:        rows,
		RetrievalHints: []string{
			"`adoptions reconverge --apply true` — re-apply the CLEAN behind copies",
			"`adoptions reapply <id> --confirm-local-overwrite true` — reconverge a flagged MODIFIED copy after review",
		},
	})
}

func reconvergeActionLabel(a adoptionsv1.ReconvergeAction) string {
	switch a {
	case adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_REAPPLIED:
		return "reapplied"
	case adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_WOULD_REAPPLY:
		return "would-reapply"
	case adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_FLAGGED_MODIFIED:
		return "flagged-modified"
	case adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_SKIPPED_UNRESOLVED:
		return "skipped-unresolved"
	case adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_ERROR:
		return "error"
	}
	return "unspecified"
}

func (h *handlers) discover(ctx cliapp.RunContext) error {
	req := &adoptionsv1.DiscoverAdoptionsRequest{Scenario: ctx.Flag("scenario")}
	if raw := ctx.Flag("min-similarity"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("invalid --min-similarity %q: %w", raw, err)
		}
		req.MinSimilarity = f
	}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid --limit %q: %w", raw, err)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.DiscoverAdoptions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("discover adoptions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no discover response")
	}
	rows := make([]string, 0, len(resp.Msg.Candidates))
	for _, c := range resp.Msg.Candidates {
		rows = append(rows, fmt.Sprintf("%s:%s ~ %s@%s (dice %.3f, shared %d) %s",
			c.Scenario, c.AdoptedPath, c.LibraryId, c.Version, c.Similarity, c.SharedLines, strings.Join(c.Evidence, " | ")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Discovery: scanned %d header-less file(s) at min similarity %.2f, surfaced %d candidate(s).", resp.Msg.Scanned, resp.Msg.MinSimilarity, len(rows))},
		ResultsHeading: "Candidates (review evidence before confirming)",
		Results:        rows,
		RetrievalHints: []string{"`adoptions confirm-discovery <scenario> <adopted-path> <component-id> <version>` — inject the provenance header and create the record for a confirmed true positive"},
	})
}

func (h *handlers) confirmDiscovery(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ConfirmDiscoveryRequest{
		Scenario:    ctx.Positional("scenario"),
		AdoptedPath: ctx.Positional("adopted-path"),
		ComponentId: ctx.Positional("component-id"),
		Version:     ctx.Positional("version"),
	}
	resp, err := h.client.ConfirmDiscovery(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("confirm discovery", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no confirm response")
	}
	a := resp.Msg.Adoption
	results := []string{
		fmt.Sprintf("id:            %s", a.Id),
		fmt.Sprintf("scenario:      %s", a.Scenario),
		fmt.Sprintf("adopted_path:  %s", a.AdoptedPath),
		fmt.Sprintf("component:     %s@%s", a.LibraryId, a.AdoptedVersion),
		fmt.Sprintf("written_path:  %s", resp.Msg.WrittenPath),
		fmt.Sprintf("similarity:    %.3f", resp.Msg.Similarity),
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Injected provenance header and recorded adoption for %s:%s.", a.Scenario, a.AdoptedPath)},
		ResultsHeading: "Adoption",
		Results:        results,
		RetrievalHints: []string{"`adoptions refresh --component-id " + a.ComponentId + "` — inspect drift on the freshly tracked copy"},
	})
}

func (h *handlers) resolvePath(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ResolveAdoptionPathRequest{
		ComponentId:  ctx.Positional("component-id"),
		Scenario:     ctx.Positional("scenario"),
		OverridePath: ctx.Flag("override"),
		Feature:      ctx.Flag("feature"),
		Version:      ctx.Flag("version"),
		Template:     ctx.Flag("template"),
	}
	resp, err := h.client.ResolveAdoptionPath(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("resolve adoption path", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no resolve response")
	}
	results := []string{
		fmt.Sprintf("path:   %s", resp.Msg.Path),
		fmt.Sprintf("source: %s", resolveSourceLabel(resp.Msg.Source)),
		fmt.Sprintf("slot:   %s", nonEmpty(resp.Msg.Slot, "(unset)")),
	}
	// Full version file-set placement: where every companion lands. The tree
	// mirrors the code panel's placement view so operators verify multi-file
	// components (hooks, tokens) from the CLI.
	if len(resp.Msg.Files) > 0 {
		manifest := "no template manifest — flat fallback placement"
		if resp.Msg.ManifestResolved {
			manifest = fmt.Sprintf("template manifest: %s", nonEmpty(resp.Msg.Template, "(unknown)"))
		}
		results = append(results, "", "placement ("+manifest+"):")
		for _, f := range resp.Msg.Files {
			entry := ""
			if f.IsEntry {
				entry = " [entry]"
			}
			results = append(results, fmt.Sprintf("  %-22s -> %s  (%s · %s)%s",
				f.LibraryPath, f.TargetPath, nonEmpty(f.Slot, "?"), f.SlotSource, entry))
			for _, w := range f.Warnings {
				results = append(results, "    warning: "+w)
			}
		}
	}
	for _, w := range resp.Msg.Warnings {
		results = append(results, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resolved %s -> %s", req.ComponentId, resp.Msg.Path)},
		ResultsHeading: "Resolution",
		Results:        results,
		RetrievalHints: []string{
			"`adoptions apply <component-id> <scenario> <adopted-path>` — copy the file into place",
		},
	})
}

func resolveSourceLabel(s adoptionsv1.ResolveSource) string {
	switch s {
	case adoptionsv1.ResolveSource_RESOLVE_SOURCE_EXPLICIT:
		return "explicit"
	case adoptionsv1.ResolveSource_RESOLVE_SOURCE_TEMPLATE_MANIFEST:
		return "template-manifest"
	case adoptionsv1.ResolveSource_RESOLVE_SOURCE_HEURISTIC:
		return "heuristic"
	case adoptionsv1.ResolveSource_RESOLVE_SOURCE_FALLBACK:
		return "fallback"
	}
	return "unspecified"
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func formatAdoption(a *adoptionsv1.Adoption) string {
	if a == nil {
		return "(nil)"
	}
	refreshed := "never"
	if a.RefreshedAt != nil {
		refreshed = a.RefreshedAt.AsTime().Format(time.RFC3339)
	}
	status := fmt.Sprintf("library=%s local=%s", libraryStatusLabel(a.LibraryVersionStatus), localStatusLabel(a.LocalStatus))
	detail := ""
	if a.StatusDetail != "" {
		detail = " (" + a.StatusDetail + ")"
	}
	return fmt.Sprintf("%s — %s:%s [%s%s] fork=%s mode=%s v=%s adopted=%s refreshed=%s",
		a.Id, a.Scenario, a.AdoptedPath, status, detail, nonEmpty(a.ForkStatus, "unclassified"), nonEmpty(a.Mode, "copied"), a.AdoptedVersion, a.LibraryId, refreshed)
}

func libraryStatusLabel(s adoptionsv1.LibraryVersionStatus) string {
	switch s {
	case adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_CURRENT:
		return "current"
	case adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_BEHIND:
		return "behind"
	case adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_DEPRECATED:
		return "deprecated"
	case adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_MISSING:
		return "missing"
	case adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_UNKNOWN:
		return "unknown"
	}
	return "unknown"
}

func localStatusLabel(s adoptionsv1.LocalStatus) string {
	switch s {
	case adoptionsv1.LocalStatus_LOCAL_STATUS_CLEAN:
		return "clean"
	case adoptionsv1.LocalStatus_LOCAL_STATUS_MODIFIED:
		return "modified"
	case adoptionsv1.LocalStatus_LOCAL_STATUS_MISSING:
		return "missing"
	case adoptionsv1.LocalStatus_LOCAL_STATUS_UNKNOWN:
		return "unknown"
	}
	return "unknown"
}
