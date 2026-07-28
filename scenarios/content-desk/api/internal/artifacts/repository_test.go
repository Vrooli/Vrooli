package artifacts

import (
	"context"
	"testing"

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

func TestUpdateBodyPersistsAttributedRevisionAndRejectsTerminalDraft(t *testing.T) {
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
}

func TestPublishAtomicallyTransitionsApprovedDraftAndAppendsLedgerRecord(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	draft, err := repo.Create(ctx, Draft{ID: "publish-draft", CampaignID: "campaign-1", PostTypeID: "dev-log", Channel: "x-twitter", Format: "thread"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Publish(ctx, draft.ID, PublishInput{PublishedURL: "https://example.test/p/1", PlatformPostID: "p-1"}); err == nil {
		t.Fatal("non-approved draft published")
	}
	if _, err := repo.(*sqliteRepository).db.ExecContext(ctx, `UPDATE drafts SET status = 'approved' WHERE id = ?`, draft.ID); err != nil {
		t.Fatal(err)
	}
	published, recordID, err := repo.Publish(ctx, draft.ID, PublishInput{Audience: "operators", PublishedURL: "https://example.test/p/1", PlatformPostID: "p-1", SeriesID: "series-1"})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != DraftPublished || recordID == "" {
		t.Fatalf("publish = %#v record=%q", published, recordID)
	}
	var storedDraft, channel string
	if err := repo.(*sqliteRepository).db.QueryRowContext(ctx, `SELECT draft_id, channel FROM ledger_publish_records WHERE id = ?`, recordID).Scan(&storedDraft, &channel); err != nil {
		t.Fatal(err)
	}
	if storedDraft != draft.ID || channel != "x-twitter" {
		t.Fatalf("ledger = %q/%q", storedDraft, channel)
	}
}
