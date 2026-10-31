package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"git-control-tower/internal/dbschema"
	"git-control-tower/internal/proposals"
	"git-control-tower/internal/reviewjobs"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
)

// Operator notifications are generic `*.notification.v1` Vrooli Events domain
// events. Notification Hub subscribes to that suffix, owns delivery, and reads
// the categories declared in .vrooli/notifications.json; the keys below must
// match that file so recipient preferences apply.
const (
	gctNotificationSource              = "git-control-tower"
	proposalNotificationEventType      = "git_control_tower.proposal.notification.v1"
	commitMessageNotificationEventType = "git_control_tower.commit_message.notification.v1"

	notificationCategoryProposalReceived    = "proposal-received"
	notificationCategoryCommitMessageReady  = "commit-message-ready"
	notificationCategoryCommitMessageFailed = "commit-message-failed"

	notificationSweepInterval = 15 * time.Second
	// A drafting result older than this when first seen is not announced.
	notificationMaxAge     = 24 * time.Hour
	notificationSweepLimit = 50
	notificationWatermark  = "__watermark__"
	notificationTextLimit  = 240
)

// gctNotifier publishes operator notifications. Publication is best effort:
// a failure is logged and never fails the operation that caused it.
type gctNotifier struct {
	publisher domainEventPublisher
	db        dbschema.DB
	jobs      *reviewjobs.Store
	now       func() time.Time
	// resolveBase returns the operator-facing GCT UI URL, or "" when unknown.
	resolveBase func(context.Context) string
	// repoName returns a display name for a repository id.
	repoName func(context.Context, string) string

	baseMu sync.Mutex
	base   string
}

func newGCTNotifier(publisher domainEventPublisher, db dbschema.DB, jobs *reviewjobs.Store) (*gctNotifier, error) {
	n := &gctNotifier{publisher: publisher, db: db, jobs: jobs, now: func() time.Time { return time.Now().UTC() }}
	n.resolveBase = func(ctx context.Context) string {
		base, err := discovery.ResolveExternalURLForNotice(ctx, gctNotificationSource)
		if err != nil {
			return ""
		}
		return base
	}
	n.repoName = func(context.Context, string) string { return "" }
	if db == nil {
		return n, nil
	}
	return n, n.ensureSchema(context.Background())
}

// ensureSchema creates the receipt table. The first creation records a
// watermark so drafting jobs that ended before notifications existed are
// never announced as new.
func (n *gctNotifier) ensureSchema(ctx context.Context) error {
	if _, err := n.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gct_notification_receipts (
		subject TEXT PRIMARY KEY, kind TEXT NOT NULL, created_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	_, err := n.db.ExecContext(ctx, `INSERT INTO gct_notification_receipts (subject,kind,created_at) VALUES (?,?,?) ON CONFLICT(subject) DO NOTHING`,
		notificationWatermark, "watermark", n.now().Format(time.RFC3339Nano))
	return err
}

func (n *gctNotifier) watermark(ctx context.Context) (time.Time, error) {
	var raw string
	if err := n.db.QueryRowContext(ctx, `SELECT created_at FROM gct_notification_receipts WHERE subject=?`, notificationWatermark).Scan(&raw); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, raw)
}

