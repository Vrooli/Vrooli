// Package operator serves the operator's side of the desk: the needs-you
// feed, answers, direction, verdicts and the courier (SYSTEM.md §7, §9).
// Every write goes through internal/writer.
package operatorapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/courier"
	"research-desk/internal/home"
	"research-desk/internal/wire"
	"research-desk/internal/writer"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
)

// Snapshotter is the home reader seam.
type Snapshotter interface {
	Snapshot(ctx context.Context, opts home.Options) (*home.Snapshot, error)
}

// Deps wires the operator handler.
type Deps struct {
	// ResearchDir is the research home the file route serves.
	ResearchDir string
	Reader      Snapshotter
	Writer      *writer.Writer
	Courier     *courier.Courier
	Clock       schedule.NowSource
	Logger      *log.Logger
	// TestWait bounds how long TestCourier waits for delivery.
	TestWait time.Duration
}

type handler struct{ deps Deps }

// NewHandler returns the Connect handler.
func NewHandler(deps Deps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if deps.TestWait <= 0 {
		deps.TestWait = time.Minute
	}
	return &handler{deps: deps}
}

// WindDownText is the standard wind-down when the operator gives no words.
const WindDownText = "Finish the current unit, write the closeout, then end."

func (h *handler) snapshot(ctx context.Context) (*home.Snapshot, error) {
	snap, err := h.deps.Reader.Snapshot(ctx, home.Options{})
	if err == nil {
		return snap, nil
	}
	return nil, h.fail("snapshot", err)
}

// fail maps an error to its wire code by cause (wire.Error).
func (h *handler) fail(op string, err error) error {
	return wire.Error(h.deps.Logger, "operator: "+op, err)
}

func (h *handler) commission(snap *home.Snapshot, id string) (*home.Commission, error) {
	c, err := snap.Find(id)
	return c, h.fail("commission", err)
}

func steeringWarning(c *home.Commission) string {
	if c.Steering.Value {
		return ""
	}
	return "this charter does not tell the scheduler to read OPERATOR.md, so the entry will not be read until steering is enabled"
}

func (h *handler) ListNeedsYou(ctx context.Context, _ *connect.Request[operatorv1.ListNeedsYouRequest]) (*connect.Response[operatorv1.ListNeedsYouResponse], error) {
	plan, snap, err := h.deps.Courier.Plan(ctx)
	if err != nil {
		return nil, h.fail("plan", err)
	}
	resp := &operatorv1.ListNeedsYouResponse{EntryCharsMax: int32(h.deps.Writer.EntryChars()), Courier: &operatorv1.CourierStatus{
		GlobalOn:     plan.Switches.Global.On,
		HubRead:      !plan.HubSkipped && plan.HubRead == nil,
		SwitchesFile: filepath.Join(h.deps.Writer.StateDir(), courier.SwitchFile),
	}}
	switch {
	case plan.HubSkipped:
		resp.Courier.HubReason = "no courier switch is on"
	case plan.HubRead != nil:
		resp.Courier.HubReason = plan.HubRead.Error()
	}
	// A read never takes the lock: it only asks which role this has.
	if err := h.deps.Writer.Role(); err != nil {
		resp.Courier.WriterReason = err.Error()
	} else {
		resp.Courier.Writer = true
	}
	for _, c := range snap.Commissions {
		if c.State == home.StateEnded {
			continue
		}
		on, why := plan.Switches.ForCommission(c)
		resp.Courier.Commissions = append(resp.Courier.Commissions, &operatorv1.CommissionSwitch{CommissionId: c.ID, On: on, Reason: why})
	}
	for i := range plan.Steps {
		s := &plan.Steps[i]
		if s.Item == nil {
			resp.Courier.Orphans = append(resp.Courier.Orphans, step(s))
			continue
		}
		resp.Items = append(resp.Items, item(s))
		if s.Waiting() {
			resp.ItemCount++
		}
	}
	feed := snap.NeedsYou()
	for _, l := range feed.Legacy {
		resp.Items = append(resp.Items, olderList(l))
	}
	resp.OlderListCount = int32(len(feed.Legacy))
	resp.Unreadable = wire.EvidenceList(feed.Unreadable)
	return connect.NewResponse(resp), nil
}

