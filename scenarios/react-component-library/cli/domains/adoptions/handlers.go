package adoptions

import (
	"context"
	"fmt"
	"strconv"
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
		n, err := strconv.Atoi(raw)
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

func (h *handlers) apply(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ApplyAdoptionRequest{
		ComponentId:        ctx.Positional("component-id"),
		Scenario:           ctx.Positional("scenario"),
		AdoptedPath:        ctx.Positional("adopted-path"),
		Version:            ctx.Flag("version"),
		ConfirmOverwrite:   ctx.Flag("confirm-overwrite") == "true",
		OverrideValidation: ctx.Flag("override-validation") == "true",
	}
	resp, err := h.client.ApplyAdoption(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("apply adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Applied adoption %s to %s.", resp.Msg.Adoption.Id, resp.Msg.WrittenPath)},
		ResultsHeading: "Adoption",
		Results:        []string{formatAdoption(resp.Msg.Adoption)},
		RetrievalHints: []string{"`adoptions refresh` — compute drift status now"},
	})
}

func (h *handlers) reapply(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ReapplyAdoptionRequest{
		Id:                    ctx.Positional("id"),
		Version:               ctx.Flag("version"),
		ConfirmLocalOverwrite: ctx.Flag("confirm-local-overwrite") == "true",
		OverrideValidation:    ctx.Flag("override-validation") == "true",
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
	resp, err := h.client.DeleteAdoption(context.Background(), connect.NewRequest(&adoptionsv1.DeleteAdoptionRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete adoption %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Deleted adoption %s.", id)},
	})
}

func (h *handlers) refresh(ctx cliapp.RunContext) error {
	req := &adoptionsv1.RefreshAdoptionsRequest{ComponentId: ctx.Flag("component-id")}
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
	summary := fmt.Sprintf("Refreshed %d adoption(s): library current=%d behind=%d; local clean=%d modified=%d missing=%d.",
		len(resp.Msg.Adoptions), resp.Msg.LibraryCurrent, resp.Msg.LibraryBehind, resp.Msg.LocalClean, resp.Msg.LocalModified, resp.Msg.LocalMissing)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Adoptions",
		Results:        results,
	})
}

func (h *handlers) resolvePath(ctx cliapp.RunContext) error {
	req := &adoptionsv1.ResolveAdoptionPathRequest{
		ComponentId:  ctx.Positional("component-id"),
		Scenario:     ctx.Positional("scenario"),
		OverridePath: ctx.Flag("override"),
		Feature:      ctx.Flag("feature"),
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
	return fmt.Sprintf("%s — %s:%s [%s%s] v=%s adopted=%s refreshed=%s",
		a.Id, a.Scenario, a.AdoptedPath, status, detail, a.AdoptedVersion, a.LibraryId, refreshed)
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
