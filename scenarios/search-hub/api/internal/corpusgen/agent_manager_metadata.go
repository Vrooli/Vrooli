package corpusgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

// AgentManagerCallerSource is a prompt-free source record for caller-corpus
// discovery. Query text, assistant messages, and tool contents are absent by
// construction. A label review and a caller query are still required before it
// can become an evaluation case.
type AgentManagerCallerSource struct {
	RunID            string    `json:"run_id"`
	EventID          string    `json:"event_id"`
	Sequence         int64     `json:"sequence"`
	Timestamp        time.Time `json:"timestamp"`
	MessageRole      string    `json:"message_role"`
	MessageSource    string    `json:"message_source"`
	RequestID        string    `json:"request_id,omitempty"`
	Consumer         string    `json:"consumer,omitempty"`
	PolicyRole       string    `json:"policy_role,omitempty"`
	Model            string    `json:"model,omitempty"`
	CapacityEvidence string    `json:"capacity_evidence,omitempty"`
}

// AgentManagerMetadataPage is the owner API envelope; no event payload is legal.
type AgentManagerMetadataPage struct {
	RunID        string                      `json:"run_id"`
	Events       []AgentManagerEventMetadata `json:"events"`
	HasMore      bool                        `json:"has_more"`
	MetadataOnly bool                        `json:"metadata_only"`
}

// AgentManagerEventMetadata contains stable identity and allowlisted scalars.
type AgentManagerEventMetadata struct {
	EventID          string    `json:"event_id"`
	RunID            string    `json:"run_id"`
	Sequence         int64     `json:"sequence"`
	EventType        string    `json:"event_type"`
	Timestamp        time.Time `json:"timestamp"`
	Role             string    `json:"role"`
	Source           string    `json:"source"`
	RequestID        string    `json:"request_id"`
	Consumer         string    `json:"consumer"`
	PolicyRole       string    `json:"policy_role"`
	Model            string    `json:"model"`
	CapacityEvidence string    `json:"capacity_evidence"`
}

// AgentManagerMetadataReader is the owner-supported paged, payload-free API.
type AgentManagerMetadataReader interface {
	GetRunEventMetadata(context.Context, string, int64, int) (AgentManagerMetadataPage, error)
}

// AgentManagerMetadataHTTPReader reads only
// GET /api/v1/runs/{id}/events/metadata. Agent Manager's service proto has no
// RPC for this privacy-filtered source projection, so this specific REST read
// remains; responses that do not assert the payload-free contract are rejected.
type AgentManagerMetadataHTTPReader struct {
	BaseURL string
	Client  *http.Client
}

func (r AgentManagerMetadataHTTPReader) GetRunEventMetadata(ctx context.Context, runID string, after int64, limit int) (AgentManagerMetadataPage, error) {
	var page AgentManagerMetadataPage
	if r.Client == nil || strings.TrimSpace(r.BaseURL) == "" || strings.TrimSpace(runID) == "" || limit < 1 || limit > 1000 {
		return page, fmt.Errorf("Agent Manager metadata URL, authenticated client, run ID and bounded page size are required")
	}
	base, err := url.Parse(strings.TrimRight(r.BaseURL, "/") + "/api/v1/runs/" + url.PathEscape(runID) + "/events/metadata")
	if err != nil {
		return page, fmt.Errorf("parse Agent Manager metadata URL: %w", err)
	}
	query := base.Query()
	query.Set("after_sequence", fmt.Sprint(after))
	query.Set("limit", fmt.Sprint(limit))
	base.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return page, fmt.Errorf("build Agent Manager metadata request: %w", err)
	}
	response, err := r.Client.Do(req)
	if err != nil {
		return page, fmt.Errorf("request Agent Manager metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return page, fmt.Errorf("Agent Manager metadata request returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&page); err != nil {
		return AgentManagerMetadataPage{}, fmt.Errorf("decode Agent Manager metadata: %w", err)
	}
	if !page.MetadataOnly || page.RunID != runID {
		return AgentManagerMetadataPage{}, fmt.Errorf("Agent Manager response did not verify the metadata-only run identity")
	}
	return page, nil
}

