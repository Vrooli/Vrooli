package backlog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/internal/backlogrank"
)

// ConnectService implements the typed BacklogService Connect contract
// (CreateItem/GetItem) that cross-scenario feedback consumers call. It wraps the
// existing *Handler so create/read go through the same Service chokepoint and
// store as the REST surface — only the transport differs. The shared generated
// proto client is what prevents the per-consumer wire drift that the old
// hand-rolled issue-tracker HTTP clients suffered.
type ConnectService struct {
	h *Handler
}

// NewConnectService builds the Connect BacklogService over an existing Handler.
func NewConnectService(h *Handler) *ConnectService { return &ConnectService{h: h} }

// registerBacklogConnectRoutes mounts the BacklogService Connect handler.
func registerBacklogConnectRoutes(router *mux.Router, h *Handler) {
	path, handler := apiconnect.NewBacklogServiceHandler(NewConnectService(h))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

// pendingQueueStatuses are the statuses an item passes through while waiting to
// be picked up. queue_position is only meaningful for these; once an item is
// in_progress / in_review / terminal / archived there is no position to report.
var pendingQueueStatuses = map[string]struct{}{
	backlogstatus.Backlog:     {},
	backlogstatus.Researching: {},
	backlogstatus.Ready:       {},
	backlogstatus.Queued:      {},
}

// CreateItem files a backlog item with creation-time dedup. If an open
// (non-archived, non-terminal) item already exists for the same target +
// signature, that item is returned with deduped=true instead of creating a
// duplicate.
func (s *ConnectService) CreateItem(ctx context.Context, req *connect.Request[apipb.CreateBacklogItemRequest]) (*connect.Response[apipb.BacklogItemResponse], error) {
	in := req.Msg
	if in == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}

	prov := identity.FromContext(ctx)
	item, err := buildItemFromCreateRequest(in, prov, s.h.validateInitiativeReference)
	if err != nil {
		var ve *CreateValidationError
		if errors.As(err, &ve) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(ve.Msg))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sig := dedupSignature(item)

	// Dedup: return an existing open item with the same signature.
	if existing, found, derr := s.findOpenBySignature(item.Kind, sig); derr != nil {
		return nil, connect.NewError(connect.CodeInternal, derr)
	} else if found {
		return connect.NewResponse(s.itemResponse(existing, true)), nil
	}

	// Embed the dedup signature as a tag so future creations collapse onto it.
	item.Tags = appendUniqueTag(item.Tags, sig)

	// Triage-class reports are pre-formed remediation stubs filed by another
	// scenario; like the auto fix-discovery path they should not spawn a
	// workshop agent on creation (refinement happens via the deep link).
	if cerr := s.h.creationService().Create(item, CreationContext{
		Context:    ctx,
		Source:     SourceFixDiscovery,
		Entrypoint: "connect.create",
	}); cerr != nil {
		return nil, mapCreateConnectError(cerr)
	}

	created, lerr := s.h.store.LoadItem(item.Kind, item.Name)
	if lerr != nil {
		// Created but couldn't reload — return what we built rather than failing.
		return connect.NewResponse(s.itemResponse(item, false)), nil
	}
	return connect.NewResponse(s.itemResponse(created, false)), nil
}

