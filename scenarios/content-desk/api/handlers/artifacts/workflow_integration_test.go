package artifacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"content-desk/handlers/intents"
	"content-desk/handlers/review"
	"content-desk/integrations/channelmanager"
	internalartifacts "content-desk/internal/artifacts"
	internalcampaigns "content-desk/internal/campaigns"
	internalclaims "content-desk/internal/claims"
	localdb "content-desk/internal/database"
	internalintents "content-desk/internal/intents"
	internalledger "content-desk/internal/ledger"
	internalposttypes "content-desk/internal/posttypes"
	internalprojects "content-desk/internal/projects"
	internalreview "content-desk/internal/review"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
	intentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intents"
	intentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intents/intents_v1connect"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review/review_v1connect"
)

type localQueue struct{ calls int }

func (*localQueue) CheckEligibility(context.Context, string, string) (string, error) {
	return "eligible", nil
}
func (q *localQueue) SubmitRelease(context.Context, channelmanager.Submission) (channelmanager.Receipt, error) {
	q.calls++
	return channelmanager.Receipt{ID: "test-release", ActionID: "test-action", Status: "queued"}, nil
}

func TestCampaignFreeDraftReviewApprovalPlanAndQueueWorkflowPersists(t *testing.T) {
	ctx := context.Background()
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(internalartifacts.Schema),
		database.SchemaProviderFunc(internalcampaigns.Schema), database.SchemaProviderFunc(internalclaims.Schema),
		database.SchemaProviderFunc(internalintents.Schema), database.SchemaProviderFunc(internalledger.Schema),
		database.SchemaProviderFunc(internalposttypes.Schema), database.SchemaProviderFunc(internalprojects.Schema),
		database.SchemaProviderFunc(internalreview.Schema)))
	_, err := db.ExecContext(ctx, `UPDATE post_types SET status='active' WHERE id='dev-log'`)
	require.NoError(t, err)
	repo := internalartifacts.NewSQLiteRepository(db)
	queue := &localQueue{}
	artifactPath, artifactHandler := artifactsconnect.NewArtifactsServiceHandler(handler{repo: repo, submitter: queue, eligibility: queue})
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: artifactPath, Handler: artifactHandler})
	review.Module(db).Mount(router)
	intents.Module(db).Mount(router)
	server := httptest.NewServer(router)
	artifactClient := artifactsconnect.NewArtifactsServiceClient(http.DefaultClient, server.URL)
	reviewClient := reviewconnect.NewReviewServiceClient(http.DefaultClient, server.URL)
	intentClient := intentsconnect.NewPublicationIntentServiceClient(http.DefaultClient, server.URL)

	created, err := artifactClient.CreateDraft(ctx, connect.NewRequest(&artifactsv1.CreateDraftRequest{PostTypeId: "dev-log", Body: "Synthetic draft placeholder.", Channel: "linkedin", Format: "post"}))
	require.NoError(t, err)
	draft := created.Msg.Draft
	require.Empty(t, draft.CampaignId)
	require.Equal(t, "unassigned", draft.ProjectId)
	require.Equal(t, "requested", draft.Status)
	edited, err := artifactClient.UpdateDraftBody(ctx, connect.NewRequest(&artifactsv1.UpdateDraftBodyRequest{Id: draft.Id, Body: "A synthetic note tied to a local evidence receipt."}))
	require.NoError(t, err)
	draft = edited.Msg.Draft
	for _, event := range []string{"begin", "complete", "check"} {
		response, transitionErr := artifactClient.TransitionDraft(ctx, connect.NewRequest(&artifactsv1.TransitionDraftRequest{Id: draft.Id, Event: event}))
		require.NoError(t, transitionErr)
		draft = response.Msg.Draft
	}
	require.Equal(t, "checking", draft.Status)
	currentResponse, err := artifactClient.GetDraftCurrentRevision(ctx, connect.NewRequest(&artifactsv1.GetDraftCurrentRevisionRequest{Id: draft.Id}))
	require.NoError(t, err)
	current := currentResponse.Msg.CurrentRevision
	modes, err := reviewClient.ListRequiredReviewModes(ctx, connect.NewRequest(&reviewv1.ListRequiredReviewModesRequest{PostTypeId: draft.PostTypeId}))
	require.NoError(t, err)
	require.NotEmpty(t, modes.Msg.Modes)
	var verdicts []*reviewv1.Verdict
	for _, mode := range modes.Msg.Modes {
		verdicts = append(verdicts, &reviewv1.Verdict{Mode: mode, Passed: true, Evidence: "Synthetic local check: no unsupported claim or imitation was found."})
	}
	reviewed, err := reviewClient.RecordReviewRun(ctx, connect.NewRequest(&reviewv1.RecordReviewRunRequest{DraftId: draft.Id, ExpectedRevisionId: current.RevisionId, ExpectedBody: current.Body, Verdicts: verdicts}))
	require.NoError(t, err)
	require.Equal(t, current.RevisionId, reviewed.Msg.ReviewRun.SourceRevisionId)
	transitioned, err := artifactClient.TransitionDraft(ctx, connect.NewRequest(&artifactsv1.TransitionDraftRequest{Id: draft.Id, Event: "review_pass"}))
	require.NoError(t, err)
	require.Equal(t, "reviewed", transitioned.Msg.Draft.Status)
	approved, err := artifactClient.ApproveDraft(ctx, connect.NewRequest(&artifactsv1.ApproveDraftRequest{Id: draft.Id, IdentityId: "synthetic-test-identity", Lane: "test-lane", ExpectedRevisionId: current.RevisionId, ExpectedBody: current.Body}))
	require.NoError(t, err)
	require.Equal(t, "approved", approved.Msg.Draft.Status)
	intent, err := intentClient.SetPublicationIntent(ctx, connect.NewRequest(&intentsv1.SetPublicationIntentRequest{DraftId: draft.Id, Destination: draft.Channel, DesiredLocalDate: "2026-12-01", TimeZone: "UTC", Daypart: "morning", ExpectedRevisionId: current.RevisionId, ExpectedBody: current.Body, IdempotencyKey: "synthetic-workflow-plan"}))
	require.NoError(t, err)
	require.Equal(t, "desired", intent.Msg.Intent.Status)
	release, err := artifactClient.SubmitReleaseDraft(ctx, connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: draft.Id, IdempotencyKey: "synthetic-workflow-handoff", DisclosureVisible: true}))
	require.NoError(t, err)
	require.Equal(t, "queued", release.Msg.ReleaseStatus)
	require.Equal(t, 1, queue.calls, "only the local fake queue receives the handoff")
	server.Close()

	// Re-mount the scenario API over the same SQLite state to prove that the
	// review basis, exact approval, plan intent and queue handoff survive a restart.
	router = mux.NewRouter()
	artifactPath, artifactHandler = artifactsconnect.NewArtifactsServiceHandler(handler{repo: repo, submitter: queue, eligibility: queue})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: artifactPath, Handler: artifactHandler})
	review.Module(db).Mount(router)
	intents.Module(db).Mount(router)
	server = httptest.NewServer(router)
	defer server.Close()
	artifactClient = artifactsconnect.NewArtifactsServiceClient(http.DefaultClient, server.URL)
	reviewClient = reviewconnect.NewReviewServiceClient(http.DefaultClient, server.URL)
	intentClient = intentsconnect.NewPublicationIntentServiceClient(http.DefaultClient, server.URL)
	persisted, err := artifactClient.GetDraftCurrentRevision(ctx, connect.NewRequest(&artifactsv1.GetDraftCurrentRevisionRequest{Id: draft.Id}))
	require.NoError(t, err)
	require.Equal(t, "approved", persisted.Msg.CurrentRevision.Status)
	require.True(t, persisted.Msg.CurrentRevision.HasApplicableReview)
	require.True(t, persisted.Msg.CurrentRevision.HasApplicableApproval)
	runs, err := reviewClient.ListReviewRuns(ctx, connect.NewRequest(&reviewv1.ListReviewRunsRequest{}))
	require.NoError(t, err)
	require.Len(t, runs.Msg.ReviewRuns, 1)
	planned, err := intentClient.ListPublicationIntents(ctx, connect.NewRequest(&intentsv1.ListPublicationIntentsRequest{DraftId: draft.Id}))
	require.NoError(t, err)
	require.Len(t, planned.Msg.Intents, 1)
	require.Equal(t, intent.Msg.Intent.Id, planned.Msg.Intents[0].Id)
}
