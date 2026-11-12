// Package activity serves the read-only projection of the research home:
// what is moving, which commissions exist, which topics they cover, and what
// each role must read (SYSTEM.md §3, §4, §9).
package activityapi

import (
	"context"
	"errors"
	"log"
	"slices"
	"time"

	"connectrpc.com/connect"

	"research-desk/internal/home"
	"research-desk/internal/wire"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// Snapshotter is the home reader seam.
type Snapshotter interface {
	Snapshot(ctx context.Context, opts home.Options) (*home.Snapshot, error)
}

// WebConsoleLens is the overview lens for Web Console access.
const WebConsoleLens = "web-console access"

// Deps wires the activity handler.
type Deps struct {
	Reader Snapshotter
	// Looks reads the look snapshots OperatorService.MarkSeen records in the
	// desk state folder; nil refuses GetChanges (failed_precondition).
	Looks Looks
	// Routes reports the desk's own routes to other scenarios (optional
	// scenarios, the courier's health), shown with the overview's lenses;
	// nil shows none.
	Routes func(context.Context) []home.Lens
	// WebConsoleAccess says how the desk reaches Web Console:
	// the overview's kind and sentence, and its WebConsoleLens. Nil shows
	// neither.
	WebConsoleAccess func(context.Context) wire.Access
	// EntryChars is limits.json's entry_chars_max, shown beside the
	// commission's entry fields; nil or an error leaves it 0.
	EntryChars func() (int, error)
	Logger     *log.Logger
}

// Looks reads the desk's look snapshots (internal/writer).
type Looks interface {
	ReadState(name string, into any) (bool, error)
}

type handler struct{ deps Deps }

// NewHandler returns the Connect handler.
func NewHandler(deps Deps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &handler{deps: deps}
}

func (h *handler) snapshot(ctx context.Context, opts home.Options) (*home.Snapshot, error) {
	snap, err := h.deps.Reader.Snapshot(ctx, opts)
	return snap, h.fail("snapshot", err)
}

// fail maps an error to its wire code by cause (wire.Error).
func (h *handler) fail(op string, err error) error {
	return wire.Error(h.deps.Logger, "activity: "+op, err)
}

func (h *handler) GetOverview(ctx context.Context, _ *connect.Request[activityv1.GetOverviewRequest]) (*connect.Response[activityv1.GetOverviewResponse], error) {
	snap, err := h.snapshot(ctx, home.Options{})
	if err != nil {
		return nil, err
	}
	counts := map[home.State]int32{}
	resp := &activityv1.GetOverviewResponse{GeneratedAt: wire.Timestamp(snap.Now), ResearchHome: snap.ResearchDir}
	for _, c := range snap.Commissions {
		counts[c.State]++
		if c.State.Running() {
			resp.Running = append(resp.Running, summary(c))
		}
	}
	for _, state := range stateOrder {
		if n := counts[state]; n > 0 {
			resp.StateCounts = append(resp.StateCounts, &activityv1.StateCount{State: wire.State(state), Count: n})
		}
	}
	for _, o := range snap.Overlaps() {
		resp.Overlaps = append(resp.Overlaps, &activityv1.Overlap{Topic: o.Topic, CommissionIds: o.CommissionIDs})
	}
	for _, o := range snap.OpenOverlaps() {
		resp.OpenOverlaps = append(resp.OpenOverlaps, &activityv1.Overlap{Topic: o.Topic, CommissionIds: o.CommissionIDs})
	}
	resp.Lenses = lenses(snap.Lenses)
	if h.deps.Routes != nil {
		resp.Lenses = append(resp.Lenses, lenses(h.deps.Routes(ctx))...)
	}
	if h.deps.WebConsoleAccess != nil {
		a := h.deps.WebConsoleAccess(ctx)
		resp.WebConsoleAccessKind, resp.WebConsoleAccess = a.Kind, a.Said
		resp.Lenses = append(resp.Lenses, &activityv1.Lens{Name: WebConsoleLens, Available: a.Automatic, Reason: a.Said})
	}
	resp.Unreadable = wire.EvidenceList(snap.AllUnreadable())
	resp.LatestResearchWrite = wire.ResearchWrite(snap.LatestResearchWrite())
	return connect.NewResponse(resp), nil
}

func (h *handler) ListCommissions(ctx context.Context, req *connect.Request[activityv1.ListCommissionsRequest]) (*connect.Response[activityv1.ListCommissionsResponse], error) {
	snap, err := h.snapshot(ctx, home.Options{})
	if err != nil {
		return nil, err
	}
	resp := &activityv1.ListCommissionsResponse{}
	for _, c := range snap.Commissions {
		if len(req.Msg.States) == 0 || slices.Contains(req.Msg.States, wire.State(c.State)) {
			resp.Commissions = append(resp.Commissions, summary(c))
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) GetCommission(ctx context.Context, req *connect.Request[activityv1.GetCommissionRequest]) (*connect.Response[activityv1.GetCommissionResponse], error) {
	snap, err := h.snapshot(ctx, home.Options{})
	if err != nil {
		return nil, err
	}
	c, err := snap.Find(req.Msg.Id)
	if err != nil {
		return nil, h.fail("get commission", err)
	}
	resp := &activityv1.GetCommissionResponse{
		Commission:      summary(c),
		Files:           wire.Files(c.Files),
		RecordErrors:    c.RecordErrors,
		Steering:        wire.Flag(c.Steering),
		Decisions:       wire.Flag(c.Decisions),
		OperatorEntries: wire.Entries(c.Operator),
		Unreadable:      wire.EvidenceList(c.Unreadable),
	}
	if c.Operator != nil {
		resp.OperatorUnparsedHeadings = wire.UnparsedHeadings(c.Operator.Unparsed)
	}
	for i := range c.Matches {
		resp.Matches = append(resp.Matches, match(&c.Matches[i]))
	}
	for _, run := range c.Runs {
		resp.Runs = append(resp.Runs, &activityv1.RunFolder{Topic: run.Topic, Path: run.Path, LastWrite: wire.Time(run.LastWrite), Checkpoint: wire.File(run.Checkpoint)})
	}
	resp.ReadSets = wire.ReadSets(snap.ReadSets(c))
	resp.Ceilings = ceilingsEvidence(snap)
	if h.deps.EntryChars != nil {
		if max, err := h.deps.EntryChars(); err == nil {
			resp.EntryCharsMax = int32(max)
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) ListTopics(ctx context.Context, _ *connect.Request[activityv1.ListTopicsRequest]) (*connect.Response[activityv1.ListTopicsResponse], error) {
	snap, err := h.snapshot(ctx, home.Options{MissionCheck: true})
	if err != nil {
		return nil, err
	}
	resp := &activityv1.ListTopicsResponse{}
	for _, t := range snap.Topics {
		check := &activityv1.MissionCheck{
			Mode: t.Check.Mode, Open: int32(t.Check.Open), Active: int32(t.Check.Active),
			WaitingOperator: int32(t.Check.WaitingOperator), Landed: int32(t.Check.Landed),
			Errors: t.Check.Errors, Warnings: t.Check.Warnings, Evidence: wire.Evidence(t.Check.Evidence),
		}
		check.Workstreams = wire.Workstreams(t.Check.Workstreams)
		resp.Topics = append(resp.Topics, &activityv1.TopicSummary{
			Topic:                t.Slug,
			Title:                wire.Text(t.Title),
			LastResearchWrite:    wire.ResearchWrite(t.LastResearchWrite),
			LastReview:           reviewTouch(t.LastReview),
			CommissionIds:        t.CommissionIDs,
			LiveCommissionIds:    t.Live,
			PausedCommissionIds:  t.Paused,
			LimitedCommissionIds: t.Limited,
			Aliases:              t.Aliases,
			Mission:              wire.File(t.Mission),
			MissionCheck:         check,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) GetChanges(ctx context.Context, req *connect.Request[activityv1.GetChangesRequest]) (*connect.Response[activityv1.GetChangesResponse], error) {
	if h.deps.Looks == nil {
		return nil, h.fail("changes", home.Cause(home.ErrRefused, errors.New("changes since the last look are not available in this instance")))
	}
	subject := home.LookCommission
	switch req.Msg.Subject {
	case activityv1.ChangeSubject_CHANGE_SUBJECT_COMMISSION:
	case activityv1.ChangeSubject_CHANGE_SUBJECT_TOPIC:
		subject = home.LookTopic
	default:
		return nil, h.fail("changes", home.Cause(home.ErrInvalid, errors.New("subject must be CHANGE_SUBJECT_COMMISSION or CHANGE_SUBJECT_TOPIC")))
	}
	snap, err := h.snapshot(ctx, home.Options{MissionCheck: true})
	if err != nil {
		return nil, err
	}
	var prev *home.Look
	var since time.Time
	if req.Msg.Since != nil {
		since = req.Msg.Since.AsTime()
	} else {
		var look home.Look
		found, err := h.deps.Looks.ReadState(home.LookName(subject, req.Msg.Id), &look)
		if err != nil {
			return nil, h.fail("read the last look", err)
		}
		if found {
			prev = &look
		}
	}
	changes, err := snap.Changes(prev, since, subject, req.Msg.Id)
	if err != nil {
		return nil, h.fail("changes", err)
	}
	resp := &activityv1.GetChangesResponse{
		Since: wire.Timestamp(changes.Since), Checkpoints: wire.Files(changes.Checkpoints), ShowcaseLines: changes.ShowcaseLines,
		Files: wire.Files(changes.Files), Truncated: changes.Truncated, Baseline: changes.Baseline,
		Unreadable: wire.EvidenceList(changes.Unreadable), BaselineKind: baselines[changes.BaselineKind],
		NotShown: changes.NotShown,
	}
	for _, l := range changes.Ledger {
		resp.Ledger = append(resp.Ledger, &activityv1.LedgerAddition{Path: l.Path, Text: l.Text, Truncated: l.Truncated, FromDatedHeadings: l.FromDatedHeadings})
	}
	for _, c := range changes.StatusChanges {
		resp.StatusChanges = append(resp.StatusChanges, &activityv1.StatusChange{Topic: c.Topic, Workstream: c.Workstream, Before: c.Before, After: c.After})
	}
	return connect.NewResponse(resp), nil
}

func summary(c *home.Commission) *activityv1.CommissionSummary {
	out := &activityv1.CommissionSummary{
		Id:              c.ID,
		Folder:          c.Dir,
		Title:           wire.Text(c.Title),
		State:           wire.State(c.State),
		StateRule:       c.StateRule,
		StateEvidence:   wire.EvidenceList(c.StateEvidence),
		Stalled:         c.Stalled,
		TopicsEvidence:  wire.Evidence(c.TopicsEvidence),
		Started:         wire.Time(c.Started),
		AllowanceEnd:    wire.Time(c.AllowanceEnd),
		LastWrite:       wire.Time(c.LastWrite),
		Ended:           wire.Time(c.Ended),
		TouchedAfterEnd: wire.Time(c.TouchedAfterEnd),
		Claim:           wire.Text(c.Claim),
		LaunchedBy:      c.LaunchedBy(),
		EntryReading:    wire.EntryReading(c.EntryReading()),
	}
	if c.State == home.StateUnknown {
		out.NeverStarted = wire.NeverStarted(c.NeverStarted)
	}
	for _, ref := range c.Topics {
		out.Topics = append(out.Topics, &activityv1.TopicRef{Topic: ref.Topic, Basis: ref.Basis, Source: ref.Source})
	}
	if c.Session != nil {
		out.Session = match(c.Session)
	}
	return out
}

var baselines = map[home.BaselineKind]activityv1.Baseline{
	home.BaselineLook:    activityv1.Baseline_BASELINE_LOOK,
	home.BaselineAsked:   activityv1.Baseline_BASELINE_ASKED,
	home.BaselineLastDay: activityv1.Baseline_BASELINE_LAST_DAY,
}

func reviewTouch(r *home.ReviewTouch) *activityv1.ReviewTouch {
	if r == nil {
		return nil
	}
	return &activityv1.ReviewTouch{ReviewId: r.ReviewID, File: wire.File(&r.File)}
}

func match(m *home.Match) *activityv1.SessionMatch {
	kind := activityv1.MatchKind_MATCH_KIND_RELATED
	if m.Kind == home.MatchExact {
		kind = activityv1.MatchKind_MATCH_KIND_EXACT
	}
	return &activityv1.SessionMatch{
		Kind: kind, Basis: m.Basis, WebConsoleSessionId: m.SessionID, ThreadId: m.ThreadID,
		GoalStatus: m.GoalStatus, At: wire.Timestamp(m.At), Source: m.Source, WindowOpen: m.WindowOpen,
	}
}

func lenses(list []home.Lens) []*activityv1.Lens {
	out := make([]*activityv1.Lens, 0, len(list))
	for _, l := range list {
		out = append(out, &activityv1.Lens{Name: l.Name, Available: l.Available, Reason: l.Reason, Command: l.Command})
	}
	return out
}

// ceilingsEvidence names the limits.json the read sets were measured
// against, or why there were no ceilings.
func ceilingsEvidence(snap *home.Snapshot) *sharedv1.Evidence {
	if snap.Ceilings != nil {
		return wire.Evidence(home.From(snap.Ceilings.Source, snap.Now))
	}
	for _, lens := range snap.Lenses {
		if lens.Name == home.LimitsLens {
			return wire.Evidence(home.Unknown(lens.Reason))
		}
	}
	return wire.Evidence(home.Unknown("no ceilings source configured"))
}
