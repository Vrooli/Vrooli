package conversations_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	"google.golang.org/protobuf/types/known/structpb"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

func TestStatusUpdateStoresSteeringImagesAndSubjectRecurrence(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hubtest.New(t, db, nil, nil)
	service.SetAttachmentStore(hub.StaticBlobStore(blobstore.NewFilesystemBlobStore(t.TempDir())))
	service.SetDefaultRecipient("alice")
	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL, asSubject("alice"))
	status := func(ref, summary string) hub.AskRecord {
		sample, err := os.ReadFile("../../../skills/notification-hub/kit/samples/daily-briefing.json")
		require.NoError(t, err)
		var dataObject map[string]any
		require.NoError(t, json.Unmarshal(sample, &dataObject))
		dataObject["meta"].(map[string]any)["title"] = summary
		data, err := structpb.NewStruct(dataObject)
		require.NoError(t, err)
		response, err := client.Ask(ctx, connect.NewRequest(&v1.AskRequest{
			Kind: "status", Question: summary, SubjectKey: "daily-briefing", IdempotencyKey: ref,
			SensitivityLabel: "private", TemplateName: "daily-briefing", TemplateData: data,
		}))
		require.NoError(t, err)
		record, err := service.GetAsk(ctx, response.Msg.GetAskId())
		require.NoError(t, err)
		require.Empty(t, record.Deadline)
		return record
	}

	first := status("daily-1", "Today's portfolio status")
	got, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: first.ID}))
	require.NoError(t, err)
	require.Equal(t, "status", got.Msg.GetAsk().GetKind())
	require.Equal(t, "daily-briefing", got.Msg.GetAsk().GetTemplateName())
	require.Equal(t, "Today's portfolio status", got.Msg.GetAsk().GetTemplateData().GetFields()["meta"].GetStructValue().GetFields()["title"].GetStringValue())
	require.Empty(t, got.Msg.GetAsk().GetOptions())
	require.Equal(t, int32(1), got.Msg.GetAsk().GetRecurrence().GetCount())
	pageRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/asks/"+first.ID+"/attachments/decision.html", nil)
	require.NoError(t, err)
	pageRequest.Header.Set("X-Vrooli-Identity-Subject", "alice")
	pageResponse, err := server.Client().Do(pageRequest)
	require.NoError(t, err)
	pageBytes, err := io.ReadAll(pageResponse.Body)
	require.NoError(t, err)
	require.NoError(t, pageResponse.Body.Close())
	require.Equal(t, http.StatusOK, pageResponse.StatusCode)
	require.Contains(t, string(pageBytes), `id="hub-data"`)
	require.Contains(t, string(pageBytes), `"title":"Today's portfolio status"`)
	oversizedChanges, err := structpb.NewStruct(map[string]any{"steer": strings.Repeat("x", 70<<10)})
	require.NoError(t, err)
	_, err = client.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{AskId: first.ID, Changes: oversizedChanges}))
	var oversized *connect.Error
	require.ErrorAs(t, err, &oversized)
	require.Equal(t, connect.CodeInvalidArgument, oversized.Code())

	changes, err := structpb.NewStruct(map[string]any{"pace": map[string]any{"tier": "wednesday"}, "review": true})
	require.NoError(t, err)
	imageBytes := []byte{0x89, 'P', 'N', 'G', 13, 10, 26, 10, 0, 0, 0, 0}
	imageHash := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	fileBytes := []byte("response evidence, v1\n")
	fileHash := fmt.Sprintf("%x", sha256.Sum256(fileBytes))
	_, err = client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: imageHash, MediaType: "image/png", Content: imageBytes}))
	require.NoError(t, err)
	_, err = client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: fileHash, MediaType: "text/plain", Content: fileBytes}))
	require.NoError(t, err)
	_, err = client.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{
		AskId: first.ID, Note: "Use the Wednesday capacity tier.", Changes: changes,
		Images:      []*v1.AnswerImage{{Name: "schedule.png", Sha256: imageHash, MediaType: "image/png", SizeBytes: int64(len(imageBytes))}},
		Attachments: []*v1.AnswerAttachment{{Name: "evidence.txt", Sha256: fileHash, MediaType: "text/plain", SizeBytes: int64(len(fileBytes))}},
	}))
	require.NoError(t, err)

	second := status("daily-2", "Updated portfolio status")
	readback, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: second.ID}))
	require.NoError(t, err)
	require.Equal(t, int32(2), readback.Msg.GetAsk().GetRecurrence().GetCount())
	require.Len(t, readback.Msg.GetAsk().GetRecurrence().GetDates(), 2)
	require.Len(t, readback.Msg.GetAsk().GetRecurrence().GetPreviousAnswers(), 1)
	require.Equal(t, "Use the Wednesday capacity tier.", readback.Msg.GetAsk().GetRecurrence().GetPreviousAnswers()[0].GetNote())

	firstReadback, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: first.ID}))
	require.NoError(t, err)
	require.Equal(t, "Use the Wednesday capacity tier.", firstReadback.Msg.GetAsk().GetNote())
	require.Equal(t, map[string]any{"pace": map[string]any{"tier": "wednesday"}, "review": true}, firstReadback.Msg.GetAsk().GetAnswerChanges().AsMap())
	require.Equal(t, imageHash, firstReadback.Msg.GetAsk().GetAnswerImages()[0].GetSha256())
	require.Equal(t, fileHash, firstReadback.Msg.GetAsk().GetAnswerAttachments()[0].GetSha256())

	imageURL := server.URL + "/api/v1/asks/" + first.ID + "/answer-images/schedule.png"
	request, err := http.NewRequest(http.MethodGet, imageURL, nil)
	require.NoError(t, err)
	request.Header.Set("X-Vrooli-Identity-Subject", "alice")
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	served, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, imageBytes, served)
	fileURL := server.URL + "/api/v1/asks/" + first.ID + "/answer-files/evidence.txt"
	fileRequest, err := http.NewRequest(http.MethodGet, fileURL, nil)
	require.NoError(t, err)
	fileRequest.Header.Set("X-Vrooli-Identity-Subject", "alice")
	foreignFileRequest, err := http.NewRequest(http.MethodGet, fileURL, nil)
	require.NoError(t, err)
	foreignFileRequest.Header.Set("X-Vrooli-Identity-Subject", "mallory")
	foreignFileResponse, err := server.Client().Do(foreignFileRequest)
	require.NoError(t, err)
	defer foreignFileResponse.Body.Close()
	require.Equal(t, http.StatusNotFound, foreignFileResponse.StatusCode)
	fileResponse, err := server.Client().Do(fileRequest)
	require.NoError(t, err)
	defer fileResponse.Body.Close()
	require.Equal(t, http.StatusOK, fileResponse.StatusCode)
	require.Equal(t, "attachment; filename=evidence.txt", fileResponse.Header.Get("Content-Disposition"))
	servedFile, err := io.ReadAll(fileResponse.Body)
	require.NoError(t, err)
	require.Equal(t, fileBytes, servedFile)

	marked, err := client.MarkRead(ctx, connect.NewRequest(&v1.MarkReadRequest{AskId: second.ID}))
	require.NoError(t, err)
	require.Equal(t, "read", marked.Msg.GetState())
}

