package database

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func authorizationGrantFixture(now time.Time) *domain.AuthorizationGrant {
	return &domain.AuthorizationGrant{
		ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"},
		DurationSeconds: 57600, HarnessKind: "codex", HarnessSession: "fixture",
		WebConsoleSessionID: "verified-web-console-session",
		CreatedAt:           now.UTC(), RequestExpiresAt: now.Add(24 * time.Hour).UTC(),
	}
}

func authorizationNotificationSnapshot(g *domain.AuthorizationGrant) *domain.AuthorizationNotificationSnapshot {
	return &domain.AuthorizationNotificationSnapshot{
		Source: "agent-manager", EventType: "agent_manager.authorization_requested.notification.v1",
		EventID: "stable-event-id", Occurred: g.CreatedAt,
		Payload: map[string]any{"body": "16-hour permission starts only after approval", "context_url": "https://agent-manager.example.test/api/v1/authorizations/" + g.ID.String() + "/approve"},
	}
}

func TestAuthorizationRepositoryNotificationCodecIsDurableAndRedacted(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewAuthorizationRepository(db)
	ctx := context.Background()
	g := authorizationGrantFixture(time.Now())
	g.NotificationSnapshot = authorizationNotificationSnapshot(g)
	published := g.CreatedAt.Add(time.Minute)
	g.NotificationPublishedAt = &published
	encoded, err := encodeAuthorization(g)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeAuthorization(string(encoded))
	if err != nil || decoded.NotificationSnapshot == nil || decoded.NotificationSnapshot.EventID != g.NotificationSnapshot.EventID || decoded.NotificationPublishedAt == nil || !decoded.NotificationPublishedAt.Equal(published) || decoded.WebConsoleSessionID != g.WebConsoleSessionID {
		t.Fatalf("production wrapper codec lost notification metadata: %+v err=%v", decoded, err)
	}
	if err := repo.Create(ctx, g); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Get(ctx, g.ID)
	if err != nil || stored.NotificationSnapshot == nil || stored.NotificationPublishedAt == nil || stored.WebConsoleSessionID != g.WebConsoleSessionID {
		t.Fatalf("repository round trip lost notification metadata: %+v err=%v", stored, err)
	}
	publicJSON, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"notification_snapshot", "notification_published_at", "stable-event-id", "context_url"} {
		if strings.Contains(string(publicJSON), private) {
			t.Fatalf("public grant JSON leaked private notification metadata %q: %s", private, publicJSON)
		}
	}
}