var reasonWire = map[home.DeliveryReason]operatorv1.DeliveryReason{
	home.ReasonSchedulerDelivered: operatorv1.DeliveryReason_DELIVERY_REASON_SCHEDULER_DELIVERED,
	home.ReasonAnswered:           operatorv1.DeliveryReason_DELIVERY_REASON_ANSWERED,
	home.ReasonSecret:             operatorv1.DeliveryReason_DELIVERY_REASON_SECRET,
	home.ReasonNoCommission:       operatorv1.DeliveryReason_DELIVERY_REASON_NO_COMMISSION,
	home.ReasonNoSteering:         operatorv1.DeliveryReason_DELIVERY_REASON_NO_STEERING,
	home.ReasonNoRecommendation:   operatorv1.DeliveryReason_DELIVERY_REASON_NO_RECOMMENDATION,
	home.ReasonAsk:                operatorv1.DeliveryReason_DELIVERY_REASON_ASK,
	home.ReasonUnreadable:         operatorv1.DeliveryReason_DELIVERY_REASON_UNREADABLE,
	home.ReasonIDReused:           operatorv1.DeliveryReason_DELIVERY_REASON_ID_REUSED,
	home.ReasonSwitchOff:          operatorv1.DeliveryReason_DELIVERY_REASON_SWITCH_OFF,
	home.ReasonSchedulerGrace:     operatorv1.DeliveryReason_DELIVERY_REASON_SCHEDULER_GRACE,
	home.ReasonHubUnavailable:     operatorv1.DeliveryReason_DELIVERY_REASON_HUB_UNAVAILABLE,
	home.ReasonInHub:              operatorv1.DeliveryReason_DELIVERY_REASON_IN_HUB,
	home.ReasonHubDelivered:       operatorv1.DeliveryReason_DELIVERY_REASON_HUB_DELIVERED,
	home.ReasonHubAnswered:        operatorv1.DeliveryReason_DELIVERY_REASON_HUB_ANSWERED,
	home.ReasonOrphaned:           operatorv1.DeliveryReason_DELIVERY_REASON_ORPHANED,
	home.ReasonBaseline:           operatorv1.DeliveryReason_DELIVERY_REASON_BASELINE,
	home.ReasonNotice:             operatorv1.DeliveryReason_DELIVERY_REASON_NOTICE,
}

func step(s *courier.Step) *operatorv1.CourierStep {
	out := &operatorv1.CourierStep{Action: string(s.Action), Why: s.Why, On: s.On, SwitchReason: s.Switch, Key: s.Key, Reason: reasonWire[s.Reason]}
	if s.Ask != nil {
		out.AskId, out.AskState = s.Ask.ID, s.Ask.State
	}
	return out
}

func item(s *courier.Step) *operatorv1.NeedsYouItem {
	f := s.Item
	answerable, why := f.Answerable()
	delivery, deliveryCode, deliveryWhy := f.Delivery()
	out := &operatorv1.NeedsYouItem{
		Kind: operatorv1.NeedsYouKind_NEEDS_YOU_KIND_ITEM, Id: f.Item.ID, Title: f.Item.Title, Body: f.Item.Body, CommissionId: f.CommissionID, Sources: f.Sources,
		RecommendedProblem: f.Item.RecommendedProblem, Options: options(f.Item.Choices()),
		Packet: f.Item.Packet, Secret: f.Item.Secret, DeliveredByScheduler: f.Item.Delivered,
		Answerable: answerable, NotAnswerableReason: why,
		Answer: wire.Entry(f.Answer), Delivered: wire.Entry(f.Delivered),
		Delivery: deliveryProto(delivery), DeliveryReason: deliveryWhy, DeliveryReasonCode: reasonWire[deliveryCode], Courier: step(s),
	}
	if !answerable {
		// What the item itself says to do, so it is acted on where it says.
		out.Instruction = f.Item.Instruction()
		for _, l := range f.Item.Links() {
			out.Links = append(out.Links, &operatorv1.Link{Label: l.Label, Target: l.Target})
		}
	}
	if f.Item.Recommended != nil {
		out.Recommended = &operatorv1.Option{Key: f.Item.Recommended.Key, Label: f.Item.Recommended.Label}
	}
	if f.Answer != nil {
		out.Read = wire.Flag(f.Read)
	}
	if s.Reused {
		// The answer and delivery on record belong to an earlier item.
		out.Answer, out.Read, out.Delivered = nil, nil, nil
		out.Delivery, out.DeliveryReason, out.DeliveryReasonCode = operatorv1.Delivery_DELIVERY_NONE, s.Why, operatorv1.DeliveryReason_DELIVERY_REASON_ID_REUSED
	}
	return out
}