// GetItem returns a single backlog item with its computed queue_position.
func (s *ConnectService) GetItem(ctx context.Context, req *connect.Request[apipb.GetBacklogItemRequest]) (*connect.Response[apipb.BacklogItemResponse], error) {
	in := req.Msg
	if in == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	kind, err := ParseBacklogKind(strings.TrimSpace(in.Kind))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	name := sanitizeName(strings.TrimSpace(in.Name))
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	item, err := s.h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("backlog item not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(s.itemResponse(item, false)), nil
}

// itemResponse builds the proto response, attaching the computed queue_position
// when the item is pending.
func (s *ConnectService) itemResponse(item BacklogItem, deduped bool) *apipb.BacklogItemResponse {
	proto := backlogToProto(item)
	if pos, ok := s.computeQueuePosition(item); ok {
		proto.QueuePosition = &pos
	}
	return &apipb.BacklogItemResponse{Item: proto, Deduped: deduped}
}

// computeQueuePosition returns the zero-based index of item within the ranked
// pending set (backlogrank order), and whether the item is pending at all.
func (s *ConnectService) computeQueuePosition(item BacklogItem) (int32, bool) {
	if _, pending := pendingQueueStatuses[string(item.Status)]; !pending {
		return 0, false
	}
	all, err := s.h.store.LoadAll(nil)
	if err != nil {
		return 0, false
	}

	rankItems := make([]backlogrank.Item, 0, len(all))
	for _, it := range all {
		rankItems = append(rankItems, toRankItem(it))
	}
	depthMap := backlogrank.ComputeDepthMap(rankItems)
	unblockingMap := backlogrank.ComputeUnblockingMap(rankItems)

	// Restrict to the pending subset, then sort by ranking order.
	pending := make([]backlogrank.Item, 0, len(rankItems))
	for _, ri := range rankItems {
		if _, ok := pendingQueueStatuses[ri.Status]; ok && !ri.Archived {
			pending = append(pending, ri)
		}
	}
	sort.SliceStable(pending, func(i, j int) bool {
		return backlogrank.Less(pending[i], pending[j], depthMap, unblockingMap)
	})

	targetKey := backlogrank.Key(string(item.Kind), item.Name)
	for idx, ri := range pending {
		if backlogrank.ItemKey(ri) == targetKey {
			return int32(idx), true
		}
	}
	return 0, false
}

// findOpenBySignature returns the first non-archived, non-terminal item of the
// given kind carrying the dedup signature tag.
func (s *ConnectService) findOpenBySignature(kind BacklogKind, sig string) (BacklogItem, bool, error) {
	items, err := s.h.store.LoadAll([]BacklogKind{kind})
	if err != nil {
		return BacklogItem{}, false, err
	}
	for _, it := range items {
		if it.ArchivedAt != nil && strings.TrimSpace(*it.ArchivedAt) != "" {
			continue
		}
		if backlogstatus.IsTerminal(string(it.Status)) {
			continue
		}
		for _, tag := range it.Tags {
			if tag == sig {
				return it, true, nil
			}
		}
	}
	return BacklogItem{}, false, nil
}

// toRankItem converts a BacklogItem into a backlogrank.Item.
func toRankItem(item BacklogItem) backlogrank.Item {
	archived := item.ArchivedAt != nil && strings.TrimSpace(*item.ArchivedAt) != ""
	return backlogrank.Item{
		Kind:      string(item.Kind),
		Name:      item.Name,
		Status:    string(item.Status),
		DependsOn: item.DependsOn,
		Archived:  archived,
		Priority:  item.Priority,
		UpdatedAt: parseBacklogUpdatedAt(item.Updated),
	}
}

// dedupSignature derives a stable "sig:<hash>" tag from the fields that
// identify a duplicate report: kind + target (acceptance_allow) + normalized
// title + origin. Conservative by design — when any of these differ a new item
// is created rather than collapsing distinct problems.
func dedupSignature(item BacklogItem) string {
	allow := append([]string(nil), item.AcceptanceAllow...)
	sort.Strings(allow)

	origin := ""
	for _, tag := range item.Tags {
		if strings.HasPrefix(tag, "origin:") {
			origin = tag
			break
		}
	}

	h := sha256.New()
	h.Write([]byte(string(item.Kind)))
	h.Write([]byte{0})
	h.Write([]byte(strings.ToLower(strings.TrimSpace(item.Title))))
	h.Write([]byte{0})
	h.Write([]byte(strings.Join(allow, "|")))
	h.Write([]byte{0})
	h.Write([]byte(origin))
	return "sig:" + hex.EncodeToString(h.Sum(nil))[:12]
}

// appendUniqueTag appends tag unless already present.
func appendUniqueTag(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	return append(tags, tag)
}

// mapCreateConnectError translates Service.Create errors into Connect codes.
func mapCreateConnectError(err error) error {
	switch {
	case errors.Is(err, ErrItemExists):
		return connect.NewError(connect.CodeAlreadyExists, errors.New("backlog item already exists"))
	case strings.HasPrefix(err.Error(), "depends_on:") ||
		strings.HasPrefix(err.Error(), "dependency cycle"):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, errors.New("failed to create backlog item"))
	}
}
