package courier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"

	"research-desk/internal/home"
	"research-desk/internal/peer"

	convv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	convconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	notifv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notifconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	hubsharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
)

// Ask is the courier's view of one Notification Hub ask.
type Ask struct {
	ID string
	// Key is the desk's identity for the ask (AskPrefix + commission/ID, or
	// TestPrefix + stamp), read back from the "Desk key:" line of the
	// technical details the desk wrote, or from an older ask's question.
	Key   string
	State string
	// RequesterRef is the id of the event that opened the ask; the ask is
	// closed by this id, whether or not it carries a subject key.
	RequesterRef string
	// Question, BigPicture, TechnicalDetails, Recommended and Options are the
	// decision as the Hub holds it, to tell whether it is still the item's
	// current question.
	Question, BigPicture, TechnicalDetails, Recommended string
	Options                                             []home.Option
	Answer                                              string
	AnswerLabel                                         string
	Note                                                string
	// AnsweredAt is when the operator answered; zero while no one has.
	AnsweredAt time.Time
	CreatedAt  time.Time
	Deadline   time.Time
	// FirstDelivered is when a device first received it; zero until then.
	FirstDelivered time.Time
}

// Open reports whether the ask still awaits the operator.
func (a Ask) Open() bool { return a.State == "pending" || a.State == "escalated" }

// AskSpec is one decision to raise. Its technical details start with a
// "Desk key: <Key>" line, which is how the desk finds the ask again; the
// question and big picture are for the operator and carry no desk IDs.
type AskSpec struct {
	Key string
	// Subject is the short name in the Hub's "Decision needed: <Subject>"
	// title; empty falls back to the last segment of Key.
	Subject          string
	Question         string
	BigPicture       string
	TechnicalDetails string
	Options          []home.Option
	Recommended      string
	Reason           string
	// Deadline is zero for the Hub's default; a set deadline changes the
	// event on every pass, so only a one-off ask (the delivery test) sets it.
	Deadline time.Time
	// Explainer is the decision's page, carried inline in the event; the Hub
	// stores it as the ask's explainer. Every ask the courier raises has one.
	Explainer string
}

// NoticeSpec is one plain notice, without answer options.
type NoticeSpec struct {
	Key   string
	Title string
	Body  string
}

// Notice is the Hub's record of a notice.
type Notice struct {
	ID string
	// Key is the notice's dedupe key: NoticePrefix + owner/ID.
	Key       string
	Delivered bool
	State     string
}

// Hub is the courier's Notification Hub seam. Decisions and notices go out as
// events through Vrooli Events; reading comes back from the Hub.
type Hub interface {
	// Sendable says why nothing can be sent now, or nil.
	Sendable(ctx context.Context) error
	// Ask sends one decision request for the item's subject. The same request
	// is a no-op at the Hub; a changed request updates the open decision in
	// place, or replaces it when its answer menu changed.
	Ask(ctx context.Context, spec AskSpec) error
	// RequestRef is the id of the request Ask sends for spec. An open ask with
	// another RequesterRef was raised before the item changed, or before asks
	// carried a subject key, and is raised again.
	RequestRef(spec AskSpec) string
	// Withdraw closes an open ask whose item no longer needs the operator, by
	// the id of the event that opened it. The reason is shown to the operator
	// on the closed decision. Closing is never an answer.
	Withdraw(ctx context.Context, ask Ask, reason string, at time.Time) error
	// Notify publishes one notice, deduplicated the same way.
	Notify(ctx context.Context, spec NoticeSpec) error
	// Asks returns every ask the desk raised whose key starts with prefix.
	Asks(ctx context.Context, prefix string) ([]Ask, error)
	// Notices returns the desk's notices, by key.
	Notices(ctx context.Context) (map[string]Notice, error)
}

// Source is the desk's scenario name on its events and Hub records.
const Source = "research-desk"

// Event types. The Hub subscribes to **.decision_requested.v1 and
// **.notification.v1 (notification-hub internal/integrations), so a new
// producer needs no Hub change and no grant.
var (
	DecisionEvent = operatordecision.DecisionRequestedEventType("research_desk", "needs_you")
	NoticeEvent   = "research_desk.needs_you.notification.v1"
)

// EventHub raises decisions and notices through Vrooli Events, the route every
// scenario-originated decision takes (api-core operatordecision); a decision
// carries its page inline. Reading asks and notices back comes from the Hub,
// which shows the desk its asks only under its ask-reader identity.
type EventHub struct {
	// Events resolves the Vrooli Events API base URL on each call.
	Events func(ctx context.Context) (string, error)
	// Hub dials Notification Hub; it carries no credential of its own.
	Hub *peer.Peer
	// AskReader is the desk's ask-reader identity, presented on every read of
	// an ask and never on a notice read. Without one the Hub refuses to show
	// the desk its asks, and Asks says so.
	AskReader *askreader.Tokens
}

