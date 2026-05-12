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
			"`adoptions create <component-id> <scenario> <adopted-path>` — link a copy",
		},
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	req := &adoptionsv1.CreateAdoptionRequest{
		ComponentId:    ctx.Positional("component-id"),
		Scenario:       ctx.Positional("scenario"),
		AdoptedPath:    ctx.Positional("adopted-path"),
		AdoptedVersion: ctx.Flag("adopted-version"),
	}
	resp, err := h.client.CreateAdoption(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("create adoption", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Adoption == nil {
		return fmt.Errorf("server returned no adoption")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Created adoption %s.", resp.Msg.Adoption.Id)},
		ResultsHeading: "Adoption",
		Results:        []string{formatAdoption(resp.Msg.Adoption)},
		RetrievalHints: []string{"`adoptions refresh` — compute drift status now"},
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
	summary := fmt.Sprintf("Refreshed %d adoption(s): current=%d behind=%d modified=%d unknown=%d.",
		len(resp.Msg.Adoptions), resp.Msg.Current, resp.Msg.Behind, resp.Msg.Modified, resp.Msg.Unknown)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Adoptions",
		Results:        results,
	})
}

func formatAdoption(a *adoptionsv1.Adoption) string {
	if a == nil {
		return "(nil)"
	}
	refreshed := "never"
	if a.RefreshedAt != nil {
		refreshed = a.RefreshedAt.AsTime().Format(time.RFC3339)
	}
	status := statusLabel(a.Status)
	detail := ""
	if a.StatusDetail != "" {
		detail = " (" + a.StatusDetail + ")"
	}
	return fmt.Sprintf("%s — %s:%s [%s%s] v=%s adopted=%s refreshed=%s",
		a.Id, a.Scenario, a.AdoptedPath, status, detail, a.AdoptedVersion, a.LibraryId, refreshed)
}

func statusLabel(s adoptionsv1.AdoptionStatus) string {
	switch s {
	case adoptionsv1.AdoptionStatus_ADOPTION_STATUS_CURRENT:
		return "current"
	case adoptionsv1.AdoptionStatus_ADOPTION_STATUS_BEHIND:
		return "behind"
	case adoptionsv1.AdoptionStatus_ADOPTION_STATUS_MODIFIED:
		return "modified"
	case adoptionsv1.AdoptionStatus_ADOPTION_STATUS_UNKNOWN:
		return "unknown"
	}
	return "unknown"
}
