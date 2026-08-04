package discovery

import (
	"context"
	"log"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"

	"connectrpc.com/connect"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
)

// Deps wires the seams the Connect discovery handler needs.
type Deps struct {
	Service discovery.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the discovery Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListTargetSuggestions(ctx context.Context, _ *connect.Request[discoveryv1.ListTargetSuggestionsRequest]) (*connect.Response[discoveryv1.ListTargetSuggestionsResponse], error) {
	list, err := h.deps.Service.ListTargetSuggestions(ctx)
	if err != nil {
		return nil, h.translate("ListTargetSuggestions", err)
	}
	resp := &discoveryv1.ListTargetSuggestionsResponse{Suggestions: make([]*discoveryv1.TargetSuggestion, 0, len(list))}
	for _, s := range list {
		resp.Suggestions = append(resp.Suggestions, targetToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListDestinationSuggestions(ctx context.Context, _ *connect.Request[discoveryv1.ListDestinationSuggestionsRequest]) (*connect.Response[discoveryv1.ListDestinationSuggestionsResponse], error) {
	list, err := h.deps.Service.ListDestinationSuggestions(ctx)
	if err != nil {
		return nil, h.translate("ListDestinationSuggestions", err)
	}
	resp := &discoveryv1.ListDestinationSuggestionsResponse{Suggestions: make([]*discoveryv1.DestinationSuggestion, 0, len(list))}
	for _, s := range list {
		resp.Suggestions = append(resp.Suggestions, destinationToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DismissSuggestion(ctx context.Context, req *connect.Request[discoveryv1.DismissSuggestionRequest]) (*connect.Response[discoveryv1.DismissSuggestionResponse], error) {
	dismissed, err := h.deps.Service.DismissSuggestion(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("DismissSuggestion", err)
	}
	return connect.NewResponse(&discoveryv1.DismissSuggestionResponse{Dismissed: dismissed}), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := discovery.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("discovery.%s: %v", op, err)
	}
	return connectErr
}

// targetToProto converts an internal target suggestion to its wire shape.
func targetToProto(s discovery.TargetSuggestion) *discoveryv1.TargetSuggestion {
	return &discoveryv1.TargetSuggestion{
		Id:          s.ID,
		Owner:       s.Owner,
		Name:        s.Name,
		SourceKind:  kindToProto(s.SourceKind),
		Locator:     s.Locator,
		Rationale:   s.Rationale,
		ApproxBytes: s.ApproxBytes,
		Sensitive:   s.Sensitive,
		Warning:     targetWarning(s),
	}
}

// sensitiveWarning returns the operator-facing warning for a sensitive
// suggestion (empty otherwise). Sensitive suggestions are surfaced but never
// auto-accepted — registering one is a deliberate operator action.
func sensitiveWarning(sensitive bool) string {
	if !sensitive {
		return ""
	}
	return "Includes credentials/tokens — review before backing up; restoring stale tokens can silently break auth."
}

func targetWarning(s discovery.TargetSuggestion) string {
	warning := sensitiveWarning(s.Sensitive)
	if len(s.Findings) == 0 {
		return warning
	}
	if warning == "" {
		return "Owner storage metadata has declaration findings; review before registering this target."
	}
	return warning + " Owner storage metadata also has declaration findings; review before registering."
}

// destinationToProto converts an internal destination suggestion to its wire
// shape. Discovered volumes are always filesystem-backed destinations.
func destinationToProto(s discovery.DestinationSuggestion) *discoveryv1.DestinationSuggestion {
	return &discoveryv1.DestinationSuggestion{
		Id:             s.ID,
		Label:          s.Label,
		BackendKind:    destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM,
		Location:       s.Location,
		DriveClass:     classToProto(s.Class),
		FreeBytes:      s.FreeBytes,
		TotalBytes:     s.TotalBytes,
		Removable:      s.Removable,
		SeparateRootOk: s.SeparateRootOK,
		Rationale:      s.Rationale,
	}
}

// classToProto translates the domain DriveClass to the proto enum so domain
// code never imports the generated enum.
func classToProto(c discovery.DriveClass) discoveryv1.DriveClass {
	switch c {
	case sysmounts.ClassRemovable:
		return discoveryv1.DriveClass_DRIVE_CLASS_REMOVABLE
	case sysmounts.ClassFixed:
		return discoveryv1.DriveClass_DRIVE_CLASS_FIXED
	case sysmounts.ClassNetwork:
		return discoveryv1.DriveClass_DRIVE_CLASS_NETWORK
	default:
		return discoveryv1.DriveClass_DRIVE_CLASS_UNSPECIFIED
	}
}

// kindToProto mirrors the targets handler's translation (kept local so the
// discovery handler does not import the targets handler package).
func kindToProto(k sources.SourceKind) sourcesv1.SourceKind {
	switch k {
	case sources.KindFilesystem:
		return sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM
	case sources.KindSQLite:
		return sourcesv1.SourceKind_SOURCE_KIND_SQLITE
	case sources.KindPostgres:
		return sourcesv1.SourceKind_SOURCE_KIND_POSTGRES
	case sources.KindRedis:
		return sourcesv1.SourceKind_SOURCE_KIND_REDIS
	case sources.KindQdrant:
		return sourcesv1.SourceKind_SOURCE_KIND_QDRANT
	case sources.KindObjectStorage:
		return sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE
	default:
		return sourcesv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}
