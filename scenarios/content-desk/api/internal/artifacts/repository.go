package artifacts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/provenance"
)

type Draft struct {
	ID, CampaignID, PostTypeID, Body, Channel, Format, Lane, SKU string
	Status                                                       DraftStatus
}

type EventRecord struct {
	ID, DraftID          string
	Event                DraftEvent
	FromStatus, ToStatus DraftStatus
	OccurredAt           time.Time
}

// ReleaseOutcome is the credential-free completion evidence received from
// Channel Manager. Its ReceiptID is the durable idempotency boundary.
type ReleaseOutcome struct {
	ReceiptID, DraftID, Status, PlatformPostID, PublishedURL string
	PublishedAt                                              time.Time
}

type ReleaseTarget struct {
	IdentityID, Lane, Eligibility string
	CheckedAt                     time.Time
}

// Attachment is a non-byte reference to one released Asset Studio asset.
type Attachment struct {
	ID, DraftID, AssetID, Role, AspectRatio, AltText string
	Position                                         int
	CreatedAt                                        time.Time
}
type AgentCommission struct {
	ID, DraftID, Action, TaskID, RunID, Status string
	CreatedAt                                  time.Time
}

type ApprovalError struct{ Blockers []string }

func (e *ApprovalError) Error() string { return fmt.Sprintf("draft approval blocked: %v", e.Blockers) }

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	Create(context.Context, Draft) (Draft, error)
	List(context.Context) ([]Draft, error)
	Get(context.Context, string) (Draft, error)
	Transition(context.Context, string, DraftEvent) (Draft, error)
	Approve(context.Context, string) (Draft, error)
	UpdateBody(context.Context, string, string) (Draft, error)
	RecordEligibility(context.Context, string, ReleaseTarget) error
	RevalidateForRelease(context.Context, string) (Draft, error)
	RecordReleaseOutcome(context.Context, ReleaseOutcome) (Draft, string, error)
	Attach(context.Context, Attachment) (Attachment, error)
	ListAttachments(context.Context, string) ([]Attachment, error)
	RecordAgentCommission(context.Context, AgentCommission) (AgentCommission, error)
	GetAgentCommission(context.Context, string) (AgentCommission, error)
	RecordAgentAdoption(context.Context, string, string, string) error
	Events(context.Context, string) ([]EventRecord, error)
}