// olderList is an older needs-you list as one feed row: readable in place,
// never sent and never answerable here.
func olderList(l home.Legacy) *operatorv1.NeedsYouItem {
	const why = "an older list, not in the item format: shown, never sent"
	where := "Act where each line says."
	if l.CommissionID != "" {
		where = "Act where each line says; direction for " + l.CommissionID + " goes in its OPERATOR.md (Say)."
	}
	out := &operatorv1.NeedsYouItem{
		Kind: operatorv1.NeedsYouKind_NEEDS_YOU_KIND_OLDER_LIST, Title: "Older list in " + filepath.Base(l.Source),
		Body: l.Text, CommissionId: l.CommissionID, Sources: []string{l.Source},
		NotAnswerableReason: "an older list has no item IDs or options, so it cannot be answered here",
		Instruction:         where,
		Delivery:            operatorv1.Delivery_DELIVERY_NONE, DeliveryReason: why,
		DeliveryReasonCode: operatorv1.DeliveryReason_DELIVERY_REASON_OLDER_LIST,
		Courier:            &operatorv1.CourierStep{Action: "none", Why: why, Reason: operatorv1.DeliveryReason_DELIVERY_REASON_OLDER_LIST},
	}
	for _, link := range l.Links() {
		out.Links = append(out.Links, &operatorv1.Link{Label: link.Label, Target: link.Target})
	}
	return out
}

func options(list []home.Option) []*operatorv1.Option {
	out := make([]*operatorv1.Option, 0, len(list))
	for _, o := range list {
		out = append(out, &operatorv1.Option{Key: o.Key, Label: o.Label})
	}
	return out
}

func deliveryProto(d home.Delivery) operatorv1.Delivery {
	switch d {
	case home.DeliverAsk:
		return operatorv1.Delivery_DELIVERY_ASK
	case home.DeliverNotice:
		return operatorv1.Delivery_DELIVERY_NOTICE
	case home.DeliverNone:
		return operatorv1.Delivery_DELIVERY_NONE
	}
	return operatorv1.Delivery_DELIVERY_UNSPECIFIED
}

func (h *handler) Answer(ctx context.Context, req *connect.Request[operatorv1.AnswerRequest]) (*connect.Response[operatorv1.AnswerResponse], error) {
	msg := req.Msg
	snap, err := h.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.commission(snap, msg.CommissionId)
	if err != nil {
		return nil, err
	}
	feed := snap.NeedsYou()
	if why := feed.Blocked(c.ID); why != "" {
		return nil, h.fail("answer", fmt.Errorf("%w: the answer is not written while the desk cannot read the item or the answers already given: %s", home.ErrRefused, why))
	}
	var f *home.FeedItem
	for i := range feed.Items {
		if feed.Items[i].CommissionID == c.ID && feed.Items[i].Item.ID == msg.ItemId {
			f = &feed.Items[i]
		}
	}
	if f == nil {
		return nil, h.fail("answer", home.WithCommand(home.Cause(home.ErrNotFound, fmt.Errorf("commission %s has no needs-you item %q (an ended commission's items are not answerable)", c.ID, msg.ItemId)), "research-desk needs-you"))
	}
	if ok, why := f.Answerable(); !ok {
		return nil, h.fail("answer", home.Cause(home.ErrRefused, errors.New(why)))
	}
	key, note := strings.TrimSpace(msg.OptionKey), strings.TrimSpace(msg.Note)
	label := ""
	if key == "other" {
		if note == "" {
			return nil, h.fail("answer", home.Cause(home.ErrInvalid, errors.New(`an "other" answer needs the operator's words in the note`)))
		}
	} else {
		choice, ok := f.Item.Choice(key)
		if !ok {
			return nil, h.fail("answer", home.Cause(home.ErrInvalid, fmt.Errorf(`item %s offers %s, or "other" with a note; not %q`, f.Item.ID, optionKeys(f.Item.Choices()), key)))
		}
		label = choice.Label
	}
	entry := home.Entry{At: h.deps.Clock.Now().UTC(), Kind: home.EntryAnswer, Subject: f.Item.ID, Via: home.ViaDesk, Body: home.AnswerBody(key, label, note)}
	path, text, written, err := h.operatorEntry(c.ID, entry, msg.DryRun)
	if err != nil {
		return nil, h.fail("answer", err)
	}
	return connect.NewResponse(&operatorv1.AnswerResponse{Entry: wire.Entry(&entry), Path: path, Warning: steeringWarning(c), Text: text, Written: written, Reading: wire.EntryReading(c.EntryReading())}), nil
}

