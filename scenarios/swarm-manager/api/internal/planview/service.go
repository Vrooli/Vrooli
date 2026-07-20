package planview

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/backlogrank"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/gates"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operations"
)

// ErrGoalScope wraps a failure to resolve a goal-scoped board request (goal
// scoping unavailable, or the named goal not found) so the handler can return a
// client error rather than a 500.
var ErrGoalScope = errors.New("planview: goal scope")

// Service builds the plan-board projection.
type Service struct {
	cfg Config
}

// NewService creates a Service. Backlog and Gates are required.
func NewService(cfg Config) (*Service, error) {
	if cfg.Backlog == nil {
		return nil, fmt.Errorf("planview: Config.Backlog is required")
	}
	if cfg.Gates == nil {
		return nil, fmt.Errorf("planview: Config.Gates is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg}, nil
}

// clampWindow normalizes the Done-column window.
func clampWindow(seconds int) int {
	if seconds <= 0 {
		return DefaultWindowSeconds
	}
	if seconds < MinWindowSeconds {
		return MinWindowSeconds
	}
	if seconds > MaxWindowSeconds {
		return MaxWindowSeconds
	}
	return seconds
}

// Build composes the board projection.
func (s *Service) Build(ctx context.Context, params Params) (Board, error) {
	window := clampWindow(params.WindowSeconds)
	now := s.cfg.Now().UTC()

	items, err := s.cfg.Backlog.LoadAll(nil)
	if err != nil {
		return Board{}, fmt.Errorf("planview: load backlog: %w", err)
	}
	gateList := s.cfg.Gates.Enumerate(ctx)

	var execs []execution.Record
	if s.cfg.Executions != nil {
		execs, err = s.cfg.Executions.List(ctx, execution.ListFilters{})
		if err != nil {
			slog.Warn("planview: executions read failed; omitting execution outcomes", "error", err)
			execs = nil
		}
	}

	var inits []initiatives.Initiative
	if s.cfg.Initiatives != nil {
		inits, err = s.cfg.Initiatives.LoadAll()
		if err != nil {
			slog.Warn("planview: initiatives read failed; omitting initiative gate", "error", err)
			inits = nil
		}
	}

	// Goal scoping (optional): subset items, executions, and gates to the
	// goal's transitive prerequisite closure. Absent a goal this is a no-op, so
	// the unscoped projection is byte-identical.
	if params.Goal != "" {
		inScope, err := s.goalScope(params.Goal)
		if err != nil {
			return Board{}, err
		}
		items = filterItemsToScope(items, inScope)
		execs = filterExecsToScope(execs, inScope)
		gateList = filterGatesToScope(gateList, inScope, inits)
	}

	proj := newProjection(items, gateList, inits)

	board := Board{
		Now:   s.buildNowSummary(ctx),
		Next:  proj.buildNext(),
		Later: proj.buildLater(),
		Done:  proj.buildDone(execs, now, window),
		Meta: Meta{
			GeneratedAt:   now.Format(time.RFC3339),
			WindowSeconds: window,
			MaxWave:       proj.waves.MaxWave,
			Cycles:        proj.waves.Cycles,
			ETA:           s.buildETA(proj),
		},
	}
	if board.Meta.Cycles == nil {
		board.Meta.Cycles = []string{}
	}
	return board, nil
}

// buildETA computes the board-wide completion band over the projection's
// items, tolerating a nil factory or build error (band omitted).
func (s *Service) buildETA(proj *projection) *eta.Band {
	if s.cfg.ETA == nil {
		return nil
	}
	est, err := s.cfg.ETA()
	if err != nil || est == nil {
		return nil
	}
	if band, ok := est.EstimateGoal(proj.etaClosureInput()); ok {
		return &band
	}
	return nil
}

// goalScope resolves a goal name to the set of in-scope item keys
// ("<kind>/<name>"), returning ErrGoalScope when goal scoping is unavailable or
// the goal cannot be resolved.
func (s *Service) goalScope(goal string) (map[string]bool, error) {
	if s.cfg.Goals == nil {
		return nil, fmt.Errorf("%w: goal scoping unavailable", ErrGoalScope)
	}
	refs, err := s.cfg.Goals.ClosureRefs(goal)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoalScope, err)
	}
	inScope := make(map[string]bool, len(refs))
	for _, r := range refs {
		inScope[r] = true
	}
	return inScope, nil
}