func (r *sqliteRepository) GetAgentCommission(ctx context.Context, id string) (AgentCommission, error) {
	var commission AgentCommission
	var created string
	err := r.db.QueryRowContext(ctx, `SELECT id, draft_id, action, task_id, run_id, status, created_at FROM draft_agent_commissions WHERE id = ?`, id).Scan(&commission.ID, &commission.DraftID, &commission.Action, &commission.TaskID, &commission.RunID, &commission.Status, &created)
	if err != nil {
		return AgentCommission{}, err
	}
	commission.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return commission, err
}
func (r *sqliteRepository) RecordAgentAdoption(ctx context.Context, commissionID, draftID, runID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO draft_agent_adoptions (commission_id, draft_id, run_id, adopted_at) VALUES (?, ?, ?, ?)`, commissionID, draftID, runID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (r *sqliteRepository) RecordAgentCommission(ctx context.Context, commission AgentCommission) (AgentCommission, error) {
	if commission.DraftID == "" || commission.Action == "" || commission.TaskID == "" || commission.RunID == "" || commission.Status == "" {
		return AgentCommission{}, fmt.Errorf("agent commission requires draft, action, task, run, and status")
	}
	if commission.Action != "draft" && commission.Action != "evidence-hunt" && commission.Action != "review" {
		return AgentCommission{}, fmt.Errorf("unsupported agent action %q", commission.Action)
	}
	if commission.ID == "" {
		commission.ID = uuid.NewString()
	}
	if commission.CreatedAt.IsZero() {
		commission.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO draft_agent_commissions (id, draft_id, action, task_id, run_id, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, commission.ID, commission.DraftID, commission.Action, commission.TaskID, commission.RunID, commission.Status, commission.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return AgentCommission{}, err
	}
	return commission, nil
}

func (r *sqliteRepository) Attach(ctx context.Context, attachment Attachment) (Attachment, error) {
	if attachment.DraftID == "" || attachment.AssetID == "" || attachment.Role == "" || attachment.AspectRatio == "" || attachment.AltText == "" || attachment.Position < 0 {
		return Attachment{}, fmt.Errorf("attachment requires draft, asset, role, aspect ratio, alt text, and non-negative position")
	}
	if attachment.ID == "" {
		attachment.ID = uuid.NewString()
	}
	if attachment.CreatedAt.IsZero() {
		attachment.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO draft_attachments (id, draft_id, asset_id, role, aspect_ratio, alt_text, position, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, attachment.ID, attachment.DraftID, attachment.AssetID, attachment.Role, attachment.AspectRatio, attachment.AltText, attachment.Position, attachment.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Attachment{}, err
	}
	return attachment, nil
}

func (r *sqliteRepository) ListAttachments(ctx context.Context, draftID string) ([]Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, draft_id, asset_id, role, aspect_ratio, alt_text, position, created_at FROM draft_attachments WHERE draft_id = ? ORDER BY position, id`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attachments []Attachment
	for rows.Next() {
		var attachment Attachment
		var createdAt string
		if err := rows.Scan(&attachment.ID, &attachment.DraftID, &attachment.AssetID, &attachment.Role, &attachment.AspectRatio, &attachment.AltText, &attachment.Position, &createdAt); err != nil {
			return nil, err
		}
		if attachment.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, rows.Err()
}

func (r *sqliteRepository) RecordEligibility(ctx context.Context, draftID string, target ReleaseTarget) error {
	if draftID == "" || target.IdentityID == "" || target.Lane == "" || target.Eligibility == "" {
		return fmt.Errorf("eligibility record requires draft, identity, lane, and result")
	}
	if target.CheckedAt.IsZero() {
		target.CheckedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO draft_release_targets(draft_id, identity_id, lane, eligibility, checked_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(draft_id) DO UPDATE SET identity_id = excluded.identity_id, lane = excluded.lane, eligibility = excluded.eligibility, checked_at = excluded.checked_at`, draftID, target.IdentityID, target.Lane, target.Eligibility, target.CheckedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// RecordReleaseOutcome atomically accepts a completed Channel Manager receipt,
// advances the approved draft, and appends the immutable publication record.
// Replays return the original record without a second lifecycle transition.
func (r *sqliteRepository) RecordReleaseOutcome(ctx context.Context, outcome ReleaseOutcome) (Draft, string, error) {
	if outcome.ReceiptID == "" || outcome.DraftID == "" || outcome.PlatformPostID == "" || outcome.PublishedURL == "" {
		return Draft{}, "", fmt.Errorf("release outcome requires receipt, draft, post id, and URL")
	}
	if outcome.Status != "published" && outcome.Status != "partial" {
		return Draft{}, "", fmt.Errorf("release outcome status %q is not publishable", outcome.Status)
	}
	if outcome.PublishedAt.IsZero() {
		outcome.PublishedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, "", err
	}
	defer func() { _ = tx.Rollback() }()
	var existingID string
	lookupErr := tx.QueryRowContext(ctx, `SELECT id FROM ledger_publish_records WHERE import_key = ?`, "channel-manager:"+outcome.ReceiptID).Scan(&existingID)
	if lookupErr == nil {
		draft, err := scanDraft(ctx, tx, outcome.DraftID)
		if err != nil {
			return Draft{}, "", err
		}
		return draft, existingID, tx.Commit()
	}
	if lookupErr != sql.ErrNoRows {
		return Draft{}, "", lookupErr
	}
	draft, err := scanDraft(ctx, tx, outcome.DraftID)
	if err != nil {
		return Draft{}, "", err
	}
	next, err := TransitionDraft(DraftState{Status: draft.Status}, DraftPublish)
	if err != nil {
		return Draft{}, "", err
	}
	now := outcome.PublishedAt.UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `UPDATE drafts SET status = ?, updated_at = ? WHERE id = ? AND status = ?`, next.Status, now, draft.ID, draft.Status); err != nil {
		return Draft{}, "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO draft_events (id, draft_id, event, from_status, to_status, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), draft.ID, DraftPublish, draft.Status, next.Status, now); err != nil {
		return Draft{}, "", err
	}
	recordID := uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO ledger_publish_records (id, import_key, draft_id, series_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json) VALUES (?, ?, ?, NULL, ?, '', ?, ?, 'channel-manager', ?, ?)`, recordID, "channel-manager:"+outcome.ReceiptID, draft.ID, draft.Channel, outcome.PublishedURL, outcome.PlatformPostID, now, `{"status":"`+outcome.Status+`"}`); err != nil {
		return Draft{}, "", err
	}
	if err = tx.Commit(); err != nil {
		return Draft{}, "", err
	}
	draft.Status = next.Status
	return draft, recordID, nil
}

func scanDraft(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (Draft, error) {
	var draft Draft
	err := queryer.QueryRowContext(ctx, `SELECT d.id, d.campaign_id, d.post_type_id, d.body, d.status, d.lane, d.sku, COALESCE(s.channel,''), COALESCE(s.format,'') FROM drafts d LEFT JOIN draft_slots s ON s.draft_id = d.id WHERE d.id = ?`, id).Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status, &draft.Lane, &draft.SKU, &draft.Channel, &draft.Format)
	return draft, err
}

// RevalidateForRelease reruns the moving claim gate immediately before the
// outbound handoff. If evidence changed since approval, the draft becomes
// blocked and no Channel Manager action may be submitted.
func (r *sqliteRepository) RevalidateForRelease(ctx context.Context, id string) (Draft, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	draft, err := scanDraft(ctx, tx, id)
	if err != nil {
		return Draft{}, err
	}
	if draft.Status != DraftApproved {
		return Draft{}, fmt.Errorf("draft %q is %s; only approved drafts may be released", draft.ID, draft.Status)
	}
	var unverified int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM claim_citations cc JOIN claims c ON c.id = cc.claim_id WHERE cc.draft_id = ? AND c.verification_status <> 'verified'`, draft.ID).Scan(&unverified); err != nil {
		return Draft{}, err
	}
	if unverified == 0 {
		return draft, tx.Commit()
	}
	next, err := TransitionDraft(DraftState{Status: draft.Status}, DraftBlock)
	if err != nil {
		return Draft{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `UPDATE drafts SET status = ?, updated_at = ? WHERE id = ? AND status = ?`, next.Status, now, draft.ID, draft.Status); err != nil {
		return Draft{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO draft_events (id, draft_id, event, from_status, to_status, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), draft.ID, DraftBlock, draft.Status, next.Status, now); err != nil {
		return Draft{}, err
	}
	if err = tx.Commit(); err != nil {
		return Draft{}, err
	}
	draft.Status = next.Status
	return Draft{}, fmt.Errorf("draft %q was blocked because cited evidence is no longer verified", draft.ID)
}

// UpdateBody creates an immutable revision owned by the current provenance
// actor. Published and abandoned drafts are terminal editorial records.
func (r *sqliteRepository) UpdateBody(ctx context.Context, id, body string) (Draft, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var draft Draft
	if err := tx.QueryRowContext(ctx, `SELECT d.id, d.campaign_id, d.post_type_id, d.body, d.status, d.lane, d.sku, COALESCE(s.channel,''), COALESCE(s.format,'') FROM drafts d LEFT JOIN draft_slots s ON s.draft_id = d.id WHERE d.id = ?`, id).Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status, &draft.Lane, &draft.SKU, &draft.Channel, &draft.Format); err != nil {
		return Draft{}, err
	}
	if draft.Status == DraftPublished || draft.Status == DraftAbandoned {
		return Draft{}, fmt.Errorf("terminal draft %q cannot be revised", id)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE drafts SET body = ?, updated_at = ? WHERE id = ?`, body, now, id); err != nil {
		return Draft{}, err
	}
	actor := provenance.FromContext(ctx)
	if _, err := tx.ExecContext(ctx, `INSERT INTO draft_revisions (id, draft_id, body, actor_kind, capacity, created_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), id, body, actor.Actor, "author", now); err != nil {
		return Draft{}, err
	}
	if err := tx.Commit(); err != nil {
		return Draft{}, err
	}
	draft.Body = body
	return draft, nil
}

func (r *sqliteRepository) Approve(ctx context.Context, id string) (Draft, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var draft Draft
	if err := tx.QueryRowContext(ctx, `SELECT id, campaign_id, post_type_id, body, status FROM drafts WHERE id = ?`, id).Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status); err != nil {
		return Draft{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT c.id FROM claim_citations cc JOIN claims c ON c.id = cc.claim_id WHERE cc.draft_id = ? AND c.verification_status <> 'verified' ORDER BY c.id`, id)
	if err != nil {
		return Draft{}, err
	}
	var unverified []string
	for rows.Next() {
		var claimID string
		if err := rows.Scan(&claimID); err != nil {
			rows.Close()
			return Draft{}, err
		}
		unverified = append(unverified, claimID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Draft{}, err
	}
	rows.Close()
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_types WHERE id = ? AND status = 'active'`, draft.PostTypeID).Scan(&active); err != nil {
		return Draft{}, err
	}
	var passed int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_runs WHERE draft_id = ? AND outcome = 'passed'`, id).Scan(&passed); err != nil {
		return Draft{}, err
	}
	verdict := EvaluateApproval(ctx, ApprovalInput{DraftStatus: draft.Status, UnverifiedClaimIDs: unverified, PostTypeActive: active > 0, ReviewPassed: passed > 0})
	if !verdict.Allowed {
		return Draft{}, &ApprovalError{Blockers: verdict.Blockers}
	}
	next, err := TransitionDraft(DraftState{Status: draft.Status}, DraftApprove)
	if err != nil {
		return Draft{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE drafts SET status = ?, updated_at = ? WHERE id = ?`, next.Status, now, id); err != nil {
		return Draft{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO draft_events (id, draft_id, event, from_status, to_status, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), id, DraftApprove, draft.Status, next.Status, now); err != nil {
		return Draft{}, err
	}
	actor := provenance.FromContext(ctx)
	if _, err := tx.ExecContext(ctx, `INSERT INTO draft_approvals (draft_id, actor_kind, capacity, approved_at) VALUES (?, ?, ?, ?)`, id, actor.Actor, "operator", now); err != nil {
		return Draft{}, err
	}
	if err := tx.Commit(); err != nil {
		return Draft{}, err
	}
	draft.Status = next.Status
	return draft, nil
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, draft Draft) (Draft, error) {
	if draft.ID == "" {
		draft.ID = uuid.NewString()
	}
	if draft.Status == "" {
		draft.Status = DraftRequested
	}
	if draft.Status != DraftRequested {
		return Draft{}, fmt.Errorf("new draft must start requested")
	}
	if draft.Channel == "" || draft.Format == "" {
		return Draft{}, fmt.Errorf("new draft requires channel and format")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var postTypeExists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_types WHERE id = ?`, draft.PostTypeID).Scan(&postTypeExists); err != nil {
		return Draft{}, fmt.Errorf("validate post type: %w", err)
	}
	if postTypeExists != 1 {
		return Draft{}, fmt.Errorf("post type %q is not registered", draft.PostTypeID)
	}
	reserved, err := tx.ExecContext(ctx, `UPDATE campaign_slots SET reserved = reserved + 1 WHERE campaign_id = ? AND channel = ? AND format = ? AND reserved < capacity`, draft.CampaignID, draft.Channel, draft.Format)
	if err != nil {
		return Draft{}, fmt.Errorf("reserve campaign slot: %w", err)
	}
	count, err := reserved.RowsAffected()
	if err != nil {
		return Draft{}, err
	}
	if count != 1 {
		return Draft{}, fmt.Errorf("campaign artifact slot is unavailable")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `INSERT INTO drafts (id, campaign_id, post_type_id, lane, sku, body, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, draft.ID, draft.CampaignID, draft.PostTypeID, draft.Lane, draft.SKU, draft.Body, draft.Status, now, now)
	if err != nil {
		return Draft{}, fmt.Errorf("create draft: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO draft_slots (draft_id, campaign_id, channel, format) VALUES (?, ?, ?, ?)`, draft.ID, draft.CampaignID, draft.Channel, draft.Format); err != nil {
		return Draft{}, fmt.Errorf("record draft slot: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Draft{}, err
	}
	return draft, nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]Draft, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT d.id, d.campaign_id, d.post_type_id, d.body, d.status, d.lane, d.sku, COALESCE(s.channel,''), COALESCE(s.format,'') FROM drafts d LEFT JOIN draft_slots s ON s.draft_id = d.id ORDER BY d.updated_at DESC, d.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drafts []Draft
	for rows.Next() {
		var draft Draft
		if err := rows.Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status, &draft.Lane, &draft.SKU, &draft.Channel, &draft.Format); err != nil {
			return nil, err
		}
		drafts = append(drafts, draft)
	}
	return drafts, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Draft, error) {
	var draft Draft
	err := r.db.QueryRowContext(ctx, `SELECT d.id, d.campaign_id, d.post_type_id, d.body, d.status, d.lane, d.sku, COALESCE(s.channel,''), COALESCE(s.format,'') FROM drafts d LEFT JOIN draft_slots s ON s.draft_id = d.id WHERE d.id = ?`, id).Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status, &draft.Lane, &draft.SKU, &draft.Channel, &draft.Format)
	return draft, err
}