// Sendable reports whether Vrooli Events can be reached.
func (h *EventHub) Sendable(ctx context.Context) error {
	if _, err := h.Events(ctx); err != nil {
		return fmt.Errorf("vrooli-events is not reachable, so nothing can be sent: %w", err)
	}
	return nil
}

func (h *EventHub) publish(ctx context.Context, event eventbus.DomainEvent) error {
	base, err := h.Events(ctx)
	if err != nil {
		return fmt.Errorf("vrooli-events is not reachable, so nothing can be sent: %w", err)
	}
	if err := (eventbus.Client{BaseURL: base, HTTPClient: peer.Client()}).PublishDomainEvent(ctx, event); err != nil {
		return fmt.Errorf("publish %s: %w", event.EventType, err)
	}
	return nil
}

// event builds an event with a zero Occurred, which keeps the event id a
// function of the content alone: every pass that publishes the same item
// publishes the same event.
func event(eventType string, payload map[string]any) eventbus.DomainEvent {
	return eventbus.DomainEvent{Source: Source, EventType: eventType, Payload: payload}
}

// subjectKey is the Hub subject key of an ask key: the Hub keeps one open
// decision per subject, so an item's ask is updated or closed, never repeated.
func subjectKey(key string) string { return operatordecision.SubjectKeyFrom(key) }

// askFacts is the decision spec states, checked against the contract every
// producer meets.
func askFacts(spec AskSpec) (operatordecision.Facts, error) {
	subject := spec.Subject
	if strings.TrimSpace(subject) == "" {
		subject = spec.Key[strings.LastIndex(spec.Key, "/")+1:]
	}
	facts := operatordecision.Facts{
		Subject:              operatordecision.Clip(subject, 60),
		Question:             spec.Question,
		Recommended:          spec.Recommended,
		RecommendationReason: spec.Reason,
		BigPicture:           spec.BigPicture,
		TechnicalDetails:     spec.TechnicalDetails,
		Urgency:              "normal",
		Deadline:             spec.Deadline,
		SubjectKey:           subjectKey(spec.Key),
		Explainer:            spec.Explainer,
	}
	for _, o := range spec.Options {
		facts.Options = append(facts.Options, operatordecision.Option{Key: o.Key, Label: o.Label})
	}
	if err := facts.Validate(); err != nil {
		return operatordecision.Facts{}, fmt.Errorf("the item cannot be a Hub decision (%w); answer it in the desk", err)
	}
	return facts, nil
}

// askEvent is the decision request for spec.
func askEvent(spec AskSpec) (eventbus.DomainEvent, error) {
	facts, err := askFacts(spec)
	if err != nil {
		return eventbus.DomainEvent{}, err
	}
	return event(DecisionEvent, facts.Payload()), nil
}

// Ask publishes the decision, its page inline.
func (h *EventHub) Ask(ctx context.Context, spec AskSpec) error {
	e, err := askEvent(spec)
	if err != nil {
		return err
	}
	return h.publish(ctx, e)
}

// RequestRef is the id of the event Ask publishes; "" when the item cannot be
// a Hub decision.
func (h *EventHub) RequestRef(spec AskSpec) string {
	e, err := askEvent(spec)
	if err != nil {
		return ""
	}
	return operatordecision.EventID(e)
}

// Withdraw closes an ask by the id of the event that opened it. at is when the
// desk saw the ask was no longer needed; it makes each pass's withdrawal a
// distinct event, so one lost before the Hub subscribed is sent again.
func (h *EventHub) Withdraw(ctx context.Context, ask Ask, reason string, at time.Time) error {
	if ask.RequesterRef == "" {
		return fmt.Errorf("the Hub gave no request id for ask %s, so it cannot be closed", ask.ID)
	}
	w := operatordecision.Withdrawal{RequesterRef: ask.RequesterRef, Reason: reason, AssertedAt: at}
	if err := w.Validate(); err != nil {
		return err
	}
	return h.publish(ctx, operatordecision.BuildWithdrawalEventTyped(Source, DecisionEvent, at, w))
}

// Notify publishes the notice. Its dedupe key is the desk's key, so the Hub
// never folds two items into one notice.
func (h *EventHub) Notify(ctx context.Context, spec NoticeSpec) error {
	return h.publish(ctx, event(NoticeEvent, map[string]any{
		"title": spec.Title, "body": spec.Body, "urgency": "normal", "sensitivity": sensitivity,
		"category": "research", "dedupe_key": spec.Key,
	}))
}

// sensitivity is the Hub label for research notices.
const sensitivity = "private"

func (h *EventHub) clients(ctx context.Context) (convconnect.ConversationsServiceClient, notifconnect.NotificationsServiceClient, error) {
	client, base, opt, err := h.Hub.Dial(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("%w; answer in the desk instead", err)
	}
	asks := []connect.ClientOption{opt}
	if h.AskReader != nil {
		asks = append(asks, connect.WithInterceptors(h.AskReader.Interceptor()))
	}
	return convconnect.NewConversationsServiceClient(client, base, asks...), notifconnect.NewNotificationsServiceClient(client, base, opt), nil
}