// filterItemsToScope keeps only backlog items whose key is in the goal closure.
func filterItemsToScope(items []backlog.BacklogItem, inScope map[string]bool) []backlog.BacklogItem {
	out := make([]backlog.BacklogItem, 0, len(inScope))
	for _, item := range items {
		if inScope[itemKey(item)] {
			out = append(out, item)
		}
	}
	return out
}

// filterExecsToScope keeps only execution records whose backlog item is in
// scope, so the Done column reflects the goal, not the whole backlog.
func filterExecsToScope(execs []execution.Record, inScope map[string]bool) []execution.Record {
	out := make([]execution.Record, 0, len(execs))
	for _, rec := range execs {
		if inScope[rec.BacklogKind+"/"+rec.BacklogName] {
			out = append(out, rec)
		}
	}
	return out
}

// filterGatesToScope keeps backlog/execution gates whose owning item is in
// scope. Classify gates (capture-owned, not item-closure work) are dropped when
// the board is goal-scoped.
func filterGatesToScope(gateList []gates.Gate, inScope map[string]bool, inits []initiatives.Initiative) []gates.Gate {
	initiativesInScope := make(map[string]bool, len(inits))
	for _, init := range inits {
		for _, ref := range init.Items {
			if inScope[ref] {
				initiativesInScope[init.Name] = true
				break
			}
		}
	}
	out := make([]gates.Gate, 0, len(gateList))
	for _, g := range gateList {
		switch g.OwnerType {
		case "backlog", "execution":
			if inScope[g.OwnerKind+"/"+g.OwnerName] {
				out = append(out, g)
			}
		case "initiative":
			if initiativesInScope[g.OwnerName] {
				out = append(out, g)
			}
		}
	}
	return out
}

// buildNowSummary reads the operations aggregate for Now-header counts,
// degrading to zeros when the aggregator is absent or failing (the Now
// column's cards are client-fetched from the operations endpoint anyway).
func (s *Service) buildNowSummary(ctx context.Context) NowSummary {
	summary := NowSummary{Lanes: []LaneStatus{}}
	if s.cfg.Ops == nil {
		return summary
	}
	view, err := s.cfg.Ops.Aggregate(ctx, operations.Filters{})
	if err != nil {
		slog.Warn("planview: operations aggregate failed; Now counts degraded to zero", "error", err)
		return summary
	}
	summary.ActiveCount = len(view.Activities)
	summary.QueueDepth = view.Queue.Depth
	summary.MaxQueueDepth = view.Queue.MaxDepth
	for _, lane := range view.Lanes {
		summary.Lanes = append(summary.Lanes, LaneStatus{
			Lane:     lane.Lane,
			Active:   lane.Active,
			Capacity: lane.Capacity,
		})
	}
	return summary
}

// projection holds the shared intermediate state the column builders read.
type projection struct {
	items      []backlog.BacklogItem
	itemsByKey map[string]backlog.BacklogItem
	satisfied  map[string]bool
	waves      depgraph.WaveResult
	depthMap   map[string]int
	unblocking map[string]int

	decideByKey         map[string]gates.Gate
	reviewByKey         map[string]gates.Gate
	workshopByKey       map[string]gates.Gate
	execReview          []gates.Gate
	classify            []gates.Gate
	initiativeProposals []gates.Gate
	etaInput            eta.GoalClosureInput
}

