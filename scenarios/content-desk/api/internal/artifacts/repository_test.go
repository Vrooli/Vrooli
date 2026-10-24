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
	if _, err = db.ExecContext(context.Background(), internalreview.Schema()); err != nil {
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

// [REQ:CONTENTD-P1-006] A receipt stored out of band (ledger inbox or
// publish-log importer) converges the draft lifecycle exactly once when the
// Channel Manager outcome is later recorded, reusing the durable record and
// appending no second publication.
func TestRecordReleaseOutcomeReconcilesReceiptStoredOutOfBand(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	draft, err := repo.Create(ctx, Draft{ID: "reconcile-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE drafts SET status = 'approved' WHERE id = ?`, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO ledger_publish_records (id, import_key, draft_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json) VALUES ('out-of-band', 'channel-manager:receipt-ob', ?, 'x-twitter', '', 'https://example.test/ob', 'post-ob', 'channel-manager', '2026-07-29T00:00:00Z', '{}')`, draft.ID); err != nil {
		t.Fatal(err)
	}
	recorded, recordID, err := repo.RecordReleaseOutcome(ctx, ReleaseOutcome{ReceiptID: "receipt-ob", DraftID: draft.ID, Status: "published", PlatformPostID: "post-ob", PublishedURL: "https://example.test/ob", PublishedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Status != DraftPublished || recordID != "out-of-band" {
		t.Fatalf("recorded = %#v id=%q", recorded, recordID)
	}
	var records, events int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_publish_records WHERE import_key = 'channel-manager:receipt-ob'`).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM draft_events WHERE draft_id = ? AND to_status = 'published'`, draft.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if records != 1 || events != 1 {
		t.Fatalf("records=%d published events=%d", records, events)
	}
}

// [REQ:CONTENTD-P1-006] A replayed receipt can never be re-pointed at another
// draft; the durable record's draft identity is authoritative.
func TestRecordReleaseOutcomeRejectsReceiptForAnotherDraft(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	for _, id := range []string{"receipt-draft-a", "receipt-draft-b"} {
		if _, err := repo.Create(ctx, Draft{ID: id, CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE drafts SET status = 'approved' WHERE id = ?`, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := repo.RecordReleaseOutcome(ctx, ReleaseOutcome{ReceiptID: "receipt-mismatch", DraftID: "receipt-draft-a", Status: "published", PlatformPostID: "post-a", PublishedURL: "https://example.test/a", PublishedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.RecordReleaseOutcome(ctx, ReleaseOutcome{ReceiptID: "receipt-mismatch", DraftID: "receipt-draft-b", Status: "published", PlatformPostID: "post-b", PublishedURL: "https://example.test/b", PublishedAt: time.Now().UTC()}); err == nil {
		t.Fatal("receipt replay claimed a different draft")
	}
}

// GetReleaseTarget reads the approved channel target recorded at approval, and
// reports absence distinctly from an ineligible target.
func TestGetReleaseTargetReadsApprovedTargetAndAbsence(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	if _, err := repo.Create(ctx, Draft{ID: "target-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := repo.GetReleaseTarget(ctx, "target-draft"); err != nil || found {
		t.Fatalf("empty lookup = found:%v err:%v", found, err)
	}
	if err := repo.RecordEligibility(ctx, "target-draft", ReleaseTarget{IdentityID: "identity-1", Lane: "main", Eligibility: "eligible", CheckedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	target, found, err := repo.GetReleaseTarget(ctx, "target-draft")
	if err != nil || !found {
		t.Fatalf("recorded lookup = found:%v err:%v", found, err)
	}
	if target.IdentityID != "identity-1" || target.Lane != "main" || target.Eligibility != "eligible" {
		t.Fatalf("target = %#v", target)
	}
}

// [REQ:CONTENTD-P1-006] An out-of-band receipt that does not correspond to a
// publishable draft is refused rather than used to fabricate a publication.
func TestRecordReleaseOutcomeRefusesReceiptWhenDraftNotPublishable(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	draft, err := repo.Create(ctx, Draft{ID: "not-publishable-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO ledger_publish_records (id, import_key, draft_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json) VALUES ('orphan-receipt', 'channel-manager:receipt-orphan', ?, 'x-twitter', '', 'https://example.test/orphan', 'post-orphan', 'channel-manager', '2026-07-29T00:00:00Z', '{}')`, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.RecordReleaseOutcome(ctx, ReleaseOutcome{ReceiptID: "receipt-orphan", DraftID: draft.ID, Status: "published", PlatformPostID: "post-orphan", PublishedURL: "https://example.test/orphan", PublishedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}); err == nil {
		t.Fatal("receipt fabricated a publish for a requested draft")
	}
	stored, err := repo.Get(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != DraftRequested {
		t.Fatalf("status = %s", stored.Status)
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

// [REQ:CONTENTD-P0-015] A body revision is a new artifact revision. It must
// withdraw the operator approval and invalidate the review that described the
// prior body, so a changed artifact cannot be released or re-approved under
// authority that no longer matches it.
func TestUpdateBodyInvalidatesReviewAndApprovalOnRevision(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	for _, schema := range []string{internalclaims.Schema(), internalreview.Schema()} {
		if _, err := db.ExecContext(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE post_types SET status = 'active' WHERE id = ?`, "single-image-ad"); err != nil {
		t.Fatal(err)
	}
	draft, err := repo.Create(ctx, Draft{ID: "revision-authority-draft", CampaignID: "campaign", PostTypeID: "single-image-ad", Body: "original", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftReviewPass} {
		if draft, err = repo.Transition(ctx, draft.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'passed', ?)`, uuid.NewString(), draft.ID, "2026-07-28T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Approve(ctx, draft.ID); err != nil {
		t.Fatalf("approve original body: %v", err)
	}
	if _, err := repo.RevalidateForRelease(ctx, draft.ID); err != nil {
		t.Fatalf("approved original body should be releasable: %v", err)
	}

	revised, err := repo.UpdateBody(ctx, draft.ID, "revised")
	if err != nil {
		t.Fatal(err)
	}
	if revised.Status != DraftBlocked {
		t.Fatalf("revised status = %s, want %s", revised.Status, DraftBlocked)
	}
	var approvals int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM draft_approvals WHERE draft_id = ?`, draft.ID).Scan(&approvals); err != nil {
		t.Fatal(err)
	}
	if approvals != 0 {
		t.Fatalf("stale approval rows = %d", approvals)
	}
	var liveReviews int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_runs WHERE draft_id = ? AND invalidated_at = ''`, draft.ID).Scan(&liveReviews); err != nil {
		t.Fatal(err)
	}
	if liveReviews != 0 {
		t.Fatalf("live review runs after revision = %d", liveReviews)
	}
	if _, err := repo.RevalidateForRelease(ctx, draft.ID); err == nil {
		t.Fatal("revised draft was released under the prior approval")
	}

	// The invalidated review cannot satisfy approval even if the draft is
	// walked back to reviewed without a fresh review.
	for _, event := range []DraftEvent{DraftCheck, DraftReviewPass} {
		if _, err = repo.Transition(ctx, draft.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.Approve(ctx, draft.ID); err == nil {
		t.Fatal("invalidated review satisfied approval")
	}
	// A fresh passed review re-earns approval for the current body.
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'passed', ?)`, uuid.NewString(), draft.ID, "2026-07-29T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	approved, err := repo.Approve(ctx, draft.ID)
	if err != nil {
		t.Fatalf("approve after fresh review: %v", err)
	}
	if approved.Status != DraftApproved {
		t.Fatalf("status = %s", approved.Status)
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

// [REQ:CONTENTD-P0-015] Consumers resolve the current body and only the
// review/approval authority that still applies to it. Invalidated, superseded
// or withdrawn evidence must read as absent rather than as the prior evidence.
func TestGetCurrentRevisionSelectsCurrentBodyAndApplicableAuthority(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	if _, err := db.ExecContext(ctx, internalclaims.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE post_types SET status = 'active' WHERE id = ?`, "single-image-ad"); err != nil {
		t.Fatal(err)
	}
	draft, err := repo.Create(ctx, Draft{ID: "current-revision-draft", CampaignID: "campaign", PostTypeID: "single-image-ad", Body: "original", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetCurrentRevision(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Body != "original" || current.HasStoredRevision() || current.HasApplicableReview() || current.HasApplicableApproval() {
		t.Fatalf("fresh current revision = %#v", current)
	}
	for _, event := range []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftReviewPass} {
		if draft, err = repo.Transition(ctx, draft.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'passed', ?)`, "review-passed", draft.ID, "2026-07-28T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Approve(ctx, draft.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	current, err = repo.GetCurrentRevision(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !current.HasApplicableReview() || current.ReviewRunID != "review-passed" {
		t.Fatalf("applicable review = %#v", current)
	}
	if !current.HasApplicableApproval() || current.ApprovalActorKind != "operator" {
		t.Fatalf("applicable approval = %#v", current)
	}

	revised, err := repo.UpdateBody(ctx, draft.ID, "revised")
	if err != nil {
		t.Fatal(err)
	}
	if revised.Status != DraftBlocked {
		t.Fatalf("revised status = %s", revised.Status)
	}
	current, err = repo.GetCurrentRevision(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Body != "revised" || !current.HasStoredRevision() || current.ActorKind != "operator" {
		t.Fatalf("revised current = %#v", current)
	}
	if current.HasApplicableReview() || current.HasApplicableApproval() {
		t.Fatalf("revised authority was not withdrawn: %#v", current)
	}

	// A superseding blocked review leaves no applicable passed authority.
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'blocked', ?)`, "review-blocked", draft.ID, "2026-07-30T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO review_supersessions (superseded_run_id, superseding_run_id) VALUES (?, ?)`, "review-passed", "review-blocked"); err != nil {
		t.Fatal(err)
	}
	current, err = repo.GetCurrentRevision(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.HasApplicableReview() {
		t.Fatalf("superseded/blocked review reported as applicable: %#v", current)
	}
}

// [REQ:CONTENTD-P0-015] A revision withdraws review authority in every
// non-terminal status, not only reviewed/approved, so a stale passed review
// from a blocked draft can never satisfy a later approval.
func TestUpdateBodyInvalidatesStaleReviewOnBlockedDraft(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	db := repo.(*sqliteRepository).db
	if _, err := db.ExecContext(ctx, internalclaims.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE post_types SET status = 'active' WHERE id = ?`, "single-image-ad"); err != nil {
		t.Fatal(err)
	}
	draft, err := repo.Create(ctx, Draft{ID: "blocked-revision-draft", CampaignID: "campaign", PostTypeID: "single-image-ad", Body: "original", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftReviewPass} {
		if draft, err = repo.Transition(ctx, draft.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO review_runs (id, draft_id, outcome, created_at) VALUES (?, ?, 'passed', ?)`, "stale-passed", draft.ID, "2026-07-28T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if draft, err = repo.Transition(ctx, draft.ID, DraftBlock); err != nil {
		t.Fatal(err)
	}
	if draft.Status != DraftBlocked {
		t.Fatalf("status = %s", draft.Status)
	}
	if _, err := repo.UpdateBody(ctx, draft.ID, "revised while blocked"); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetCurrentRevision(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.HasApplicableReview() {
		t.Fatalf("stale review survived a blocked-draft revision: %#v", current)
	}
	for _, event := range []DraftEvent{DraftCheck, DraftReviewPass} {
		if _, err = repo.Transition(ctx, draft.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.Approve(ctx, draft.ID); err == nil {
		t.Fatal("stale review satisfied approval after a blocked-draft revision")
	}
}
