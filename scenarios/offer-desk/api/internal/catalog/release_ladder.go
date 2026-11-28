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

func (s *Store) SetReleaseRank(ctx context.Context, nodeID string, rank int32, actor string, reasons ...string) (*offerspb.Node, int32, error) {
	reason := ""
	if len(reasons) > 0 {
		reason = reasons[0]
	}
	if strings.TrimSpace(nodeID) == "" || rank < 0 {
		return nil, 0, errors.New("release rank requires a node id and a non-negative rank")
	}
	if strings.TrimSpace(actor) == "" {
		return nil, 0, errors.New("release rank requires an actor")
	}
	var n offerspb.Node
	var kind, status, class, finish int32
	var created string
	if err := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank,deliverable_class,finish_bar FROM nodes WHERE id=?`, nodeID).
		Scan(&n.Id, &kind, &n.Name, &status, &n.TriggerId, &created, &n.ActualAccountId, &n.ReleaseRank, &class, &finish); err != nil {
		return nil, 0, fmt.Errorf("node %q not found: %w", nodeID, err)
	}
	n.Kind, n.Status = offerspb.NodeKind(kind), offerspb.Status(status)
	n.DeliverableClass, n.FinishBar = offerspb.DeliverableClass(class), offerspb.FinishBar(finish)
	if n.Kind != offerspb.NodeKind_DELIVERABLE {
		return nil, 0, errors.New("release rank applies only to deliverable nodes")
	}
	if n.DeliverableClass == offerspb.DeliverableClass_ENABLING && rank > 0 {
		return nil, 0, fmt.Errorf("rule enabling_deliverables_are_unranked refused rank %d for %q; reclassify it as MARKETED to put it on the schedule", rank, n.Name)
	}
	if rank > 0 {
		var other string
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE kind=? AND deliverable_class=? AND release_rank=? AND id<>? AND status<>? LIMIT 1`, int32(offerspb.NodeKind_DELIVERABLE), int32(offerspb.DeliverableClass_MARKETED), rank, nodeID, int32(offerspb.Status_RETIRED)).Scan(&other); err == nil {
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
	if strings.TrimSpace(reason) == "" {
		reason = fmt.Sprintf("release rank changed from %d to %d", prior, rank)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`,
		uuid.NewString(), nodeID, actor, status, status, reason, s.now().UTC().Format(time.RFC3339Nano)); err != nil {
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
	edges, err := s.ListEdges(ctx, "")
	if err != nil {
		return nil, err
	}
	urgency, err := s.derivedUrgencies(nodes, edges)
	if err != nil {
		return nil, err
	}
	from := make(map[string][]*offerspb.Edge)
	to := make(map[string][]*offerspb.Edge)
	for _, edge := range edges {
		from[edge.FromId] = append(from[edge.FromId], edge)
		to[edge.ToId] = append(to[edge.ToId], edge)
	}
	response := &offerspb.ReleaseLadderResponse{}
	for _, n := range nodes {
		if !includeRetired && n.Status == offerspb.Status_RETIRED {
			continue
		}
		switch n.Kind {
		case offerspb.NodeKind_RAMP:
			response.Ramps = append(response.Ramps, n)
		case offerspb.NodeKind_STREAM:
			response.Streams = append(response.Streams, n)
		case offerspb.NodeKind_AUDIENCE:
			response.Audiences = append(response.Audiences, n)
		}
		if n.Kind == offerspb.NodeKind_DELIVERABLE && n.DeliverableClass == offerspb.DeliverableClass_ENABLING && (includeRetired || n.Status != offerspb.Status_RETIRED) {
			response.Enabling = append(response.Enabling, &offerspb.PrerequisiteNode{Node: n, DerivedUrgency: urgency[n.Id]})
		}
	}
	for _, n := range nodes {
		if !includeRetired && n.Status == offerspb.Status_RETIRED {
			continue
		}
		if n.Kind != offerspb.NodeKind_DELIVERABLE || n.ReleaseRank <= 0 || (!includeRetired && n.Status == offerspb.Status_RETIRED) {
			if n.Kind == offerspb.NodeKind_DELIVERABLE && n.DeliverableClass == offerspb.DeliverableClass_MARKETED && n.ReleaseRank <= 0 && (includeRetired || n.Status != offerspb.Status_RETIRED) {
				response.Unscheduled = append(response.Unscheduled, n)
			}
			continue
		}
		entry := &offerspb.ReleaseLadderEntry{Deliverable: n}
		if s.readinessProvider != nil {
			if state, err := s.readinessProvider(ctx, n); err == nil {
				entry.ReadinessGoalExists = state.GoalExists
				entry.ReadinessGoalClosed = state.GoalClosed
				entry.ReadinessApprovedCommit = state.ApprovedCommit
			}
		}
		for _, e := range from[n.Id] {
			if e.FromId != n.Id {
				continue
			}
			target := byID[e.ToId]
			if target == nil || (!includeRetired && target.Status == offerspb.Status_RETIRED) {
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
		for _, e := range to[n.Id] {
			if e.Kind == "enables" {
				if source := byID[e.FromId]; source != nil && (includeRetired || source.Status != offerspb.Status_RETIRED) && source.Kind == offerspb.NodeKind_DELIVERABLE && source.DeliverableClass == offerspb.DeliverableClass_ENABLING {
					entry.Enablers = append(entry.Enablers, &offerspb.PrerequisiteNode{Node: source, DerivedUrgency: urgency[source.Id], Depth: 1, Path: []string{n.Name, source.Name}})
				}
			}
		}
		response.Entries = append(response.Entries, entry)
	}
	sort.Slice(response.Entries, func(i, j int) bool {
		return response.Entries[i].Deliverable.ReleaseRank < response.Entries[j].Deliverable.ReleaseRank
	})
	// Entries are now ordered, so accumulate unlocked ramps once rather than
	// rescanning the entire graph for every schedule row.
	var ramps []*offerspb.Node
	for _, entry := range response.Entries {
		for _, unlocked := range entry.UnlockedRamps {
			ramps = appendUniqueNode(ramps, unlocked)
		}
		entry.CumulativeRamps = append([]*offerspb.Node(nil), ramps...)
	}
	sort.Slice(response.Enabling, func(i, j int) bool {
		left, right := response.Enabling[i], response.Enabling[j]
		if left.DerivedUrgency == 0 {
			return false
		}
		if right.DerivedUrgency == 0 {
			return true
		}
		return left.DerivedUrgency < right.DerivedUrgency
	})
	return response, nil
}

// DerivedUrgencies resolves enabling urgency from the current graph without
// persisting a second copy of release priority on any node.
func (s *Store) DerivedUrgencies(ctx context.Context) (map[string]int32, error) {
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(ctx, "")
	if err != nil {
		return nil, err
	}
	return s.derivedUrgencies(nodes, edges)
}

func (s *Store) derivedUrgencies(nodes []*offerspb.Node, edges []*offerspb.Edge) (map[string]int32, error) {
	byID := make(map[string]*offerspb.Node, len(nodes))
	outgoing := make(map[string][]string)
	unlockedBy := make(map[string][]string)
	for _, n := range nodes {
		byID[n.Id] = n
	}
	for _, edge := range edges {
		switch edge.Kind {
		case "enables":
			outgoing[edge.FromId] = append(outgoing[edge.FromId], edge.ToId)
		case "unlocks":
			unlockedBy[edge.ToId] = append(unlockedBy[edge.ToId], edge.FromId)
		}
	}
	urgency := make(map[string]int32, len(nodes))
	var visit func(string, map[string]bool) int32
	visit = func(id string, path map[string]bool) int32 {
		if value, ok := urgency[id]; ok {
			return value
		}
		if path[id] {
			return 0
		}
		path[id] = true
		defer delete(path, id)
		best := int32(0)
		if n := byID[id]; n != nil && n.Status != offerspb.Status_RETIRED && n.Kind == offerspb.NodeKind_DELIVERABLE && n.DeliverableClass == offerspb.DeliverableClass_MARKETED && n.ReleaseRank > 0 {
			best = n.ReleaseRank
		}
		if n := byID[id]; n != nil && n.Status != offerspb.Status_RETIRED && (n.Kind == offerspb.NodeKind_RAMP || n.Kind == offerspb.NodeKind_STREAM) {
			for _, openerID := range unlockedBy[id] {
				opener := byID[openerID]
				if opener == nil || opener.Status == offerspb.Status_RETIRED || opener.Kind != offerspb.NodeKind_DELIVERABLE || opener.DeliverableClass != offerspb.DeliverableClass_MARKETED || opener.ReleaseRank <= 0 {
					continue
				}
				if best == 0 || opener.ReleaseRank < best {
					best = opener.ReleaseRank
				}
			}
		}
		for _, downstream := range outgoing[id] {
			candidate := visit(downstream, path)
			if candidate > 0 && (best == 0 || candidate < best) {
				best = candidate
			}
		}
		urgency[id] = best
		return best
	}
	for _, n := range nodes {
		visit(n.Id, map[string]bool{})
	}
	return urgency, nil
}

func (s *Store) Prerequisites(ctx context.Context, streamID string) (*offerspb.PrerequisiteWalkResponse, error) {
	return s.PrerequisitesWithOptions(ctx, streamID, 0, false)
}

func (s *Store) PrerequisitesWithOptions(ctx context.Context, streamID string, maxDepth int32, includeShipped bool) (*offerspb.PrerequisiteWalkResponse, error) {
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
	if maxDepth <= 0 {
		maxDepth = 100
	}
	return s.prerequisiteWalk(ctx, streamID, maxDepth, includeShipped)
}

func (s *Store) prerequisiteWalk(ctx context.Context, streamID string, maxDepth int32, includeShipped bool) (*offerspb.PrerequisiteWalkResponse, error) {
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*offerspb.Node, len(nodes))
	for _, n := range nodes {
		byID[n.Id] = n
	}
	if byID[streamID] == nil {
		return nil, fmt.Errorf("stream node %q not found", streamID)
	}
	edges, err := s.ListEdges(ctx, "")
	if err != nil {
		return nil, err
	}
	incoming := map[string][]*offerspb.Edge{}
	outgoingEnables := map[string][]string{}
	for _, e := range edges {
		if e.Kind == "unlocks" || e.Kind == "enables" {
			incoming[e.ToId] = append(incoming[e.ToId], e)
		}
		if e.Kind == "enables" {
			outgoingEnables[e.FromId] = append(outgoingEnables[e.FromId], e.ToId)
		}
	}
	urgency := make(map[string]int32, len(nodes))
	var urgencyFor func(string, map[string]bool) int32
	urgencyFor = func(id string, trail map[string]bool) int32 {
		if value, ok := urgency[id]; ok {
			return value
		}
		if trail[id] {
			return 0
		}
		trail[id] = true
		defer delete(trail, id)
		best := int32(0)
		if n := byID[id]; n != nil && n.Kind == offerspb.NodeKind_DELIVERABLE && n.DeliverableClass == offerspb.DeliverableClass_MARKETED && n.ReleaseRank > 0 {
			best = n.ReleaseRank
		}
		for _, downstream := range outgoingEnables[id] {
			candidate := urgencyFor(downstream, trail)
			if candidate > 0 && (best == 0 || candidate < best) {
				best = candidate
			}
		}
		urgency[id] = best
		return best
	}
	resp := &offerspb.PrerequisiteWalkResponse{}
	type item struct {
		node  *offerspb.Node
		depth int32
		path  []string
	}
	queue := make([]item, 0)
	for _, e := range incoming[streamID] {
		if n := byID[e.FromId]; n != nil {
			queue = append(queue, item{n, 1, []string{byID[streamID].Name, n.Name}})
		}
	}
	seen := map[string]bool{}
	var walk func(item, map[string]bool) error
	walk = func(cur item, trail map[string]bool) error {
		if cur.depth > maxDepth {
			return nil
		}
		if trail[cur.node.Id] {
			resp.CyclePath = append([]string{}, cur.path...)
			return nil
		}
		if seen[cur.node.Id] {
			return nil
		}
		seen[cur.node.Id] = true
		trail[cur.node.Id] = true
		defer delete(trail, cur.node.Id)
		if cur.node.Kind == offerspb.NodeKind_DELIVERABLE {
			pn := &offerspb.PrerequisiteNode{Node: cur.node, Depth: cur.depth, DerivedUrgency: urgencyFor(cur.node.Id, map[string]bool{}), Path: append([]string{}, cur.path...)}
			resp.Tree = append(resp.Tree, pn)
			resp.Deliverables = append(resp.Deliverables, cur.node)
			if includeShipped || cur.node.Status != offerspb.Status_SHIPPED {
				resp.Unshipped = append(resp.Unshipped, cur.node)
			}
		}
		for _, edge := range incoming[cur.node.Id] {
			n := byID[edge.FromId]
			if n == nil {
				continue
			}
			nextPath := append(append([]string{}, cur.path...), n.Name)
			if trail[n.Id] {
				resp.CyclePath = nextPath
				continue
			}
			if err := walk(item{n, cur.depth + 1, nextPath}, trail); err != nil {
				return err
			}
		}
		return nil
	}
	for _, entry := range queue {
		if err := walk(entry, map[string]bool{streamID: true}); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(resp.Tree, func(i, j int) bool {
		if resp.Tree[i].Depth != resp.Tree[j].Depth {
			return resp.Tree[i].Depth < resp.Tree[j].Depth
		}
		return resp.Tree[i].Node.Name < resp.Tree[j].Node.Name
	})
	return resp, nil
}

func appendUniqueNode(nodes []*offerspb.Node, n *offerspb.Node) []*offerspb.Node {
	for _, existing := range nodes {
		if existing.Id == n.Id {
			return nodes
		}
	}
	return append(nodes, n)
}