func newProjection(items []backlog.BacklogItem, gateList []gates.Gate, inits []initiatives.Initiative) *projection {
	p := &projection{
		items:         items,
		itemsByKey:    make(map[string]backlog.BacklogItem, len(items)),
		satisfied:     make(map[string]bool, len(items)),
		decideByKey:   map[string]gates.Gate{},
		reviewByKey:   map[string]gates.Gate{},
		workshopByKey: map[string]gates.Gate{},
	}

	graphMap := make(map[string][]string, len(items))
	rankItems := make([]backlogrank.Item, 0, len(items))
	for _, item := range items {
		key := itemKey(item)
		p.itemsByKey[key] = item
		graphMap[key] = item.DependsOn
		if backlog.IsArchived(item) || item.Status == backlog.StatusCompleted {
			p.satisfied[key] = true
		}
		rankItems = append(rankItems, backlogrank.Item{
			Kind:      string(item.Kind),
			Name:      item.Name,
			Status:    string(item.Status),
			DependsOn: item.DependsOn,
			Archived:  backlog.IsArchived(item),
			Priority:  item.Priority,
			UpdatedAt: parseTime(item.Updated),
		})
	}
	// Fold in the D2 initiative gate: member items inherit their initiative's
	// dependencies, blocking them until the depended-on initiatives complete.
	// Additive — with no gated initiatives the wave graph is unchanged.
	if len(inits) > 0 {
		initiativeItems := make(map[string][]string, len(inits))
		initiativeDeps := make(map[string][]string, len(inits))
		for _, ini := range inits {
			initiativeItems[ini.Name] = ini.Items
			initiativeDeps[ini.Name] = ini.DependsOn
		}
		initSatisfied := goals.ApplyInitiativeGate(graphMap, func(k string) bool { return p.satisfied[k] }, initiativeItems, initiativeDeps)
		for node, done := range initSatisfied {
			p.satisfied[node] = done
		}
	}

	p.waves = depgraph.Waves(graphMap, func(k string) bool { return p.satisfied[k] })
	p.depthMap = backlogrank.ComputeDepthMap(rankItems)
	p.unblocking = backlogrank.ComputeUnblockingMap(rankItems)
	p.etaInput = eta.BuildClosureInput(items, inits)

	for _, g := range gateList {
		switch {
		case g.Kind == gates.KindDecide && g.OwnerType == "backlog":
			p.decideByKey[g.OwnerKind+"/"+g.OwnerName] = g
		case g.Kind == gates.KindReview && g.OwnerType == "backlog":
			p.reviewByKey[g.OwnerKind+"/"+g.OwnerName] = g
		case g.Kind == gates.KindProposal && g.OwnerType == "backlog":
			p.decideByKey[g.OwnerKind+"/"+g.OwnerName] = g
		case g.Kind == gates.KindProposal && g.OwnerType == "initiative":
			p.initiativeProposals = append(p.initiativeProposals, g)
		case g.Kind == gates.KindReview && g.OwnerType == "execution":
			p.execReview = append(p.execReview, g)
		case g.Kind == gates.KindWorkshop && g.OwnerType == "backlog":
			p.workshopByKey[g.OwnerKind+"/"+g.OwnerName] = g
		case g.Kind == gates.KindClassify:
			p.classify = append(p.classify, g)
		}
	}
	return p
}

func itemKey(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}

// etaClosureInput maps the projection's items into the ETA rollup input: the
// whole backlog is treated as the board's implicit closure. Done mirrors the
// wave-graph satisfied set (completed/archived); Gated marks pending items not
// yet at wave 0 (blocked on a prerequisite or gate). Dependency edges outside
// the known item set are dropped.
func (p *projection) etaClosureInput() eta.GoalClosureInput {
	return p.etaInput
}

func parseTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return t
}

// lockedStatuses are in-flight backlog statuses: those items belong to the
// Now column (via the operations endpoint), not Next/Later.
var lockedStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusQueued:     true,
	backlog.StatusInProgress: true,
	backlog.StatusInReview:   true,
}