// HarvestAgentManagerCallerSources pages selected runs through the metadata
// projection and retains only exact operator-authored user-message identities.
// It does not fetch event pages, query local databases, or retain message text.
func HarvestAgentManagerCallerSources(ctx context.Context, reader AgentManagerMetadataReader, runIDs []string, pageSize int) ([]AgentManagerCallerSource, error) {
	if reader == nil || len(runIDs) == 0 || pageSize < 1 || pageSize > 1000 {
		return nil, fmt.Errorf("metadata reader, run IDs and a bounded page size are required")
	}
	runs := append([]string(nil), runIDs...)
	sort.Strings(runs)
	seen := make(map[string]struct{}, len(runs))
	seenEventIDs := make(map[string]struct{})
	var result []AgentManagerCallerSource
	for _, runID := range runs {
		if strings.TrimSpace(runID) == "" {
			return nil, fmt.Errorf("run IDs must be non-empty and unique")
		}
		if _, ok := seen[runID]; ok {
			return nil, fmt.Errorf("run IDs must be non-empty and unique")
		}
		seen[runID] = struct{}{}
		var after int64 = -1
		for pageNo := 0; pageNo < 1000; pageNo++ {
			page, err := reader.GetRunEventMetadata(ctx, runID, after, pageSize)
			if err != nil {
				return nil, fmt.Errorf("read Agent Manager metadata for run %s: %w", runID, err)
			}
			if !page.MetadataOnly || page.RunID != runID {
				return nil, fmt.Errorf("Agent Manager response did not verify the metadata-only run identity")
			}
			last := after
			for _, event := range page.Events {
				if event.RunID != runID || event.EventID == "" || event.Sequence <= last || event.Timestamp.IsZero() {
					return nil, fmt.Errorf("Agent Manager metadata contains an invalid or unstable event identity")
				}
				if _, exists := seenEventIDs[event.EventID]; exists {
					return nil, fmt.Errorf("Agent Manager metadata contains a duplicate event identity")
				}
				seenEventIDs[event.EventID] = struct{}{}
				last = event.Sequence
				if event.EventType != "message" || !strings.EqualFold(event.Role, "user") || event.Source != "operator" {
					continue
				}
				result = append(result, AgentManagerCallerSource{
					RunID: event.RunID, EventID: event.EventID, Sequence: event.Sequence,
					Timestamp: event.Timestamp.UTC(), MessageRole: strings.ToLower(event.Role),
					MessageSource: event.Source, RequestID: event.RequestID, Consumer: event.Consumer,
					PolicyRole: event.PolicyRole, Model: event.Model, CapacityEvidence: event.CapacityEvidence,
				})
			}
			if !page.HasMore {
				break
			}
			if last <= after {
				return nil, fmt.Errorf("Agent Manager metadata page did not advance its stable sequence cursor")
			}
			after = last
			if pageNo == 999 {
				return nil, fmt.Errorf("Agent Manager metadata page limit exceeded")
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			return result[i].EventID < result[j].EventID
		}
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

// MatchAgentManagerCallerSource binds a metadata source to an executed Gateway
// receipt. Missing and mismatched joins are explicit; neither request text nor
// an inferred model role can make a case eligible.
func MatchAgentManagerCallerSource(source AgentManagerCallerSource, route GatewayRouteProof) (string, error) {
	if source.RunID == "" || source.EventID == "" || source.Sequence < 0 || source.Timestamp.IsZero() || source.MessageRole != "user" || source.MessageSource != "operator" {
		return "invalid_source_identity", fmt.Errorf("caller source identity is incomplete")
	}
	if route.SourceRunID != source.RunID || route.SourceEventID != source.EventID || route.SourceSequence != uint64(source.Sequence) {
		return "unmatched_source_identity", nil
	}
	requestID, err := searcheval.AgentManagerReplayRequestID(source.RunID, source.EventID, source.Sequence, route.TaskRole, route.Consumer)
	if err != nil || requestID != route.GatewayRequestID || (source.RequestID != "" && source.RequestID != route.GatewayRequestID) {
		return "unmatched_request_identity", nil
	}
	if (source.Consumer != "" && source.Consumer != route.Consumer) ||
		(source.PolicyRole != "" && source.PolicyRole != route.SelectedResourceRole) ||
		(source.Model != "" && source.Model != route.SelectedModel) ||
		(source.CapacityEvidence != "" && source.CapacityEvidence != route.CapacityEstimateEvidenceURI) {
		return "unmatched_metadata_identity", nil
	}
	if route.GatewayEventID == "" || route.GatewayRoleEvidenceURI != "gateway-ledger://events/"+route.GatewayEventID ||
		route.Status != "succeeded" || route.Locality != "local" || route.SelectedProvider == "" || route.SelectedModel == "" || !isModelPolicyRole(route.SelectedResourceRole) ||
		route.CapacityVerdict != "fit" || route.CapacityClaimID == "" || route.CapacityRequiredBytes <= 0 || route.CapacityGrantedBytes < route.CapacityRequiredBytes || !strings.HasPrefix(route.CapacityEstimateEvidenceURI, "capacity-footprint://") {
		return "unmatched_route_evidence", nil
	}
	return "matched", nil
}

type AgentManagerCallerSourceJoin struct {
	Source *AgentManagerCallerSource `json:"source,omitempty"`
	Route  *GatewayRouteProof        `json:"route,omitempty"`
	Status string                    `json:"status"`
}

// JoinAgentManagerCallerSources reports every source and route row, including
// unmatched rows. Duplicate event identities are rejected because choosing
// one would make caller coverage depend on enumeration order.
func JoinAgentManagerCallerSources(sources []AgentManagerCallerSource, routes []GatewayRouteProof) ([]AgentManagerCallerSourceJoin, error) {
	routeBySourceRequest := make(map[string]GatewayRouteProof, len(routes))
	routeKeysBySource := make(map[string][]string, len(routes))
	for _, route := range routes {
		if route.SourceRunID == "" || route.SourceEventID == "" || route.SourceSequence > uint64(1<<63-1) || route.GatewayRequestID == "" {
			return nil, fmt.Errorf("Gateway route receipt is missing a valid Agent Manager source/request identity")
		}
		eventKey := sourceJoinKey(route.SourceRunID, route.SourceEventID, int64(route.SourceSequence))
		key := sourceRequestJoinKey(eventKey, route.GatewayRequestID)
		if _, exists := routeBySourceRequest[key]; exists {
			return nil, fmt.Errorf("Gateway route receipts contain duplicate source/request identity")
		}
		routeBySourceRequest[key] = route
		routeKeysBySource[eventKey] = append(routeKeysBySource[eventKey], key)
	}
	sourceKeys := make(map[string]struct{}, len(sources))
	usedRoutes := make(map[string]struct{}, len(routes))
	out := make([]AgentManagerCallerSourceJoin, 0, len(sources)+len(routes))
	for i := range sources {
		source := sources[i]
		eventKey := sourceJoinKey(source.RunID, source.EventID, source.Sequence)
		if _, exists := sourceKeys[eventKey]; exists {
			return nil, fmt.Errorf("Agent Manager caller sources contain duplicate event identity")
		}
		sourceKeys[eventKey] = struct{}{}
		routeKeys := routeKeysBySource[eventKey]
		if source.RequestID == "" && len(routeKeys) > 1 {
			out = append(out, AgentManagerCallerSourceJoin{Source: &source, Status: "ambiguous_gateway_receipts"})
			continue
		}
		key := ""
		if source.RequestID != "" {
			key = sourceRequestJoinKey(eventKey, source.RequestID)
		} else if len(routeKeys) == 1 {
			key = routeKeys[0]
		}
		route, ok := routeBySourceRequest[key]
		if !ok {
			out = append(out, AgentManagerCallerSourceJoin{Source: &source, Status: "unmatched_gateway_receipt"})
			continue
		}
		usedRoutes[key] = struct{}{}
		status, err := MatchAgentManagerCallerSource(source, route)
		if err != nil {
			return nil, err
		}
		routeCopy := route
		out = append(out, AgentManagerCallerSourceJoin{Source: &source, Route: &routeCopy, Status: status})
	}
	for key, route := range routeBySourceRequest {
		if _, ok := usedRoutes[key]; ok {
			continue
		}
		routeCopy := route
		status := "unmatched_agent_manager_source"
		if _, ok := sourceKeys[sourceJoinKey(route.SourceRunID, route.SourceEventID, int64(route.SourceSequence))]; ok {
			status = "unmatched_request_identity"
		}
		out = append(out, AgentManagerCallerSourceJoin{Route: &routeCopy, Status: status})
	}
	sort.Slice(out, func(i, j int) bool {
		left := joinSortKey(out[i])
		right := joinSortKey(out[j])
		return left < right
	})
	return out, nil
}

func sourceJoinKey(runID, eventID string, sequence int64) string {
	return fmt.Sprintf("%s\x00%s\x00%d", runID, eventID, sequence)
}

func sourceRequestJoinKey(eventKey, requestID string) string {
	return eventKey + "\x00" + requestID
}

func joinSortKey(join AgentManagerCallerSourceJoin) string {
	if join.Source != nil {
		return sourceJoinKey(join.Source.RunID, join.Source.EventID, join.Source.Sequence) + "\x00" + join.Source.RequestID + "\x00" + join.Status
	}
	if join.Route != nil {
		return sourceJoinKey(join.Route.SourceRunID, join.Route.SourceEventID, int64(join.Route.SourceSequence)) + "\x00" + join.Route.GatewayRequestID + "\x00" + join.Status
	}
	return ""
}
