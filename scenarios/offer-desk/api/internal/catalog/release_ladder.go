package catalog

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Store) SetReleaseRank(ctx context.Context, nodeID string, rank int32, actor string) (*offerspb.Node, int32, error) {
	if strings.TrimSpace(nodeID) == "" || rank < 0 {
		return nil, 0, errors.New("release rank requires a node id and a non-negative rank")
	}
	if strings.TrimSpace(actor) == "" {
		return nil, 0, errors.New("release rank requires an actor")
	}
	var n offerspb.Node
	var kind, status int32
	var created string
	if err := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank FROM nodes WHERE id=?`, nodeID).
		Scan(&n.Id, &kind, &n.Name, &status, &n.TriggerId, &created, &n.ActualAccountId, &n.ReleaseRank); err != nil {
		return nil, 0, fmt.Errorf("node %q not found: %w", nodeID, err)
	}
	n.Kind, n.Status = offerspb.NodeKind(kind), offerspb.Status(status)
	if n.Kind != offerspb.NodeKind_DELIVERABLE {
		return nil, 0, errors.New("release rank applies only to deliverable nodes")
	}
	if rank > 0 {
		var other string
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE kind=? AND release_rank=? AND id<>? AND status<>? LIMIT 1`, int32(offerspb.NodeKind_DELIVERABLE), rank, nodeID, int32(offerspb.Status_RETIRED)).Scan(&other); err == nil {
			return nil, 0, fmt.Errorf("release rank %d is already assigned to deliverable %s", rank, other)
		}
	}
	prior := n.ReleaseRank
	if prior == rank {
		return &n, prior, nil
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE nodes SET release_rank=? WHERE id=?`, rank, nodeID); err != nil {
		return nil, 0, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`,
		uuid.NewString(), nodeID, actor, status, status, fmt.Sprintf("release rank changed from %d to %d", prior, rank), s.now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, 0, err
	}
	n.ReleaseRank = rank
	if parsed, err := time.Parse(time.RFC3339Nano, created); err == nil {
		n.CreatedAt = timestamppb.New(parsed)
	}
	return &n, prior, nil
}

func (s *Store) ReleaseLadder(ctx context.Context, includeRetired bool) (*offerspb.ReleaseLadderResponse, error) {
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*offerspb.Node, len(nodes))
	for _, n := range nodes {
		byID[n.Id] = n
	}
	response := &offerspb.ReleaseLadderResponse{}
	for _, n := range nodes {
		switch n.Kind {
		case offerspb.NodeKind_RAMP:
			response.Ramps = append(response.Ramps, n)
		case offerspb.NodeKind_STREAM:
			response.Streams = append(response.Streams, n)
		case offerspb.NodeKind_AUDIENCE:
			response.Audiences = append(response.Audiences, n)
		}
	}
	for _, n := range nodes {
		if n.Kind != offerspb.NodeKind_DELIVERABLE || n.ReleaseRank <= 0 || (!includeRetired && n.Status == offerspb.Status_RETIRED) {
			continue
		}
		entry := &offerspb.ReleaseLadderEntry{Deliverable: n}
		edges, err := s.ListEdges(ctx, n.Id)
		if err != nil {
			return nil, err
		}
		for _, e := range edges {
			if e.FromId != n.Id {
				continue
			}
			target := byID[e.ToId]
			if target == nil {
				continue
			}
			switch e.Kind {
			case "unlocks":
				if target.Kind == offerspb.NodeKind_RAMP {
					entry.UnlockedRamps = append(entry.UnlockedRamps, target)
				}
				if target.Kind == offerspb.NodeKind_STREAM {
					entry.UnlockedStreams = append(entry.UnlockedStreams, target)
				}
			case "serves":
				if target.Kind == offerspb.NodeKind_AUDIENCE {
					entry.Audiences = append(entry.Audiences, target)
				}
			}
		}
		for _, prior := range nodes {
			if prior.Kind != offerspb.NodeKind_DELIVERABLE || prior.ReleaseRank <= 0 || prior.ReleaseRank > n.ReleaseRank {
				continue
			}
			priorEdges, err := s.ListEdges(ctx, prior.Id)
			if err != nil {
				return nil, err
			}
			for _, e := range priorEdges {
				if e.FromId == prior.Id && e.Kind == "unlocks" {
					if target := byID[e.ToId]; target != nil && target.Kind == offerspb.NodeKind_RAMP {
						entry.CumulativeRamps = appendUniqueNode(entry.CumulativeRamps, target)
					}
				}
			}
		}
		response.Entries = append(response.Entries, entry)
	}
	sort.Slice(response.Entries, func(i, j int) bool {
		return response.Entries[i].Deliverable.ReleaseRank < response.Entries[j].Deliverable.ReleaseRank
	})
	return response, nil
}

func (s *Store) Prerequisites(ctx context.Context, streamID string) (*offerspb.PrerequisiteWalkResponse, error) {
	if strings.TrimSpace(streamID) == "" {
		return nil, errors.New("stream node id is required")
	}
	var kind int32
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM nodes WHERE id=?`, streamID).Scan(&kind); err != nil {
		return nil, err
	}
	if offerspb.NodeKind(kind) != offerspb.NodeKind_STREAM {
		return nil, errors.New("prerequisite walk requires a stream node")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT n.id,n.kind,n.name,n.status,n.trigger_id,n.created_at,n.actual_account_id,n.release_rank FROM nodes n JOIN edges e ON e.from_id=n.id AND e.to_id=? AND e.kind='unlocks' WHERE n.kind=? ORDER BY n.release_rank,n.id`, streamID, int32(offerspb.NodeKind_DELIVERABLE))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := &offerspb.PrerequisiteWalkResponse{}
	for rows.Next() {
		var n offerspb.Node
		var k, st int32
		var ts string
		if err := rows.Scan(&n.Id, &k, &n.Name, &st, &n.TriggerId, &ts, &n.ActualAccountId, &n.ReleaseRank); err != nil {
			return nil, err
		}
		n.Kind, n.Status = offerspb.NodeKind(k), offerspb.Status(st)
		if t, e := time.Parse(time.RFC3339Nano, ts); e == nil {
			n.CreatedAt = timestamppb.New(t)
		}
		resp.Deliverables = append(resp.Deliverables, &n)
		if n.Status != offerspb.Status_SHIPPED {
			resp.Unshipped = append(resp.Unshipped, &n)
		}
	}
	return resp, rows.Err()
}

func appendUniqueNode(nodes []*offerspb.Node, n *offerspb.Node) []*offerspb.Node {
	for _, existing := range nodes {
		if existing.Id == n.Id {
			return nodes
		}
	}
	return append(nodes, n)
}
