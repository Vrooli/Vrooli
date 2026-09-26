package objectives

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	objectivesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/objectives"
	objectivesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/objectives/objectives_v1connect"

	domain "prompt-manager/internal/objectives"
	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

// newTestClient mounts the generated Connect service over a real routed SQLite
// handle, so the test exercises the same transport and validation path a live
// client uses rather than calling handler methods directly.
func newTestClient(t *testing.T) objectivesconnect.ObjectivesServiceClient {
	t.Helper()
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(domain.Schema)); err != nil {
		t.Fatalf("apply objectives schema: %v", err)
	}
	service := domain.NewService(domain.NewRepository(db), func(string) bool { return true })
	path, handler := NewConnectMount(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return objectivesconnect.NewObjectivesServiceClient(server.Client(), server.URL)
}

func createObjective(t *testing.T, client objectivesconnect.ObjectivesServiceClient, id, class string) *objectivesv1.Objective {
	t.Helper()
	resp, err := client.UpsertObjective(context.Background(), connect.NewRequest(&objectivesv1.UpsertObjectiveRequest{
		Objective: &objectivesv1.ObjectiveInput{
			Id:             id,
			Title:          "Objective " + id,
			Class:          class,
			EvidenceSource: "owner evidence for " + id,
		},
	}))
	if err != nil {
		t.Fatalf("upsert objective %s: %v", id, err)
	}
	return resp.Msg
}

func TestConnectObjectiveJourney(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)

	created := createObjective(t, client, "T1", "terminal")
	if created.GetMeaningRevision() == "" {
		t.Fatal("expected a computed meaning revision over the wire")
	}
	createObjective(t, client, "I1", "instrumental")

	// Attach over the generated contract; a fresh link is restatement-pending.
	attached, err := client.AttachObjective(ctx, connect.NewRequest(&objectivesv1.AttachObjectiveRequest{
		Attachment: &objectivesv1.AttachmentInput{
			ObjectiveId: "T1",
			TeamId:      "marketing-crew",
			Role:        "primary",
			Coverage:    "full",
		},
		ExpectedTeamRevision: "",
	}))
	if err != nil {
		t.Fatalf("attach objective: %v", err)
	}
	if !attached.Msg.GetRestatementPending() {
		t.Fatal("a fresh attachment must be restatement-pending over the wire")
	}
	if attached.Msg.GetAttachmentRevision() == "" {
		t.Fatal("expected a team attachment revision over the wire")
	}

	// A duplicate link is refused with the same classification the service uses.
	if _, err := client.AttachObjective(ctx, connect.NewRequest(&objectivesv1.AttachObjectiveRequest{
		Attachment:           &objectivesv1.AttachmentInput{ObjectiveId: "T1", TeamId: "marketing-crew"},
		ExpectedTeamRevision: attached.Msg.GetAttachmentRevision(),
	})); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("expected AlreadyExists for a duplicate link, got %v", err)
	}

	// A stale team revision is refused as a conflict.
	if _, err := client.AttachObjective(ctx, connect.NewRequest(&objectivesv1.AttachObjectiveRequest{
		Attachment:           &objectivesv1.AttachmentInput{ObjectiveId: "I1", TeamId: "marketing-crew"},
		ExpectedTeamRevision: "stale",
	})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("expected Aborted for a stale revision, got %v", err)
	}

	// Reorder reads the current revision and persists the team priority.
	revision, err := client.GetTeamAttachmentRevision(ctx, connect.NewRequest(&objectivesv1.GetTeamAttachmentRevisionRequest{TeamId: "marketing-crew"}))
	if err != nil {
		t.Fatalf("get team revision: %v", err)
	}
	linked, err := client.AttachObjective(ctx, connect.NewRequest(&objectivesv1.AttachObjectiveRequest{
		Attachment:           &objectivesv1.AttachmentInput{ObjectiveId: "I1", TeamId: "marketing-crew"},
		ExpectedTeamRevision: revision.Msg.GetAttachmentRevision(),
	}))
	if err != nil {
		t.Fatalf("attach I1: %v", err)
	}
	reordered, err := client.ReorderTeamAttachments(ctx, connect.NewRequest(&objectivesv1.ReorderTeamAttachmentsRequest{
		TeamId:               "marketing-crew",
		ObjectiveIds:         []string{"I1", "T1"},
		ExpectedTeamRevision: linked.Msg.GetAttachmentRevision(),
	}))
	if err != nil {
		t.Fatalf("reorder team attachments: %v", err)
	}
	if got := reordered.Msg.GetAttachments(); len(got) != 2 || got[0].GetObjectiveId() != "I1" {
		t.Fatalf("expected I1 first after reorder, got %+v", got)
	}

	// Relations and validation share the service read and error semantics.
	if _, err := client.AddRelation(ctx, connect.NewRequest(&objectivesv1.AddRelationRequest{
		FromObjectiveId: "I1",
		ToObjectiveId:   "T1",
	})); err != nil {
		t.Fatalf("add relation: %v", err)
	}
	validation, err := client.ValidateObjectives(ctx, connect.NewRequest(&objectivesv1.ValidateObjectivesRequest{}))
	if err != nil {
		t.Fatalf("validate objectives: %v", err)
	}
	if validation.Msg.GetErrors() != 0 {
		t.Fatalf("expected no validation errors, got %+v", validation.Msg.GetFindings())
	}
}

func TestConnectNotFoundClassification(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.GetObjective(context.Background(), connect.NewRequest(&objectivesv1.GetObjectiveRequest{Id: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound for an unknown objective, got %v", err)
	}
}
