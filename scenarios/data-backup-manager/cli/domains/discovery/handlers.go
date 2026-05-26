package discovery

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery/discovery_v1connect"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client discoveryconnect.DiscoveryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: discoveryconnect.NewDiscoveryServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) targets(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTargetSuggestions(context.Background(), connect.NewRequest(&discoveryv1.ListTargetSuggestionsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list target suggestions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no target-suggestions response")
	}
	results := make([]string, 0, len(resp.Msg.Suggestions))
	for _, s := range resp.Msg.Suggestions {
		results = append(results, formatTargetSuggestion(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d target suggestion(s) worth protecting.", len(resp.Msg.Suggestions))},
		ResultsHeading: "Suggested targets",
		Results:        results,
		RetrievalHints: []string{
			"`targets register --owner <owner> --name <name> --kind <kind> --locator <locator>` — enable a suggestion",
			"`discovery dismiss --id <id>` — hide a suggestion permanently",
		},
	})
}

func (h *handlers) destinations(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDestinationSuggestions(context.Background(), connect.NewRequest(&discoveryv1.ListDestinationSuggestionsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list destination suggestions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no destination-suggestions response")
	}
	results := make([]string, 0, len(resp.Msg.Suggestions))
	for _, s := range resp.Msg.Suggestions {
		results = append(results, formatDestinationSuggestion(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d destination suggestion(s).", len(resp.Msg.Suggestions))},
		ResultsHeading: "Suggested destinations",
		Results:        results,
		RetrievalHints: []string{
			"`destinations create --name <name> --backend filesystem --location <location>` — enable a suggestion",
			"`discovery dismiss --id <id>` — hide a suggestion permanently",
		},
	})
}

func (h *handlers) dismiss(ctx cliapp.RunContext) error {
	id := ctx.Flag("id")
	resp, err := h.client.DismissSuggestion(context.Background(), connect.NewRequest(&discoveryv1.DismissSuggestionRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("dismiss suggestion %q", id), err, nil)
	}
	msg := "No matching suggestion to dismiss."
	if resp != nil && resp.Msg != nil && resp.Msg.Dismissed {
		msg = fmt.Sprintf("Dismissed suggestion %s — it won't appear again.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

func formatTargetSuggestion(s *discoveryv1.TargetSuggestion) string {
	if s == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s/%s [kind=%s locator=%s size=%s] — %s",
		s.Id, s.Owner, s.Name, kindLabel(s.SourceKind), s.Locator, humanizeBytes(s.ApproxBytes), s.Rationale)
}

func formatDestinationSuggestion(s *discoveryv1.DestinationSuggestion) string {
	if s == nil {
		return "(nil)"
	}
	safe := "ok"
	if !s.SeparateRootOk {
		safe = "UNSAFE (overlaps protected data — cannot enable)"
	}
	return fmt.Sprintf("%s — %s [%s free=%s total=%s separate-root=%s] — %s",
		s.Id, s.Location, classLabel(s.DriveClass), humanizeBytes(s.FreeBytes), humanizeBytes(s.TotalBytes), safe, s.Rationale)
}

func classLabel(c discoveryv1.DriveClass) string {
	switch c {
	case discoveryv1.DriveClass_DRIVE_CLASS_REMOVABLE:
		return "removable"
	case discoveryv1.DriveClass_DRIVE_CLASS_FIXED:
		return "fixed"
	case discoveryv1.DriveClass_DRIVE_CLASS_NETWORK:
		return "network"
	default:
		return "unknown"
	}
}

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

// humanizeBytes renders a byte count as a short human string.
func humanizeBytes(b int64) string {
	if b <= 0 {
		return "?"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
