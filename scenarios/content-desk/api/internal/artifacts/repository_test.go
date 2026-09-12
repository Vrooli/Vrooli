package artifacts

import (
	"context"
	"testing"
	"time"

	internalcampaigns "content-desk/internal/campaigns"
	internalclaims "content-desk/internal/claims"
	internalledger "content-desk/internal/ledger"
	internalposttypes "content-desk/internal/posttypes"
	internalreview "content-desk/internal/review"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func newRepository(t *testing.T) Repository {
	t.Helper()
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:artifacts-test?mode=memory&cache=shared", MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), internalcampaigns.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), internalledger.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), internalposttypes.Schema()); err != nil {
		t.Fatal(err)
	}
	for _, campaignID := range []string{"campaign-1", "campaign", "slot-campaign"} {
		if _, err = db.ExecContext(context.Background(), `INSERT INTO campaign_slots (campaign_id, channel, format, capacity, reserved) VALUES (?, 'x-twitter', 'thread', 2, 0)`, campaignID); err != nil {
			t.Fatal(err)
		}
	}
	return NewSQLiteRepository(db)
}

// [REQ:CONTENTD-P1-006]
func TestRecordReleaseOutcomeIsAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	draft, err := repo.Create(ctx, Draft{ID: "release-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.(*sqliteRepository).db.ExecContext(ctx, `UPDATE drafts SET status = 'approved' WHERE id = ?`, draft.ID); err != nil {
		t.Fatal(err)
	}
	outcome := ReleaseOutcome{ReceiptID: "receipt-1", DraftID: draft.ID, Status: "partial", PlatformPostID: "post-1", PublishedURL: "https://example.test/post-1", PublishedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}
	first, recordID, err := repo.RecordReleaseOutcome(ctx, outcome)
	if err != nil {
		t.Fatal(err)
	}
	second, repeatedID, err := repo.RecordReleaseOutcome(ctx, outcome)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != DraftPublished || second.Status != DraftPublished || recordID == "" || recordID != repeatedID {
		t.Fatalf("outcomes = %#v/%#v record=%q/%q", first, second, recordID, repeatedID)
	}
	var count int
	if err := repo.(*sqliteRepository).db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_publish_records WHERE import_key = 'channel-manager:receipt-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("receipt record count = %d", count)
	}
}