func (r *sqliteRepository) Transition(ctx context.Context, id string, event DraftEvent) (Draft, error) {
	if event == DraftApprove {
		return Draft{}, fmt.Errorf("approve requires the approval gate")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var draft Draft
	if err := tx.QueryRowContext(ctx, `SELECT d.id, d.campaign_id, d.post_type_id, d.body, d.status, COALESCE(s.channel,''), COALESCE(s.format,'') FROM drafts d LEFT JOIN draft_slots s ON s.draft_id = d.id WHERE d.id = ?`, id).Scan(&draft.ID, &draft.CampaignID, &draft.PostTypeID, &draft.Body, &draft.Status, &draft.Channel, &draft.Format); err != nil {
		return Draft{}, err
	}
	next, err := TransitionDraft(DraftState{Status: draft.Status}, event)
	if err != nil {
		return Draft{}, err
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE drafts SET status = ?, updated_at = ? WHERE id = ? AND status = ?`, next.Status, now.Format(time.RFC3339Nano), id, draft.Status)
	if err != nil {
		return Draft{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return Draft{}, fmt.Errorf("transition draft %q conflicted", id)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO draft_events (id, draft_id, event, from_status, to_status, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), id, event, draft.Status, next.Status, now.Format(time.RFC3339Nano)); err != nil {
		return Draft{}, err
	}
	if event == DraftAbandon {
		result, err := tx.ExecContext(ctx, `UPDATE draft_slots SET released_at = ? WHERE draft_id = ? AND released_at IS NULL`, now.Format(time.RFC3339Nano), id)
		if err != nil {
			return Draft{}, err
		}
		if released, err := result.RowsAffected(); err != nil {
			return Draft{}, err
		} else if released == 1 {
			if _, err := tx.ExecContext(ctx, `UPDATE campaign_slots SET reserved = reserved - 1 WHERE campaign_id = ? AND channel = ? AND format = ? AND reserved > 0`, draft.CampaignID, draft.Channel, draft.Format); err != nil {
				return Draft{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Draft{}, err
	}
	draft.Status = next.Status
	return draft, nil
}

func (r *sqliteRepository) Events(ctx context.Context, draftID string) ([]EventRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, draft_id, event, from_status, to_status, occurred_at FROM draft_events WHERE draft_id = ? ORDER BY occurred_at, id`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []EventRecord
	for rows.Next() {
		var event EventRecord
		var occurred string
		if err := rows.Scan(&event.ID, &event.DraftID, &event.Event, &event.FromStatus, &event.ToStatus, &occurred); err != nil {
			return nil, err
		}
		if event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