func TestAuthorizationRepositoryNotificationRetryScanPageIsBoundedAndKeysetOrdered(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := &authorizationRepository{db: db}
	ctx := context.Background()
	now := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	ids := []string{
		"00000000-0000-4000-8000-000000000001",
		"00000000-0000-4000-8000-000000000002",
		"00000000-0000-4000-8000-000000000003",
		"00000000-0000-4000-8000-000000000004",
	}
	for i, rawID := range ids {
		id, err := uuid.Parse(rawID)
		if err != nil {
			t.Fatal(err)
		}
		grant := authorizationGrantFixture(now.Add(time.Duration(i) * time.Minute))
		grant.ID = id
		switch i {
		case 1:
			published := now
			grant.NotificationPublishedAt = &published
		case 3:
			grant.State = domain.AuthorizationApproved
		}
		if err := repo.Create(ctx, grant); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListNotificationRetryScanPage(ctx, "", 1)
	if err != nil || len(first) != 1 || first[0].ID.String() != ids[0] {
		t.Fatalf("first bounded page = %#v, err=%v; want %s", first, err, ids[0])
	}
	second, err := repo.ListNotificationRetryScanPage(ctx, first[0].ID.String(), 2)
	if err != nil || len(second) != 2 || second[0].ID.String() != ids[1] || second[1].ID.String() != ids[2] {
		t.Fatalf("cursor page = %#v, err=%v; want bounded rows %s and %s regardless of state", second, err, ids[1], ids[2])
	}
}

func TestAuthorizationReplacePreservesReceiptInterleavingAndCASRejectsLateReceipt(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewAuthorizationRepository(db)
	ctx := context.Background()
	grant := authorizationGrantFixture(time.Now())
	if err := repo.Create(ctx, grant); err != nil {
		t.Fatal(err)
	}
	staleApproval, _ := repo.Get(ctx, grant.ID)
	beforeSnapshot, _ := repo.Get(ctx, grant.ID)
	withSnapshot := *beforeSnapshot
	withSnapshot.NotificationSnapshot = authorizationNotificationSnapshot(grant)
	if changed, err := repo.UpdateNotification(ctx, beforeSnapshot, &withSnapshot); err != nil || !changed {
		t.Fatalf("store pending snapshot: changed=%v err=%v", changed, err)
	}
	// A decision prepared from the stale pending read must merge the receipt
	// fields and retry its CAS, rather than fail due to harmless notification
	// metadata arriving between read and transition.
	staleApproval.State = domain.AuthorizationApproving
	staleApproval.ApprovedAt = &grant.CreatedAt
	if changed, err := repo.Replace(ctx, staleApproval, domain.AuthorizationPending); err != nil || !changed {
		t.Fatalf("approval transition lost to notification-only update: changed=%v err=%v", changed, err)
	}
	approved, err := repo.Get(ctx, grant.ID)
	if err != nil || approved.State != domain.AuthorizationApproving || approved.NotificationSnapshot == nil {
		t.Fatalf("approval state or snapshot lost: %+v err=%v", approved, err)
	}
	lateReceipt := withSnapshot
	published := time.Now().UTC()
	lateReceipt.NotificationPublishedAt = &published
	if changed, err := repo.UpdateNotification(ctx, &withSnapshot, &lateReceipt); err != nil || changed {
		t.Fatalf("late receipt must not overwrite approval: changed=%v err=%v", changed, err)
	}
	stillApproved, _ := repo.Get(ctx, grant.ID)
	if stillApproved.State != domain.AuthorizationApproving || stillApproved.NotificationPublishedAt != nil {
		t.Fatalf("late receipt overwrote decision state: %+v", stillApproved)
	}
}

func TestAuthorizationSupersessionPreservesSnapshotAndRejectsLateReceipt(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewAuthorizationRepository(db)
	ctx := context.Background()
	grant := authorizationGrantFixture(time.Now())
	if err := repo.Create(ctx, grant); err != nil {
		t.Fatal(err)
	}
	current, _ := repo.Get(ctx, grant.ID)
	withSnapshot := *current
	withSnapshot.NotificationSnapshot = authorizationNotificationSnapshot(grant)
	if changed, err := repo.UpdateNotification(ctx, current, &withSnapshot); err != nil || !changed {
		t.Fatalf("store pending snapshot: changed=%v err=%v", changed, err)
	}
	newID := uuid.New()
	staleReplacement, _ := repo.Get(ctx, grant.ID)
	staleReplacement.State = domain.AuthorizationSuperseded
	staleReplacement.SupersededBy = &newID
	if changed, err := repo.Replace(ctx, staleReplacement, domain.AuthorizationPending); err != nil || !changed {
		t.Fatalf("supersede pending request: changed=%v err=%v", changed, err)
	}
	lateReceipt := withSnapshot
	published := time.Now().UTC()
	lateReceipt.NotificationPublishedAt = &published
	if changed, err := repo.UpdateNotification(ctx, &withSnapshot, &lateReceipt); err != nil || changed {
		t.Fatalf("late receipt must not overwrite supersession: changed=%v err=%v", changed, err)
	}
	stored, _ := repo.Get(ctx, grant.ID)
	if stored.State != domain.AuthorizationSuperseded || stored.NotificationSnapshot == nil || stored.NotificationPublishedAt != nil {
		t.Fatalf("supersession/snapshot state incorrect: %+v", stored)
	}
}

func TestAuthorizationNotificationUpdateCannotChangeGrantTerms(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewAuthorizationRepository(db)
	ctx := context.Background()
	grant := authorizationGrantFixture(time.Now())
	if err := repo.Create(ctx, grant); err != nil {
		t.Fatal(err)
	}
	expected, _ := repo.Get(ctx, grant.ID)
	next := *expected
	next.Policy = "different-policy"
	next.NotificationSnapshot = authorizationNotificationSnapshot(grant)
	if changed, err := repo.UpdateNotification(ctx, expected, &next); err == nil || changed {
		t.Fatalf("notification update changed consent terms: changed=%v err=%v", changed, err)
	}
	stored, _ := repo.Get(ctx, grant.ID)
	if stored.Policy != grant.Policy || stored.NotificationSnapshot != nil {
		t.Fatalf("rejected notification update changed grant: %+v", stored)
	}
}

func TestAuthorizationRepositoryKeepsRequesterContextPrivateAndGenerationStable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewAuthorizationRepository(db)
	ctx := context.Background()
	grant := authorizationGrantFixture(time.Now())
	grant.RequesterContext = &domain.AuthorizationRequesterContext{BigPicture: "Explain the purpose", TechnicalDetails: "Requester context\nwith readable line breaks"}
	if err := repo.Create(ctx, grant); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Get(ctx, grant.ID)
	if err != nil || stored.RequesterContext == nil || stored.RequesterContext.BigPicture != grant.RequesterContext.BigPicture || stored.RequesterContext.TechnicalDetails != grant.RequesterContext.TechnicalDetails {
		t.Fatalf("private context did not survive storage: grant=%+v err=%v", stored, err)
	}
	publicJSON, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"Explain the purpose", "Requester context", "requester_context", "requester_big_picture", "requester_technical_details"} {
		if strings.Contains(string(publicJSON), private) {
			t.Fatalf("public grant leaked requester context %q: %s", private, publicJSON)
		}
	}

	before := *stored
	withSnapshot := *stored
	withSnapshot.NotificationSnapshot = authorizationNotificationSnapshot(stored)
	if changed, err := repo.UpdateNotification(ctx, &before, &withSnapshot); err != nil || !changed {
		t.Fatalf("store snapshot with unchanged context: changed=%v err=%v", changed, err)
	}
	changedContext := withSnapshot
	changedContext.RequesterContext = &domain.AuthorizationRequesterContext{BigPicture: "rewritten after request"}
	if changed, err := repo.UpdateNotification(ctx, &withSnapshot, &changedContext); err == nil || changed {
		t.Fatalf("notification receipt rewrote request context: changed=%v err=%v", changed, err)
	}
	transition := *stored
	transition.State = domain.AuthorizationSuperseded
	if changed, err := repo.Replace(ctx, &transition, domain.AuthorizationPending); err != nil || !changed {
		t.Fatalf("transition pending request: changed=%v err=%v", changed, err)
	}
	after, err := repo.Get(ctx, grant.ID)
	if err != nil || after.RequesterContext == nil || after.RequesterContext.BigPicture != grant.RequesterContext.BigPicture || after.RequesterContext.TechnicalDetails != grant.RequesterContext.TechnicalDetails {
		t.Fatalf("state transition lost frozen requester context: grant=%+v err=%v", after, err)
	}
}