// terminalStatuses are Done-column outcome statuses.
var terminalStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusCompleted:     true,
	backlog.StatusFailed:        true,
	backlog.StatusNeedsFollowup: true,
}

// planEligible reports whether the item can appear as a Next/Later card.
func (p *projection) planEligible(item backlog.BacklogItem) bool {
	if backlog.IsArchived(item) {
		return false
	}
	if lockedStatuses[item.Status] || terminalStatuses[item.Status] {
		return false
	}
	if item.Status == backlog.StatusReviewPending {
		return false // surfaced as a review gate card instead
	}
	return true
}

// rankLess orders item cards by dependency depth, effective priority, and
// recency — the same ordering the Command Post used.
func (p *projection) rankLess(a, b backlog.BacklogItem) bool {
	ra := backlogrank.Item{Kind: string(a.Kind), Name: a.Name, Status: string(a.Status), Priority: a.Priority, UpdatedAt: parseTime(a.Updated)}
	rb := backlogrank.Item{Kind: string(b.Kind), Name: b.Name, Status: string(b.Status), Priority: b.Priority, UpdatedAt: parseTime(b.Updated)}
	return backlogrank.Less(ra, rb, p.depthMap, p.unblocking)
}

func (p *projection) itemCard(item backlog.BacklogItem, action string) Card {
	key := itemKey(item)
	wave, ok := p.waves.Waves[key]
	if !ok {
		wave = 0
	}
	return Card{
		ID:         "backlog-item/" + key,
		CardType:   CardItem,
		Action:     action,
		ItemKind:   string(item.Kind),
		ItemName:   item.Name,
		Title:      titleOf(item),
		Status:     string(item.Status),
		Priority:   item.Priority,
		Wave:       wave,
		Initiative: item.Initiative,
		Effort:     item.Effort,
		Unblocks:   p.unblocking[key],
	}
}

func (p *projection) gateCard(g gates.Gate, action string) Card {
	gate := g
	card := Card{
		CardType: CardGate,
		Action:   action,
		Title:    g.OwnerTitle,
		Gate:     &gate,
		Unblocks: len(g.Blocks),
	}
	switch g.OwnerType {
	case "backlog":
		key := g.OwnerKind + "/" + g.OwnerName
		card.ID = "backlog-item/" + key
		card.ItemKind = g.OwnerKind
		card.ItemName = g.OwnerName
		if item, ok := p.itemsByKey[key]; ok {
			card.Status = string(item.Status)
			card.Priority = item.Priority
			card.Initiative = item.Initiative
			card.Effort = item.Effort
		}
		if wave, ok := p.waves.Waves[key]; ok {
			card.Wave = wave
		}
		card.Unblocks = p.unblocking[key]
	case "execution":
		card.ID = "execution-record/" + strings.TrimPrefix(g.ID, "review:execution/")
		card.ItemKind = g.OwnerKind
		card.ItemName = g.OwnerName
		card.ExecutionID = strings.TrimPrefix(g.ID, "review:execution/")
		key := g.OwnerKind + "/" + g.OwnerName
		if item, ok := p.itemsByKey[key]; ok {
			card.Initiative = item.Initiative
		}
	case "capture":
		card.ID = "capture/" + g.OwnerName
	case "initiative":
		card.ID = "initiative/" + g.OwnerName
		card.Initiative = g.OwnerName
	}
	return card
}

func titleOf(item backlog.BacklogItem) string {
	if t := strings.TrimSpace(item.Title); t != "" {
		return t
	}
	return item.Name
}

// actionFor resolves the item card's primary action from the gates
// read-model (workshop gate → workshop/finalize; none → run).
func (p *projection) actionFor(key string) string {
	if g, ok := p.workshopByKey[key]; ok {
		if g.Suggested == "finalize" {
			return ActionFinalize
		}
		return ActionWorkshop
	}
	return ActionRun
}