// claim records that subject was announced. Only the first claim wins.
func (n *gctNotifier) claim(ctx context.Context, subject, kind string) (bool, error) {
	res, err := n.db.ExecContext(ctx, `INSERT INTO gct_notification_receipts (subject,kind,created_at) VALUES (?,?,?) ON CONFLICT(subject) DO NOTHING`,
		subject, kind, n.now().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows == 1, err
}

func (n *gctNotifier) release(ctx context.Context, subject string) {
	if _, err := n.db.ExecContext(ctx, `DELETE FROM gct_notification_receipts WHERE subject=?`, subject); err != nil {
		log.Printf("warn: release notification receipt %s: %v", subject, err)
	}
}

func (n *gctNotifier) enabled() bool {
	if n == nil || n.publisher == nil {
		return false
	}
	if switchable, ok := n.publisher.(interface{ Enabled() bool }); ok {
		return switchable.Enabled()
	}
	return true
}

// baseURL caches only a resolved URL, so a discovery outage is retried.
func (n *gctNotifier) baseURL(ctx context.Context) string {
	n.baseMu.Lock()
	defer n.baseMu.Unlock()
	if n.base == "" && n.resolveBase != nil {
		n.base = strings.TrimRight(strings.TrimSpace(n.resolveBase(ctx)), "/")
	}
	return n.base
}

// NotifyProposalCreated announces a stored proposal. Proposals a verified
// human created are not announced back to that human. The event is a pure
// function of the stored proposal, so a repeated call has the same event id.
func (n *gctNotifier) NotifyProposalCreated(ctx context.Context, repoName string, view proposals.View) {
	if !n.enabled() || view.ID == "" || view.CreatedBy.IsHuman() {
		return
	}
	event := proposalNotificationEvent(view, repoName, n.baseURL(ctx))
	if err := n.publisher.PublishDomainEvent(ctx, event); err != nil {
		log.Printf("warn: proposal notification %s: %v", view.ID, err)
	}
}

func proposalNotificationEvent(view proposals.View, repoName, base string) eventbus.DomainEvent {
	files := len(view.Files)
	noun := "files"
	if files == 1 {
		noun = "file"
	}
	// Title: what happened and where. Body: the subject, then its scope.
	title := "Commit proposal to review"
	if repoName = strings.TrimSpace(repoName); repoName != "" {
		title += " · " + repoName
	}
	where := fmt.Sprintf("%d %s", files, noun)
	if effort := strings.TrimPrefix(view.Work.EffortRef, "effort:"); effort != "" {
		label := effort
		if view.Work.Epoch != "" {
			label += " " + view.Work.Epoch
		}
		where += " · " + label
	}
	payload := map[string]any{
		"title":       title,
		"body":        boundNotificationText(view.Message.Subject) + "\n" + where,
		"category":    notificationCategoryProposalReceived,
		"urgency":     "normal",
		"sensitivity": "private",
		"dedupe_key":  view.ID,
		"correlation": map[string]any{"proposal_id": view.ID, "repository_id": view.RepositoryID},
	}
	if base != "" {
		payload["context_url"] = base + "/?proposal=" + url.QueryEscape(view.ID)
	}
	return eventbus.DomainEvent{Source: gctNotificationSource, EventType: proposalNotificationEventType, Occurred: view.CreatedAt.UTC(), Payload: payload}
}

// commitMessageNotificationEvent describes a drafting job that ended. A
// cancelled job is the operator's own action and is not announced.
func commitMessageNotificationEvent(row reviewjobs.AdvisorySummary, repoName, base string) (eventbus.DomainEvent, bool) {
	var title, body, category, urgency string
	where := strings.TrimSpace(repoName)
	switch row.State {
	case reviewjobs.Succeeded:
		var result struct {
			Title string `json:"title"`
		}
		_ = json.Unmarshal([]byte(row.ResultJSON), &result)
		title, category, urgency = "Commit message ready", notificationCategoryCommitMessageReady, "normal"
		body = boundNotificationText(result.Title)
		if body == "" {
			body = "Open Git Control Tower to review the suggestion."
		}
	case reviewjobs.Failed:
		title, category, urgency = "Commit message suggestion failed", notificationCategoryCommitMessageFailed, "normal"
		body = boundNotificationText(row.Error)
		if body == "" {
			body = "The drafting agent ended without a suggestion."
		}
	default:
		return eventbus.DomainEvent{}, false
	}
	if where != "" {
		title += " · " + where
	}
	payload := map[string]any{
		"title":       title,
		"body":        body,
		"category":    category,
		"urgency":     urgency,
		"sensitivity": "private",
		"dedupe_key":  row.ID,
		"correlation": map[string]any{"job_id": row.ID, "repository_id": row.RepositoryID},
	}
	if base != "" {
		payload["context_url"] = base + "/"
	}
	return eventbus.DomainEvent{Source: gctNotificationSource, EventType: commitMessageNotificationEventType, Occurred: row.UpdatedAt.UTC(), Payload: payload}, true
}

// NotifyFinishedDrafts announces each drafting job that ended after the
// watermark and within notificationMaxAge, once. It reports how many it
// published.
func (n *gctNotifier) NotifyFinishedDrafts(ctx context.Context) (int, error) {
	if !n.enabled() || n.jobs == nil || n.db == nil {
		return 0, nil
	}
	since, err := n.watermark(ctx)
	if err != nil {
		return 0, fmt.Errorf("notification watermark: %w", err)
	}
	if floor := n.now().Add(-notificationMaxAge); floor.After(since) {
		since = floor
	}
	rows, err := n.jobs.ListAdvisory(ctx, "staged-message", []reviewjobs.State{reviewjobs.Succeeded, reviewjobs.Failed}, notificationSweepLimit)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, row := range rows {
		if row.UpdatedAt.Before(since) {
			continue
		}
		event, ok := commitMessageNotificationEvent(row, n.repoName(ctx, row.RepositoryID), n.baseURL(ctx))
		if !ok {
			continue
		}
		subject := "staged-message:" + row.ID
		claimed, err := n.claim(ctx, subject, "staged-message")
		if err != nil {
			return published, err
		}
		if !claimed {
			continue
		}
		if err := n.publisher.PublishDomainEvent(ctx, event); err != nil {
			// Retry on a later pass; the event id is stable, so Vrooli Events
			// drops a duplicate if this publish actually landed.
			n.release(ctx, subject)
			log.Printf("warn: commit message notification %s: %v", row.ID, err)
			continue
		}
		published++
	}
	return published, nil
}

func boundNotificationText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= notificationTextLimit {
		return value
	}
	cut := notificationTextLimit
	for cut > 0 && (value[cut]&0xC0) == 0x80 {
		cut--
	}
	return value[:cut] + "…"
}