// [REQ:CONTENTD-P1-006]
func TestRevalidateForReleaseBlocksAnApprovedDraftWhenClaimEvidenceChanged(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	if _, err := db.ExecContext(ctx, internalclaims.Schema()); err != nil {
		t.Fatal(err)
	}
	draft, err := repo.Create(ctx, Draft{ID: "stale-at-handoff", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE drafts SET status = 'approved' WHERE id = ?`, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO claims(id, statement, kind, verification_status, created_at) VALUES ('changed-claim', 'changed', 'fact', 'stale', '2026-07-29T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO claim_citations(draft_id, claim_id, span_start, span_end) VALUES (?, 'changed-claim', 0, 1)`, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RevalidateForRelease(ctx, draft.ID); err == nil {
		t.Fatal("stale claim was allowed to release")
	}
	stored, err := repo.Get(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != DraftBlocked {
		t.Fatalf("status = %s", stored.Status)
	}
}

func TestRepositoryPersistsConstrainedLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	draft, err := repo.Create(ctx, Draft{ID: "draft-1", CampaignID: "campaign-1", PostTypeID: "dev-log", Body: "Hello", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != DraftRequested {
		t.Fatalf("status = %s", draft.Status)
	}
	if _, err := repo.Transition(ctx, draft.ID, DraftApprove); err == nil {
		t.Fatal("approve from requested succeeded")
	}
	for _, event := range []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftReviewPass} {
		draft, err = repo.Transition(ctx, draft.ID, event)
		if err != nil {
			t.Fatalf("%s: %v", event, err)
		}
	}
	if draft.Status != DraftReviewed {
		t.Fatalf("status = %s", draft.Status)
	}
	if _, err := repo.Transition(ctx, draft.ID, DraftApprove); err == nil {
		t.Fatal("ungated approval succeeded")
	}
	events, err := repo.Events(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 || events[0].FromStatus != DraftRequested || events[3].ToStatus != DraftReviewed {
		t.Fatalf("events = %#v", events)
	}
}

func TestCreateRejectsUnregisteredPostType(t *testing.T) {
	repo := newRepository(t)
	_, err := repo.Create(context.Background(), Draft{ID: "unknown-type", CampaignID: "campaign-1", PostTypeID: "not-in-canon", Channel: "x-twitter", Format: "thread"})
	if err == nil || err.Error() != `post type "not-in-canon" is not registered` {
		t.Fatalf("create error = %v", err)
	}
}

func TestApprovalPersistsOnlyAfterEveryStoredGatePasses(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	for _, schema := range []string{internalclaims.Schema(), internalreview.Schema()} {
		if _, err := db.ExecContext(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := repo.Create(ctx, Draft{ID: "approval-draft", CampaignID: "campaign", PostTypeID: "single-image-ad", Body: "body", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftReviewPass} {
		draft, err = repo.Transition(ctx, draft.ID, event)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.Approve(ctx, draft.ID); err == nil {
		t.Fatal("approval without post type and review succeeded")
	}
	if _, err := db.ExecContext(ctx, `UPDATE post_types SET status = 'active' WHERE id = ?`, "single-image-ad"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'passed', ?)`, uuid.NewString(), draft.ID, "2026-07-28T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	approved, err := repo.Approve(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != DraftApproved {
		t.Fatalf("status = %s", approved.Status)
	}
	var actor string
	if err := db.QueryRowContext(ctx, `SELECT actor_kind FROM draft_approvals WHERE draft_id = ?`, draft.ID).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if actor != "operator" {
		t.Fatalf("actor = %q", actor)
	}
}

func TestCreateReservesCampaignSlotAndAbandonReleasesItOnce(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	create := func(id string) (Draft, error) {
		return repo.Create(ctx, Draft{ID: id, CampaignID: "slot-campaign", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	}
	first, err := create("slot-draft-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = create("slot-draft-2"); err != nil {
		t.Fatal(err)
	}
	if _, err = create("slot-draft-3"); err == nil {
		t.Fatal("draft exceeded campaign slot capacity")
	}
	if _, err = repo.Transition(ctx, first.ID, DraftAbandon); err != nil {
		t.Fatal(err)
	}
	if _, err = create("slot-draft-3"); err != nil {
		t.Fatalf("released slot did not admit replacement: %v", err)
	}
	if _, err = repo.Transition(ctx, first.ID, DraftAbandon); err == nil {
		t.Fatal("terminal abandoned draft transitioned twice")
	}
	var reserved int
	if err := repo.(*sqliteRepository).db.QueryRowContext(ctx, `SELECT reserved FROM campaign_slots WHERE campaign_id = 'slot-campaign'`).Scan(&reserved); err != nil {
		t.Fatal(err)
	}
	if reserved != 2 {
		t.Fatalf("reserved = %d, want 2", reserved)
	}
}

// [REQ:CONTENTD-P0-015]
func TestUpdateBodyPersistsAttributedRevisionAndRejectsTerminalDraft(t *testing.T) {
	t.Run("[CONTENTD-P0-015] revision persists attributed authoring", func(t *testing.T) {
		ctx := context.Background()
		repo := newRepository(t)
		draft, err := repo.Create(ctx, Draft{ID: "revision-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Body: "before", Channel: "x-twitter", Format: "thread"})
		if err != nil {
			t.Fatal(err)
		}
		updated, err := repo.UpdateBody(ctx, draft.ID, "after")
		if err != nil {
			t.Fatal(err)
		}
		if updated.Body != "after" {
			t.Fatalf("body = %q", updated.Body)
		}
		var body, actor string
		if err := repo.(*sqliteRepository).db.QueryRowContext(ctx, `SELECT body, actor_kind FROM draft_revisions WHERE draft_id = ?`, draft.ID).Scan(&body, &actor); err != nil {
			t.Fatal(err)
		}
		if body != "after" || actor != "operator" {
			t.Fatalf("revision = %q/%q", body, actor)
		}
		if _, err := repo.Transition(ctx, draft.ID, DraftAbandon); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.UpdateBody(ctx, draft.ID, "never"); err == nil {
			t.Fatal("terminal draft was revised")
		}
	})
}

// [REQ:CONTENTD-P1-009] Attachments retain metadata references only. The
// Content Desk never receives or stores an image byte payload.
func TestAttachmentRoundTripsReleasedAssetMetadataWithoutBytes(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	draft, err := repo.Create(ctx, Draft{ID: "attachment-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	attachment, err := repo.Attach(ctx, Attachment{DraftID: draft.ID, AssetID: "asset-released-1", Role: "hero", AspectRatio: "16:9", AltText: "A descriptive image", Position: 0})
	if err != nil {
		t.Fatal(err)
	}
	if attachment.AssetID != "asset-released-1" {
		t.Fatalf("asset id = %q", attachment.AssetID)
	}
	attachments, err := repo.ListAttachments(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 || attachments[0].AltText != "A descriptive image" {
		t.Fatalf("attachments = %#v", attachments)
	}
	if _, err = repo.Attach(ctx, Attachment{DraftID: draft.ID, AssetID: "asset-released-2", Role: "inline", AspectRatio: "1:1", AltText: "", Position: 1}); err == nil {
		t.Fatal("empty alt text was accepted")
	}
}

// [REQ:CONTENTD-P1-011] Agent work is an attributable, output-only editorial
// commission. The ledger records no transcript, credentials, approval, or
// publish authority.
func TestAgentCommissionPersistsOnlyDurableProvenance(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	draft, err := repo.Create(ctx, Draft{ID: "agent-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Body: "Draft", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	commission, err := repo.RecordAgentCommission(ctx, AgentCommission{DraftID: draft.ID, Action: "evidence-hunt", TaskID: "task-1", RunID: "run-1", Status: "RUN_STATUS_QUEUED"})
	if err != nil {
		t.Fatal(err)
	}
	if commission.ID == "" || commission.RunID != "run-1" {
		t.Fatalf("commission = %#v", commission)
	}
	if _, err = repo.RecordAgentCommission(ctx, AgentCommission{DraftID: draft.ID, Action: "publish", TaskID: "task-2", RunID: "run-2", Status: "RUN_STATUS_QUEUED"}); err == nil {
		t.Fatal("unsupported agent action accepted")
	}
}
