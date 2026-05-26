package targets

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets/targets_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client targetsconnect.TargetsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: targetsconnect.NewTargetsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) register(ctx cliapp.RunContext) error {
	kind, err := parseKind(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	resp, err := h.client.RegisterTarget(context.Background(), connect.NewRequest(&targetsv1.RegisterTargetRequest{
		Owner:      ctx.Flag("owner"),
		Name:       ctx.Flag("name"),
		SourceKind: kind,
		Locator:    ctx.Flag("locator"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("register target", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Target == nil {
		return fmt.Errorf("server returned no target")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Registered target %s.", resp.Msg.Target.Id)},
		Changes: []string{formatTarget(resp.Msg.Target)},
		NextCommand: []string{
			fmt.Sprintf("`targets get %s` — show this target", resp.Msg.Target.Id),
			"`targets list` — show all targets",
		},
	})
}

func (h *handlers) deregister(ctx cliapp.RunContext) error {
	resp, err := h.client.DeregisterTarget(context.Background(), connect.NewRequest(&targetsv1.DeregisterTargetRequest{
		Owner: ctx.Flag("owner"),
		Name:  ctx.Flag("name"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("deregister target", err, nil)
	}
	msg := "No matching target to deregister."
	if resp != nil && resp.Msg != nil && resp.Msg.Removed {
		msg = "Deregistered target."
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetTarget(context.Background(), connect.NewRequest(&targetsv1.GetTargetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get target %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Target == nil {
		return fmt.Errorf("server returned no target")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched target %s.", resp.Msg.Target.Id)},
		ResultsHeading: "Target",
		Results:        []string{formatTarget(resp.Msg.Target)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTargets(context.Background(), connect.NewRequest(&targetsv1.ListTargetsRequest{
		Owner: ctx.Flag("owner"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list targets", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no targets response")
	}
	results := make([]string, 0, len(resp.Msg.Targets))
	for _, t := range resp.Msg.Targets {
		results = append(results, formatTarget(t))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d target(s).", len(resp.Msg.Targets))},
		ResultsHeading: "Targets",
		Results:        results,
		RetrievalHints: []string{
			"`targets get <id>` — show a single target",
			"`targets register --owner <o> --name <n> --kind <k> --locator <l>` — register a target",
		},
	})
}

// parseKind maps the --kind flag string to the proto SourceKind enum.
func parseKind(s string) (sourcesv1.SourceKind, error) {
	switch s {
	case "filesystem":
		return sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM, nil
	case "sqlite":
		return sourcesv1.SourceKind_SOURCE_KIND_SQLITE, nil
	case "postgres":
		return sourcesv1.SourceKind_SOURCE_KIND_POSTGRES, nil
	case "redis":
		return sourcesv1.SourceKind_SOURCE_KIND_REDIS, nil
	case "qdrant":
		return sourcesv1.SourceKind_SOURCE_KIND_QDRANT, nil
	case "object-storage":
		return sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE, nil
	default:
		return sourcesv1.SourceKind_SOURCE_KIND_UNSPECIFIED,
			fmt.Errorf("invalid --kind %q: must be one of filesystem, sqlite, postgres, redis, qdrant, object-storage", s)
	}
}

// kindLabel renders the proto enum back to the short domain label for output.
func kindLabel(k sourcesv1.SourceKind) string {
	switch k {
	case sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM:
		return "filesystem"
	case sourcesv1.SourceKind_SOURCE_KIND_SQLITE:
		return "sqlite"
	case sourcesv1.SourceKind_SOURCE_KIND_POSTGRES:
		return "postgres"
	case sourcesv1.SourceKind_SOURCE_KIND_REDIS:
		return "redis"
	case sourcesv1.SourceKind_SOURCE_KIND_QDRANT:
		return "qdrant"
	case sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE:
		return "object-storage"
	default:
		return "unspecified"
	}
}

func formatTarget(t *targetsv1.Target) string {
	if t == nil {
		return "(nil)"
	}
	created := ""
	if t.CreatedAt != nil {
		created = t.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s/%s [kind=%s locator=%s created=%s]",
		t.Id, t.Owner, t.Name, kindLabel(t.SourceKind), t.Locator, created)
}
