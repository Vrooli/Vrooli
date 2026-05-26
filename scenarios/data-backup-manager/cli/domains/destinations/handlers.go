package destinations

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client destinationsconnect.DestinationsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: destinationsconnect.NewDestinationsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	backend, err := parseBackendKind(ctx.Flag("backend"))
	if err != nil {
		return err
	}
	capBytes, err := parseOptionalInt64(ctx.Flag("cap-bytes"))
	if err != nil {
		return fmt.Errorf("--cap-bytes: %w", err)
	}
	capPolicy, err := parseCapPolicy(ctx.Flag("cap-policy"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateDestination(context.Background(), connect.NewRequest(&destinationsv1.CreateDestinationRequest{
		Name:        ctx.Flag("name"),
		BackendKind: backend,
		Location:    ctx.Flag("location"),
		CapBytes:    capBytes,
		CapPolicy:   capPolicy,
	}))
	if err != nil {
		return cliapp.WrapAPIError("create destination", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created destination %s.", resp.Msg.Destination.Id)},
		Changes: []string{formatDestination(resp.Msg.Destination)},
		NextCommand: []string{
			fmt.Sprintf("`destinations get %s` — show this destination", resp.Msg.Destination.Id),
			"`destinations list` — show all destinations",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDestination(context.Background(), connect.NewRequest(&destinationsv1.GetDestinationRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get destination %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched destination %s.", resp.Msg.Destination.Id)},
		ResultsHeading: "Destination",
		Results:        []string{formatDestination(resp.Msg.Destination)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDestinations(context.Background(), connect.NewRequest(&destinationsv1.ListDestinationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list destinations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no destinations response")
	}
	results := make([]string, 0, len(resp.Msg.Destinations))
	for _, d := range resp.Msg.Destinations {
		results = append(results, formatDestination(d))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d destination(s).", len(resp.Msg.Destinations))},
		ResultsHeading: "Destinations",
		Results:        results,
		RetrievalHints: []string{
			"`destinations get <id>` — show a single destination",
			"`destinations create --name <n> --backend <b> --location <l>` — create a destination",
		},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	capBytes, err := parseOptionalInt64(ctx.Flag("cap-bytes"))
	if err != nil {
		return fmt.Errorf("--cap-bytes: %w", err)
	}
	capPolicy, err := parseCapPolicy(ctx.Flag("cap-policy"))
	if err != nil {
		return err
	}
	resp, err := h.client.UpdateDestination(context.Background(), connect.NewRequest(&destinationsv1.UpdateDestinationRequest{
		Id:        ctx.Flag("id"),
		CapBytes:  capBytes,
		CapPolicy: capPolicy,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update destination", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated destination %s.", resp.Msg.Destination.Id)},
		Changes: []string{formatDestination(resp.Msg.Destination)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	deleteRepo := false
	if s := ctx.Flag("delete-repository"); s != "" {
		deleteRepo, _ = strconv.ParseBool(s)
	}
	resp, err := h.client.DeleteDestination(context.Background(), connect.NewRequest(&destinationsv1.DeleteDestinationRequest{
		Id:               ctx.Flag("id"),
		DeleteRepository: deleteRepo,
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete destination", err, nil)
	}
	msg := "No matching destination to delete."
	if resp != nil && resp.Msg != nil && resp.Msg.Removed {
		msg = "Deleted destination."
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

func (h *handlers) usage(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDestinationUsage(context.Background(), connect.NewRequest(&destinationsv1.GetDestinationUsageRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get destination usage %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no usage response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Usage for destination %s.", id)},
		ResultsHeading: "Usage",
		Results: []string{
			fmt.Sprintf("usage=%d bytes cap=%d bytes state=%s policy=%s",
				resp.Msg.UsageBytes, resp.Msg.CapBytes,
				usageStateLabel(resp.Msg.UsageState),
				capPolicyLabel(resp.Msg.CapPolicy)),
		},
	})
}

// parseBackendKind maps the --backend flag string to the proto BackendKind enum.
func parseBackendKind(s string) (destinationsv1.BackendKind, error) {
	switch s {
	case "filesystem":
		return destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM, nil
	case "s3":
		return destinationsv1.BackendKind_BACKEND_KIND_S3, nil
	default:
		return destinationsv1.BackendKind_BACKEND_KIND_UNSPECIFIED,
			fmt.Errorf("invalid --backend %q: must be one of filesystem, s3", s)
	}
}

// parseCapPolicy maps the --cap-policy flag string to the proto CapPolicy enum.
// Empty string returns UNSPECIFIED (no cap policy change).
func parseCapPolicy(s string) (destinationsv1.CapPolicy, error) {
	switch s {
	case "":
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED, nil
	case "alert-block":
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK, nil
	case "alert-only":
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY, nil
	default:
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED,
			fmt.Errorf("invalid --cap-policy %q: must be one of alert-block, alert-only", s)
	}
}

// parseOptionalInt64 parses an optional integer flag; returns 0 for empty.
func parseOptionalInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

func backendKindLabel(k destinationsv1.BackendKind) string {
	switch k {
	case destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM:
		return "filesystem"
	case destinationsv1.BackendKind_BACKEND_KIND_S3:
		return "s3"
	default:
		return "unspecified"
	}
}

func capPolicyLabel(p destinationsv1.CapPolicy) string {
	switch p {
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK:
		return "alert-block"
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY:
		return "alert-only"
	default:
		return "unspecified"
	}
}

func usageStateLabel(s destinationsv1.UsageState) string {
	switch s {
	case destinationsv1.UsageState_USAGE_STATE_WITHIN:
		return "within"
	case destinationsv1.UsageState_USAGE_STATE_NEAR:
		return "near"
	case destinationsv1.UsageState_USAGE_STATE_OVER:
		return "over"
	default:
		return "unspecified"
	}
}

func formatDestination(d *destinationsv1.Destination) string {
	if d == nil {
		return "(nil)"
	}
	created := ""
	if d.CreatedAt != nil {
		created = d.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [backend=%s location=%s cap=%d policy=%s usage=%d state=%s created=%s]",
		d.Id, d.Name,
		backendKindLabel(d.BackendKind), d.Location,
		d.CapBytes, capPolicyLabel(d.CapPolicy),
		d.UsageBytes, usageStateLabel(d.UsageState),
		created)
}