// startNotifications wires the notifier and its background pass, which
// settles drafting jobs no browser is following and announces finished ones.
func (s *Server) startNotifications() {
	if s.db == nil {
		return
	}
	jobs, err := reviewjobs.New(s.db)
	if err != nil {
		log.Printf("warn: notifications unavailable: %v", err)
		return
	}
	notifier, err := newGCTNotifier(eventbus.NewDiscoveredClient(context.Background()), s.db, jobs)
	if err != nil {
		log.Printf("warn: notifications unavailable: %v", err)
		return
	}
	notifier.repoName = s.notificationRepoName
	s.notifier = notifier
	go func() {
		ticker := time.NewTicker(notificationSweepInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.notificationPass()
		}
	}()
}

func (s *Server) notificationPass() {
	ctx, cancel := context.WithTimeout(context.Background(), notificationSweepInterval)
	defer cancel()
	if s.stagedMessages != nil {
		if _, err := s.stagedMessages.Settle(ctx, s.stagedMessageDirectory); err != nil {
			log.Printf("warn: settle drafting jobs: %v", err)
		}
	}
	if _, err := s.notifier.NotifyFinishedDrafts(ctx); err != nil {
		log.Printf("warn: drafting notifications: %v", err)
	}
}

func (s *Server) stagedMessageDirectory(ctx context.Context, repositoryID string) (string, error) {
	record, err := s.mutationRepo(ctx, repositoryID)
	if err != nil {
		return "", err
	}
	return record.Path, nil
}

func (s *Server) notificationRepoName(ctx context.Context, repositoryID string) string {
	if s.repos == nil {
		return ""
	}
	record, err := s.mutationRepo(ctx, repositoryID)
	if err != nil || record == nil {
		return ""
	}
	return record.Name
}

// notifyProposalCreated publishes off the request path so a slow event
// service never delays the agent that created the proposal.
func (s *Server) notifyProposalCreated(repoName string, view proposals.View) {
	if s.notifier == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.notifier.NotifyProposalCreated(ctx, repoName, view)
	}()
}
