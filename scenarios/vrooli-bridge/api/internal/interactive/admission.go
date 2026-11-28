// Package interactive contains Bridge's transport-only desktop admission
// policy. Device Control remains the authority for native desktop actions.
package interactive

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const MaxSignalBytes = 256 * 1024

var (
	ErrUnauthorized = errors.New("interactive channel is unauthorized")
	ErrViewerWrite  = errors.New("viewer cannot open a controller channel")
	ErrRevoked      = errors.New("interactive channel is revoked")
	ErrStale        = errors.New("interactive channel grant is stale")
	ErrSignalLimit  = errors.New("interactive signal exceeds limit")
	ErrProtocol     = errors.New("interactive signal uses an unsupported lane")
	ErrInvalid      = errors.New("interactive channel request is invalid")
)

type Admission struct {
	mu       sync.Mutex
	now      func() time.Time
	routes   func() []*interactivev1.RouteCandidate
	nodes    map[string]bool
	channels map[string]*interactivev1.ChannelGrant
	queued   map[string]int
}

func NewAdmission() *Admission {
	return &Admission{now: func() time.Time { return time.Now().UTC() }, routes: func() []*interactivev1.RouteCandidate {
		return []*interactivev1.RouteCandidate{{Kind: interactivev1.RouteKind_ROUTE_KIND_DIRECT, Priority: 1}}
	}, nodes: make(map[string]bool), channels: make(map[string]*interactivev1.ChannelGrant), queued: make(map[string]int)}
}

// SetRoutes supplies deployment-owned ICE route metadata. TURN credentials
// are short-lived and returned only in the open response; Admission never
// persists or logs them. A nil provider restores the direct-only default.
func (a *Admission) SetRoutes(provider func() []*interactivev1.RouteCandidate) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if provider == nil {
		a.routes = func() []*interactivev1.RouteCandidate {
			return []*interactivev1.RouteCandidate{{Kind: interactivev1.RouteKind_ROUTE_KIND_DIRECT, Priority: 1}}
		}
		return
	}
	a.routes = provider
}

func (a *Admission) SetNode(nodeID string, authorized bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodes[strings.TrimSpace(nodeID)] = authorized
}