// buildNext assembles the Next column: the gates band (decide / review /
// classify — human-actionable now regardless of wave) followed by runnable
// and workshop item cards at wave 0.
func (p *projection) buildNext() Column {
	var gateCards []Card
	for _, item := range p.items {
		key := itemKey(item)
		if g, ok := p.decideByKey[key]; ok && p.planEligibleForGate(item) {
			gateCards = append(gateCards, p.gateCard(g, ActionDecide))
		}
		if g, ok := p.reviewByKey[key]; ok && !backlog.IsArchived(item) {
			gateCards = append(gateCards, p.gateCard(g, ActionReview))
		}
	}
	for _, g := range p.execReview {
		gateCards = append(gateCards, p.gateCard(g, ActionReview))
	}
	for _, g := range p.classify {
		gateCards = append(gateCards, p.gateCard(g, ActionClassify))
	}
	for _, g := range p.initiativeProposals {
		gateCards = append(gateCards, p.gateCard(g, ActionDecide))
	}
	sortGateCards(gateCards)

	var ready, workshop []backlog.BacklogItem
	for _, item := range p.items {
		key := itemKey(item)
		if !p.planEligible(item) {
			continue
		}
		if _, hasDecide := p.decideByKey[key]; hasDecide {
			continue // gate card already carries the item
		}
		if p.waves.Waves[key] != 0 {
			continue // Later
		}
		if _, needsWorkshop := p.workshopByKey[key]; needsWorkshop {
			workshop = append(workshop, item)
		} else {
			ready = append(ready, item)
		}
	}
	sort.SliceStable(ready, func(i, j int) bool { return p.rankLess(ready[i], ready[j]) })
	sort.SliceStable(workshop, func(i, j int) bool { return p.rankLess(workshop[i], workshop[j]) })

	var groups []CardGroup
	if len(gateCards) > 0 {
		groups = append(groups, CardGroup{
			ID:          "gates",
			Label:       "Decisions & reviews",
			BlockerKind: BlockerNone,
			Cards:       gateCards,
		})
	}
	if len(ready) > 0 {
		groups = append(groups, CardGroup{
			ID:          "ready",
			Label:       "Ready to run",
			BlockerKind: BlockerNone,
			Cards:       p.itemCards(ready),
		})
	}
	if len(workshop) > 0 {
		groups = append(groups, CardGroup{
			ID:          "workshop",
			Label:       "Needs workshop",
			BlockerKind: BlockerNone,
			Cards:       p.itemCards(workshop),
		})
	}
	return column(groups)
}

// planEligibleForGate mirrors gate eligibility for decide cards: locked and
// terminal items already live in Now/Done.
func (p *projection) planEligibleForGate(item backlog.BacklogItem) bool {
	if backlog.IsArchived(item) {
		return false
	}
	if item.Status == backlog.StatusQueued || item.Status == backlog.StatusInProgress {
		return false
	}
	return !terminalStatuses[item.Status]
}

func (p *projection) itemCards(items []backlog.BacklogItem) []Card {
	cards := make([]Card, 0, len(items))
	for _, item := range items {
		cards = append(cards, p.itemCard(item, p.actionFor(itemKey(item))))
	}
	return cards
}

// gateKindOrder ranks gate kinds for the Next gates band.
var gateKindOrder = map[gates.Kind]int{
	gates.KindDecide:   0,
	gates.KindReview:   1,
	gates.KindClassify: 2,
}

func sortGateCards(cards []Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		gi, gj := cards[i].Gate, cards[j].Gate
		if oi, oj := gateKindOrder[gi.Kind], gateKindOrder[gj.Kind]; oi != oj {
			return oi < oj
		}
		if cards[i].Unblocks != cards[j].Unblocks {
			return cards[i].Unblocks > cards[j].Unblocks
		}
		return cards[i].Title < cards[j].Title
	})
}