// operatorEntry renders the exact text of an OPERATOR.md entry and, unless
// dryRun, appends it. The text is rendered first, so a write
// appends what a dry run just showed when the file has not changed.
func (h *handler) operatorEntry(id string, e home.Entry, dryRun bool) (path, text string, written bool, err error) {
	path, text, err = h.deps.Writer.RenderOperator(id, e)
	if err != nil || dryRun {
		return path, text, false, err
	}
	if _, err := h.deps.Writer.AppendOperator(id, e); err != nil {
		return path, text, false, err
	}
	return path, text, true, nil
}

func optionKeys(list []home.Option) string {
	keys := make([]string, 0, len(list))
	for _, o := range list {
		keys = append(keys, "`"+o.Key+"`")
	}
	return strings.Join(keys, ", ")
}

func (h *handler) AddEntry(ctx context.Context, req *connect.Request[operatorv1.AddEntryRequest]) (*connect.Response[operatorv1.AddEntryResponse], error) {
	msg := req.Msg
	snap, err := h.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.commission(snap, msg.CommissionId)
	if err != nil {
		return nil, err
	}
	entry := home.Entry{At: h.deps.Clock.Now().UTC(), Via: home.ViaDesk, Body: strings.TrimSpace(msg.Text)}
	switch msg.Kind {
	case operatorv1.EntryKind_ENTRY_KIND_DIRECTION:
		entry.Kind = home.EntryDirection
		if entry.Body == "" {
			return nil, h.fail("add entry", home.Cause(home.ErrInvalid, errors.New("direction needs the operator's words")))
		}
	case operatorv1.EntryKind_ENTRY_KIND_WIND_DOWN:
		entry.Kind = home.EntryWindDown
		if entry.Body == "" {
			entry.Body = WindDownText
		}
	default:
		return nil, h.fail("add entry", home.Cause(home.ErrInvalid, errors.New("kind must be ENTRY_KIND_DIRECTION or ENTRY_KIND_WIND_DOWN")))
	}
	path, text, written, err := h.operatorEntry(c.ID, entry, msg.DryRun)
	if err != nil {
		return nil, h.fail("add entry", err)
	}
	return connect.NewResponse(&operatorv1.AddEntryResponse{Entry: wire.Entry(&entry), Path: path, Warning: steeringWarning(c), Text: text, Written: written, Reading: wire.EntryReading(c.EntryReading())}), nil
}

func (h *handler) EnableSteering(ctx context.Context, req *connect.Request[operatorv1.EnableSteeringRequest]) (*connect.Response[operatorv1.EnableSteeringResponse], error) {
	snap, err := h.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.commission(snap, req.Msg.CommissionId)
	if err != nil {
		return nil, err
	}
	if req.Msg.DryRun {
		path, text, err := h.deps.Writer.RenderSteering(c.ID)
		if err != nil {
			return nil, h.fail("enable steering", err)
		}
		return connect.NewResponse(&operatorv1.EnableSteeringResponse{Path: path, Text: text}), nil
	}
	path, text, err := h.deps.Writer.EnableSteering(c.ID)
	if err != nil {
		return nil, h.fail("enable steering", err)
	}
	return connect.NewResponse(&operatorv1.EnableSteeringResponse{Path: path, Written: text != "", Text: text}), nil
}