// The owner reads every ask addressed to the configured recipient and no one
// else's. Readers other than the owner are covered by ask_reads_test.go.
func TestOwnerAskReadsAreConfiguredRecipientScoped(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	deadline := time.Now().Add(time.Hour)
	aliceAsk, _, err := service.Ask(ctx, "alice", "Alice question", []string{"yes", "no"}, deadline, "public", "alice-read-test")
	require.NoError(t, err)
	bobAsk, _, err := service.Ask(ctx, "bob", "Bob question", []string{"yes", "no"}, deadline, "public", "bob-read-test")
	require.NoError(t, err)

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL, asSubject("alice"))
	anonymous := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)

	listRequest := connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50})
	list, err := client.ListAsks(ctx, listRequest)
	require.NoError(t, err)
	require.Len(t, list.Msg.GetAsks(), 1)
	require.Equal(t, aliceAsk, list.Msg.GetAsks()[0].GetId())

	secondAsk, _, err := service.Ask(ctx, "alice", "An older searchable question", []string{"yes", "no"}, deadline, "public", "alice-search-test")
	require.NoError(t, err)
	firstPage, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Limit: 1, StateFilter: "awaiting"}))
	require.NoError(t, err)
	require.Len(t, firstPage.Msg.GetAsks(), 1)
	require.NotEmpty(t, firstPage.Msg.GetNextCursor())
	secondPage, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Limit: 1, StateFilter: "awaiting", Cursor: firstPage.Msg.GetNextCursor()}))
	require.NoError(t, err)
	require.Len(t, secondPage.Msg.GetAsks(), 1)
	require.NotEqual(t, firstPage.Msg.GetAsks()[0].GetId(), secondPage.Msg.GetAsks()[0].GetId())
	require.Empty(t, secondPage.Msg.GetNextCursor())
	found, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Query: "searchable", StateFilter: "awaiting"}))
	require.NoError(t, err)
	require.Len(t, found.Msg.GetAsks(), 1)
	require.Equal(t, secondAsk, found.Msg.GetAsks()[0].GetId())
	_, err = client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Query: "changed", StateFilter: "awaiting", Cursor: firstPage.Msg.GetNextCursor()}))
	var invalidCursor *connect.Error
	require.ErrorAs(t, err, &invalidCursor)
	require.Equal(t, connect.CodeInvalidArgument, invalidCursor.Code())

	getRequest := connect.NewRequest(&v1.GetAskRequest{AskId: aliceAsk})
	got, err := client.GetAsk(ctx, getRequest)
	require.NoError(t, err)
	require.Equal(t, "Alice question", got.Msg.GetAsk().GetQuestion())
	keys := make([]string, 0, len(got.Msg.GetAsk().GetOptions()))
	for _, option := range got.Msg.GetAsk().GetOptions() {
		keys = append(keys, option.GetKey())
	}
	require.Contains(t, keys, "other", "every ask must expose the operator-authored alternative")
	foreignRequest := connect.NewRequest(&v1.GetAskRequest{AskId: bobAsk})
	_, err = client.GetAsk(ctx, foreignRequest)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	_, err = anonymous.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{AskId: aliceAsk, Answer: "yes"}))
	require.ErrorAs(t, err, &connectErr, "the answer mutation must keep its human identity requirement")
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	require.Equal(t, "verified owner identity is required", connectErr.Message())
	stillPending, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: aliceAsk}))
	require.NoError(t, err)
	require.Equal(t, "pending", stillPending.Msg.GetAsk().GetState())
	require.Empty(t, stillPending.Msg.GetAsk().GetAnswer())

	service.SetDefaultRecipient("")
	emptyList := connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50})
	_, err = client.ListAsks(ctx, emptyList)
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestDismissEndpointClosesOnlyTheVerifiedRecipientAsk(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hubtest.New(t, db, nil, nil)
	service.SetDefaultRecipient("alice")
	deadline := time.Now().Add(time.Hour)
	askID, _, err := service.Ask(ctx, "alice", "Choose?", []string{"approve", "reject"}, deadline, "public", "dismiss-endpoint-test")
	require.NoError(t, err)

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)

	foreign := connect.NewRequest(&v1.DismissRequest{AskId: askID, Note: "not mine"})
	foreign.Header().Set("X-Vrooli-Identity-Subject", "bob")
	_, err = client.Dismiss(ctx, foreign)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	dismiss := connect.NewRequest(&v1.DismissRequest{AskId: askID, Note: "No longer applicable"})
	dismiss.Header().Set("X-Vrooli-Identity-Subject", "alice")
	response, err := client.Dismiss(ctx, dismiss)
	require.NoError(t, err)
	require.Equal(t, "dismissed", response.Msg.GetState())
	var answers int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM answers WHERE ask_id=?`, askID).Scan(&answers))
	require.Zero(t, answers, "dismissal must not create an answer")
}

func TestGetAskExposesProviderDecisionContextAsTypedOwnerEvidence(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)
	decisionContext := &v1.ProviderDecisionContext{
		SchemaVersion: 1, CanonicalTermsSha256: strings.Repeat("a", 64),
		Scope: v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE,
	}
	askRequest := connect.NewRequest(&v1.AskRequest{
		Question: "Review provider rule terms", Options: []*v1.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private",
		IdempotencyKey: "provider-rule-test", ProviderDecisionContext: decisionContext,
	})
	askRequest.Header().Set("X-Vrooli-Identity-Subject", "alice")
	askResponse, err := client.Ask(ctx, askRequest)
	require.NoError(t, err)

	get, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: askResponse.Msg.GetAskId()}))
	require.NoError(t, err)
	gotContext := get.Msg.GetAsk().GetProviderDecisionContext()
	require.Equal(t, decisionContext.GetSchemaVersion(), gotContext.GetSchemaVersion())
	require.Equal(t, decisionContext.GetCanonicalTermsSha256(), gotContext.GetCanonicalTermsSha256())
	require.Equal(t, decisionContext.GetScope(), gotContext.GetScope())
	require.Equal(t, "unknown", get.Msg.GetAsk().GetAnsweredByIdentityKind())
}