// buildLater assembles the Later column: items not yet actionable, grouped
// by nearest blocker. Gate-blocked groups sort above item-blocked groups
// (D2); cycle-trapped items land in a diagnostics group last.
func (p *projection) buildLater() Column {
	type groupAccum struct {
		group CardGroup
		items []backlog.BacklogItem
	}
	accums := map[string]*groupAccum{}

	for _, item := range p.items {
		key := itemKey(item)
		if !p.planEligible(item) {
			continue
		}
		if _, hasDecide := p.decideByKey[key]; hasDecide {
			continue // decide gate card in Next carries the item
		}
		wave := p.waves.Waves[key]
		if wave == 0 {
			continue // Next
		}

		var groupID, label, blockerKind, gateID string
		var blockerKeys []string
		if wave == depgraph.CycleWave {
			groupID, label, blockerKind = "cycle", "Dependency cycle", BlockerCycle
		} else {
			unmet := p.unmetDeps(item)
			if g, ok := p.nearestGate(unmet); ok {
				groupID = "gate:" + g.ID
				label = fmt.Sprintf("after %s %q", g.Kind, g.OwnerTitle)
				blockerKind = BlockerGate
				gateID = g.ID
			} else {
				groupID = "items:" + strings.Join(unmet, ",")
				label = "after " + p.depLabel(unmet)
				blockerKind = BlockerItems
				blockerKeys = unmet
			}
		}

		acc, ok := accums[groupID]
		if !ok {
			acc = &groupAccum{group: CardGroup{
				ID:          groupID,
				Label:       label,
				BlockerKind: blockerKind,
				GateID:      gateID,
				BlockerKeys: blockerKeys,
			}}
			accums[groupID] = acc
		}
		acc.items = append(acc.items, item)
	}

	groups := make([]*groupAccum, 0, len(accums))
	for _, acc := range accums {
		groups = append(groups, acc)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return laterGroupLess(groups[i].group, groups[j].group)
	})

	out := make([]CardGroup, 0, len(groups))
	for _, acc := range groups {
		sort.SliceStable(acc.items, func(i, j int) bool {
			wi := p.waves.Waves[itemKey(acc.items[i])]
			wj := p.waves.Waves[itemKey(acc.items[j])]
			if wi != wj {
				return wi < wj
			}
			return p.rankLess(acc.items[i], acc.items[j])
		})
		group := acc.group
		for _, item := range acc.items {
			group.Cards = append(group.Cards, p.itemCard(item, p.actionFor(itemKey(item))))
		}
		out = append(out, group)
	}
	return column(out)
}

// blockerKindOrder ranks Later groups: gate-blocked, then item-blocked,
// then cycle diagnostics.
var blockerKindOrder = map[string]int{
	BlockerGate:  0,
	BlockerItems: 1,
	BlockerCycle: 2,
}

func laterGroupLess(a, b CardGroup) bool {
	if oa, ob := blockerKindOrder[a.BlockerKind], blockerKindOrder[b.BlockerKind]; oa != ob {
		return oa < ob
	}
	return a.Label < b.Label
}

// unmetDeps returns the item's direct dependencies that are known items and
// not yet satisfied, sorted.
func (p *projection) unmetDeps(item backlog.BacklogItem) []string {
	var out []string
	for _, dep := range item.DependsOn {
		if _, known := p.itemsByKey[dep]; !known {
			continue
		}
		if p.satisfied[dep] {
			continue
		}
		out = append(out, dep)
	}
	sort.Strings(out)
	return out
}

// nearestGate finds the human gate (decide/review) on the closest-to-
// runnable unmet dependency, if any.
func (p *projection) nearestGate(unmet []string) (gates.Gate, bool) {
	type candidate struct {
		gate gates.Gate
		wave int
		key  string
	}
	var best *candidate
	for _, dep := range unmet {
		g, ok := p.decideByKey[dep]
		if !ok {
			g, ok = p.reviewByKey[dep]
		}
		if !ok {
			continue
		}
		wave := p.waves.Waves[dep]
		c := candidate{gate: g, wave: wave, key: dep}
		if best == nil || c.wave < best.wave || (c.wave == best.wave && c.key < best.key) {
			best = &c
		}
	}
	if best == nil {
		return gates.Gate{}, false
	}
	return best.gate, true
}