func (h *handler) ListReviews(ctx context.Context, _ *connect.Request[operatorv1.ListReviewsRequest]) (*connect.Response[operatorv1.ListReviewsResponse], error) {
	snap, err := h.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	resp := &operatorv1.ListReviewsResponse{EntryCharsMax: int32(h.deps.Writer.EntryChars())}
	for _, e := range snap.Unreadable {
		if filepath.Base(e.Source) == home.ReviewsDir {
			resp.Unreadable = append(resp.Unreadable, wire.Evidence(e))
		}
	}
	for _, r := range snap.Reviews {
		out := &operatorv1.Review{
			Id: r.ID, Folder: r.Dir, Title: wire.Text(r.Title), Topics: r.Topics, TopicsEvidence: wire.Evidence(r.TopicsEvidence),
			Review: reviewFile(snap.ResearchDir, r.Review), ReviewHtml: reviewFile(snap.ResearchDir, r.HTML),
			Verdicts: wire.Entries(r.Verdicts), Unreadable: wire.EvidenceList(r.Unreadable),
		}
		if r.Verdicts != nil {
			out.UnparsedHeadings = wire.UnparsedHeadings(r.Verdicts.Unparsed)
		}
		for i := range r.Proposals {
			out.Proposals = append(out.Proposals, reviewFile(snap.ResearchDir, &r.Proposals[i]))
		}
		resp.Reviews = append(resp.Reviews, out)
	}
	return connect.NewResponse(resp), nil
}

func reviewFile(researchDir string, f *home.FileFact) *operatorv1.ReviewFile {
	if f == nil {
		return nil
	}
	out := &operatorv1.ReviewFile{File: wire.File(f)}
	if rel, err := filepath.Rel(researchDir, f.Path); err == nil {
		out.Url = FileRoute + filepath.ToSlash(rel)
	}
	return out
}

func (h *handler) AddVerdict(ctx context.Context, req *connect.Request[operatorv1.AddVerdictRequest]) (*connect.Response[operatorv1.AddVerdictResponse], error) {
	msg := req.Msg
	snap, err := h.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	r := snap.Review(msg.ReviewId)
	if r == nil {
		return nil, h.fail("add verdict", home.WithCommand(home.Cause(home.ErrNotFound, fmt.Errorf("no review folder %q under %s/reviews", msg.ReviewId, snap.ResearchDir)), "research-desk reviews"))
	}
	topic := snap.Topic(msg.Topic)
	if topic == nil {
		return nil, h.fail("add verdict", home.WithCommand(home.Cause(home.ErrNotFound, fmt.Errorf("no topic %q under %s/topics", msg.Topic, snap.ResearchDir)), "research-desk topics"))
	}
	entry := home.Entry{At: h.deps.Clock.Now().UTC(), Kind: home.EntryVerdict, Subject: topic.Slug, Via: home.ViaDesk, Body: strings.TrimSpace(msg.Text)}
	path, text, err := h.deps.Writer.RenderVerdict(r.ID, entry)
	if err != nil {
		return nil, h.fail("add verdict", err)
	}
	if msg.DryRun {
		return connect.NewResponse(&operatorv1.AddVerdictResponse{Entry: wire.Entry(&entry), Path: path, Text: text}), nil
	}
	if _, err := h.deps.Writer.AppendVerdict(r.ID, entry); err != nil {
		return nil, h.fail("add verdict", err)
	}
	return connect.NewResponse(&operatorv1.AddVerdictResponse{Entry: wire.Entry(&entry), Path: path, Text: text, Written: true}), nil
}

