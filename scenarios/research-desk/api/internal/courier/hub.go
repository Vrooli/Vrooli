package courier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
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
	// TestPrefix + stamp), read back from the question the desk wrote.
	Key         string
	State       string
	Answer      string
	AnswerLabel string
	Note        string
	CreatedAt   time.Time
	Deadline    time.Time
	// FirstDelivered is when a device first received it; zero until then.
	FirstDelivered time.Time
}

// Open reports whether the ask still awaits the operator.
func (a Ask) Open() bool { return a.State == "pending" || a.State == "escalated" }

// AskSpec is one decision to raise. Its question starts with "<commission> ·
// <ID> — ", which is how the desk finds the ask again.
type AskSpec struct {
	Key         string
	Question    string
	BigPicture  string
	Options     []home.Option
	Recommended string
	Reason      string
	// Deadline is zero for the Hub's default; a set deadline changes the
	// event on every pass, so only a one-off ask (the delivery test) sets it.
	Deadline time.Time
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

// Hub is the courier's Notification Hub seam. Raising goes out as events;
// reading comes back from the Hub. Nothing can withdraw an ask the desk raised.
type Hub interface {
	// Sendable says why nothing can be sent now, or nil.
	Sendable(ctx context.Context) error
	// Ask publishes one decision request; the same request is the same event,
	// so a repeat is deduplicated by Vrooli Events and the Hub.
	Ask(ctx context.Context, spec AskSpec) error
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

// EventHub raises through Vrooli Events, the route every scenario-originated
// decision takes (api-core operatordecision), and reads asks and notices back
// from Notification Hub, whose reads need no credential.
type EventHub struct {
	// Events resolves the Vrooli Events API base URL on each call.
	Events func(ctx context.Context) (string, error)
	// Hub dials Notification Hub; it carries no credential.
	Hub *peer.Peer
}

// Sendable reports whether Vrooli Events can be reached.
func (h *EventHub) Sendable(ctx context.Context) error {
	if _, err := h.Events(ctx); err != nil {
		return fmt.Errorf("vrooli-events is not reachable, so nothing can be sent: %w", err)
	}
	return nil
}

func (h *EventHub) publish(ctx context.Context, eventType string, payload map[string]any) error {
	base, err := h.Events(ctx)
	if err != nil {
		return fmt.Errorf("vrooli-events is not reachable, so nothing can be sent: %w", err)
	}
	// A zero Occurred keeps the event id a function of the content alone, so
	// every pass that publishes the same item publishes the same event.
	event := eventbus.DomainEvent{Source: Source, EventType: eventType, Payload: payload}
	if err := (eventbus.Client{BaseURL: base, HTTPClient: peer.Client()}).PublishDomainEvent(ctx, event); err != nil {
		return fmt.Errorf("publish %s: %w", eventType, err)
	}
	return nil
}

// Ask publishes the decision.
func (h *EventHub) Ask(ctx context.Context, spec AskSpec) error {
	facts := operatordecision.Facts{
		Subject:              clip(spec.Key[strings.LastIndex(spec.Key, "/")+1:], 60),
		Question:             spec.Question,
		Recommended:          spec.Recommended,
		RecommendationReason: spec.Reason,
		BigPicture:           spec.BigPicture,
		Urgency:              "normal",
		Deadline:             spec.Deadline,
	}
	for _, o := range spec.Options {
		facts.Options = append(facts.Options, operatordecision.Option{Key: o.Key, Label: o.Label})
	}
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("the item cannot be a Hub decision (%w); answer it in the desk", err)
	}
	return h.publish(ctx, DecisionEvent, facts.Payload())
}

// Notify publishes the notice. Its dedupe key is the desk's key, so the Hub
// never folds two items into one notice.
func (h *EventHub) Notify(ctx context.Context, spec NoticeSpec) error {
	return h.publish(ctx, NoticeEvent, map[string]any{
		"title": spec.Title, "body": spec.Body, "urgency": "normal", "sensitivity": sensitivity,
		"category": "research", "dedupe_key": spec.Key,
	})
}

// sensitivity is the Hub label for research notices.
const sensitivity = "private"

func (h *EventHub) clients(ctx context.Context) (convconnect.ConversationsServiceClient, notifconnect.NotificationsServiceClient, error) {
	client, base, opt, err := h.Hub.Dial(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("%w; answer in the desk instead", err)
	}
	return convconnect.NewConversationsServiceClient(client, base, opt), notifconnect.NewNotificationsServiceClient(client, base, opt), nil
}

// Asks pages through the Hub's asks from this source and keeps those whose
// key starts with prefix.
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

// keyOf reads the desk's key back from a question it wrote.
func keyOf(question string) string {
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
		ID: a.GetId(), Key: keyOf(a.GetQuestion()), State: a.GetState(),
		Answer: a.GetAnswer(), AnswerLabel: a.GetAnswerLabel(), Note: a.GetNote(),
		CreatedAt: parseTime(a.GetCreatedAt()), Deadline: parseTime(a.GetDeadline()), FirstDelivered: parseTime(a.GetFirstDeliveredAt()),
	}
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