// depLabel renders up to three blocking dependency titles.
func (p *projection) depLabel(deps []string) string {
	const maxNamed = 3
	names := make([]string, 0, maxNamed)
	for i, dep := range deps {
		if i == maxNamed {
			names = append(names, fmt.Sprintf("+%d more", len(deps)-maxNamed))
			break
		}
		if item, ok := p.itemsByKey[dep]; ok {
			names = append(names, titleOf(item))
		} else {
			names = append(names, dep)
		}
	}
	return strings.Join(names, ", ")
}

// buildDone assembles the Done column: recent run outcomes and terminal
// items within the window, newest first, capped.
func (p *projection) buildDone(execs []execution.Record, now time.Time, windowSeconds int) Column {
	since := now.Add(-time.Duration(windowSeconds) * time.Second)

	type dated struct {
		card Card
		at   time.Time
	}
	var out []dated

	// doneExecutionStatuses mirror the Command Post RecentSection derivation.
	for _, rec := range execs {
		var outcome string
		switch rec.Status {
		case execution.StatusCompleted:
			outcome = OutcomeOK
		case execution.StatusFailed:
			outcome = OutcomeFailed
		case execution.StatusNeedsReview:
			outcome = OutcomeNeedsReview
		default:
			continue
		}
		ts := parseTime(rec.FinishedAt)
		if ts.IsZero() {
			ts = parseTime(rec.StartedAt)
		}
		if ts.IsZero() || ts.Before(since) {
			continue
		}
		key := rec.BacklogKind + "/" + rec.BacklogName
		title := key
		initiative := ""
		if item, ok := p.itemsByKey[key]; ok {
			title = titleOf(item)
			initiative = item.Initiative
		}
		out = append(out, dated{at: ts, card: Card{
			ID:          "execution-record/" + rec.ExecutionID,
			CardType:    CardOutcome,
			Action:      ActionReview,
			ItemKind:    rec.BacklogKind,
			ItemName:    rec.BacklogName,
			Title:       title,
			Status:      string(rec.Status),
			Initiative:  initiative,
			Outcome:     outcome,
			FinishedAt:  ts.Format(time.RFC3339),
			ExecutionID: rec.ExecutionID,
		}})
	}

	for _, item := range p.items {
		if !terminalStatuses[item.Status] {
			continue
		}
		ts := parseTime(item.Updated)
		if ts.IsZero() || ts.Before(since) {
			continue
		}
		var outcome string
		switch item.Status {
		case backlog.StatusCompleted:
			outcome = OutcomeOK
		case backlog.StatusFailed:
			outcome = OutcomeFailed
		default:
			outcome = OutcomeNeedsFollowup
		}
		key := itemKey(item)
		out = append(out, dated{at: ts, card: Card{
			ID:         "backlog-item/" + key,
			CardType:   CardOutcome,
			Action:     ActionNone,
			ItemKind:   string(item.Kind),
			ItemName:   item.Name,
			Title:      titleOf(item),
			Status:     string(item.Status),
			Initiative: item.Initiative,
			Outcome:    outcome,
			FinishedAt: ts.Format(time.RFC3339),
		}})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].at.After(out[j].at) })
	if len(out) > DoneCap {
		out = out[:DoneCap]
	}

	if len(out) == 0 {
		return column(nil)
	}
	group := CardGroup{ID: "recent", Label: "Recent outcomes", BlockerKind: BlockerNone}
	for _, d := range out {
		group.Cards = append(group.Cards, d.card)
	}
	return column([]CardGroup{group})
}

func column(groups []CardGroup) Column {
	if groups == nil {
		groups = []CardGroup{}
	}
	count := 0
	for _, g := range groups {
		count += len(g.Cards)
	}
	return Column{Groups: groups, CardCount: count}
}