// Asks returns the Hub's asks from this source whose key starts with prefix,
// read as the desk's ask-reader.
func (h *EventHub) Asks(ctx context.Context, prefix string) ([]Ask, error) {
	conv, _, err := h.clients(ctx)
	if err != nil {
		return nil, err
	}
	var (
		out    []Ask
		cursor string
	)
	for page := 0; page < 50; page++ {
		resp, err := conv.ListAsks(ctx, connect.NewRequest(&convv1.ListAsksRequest{Limit: 200, Cursor: cursor, Query: Source, StateFilter: "all"}))
		if err != nil {
			return nil, fmt.Errorf("notification-hub ListAsks: %w", err)
		}
		for _, a := range resp.Msg.GetAsks() {
			if a.GetSource() != Source {
				continue
			}
			if ask := askFromProto(a); ask.Key != "" && strings.HasPrefix(ask.Key, prefix) {
				out = append(out, ask)
			}
		}
		if cursor = resp.Msg.GetNextCursor(); cursor == "" {
			return out, nil
		}
	}
	return nil, errors.New("notification-hub ListAsks did not finish within 50 pages")
}

// noticeWindow is how many recent notifications are read back for notices.
const noticeWindow = 500

// Notices reads the operator's recent notifications and keeps the desk's.
func (h *EventHub) Notices(ctx context.Context) (map[string]Notice, error) {
	_, notif, err := h.clients(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := notif.List(ctx, connect.NewRequest(&notifv1.ListRequest{Limit: noticeWindow}))
	if err != nil {
		return nil, fmt.Errorf("notification-hub List: %w", err)
	}
	out := map[string]Notice{}
	for _, n := range resp.Msg.GetNotifications() {
		if !strings.HasPrefix(n.GetDedupeKey(), NoticePrefix) {
			continue
		}
		delivered := n.GetState() == hubsharedv1.NotificationState_NOTIFICATION_STATE_DELIVERED
		if old, ok := out[n.GetDedupeKey()]; ok && old.Delivered {
			continue
		}
		out[n.GetDedupeKey()] = Notice{ID: n.GetId(), Key: n.GetDedupeKey(), State: n.GetState().String(), Delivered: delivered}
	}
	return out, nil
}

// testQuestion starts every delivery-test question; the stamp follows it.
const testQuestion = "Research desk delivery test "

// deskKeyLabel opens the technical-details line that names the ask's key.
const deskKeyLabel = "Desk key: "

// deskKeyLine is the technical-details line naming key.
func deskKeyLine(key string) string { return deskKeyLabel + key }

// keyOf reads the desk's key back from an ask it wrote: the "Desk key:" line
// of its technical details, else (an ask raised before the desk wrote that
// line) the "<commission> · <ID> — " start of its question.
func keyOf(question, technical string) string {
	for _, line := range strings.Split(technical, "\n") {
		if key, ok := strings.CutPrefix(strings.TrimSpace(line), deskKeyLabel); ok {
			if key = strings.TrimSpace(key); strings.HasPrefix(key, AskPrefix) || strings.HasPrefix(key, TestPrefix) {
				return key
			}
		}
	}
	if rest, ok := strings.CutPrefix(question, testQuestion); ok {
		stamp, _, _ := strings.Cut(rest, " ")
		return TestPrefix + stamp
	}
	commission, rest, ok := strings.Cut(question, " · ")
	if !ok {
		return ""
	}
	id, _, ok := strings.Cut(rest, " — ")
	if !ok || commission == "" || id == "" || strings.ContainsAny(commission+id, " /") {
		return ""
	}
	return AskPrefix + commission + "/" + id
}

func askFromProto(a *convv1.Ask) Ask {
	return Ask{
		ID: a.GetId(), Key: keyOf(a.GetQuestion(), a.GetTechnicalDetails()), State: a.GetState(), RequesterRef: a.GetRequesterRef(),
		Question: a.GetQuestion(), BigPicture: a.GetBigPicture(), TechnicalDetails: a.GetTechnicalDetails(), Recommended: a.GetRecommended(), Options: askOptions(a),
		Answer: a.GetAnswer(), AnswerLabel: a.GetAnswerLabel(), Note: a.GetNote(), AnsweredAt: parseTime(a.GetAnsweredAt()),
		CreatedAt: parseTime(a.GetCreatedAt()), Deadline: parseTime(a.GetDeadline()), FirstDelivered: parseTime(a.GetFirstDeliveredAt()),
	}
}

// askOptions are the options the desk offered; the Hub's own free-form
// "other" answer is not one of them.
func askOptions(a *convv1.Ask) []home.Option {
	var out []home.Option
	for _, o := range a.GetOptions() {
		if o.GetKey() != "other" {
			out = append(out, home.Option{Key: o.GetKey(), Label: o.GetLabel()})
		}
	}
	return out
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