func (h *handler) SetCourier(ctx context.Context, req *connect.Request[operatorv1.SetCourierRequest]) (*connect.Response[operatorv1.SetCourierResponse], error) {
	msg := req.Msg
	if msg.Global == (msg.CommissionId != "") {
		return nil, h.fail("set courier", home.Cause(home.ErrInvalid, errors.New("name one commission, or set global for ATTENTION.md items, not both")))
	}
	var c *home.Commission
	if !msg.Global {
		snap, err := h.snapshot(ctx)
		if err != nil {
			return nil, err
		}
		if c, err = h.commission(snap, msg.CommissionId); err != nil {
			return nil, err
		}
	}
	sw, err := h.deps.Courier.Set(ctx, msg.CommissionId, msg.On)
	if err != nil {
		return nil, h.fail("set courier", err)
	}
	resp := &operatorv1.SetCourierResponse{SwitchesFile: filepath.Join(h.deps.Writer.StateDir(), courier.SwitchFile)}
	if c == nil {
		resp.On, resp.Reason = sw.Global.On, "global ATTENTION.md items; those present now are never sent"
	} else {
		resp.On, resp.Reason = sw.ForCommission(c)
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) TestCourier(ctx context.Context, _ *connect.Request[operatorv1.TestCourierRequest]) (*connect.Response[operatorv1.TestCourierResponse], error) {
	result, err := h.deps.Courier.Test(ctx, h.deps.TestWait)
	resp := &operatorv1.TestCourierResponse{AskId: result.Ask.ID, Key: result.Key, FirstDeliveredAt: wire.Timestamp(result.FirstDelivered), ExpiresAt: wire.Timestamp(result.Expires), WaitedSeconds: int32(result.Waited.Seconds())}
	if err != nil {
		// A read-only instance stays refused; any other failure is the Hub's.
		return nil, h.fail("test courier", home.Cause(home.ErrUnavailable, err))
	}
	return connect.NewResponse(resp), nil
}

// MarkSeen records a look at one commission or topic, or with all at every
// commission and every topic, so the next changes are reported
// from now.
func (h *handler) MarkSeen(ctx context.Context, req *connect.Request[operatorv1.MarkSeenRequest]) (*connect.Response[operatorv1.MarkSeenResponse], error) {
	msg := req.Msg
	subject := home.LookCommission
	switch {
	case msg.All && (msg.Subject != activityv1.ChangeSubject_CHANGE_SUBJECT_UNSPECIFIED || msg.Id != ""):
		return nil, h.fail("mark seen", home.Cause(home.ErrInvalid, errors.New("give all, or a subject and id, not both")))
	case msg.All:
	case msg.Subject == activityv1.ChangeSubject_CHANGE_SUBJECT_COMMISSION:
	case msg.Subject == activityv1.ChangeSubject_CHANGE_SUBJECT_TOPIC:
		subject = home.LookTopic
	default:
		return nil, h.fail("mark seen", home.Cause(home.ErrInvalid, errors.New("subject must be CHANGE_SUBJECT_COMMISSION or CHANGE_SUBJECT_TOPIC, or set all")))
	}
	if err := h.deps.Writer.Held(); err != nil {
		return nil, h.fail("mark seen", err)
	}
	snap, err := h.deps.Reader.Snapshot(ctx, home.Options{MissionCheck: true})
	if err != nil {
		return nil, h.fail("snapshot", err)
	}
	resp := &operatorv1.MarkSeenResponse{At: wire.Timestamp(snap.Now)}
	mark := func(subject, id string) error {
		look, err := snap.Look(subject, id)
		if err != nil {
			return err
		}
		resp.At = wire.Timestamp(look.At)
		resp.Path, err = h.deps.Writer.WriteState(home.LookName(subject, id), look)
		if err == nil && subject == home.LookTopic {
			resp.Topics++
		} else if err == nil {
			resp.Commissions++
		}
		return err
	}
	if !msg.All {
		if err := mark(subject, msg.Id); err != nil {
			return nil, h.fail("mark seen", err)
		}
		return connect.NewResponse(resp), nil
	}
	for _, c := range snap.Commissions {
		if err := mark(home.LookCommission, c.ID); err != nil {
			return nil, h.fail("mark seen", fmt.Errorf("commission %s (after %d commissions): %w", c.ID, resp.Commissions, err))
		}
	}
	for _, t := range snap.Topics {
		if err := mark(home.LookTopic, t.Slug); err != nil {
			return nil, h.fail("mark seen", fmt.Errorf("topic %s (after %d commissions and %d topics): %w", t.Slug, resp.Commissions, resp.Topics, err))
		}
	}
	resp.Path = ""
	return connect.NewResponse(resp), nil
}