func (a *Admission) Open(actor string, req *interactivev1.OpenChannelRequest) (*interactivev1.OpenChannelResponse, error) {
	if req == nil || strings.TrimSpace(actor) == "" || strings.TrimSpace(req.GetNodeId()) == "" || strings.TrimSpace(req.GetSessionId()) == "" || strings.TrimSpace(req.GetLeaseId()) == "" || req.GetLeaseEpoch() == 0 || req.GetProtocol() != interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8 || req.GetSurface() == nil || req.GetSurface().GetTarget() == nil || strings.TrimSpace(req.GetSurface().GetTarget().GetResourceId()) == "" {
		return nil, ErrInvalid
	}
	if req.GetRole() != interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER && req.GetRole() != interactivev1.ChannelRole_CHANNEL_ROLE_VIEWER {
		return nil, ErrInvalid
	}
	if req.GetRole() != interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER && req.GetTakeover() {
		return nil, ErrInvalid
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.nodes[req.GetNodeId()] {
		return nil, ErrUnauthorized
	}
	if req.GetRole() == interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER && !req.GetTakeover() {
		for _, grant := range a.channels {
			if sameTarget(grant, req) && grant.GetRole() == interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER && grant.GetExpiresAt().AsTime().After(a.now()) {
				return nil, ErrViewerWrite
			}
		}
	}
	if req.GetRole() == interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER && req.GetTakeover() {
		for _, grant := range a.channels {
			if sameTarget(grant, req) && grant.GetRole() == interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER {
				grant.Role = interactivev1.ChannelRole_CHANNEL_ROLE_VIEWER
			}
		}
	}
	now := a.now()
	ttl := time.Duration(req.GetTtlSeconds()) * time.Second
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	id := fmt.Sprintf("channel-%d", now.UnixNano())
	routes := a.routes
	if routes == nil {
		routes = func() []*interactivev1.RouteCandidate { return nil }
	}
	grant := &interactivev1.ChannelGrant{ChannelId: id, NodeId: req.GetNodeId(), Surface: req.GetSurface(), SessionId: req.GetSessionId(), LeaseId: req.GetLeaseId(), LeaseEpoch: req.GetLeaseEpoch(), Protocol: req.GetProtocol(), Role: req.GetRole(), ExpiresAt: timestamp(now.Add(ttl)), PolicyRevision: actor, Routes: cloneRoutes(routes())}
	a.channels[id] = grant
	return &interactivev1.OpenChannelResponse{Grant: cloneGrant(grant), Routes: cloneRoutes(grant.GetRoutes())}, nil
}

func sameTarget(grant *interactivev1.ChannelGrant, req *interactivev1.OpenChannelRequest) bool {
	return grant != nil && req != nil && grant.GetNodeId() == req.GetNodeId() &&
		grant.GetSurface().GetTarget().GetResourceId() == req.GetSurface().GetTarget().GetResourceId()
}

func (a *Admission) Signal(req *interactivev1.SignalRequest) (*interactivev1.SignalResponse, error) {
	if req == nil || req.GetGrant() == nil || req.GetRequestId() == "" || req.GetGeneration() == "" || len(req.GetPayload()) == 0 {
		return nil, ErrInvalid
	}
	if len(req.GetPayload()) > MaxSignalBytes {
		return nil, ErrSignalLimit
	}
	if req.GetKind() != interactivev1.SignalKind_SIGNAL_KIND_OFFER && req.GetKind() != interactivev1.SignalKind_SIGNAL_KIND_ICE {
		return nil, ErrProtocol
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	submitted := req.GetGrant()
	grant, ok := a.channels[submitted.GetChannelId()]
	if !ok || grant.GetLeaseEpoch() != submitted.GetLeaseEpoch() || grant.GetNodeId() != submitted.GetNodeId() || grant.GetSessionId() != submitted.GetSessionId() || grant.GetLeaseId() != submitted.GetLeaseId() || grant.GetProtocol() != submitted.GetProtocol() || grant.GetRole() != submitted.GetRole() || !proto.Equal(grant.GetSurface(), submitted.GetSurface()) {
		return nil, ErrStale
	}
	if grant.GetExpiresAt().AsTime().Before(a.now()) || !a.nodes[grant.GetNodeId()] {
		return nil, ErrRevoked
	}
	if a.queued[grant.GetChannelId()] >= 64 {
		return nil, ErrSignalLimit
	}
	a.queued[grant.GetChannelId()]++
	return &interactivev1.SignalResponse{Accepted: true, Generation: req.GetGeneration()}, nil
}

// ReleaseSignal accounts for a response, timeout, or transport failure. The
// broker calls this after the bounded pending request has left its waiter.
func (a *Admission) ReleaseSignal(channelID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if depth := a.queued[channelID]; depth > 1 {
		a.queued[channelID] = depth - 1
	} else {
		delete(a.queued, channelID)
	}
}

func (a *Admission) Revoke(channelID string) bool {
	_, ok := a.RevokeGrant(channelID)
	return ok
}

func (a *Admission) RevokeGrant(channelID string) (*interactivev1.ChannelGrant, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if grant, ok := a.channels[channelID]; ok {
		delete(a.channels, channelID)
		delete(a.queued, channelID)
		return cloneGrant(grant), true
	}
	return nil, false
}

func (a *Admission) RevokeNode(nodeID string) int {
	return len(a.RevokeNodeGrants(nodeID))
}

func (a *Admission) RevokeNodeGrants(nodeID string) []*interactivev1.ChannelGrant {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodes[nodeID] = false
	grants := make([]*interactivev1.ChannelGrant, 0)
	for id, grant := range a.channels {
		if grant.GetNodeId() == nodeID {
			delete(a.channels, id)
			delete(a.queued, id)
			grants = append(grants, cloneGrant(grant))
		}
	}
	return grants
}

func (a *Admission) QueueDepth(channelID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.queued[channelID]
}

func cloneGrant(in *interactivev1.ChannelGrant) *interactivev1.ChannelGrant {
	out := proto.Clone(in).(*interactivev1.ChannelGrant)
	out.Routes = cloneRoutes(in.GetRoutes())
	return out
}

func cloneRoutes(in []*interactivev1.RouteCandidate) []*interactivev1.RouteCandidate {
	out := make([]*interactivev1.RouteCandidate, 0, len(in))
	for _, route := range in {
		if route == nil {
			continue
		}
		out = append(out, proto.Clone(route).(*interactivev1.RouteCandidate))
	}
	return out
}
func timestamp(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
