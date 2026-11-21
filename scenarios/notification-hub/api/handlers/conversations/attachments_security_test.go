package conversations_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	"google.golang.org/protobuf/proto"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
)

// ---- caching -------------------------------------------------------------------

func TestAttachmentContentRoute_ARevisionThatReusesANameIsNeverServedFromTheOldCopy(t *testing.T) {
	f := newAttachmentFixture(t)
	ask := func(event, content string) string {
		response, err := f.client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
			Question: "Approve the plan? (" + event + ")", AllowedAnswers: []string{"yes", "no"}, BigPicture: proto.String("The plan changes retention."),
			Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: event,
			SubjectKey: "the-plan", Attachments: []*v1.AskAttachment{fileRef("plan.md", f.put(t, content))},
		}))
		require.NoError(t, err)
		return response.Msg.GetAskId()
	}
	askID := ask("evt-1", "# Plan v1")
	path := "/api/v1/asks/" + askID + "/attachments/plan.md"
	first := f.get(t, path)
	require.Equal(t, "# Plan v1", body(t, first))
	oldTag := first.Header.Get("ETag")

	require.Equal(t, askID, ask("evt-2", "# Plan v2, rewritten"), "the revision updates the same ask in place")

	// A browser holding the old copy asks again. It must get the new bytes.
	second := f.get(t, path, "If-None-Match", oldTag)
	require.Equal(t, http.StatusOK, second.StatusCode, "the old validator no longer matches")
	require.Equal(t, "# Plan v2, rewritten", body(t, second))
	require.NotEqual(t, oldTag, second.Header.Get("ETag"))
	require.Equal(t, "private, no-cache", second.Header.Get("Cache-Control"), "nothing lets a browser reuse the copy without asking")
	require.Equal(t, http.StatusNotModified, f.get(t, path, "If-None-Match", second.Header.Get("ETag")).StatusCode, "an unchanged file still costs no body")
}

// ---- who may upload ------------------------------------------------------------

func TestAnonymousCanUploadButCannotInspectAttachmentInventory(t *testing.T) {
	f := newAttachmentFixture(t)
	store := blobstore.NewMemoryBlobStore()
	f.service.SetAttachmentStore(hub.StaticBlobStore(store))
	ctx := context.Background()
	content := "uploaded by nobody"

	_, err := f.anonymous.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf([]byte(content)), Content: []byte(content)}))
	require.NoError(t, err, "packet content upload does not require an identity")
	_, err = f.anonymous.StatAttachmentBlobs(ctx, connect.NewRequest(&v1.StatAttachmentBlobsRequest{Sha256: []string{hashOf([]byte(content))}}))
	require.Equal(t, connect.CodeUnauthenticated, codeOf(t, err), "nor learn what it holds")
	stored, _, err := store.Get(ctx, "sha256/"+hashOf([]byte(content))[:2]+"/"+hashOf([]byte(content)))
	require.NoError(t, err, "uploaded packet bytes are available for a later ask")
	defer stored.Close()
	storedBytes, err := io.ReadAll(stored)
	require.NoError(t, err)
	require.Equal(t, content, string(storedBytes))

	require.Equal(t, hashOf([]byte(content)), f.put(t, content), "the owner can")
}

func TestAttachmentWritesAreIdentityOptionalButInventoryReadsStayVerified(t *testing.T) {
	f := newAttachmentFixture(t)
	router := mux.NewRouter()
	conversationsH.ModuleWithVerifier(f.service, tokenVerifier{}).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := func(token string) conversationsconnect.ConversationsServiceClient {
		return conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL, withHeader("Authorization", "Bearer "+token))
	}
	put := func(c conversationsconnect.ConversationsServiceClient) error {
		_, err := c.PutAttachmentBlob(context.Background(), connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf([]byte("x")), Content: []byte("x")}))
		return err
	}

	require.NoError(t, put(client("owner-token")))
	require.NoError(t, put(client("someone-else")))
	anonymous := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)
	require.NoError(t, put(anonymous))
	_, err := anonymous.StatAttachmentBlobs(context.Background(), connect.NewRequest(&v1.StatAttachmentBlobsRequest{Sha256: []string{hashOf([]byte("x"))}}))
	require.Equal(t, connect.CodeUnauthenticated, codeOf(t, err))
}

func TestAsk_AttachmentsFromAnIdentitylessProducerAreKeptWithoutHostPathProvenance(t *testing.T) {
	f := newAttachmentFixture(t)
	hash := f.put(t, "plan bytes")
	response, err := f.anonymous.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
		Question: "Approve the plan?", AllowedAnswers: []string{"yes", "no"}, BigPicture: proto.String("The plan changes retention."),
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: "anon-ask",
		Attachments: []*v1.AskAttachment{{Name: "plan.md", Kind: v1.AskAttachment_KIND_FILE, Sha256: hash, SourcePath: "/etc/hostname", SourceHost: "any"}},
	}))

	require.NoError(t, err, "an ask from a host process is still accepted")
	got, err := f.client.GetAsk(context.Background(), connect.NewRequest(&v1.GetAskRequest{AskId: response.Msg.GetAskId()}))
	require.NoError(t, err)
	attachments := got.Msg.GetAsk().GetAttachments()
	require.Len(t, attachments, 2, "the Hub's page, then the plan")
	require.Equal(t, "plan.md", attachments[1].GetName())
	require.Empty(t, attachments[1].GetSourcePath(), "the Hub must not probe caller-selected host paths")
	require.Empty(t, attachments[1].GetSourceHost())
}

// ---- memory --------------------------------------------------------------------

// slotServer serves the module with two upload slots and a short wait.
func slotServer(t *testing.T, f *attachmentFixture) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	conversationsH.ModuleWithUploadLimits(f.service, nil, 2, 300*time.Millisecond).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

// holdUpload starts an upload whose body never finishes, so the Hub holds its
// slot for as long as the test wants.
func holdUpload(t *testing.T, server *httptest.Server) *io.PipeWriter {
	t.Helper()
	reader, writer := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, server.URL+conversationsconnect.ConversationsServicePutAttachmentBlobProcedure, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Identity-Subject", "alice")
	go func() {
		if resp, err := server.Client().Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}()
	go func() { _, _ = writer.Write([]byte(`{"sha256":"`)) }()
	t.Cleanup(func() { _ = writer.CloseWithError(io.ErrClosedPipe) })
	return writer
}

func postUpload(t *testing.T, server *httptest.Server, content string) *http.Response {
	t.Helper()
	body := `{"sha256":"` + hashOf([]byte(content)) + `","content":"` + base64.StdEncoding.EncodeToString([]byte(content)) + `"}`
	req, err := http.NewRequest(http.MethodPost, server.URL+conversationsconnect.ConversationsServicePutAttachmentBlobProcedure, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Identity-Subject", "alice")
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// Mounted, the upload procedure sits behind the gate: with both slots held, a
// further upload is told the Hub is busy, and it succeeds once a slot is free.
func TestUploads_AreLimitedToTheirSlotsAndReleaseThemWhenTheyEnd(t *testing.T) {
	f := newAttachmentFixture(t)
	server := slotServer(t, f)
	first, second := holdUpload(t, server), holdUpload(t, server)

	// The held uploads take their slots a moment after they start; until both
	// have, another upload still gets in.
	var busy *http.Response
	require.Eventually(t, func() bool {
		busy = postUpload(t, server, "while busy")
		return busy.StatusCode == http.StatusServiceUnavailable
	}, 5*time.Second, 20*time.Millisecond, "a third upload was never refused")
	require.NotEmpty(t, busy.Header.Get("Retry-After"))

	_ = first.CloseWithError(io.ErrClosedPipe)
	_ = second.CloseWithError(io.ErrClosedPipe)
	require.Eventually(t, func() bool {
		return postUpload(t, server, "after").StatusCode == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond, "the slots were never released")
}

// failingBody fails the test if the gate lets anything read the request body.
type failingBody struct{ t *testing.T }

func (b failingBody) Read([]byte) (int, error) {
	b.t.Error("the request body was read before the gate admitted the upload")
	return 0, io.ErrUnexpectedEOF
}

func (b failingBody) Close() error { return nil }

// The gate decides before a byte of the body is read: that is what keeps a
// burst of 85 MiB messages from being in memory at once.
func TestUploadGate_AdmitsOrRefusesBeforeTheBodyIsRead(t *testing.T) {
	f := newAttachmentFixture(t)
	entered, release := make(chan struct{}, 8), make(chan struct{})
	gate := conversationsH.GateUploads(f.service, nil, 2, 150*time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	request := func(owner string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, conversationsconnect.ConversationsServicePutAttachmentBlobProcedure, failingBody{t})
		if owner != "" {
			req.Header.Set("X-Vrooli-Identity-Subject", owner)
		}
		return req
	}
	var admitted sync.WaitGroup
	for i := 0; i < 2; i++ {
		admitted.Add(1)
		go func() {
			defer admitted.Done()
			gate.ServeHTTP(httptest.NewRecorder(), request("alice"))
		}()
		<-entered
	}

	full := httptest.NewRecorder()
	gate.ServeHTTP(full, request("alice"))
	require.Equal(t, http.StatusServiceUnavailable, full.Code, "no free slot: busy, and the body was not read")
	require.Equal(t, "5", full.Header().Get("Retry-After"))

	unverified := httptest.NewRecorder()
	gate.ServeHTTP(unverified, request(""))
	require.Equal(t, http.StatusServiceUnavailable, unverified.Code, "identityless uploads use the same slot limit")

	close(release)
	admitted.Wait()
	ok := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { gate.ServeHTTP(ok, request("alice")); close(done) }()
	<-entered
	<-done
	require.Equal(t, http.StatusOK, ok.Code, "the slots were released")
}

func TestOtherProcedures_RefuseACompressedMessageThatExpandsPastTheirLimit(t *testing.T) {
	f := newAttachmentFixture(t)
	// Nine MiB of one letter is a few kilobytes of gzip, far under any limit on
	// the compressed size, and over the 8 MiB the procedure accepts once expanded.
	message := `{"question":"` + strings.Repeat("q", 9<<20) + `","allowedAnswers":["yes"],"deadline":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","sensitivityLabel":"private","idempotencyKey":"bomb"}`
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, err := zw.Write([]byte(message))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.Less(t, compressed.Len(), 64<<10, "the body is small; only its expansion is large")

	req, err := http.NewRequest(http.MethodPost, f.url+conversationsconnect.ConversationsServiceAskProcedure, &compressed)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Vrooli-Identity-Subject", "alice")
	resp, err := f.http.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode, string(data))
	require.Contains(t, string(data), "resource_exhausted")
}

// ---- file names ----------------------------------------------------------------

func TestAttachmentContentRoute_ANameThatIsNotASCIIIsOfferedAsFilenameStar(t *testing.T) {
	f := newAttachmentFixture(t)
	response := f.ask(t, "utf8-1", &v1.AskAttachment{Name: "resume.md", Kind: v1.AskAttachment_KIND_FILE, Sha256: f.put(t, "# Resume"), SourcePath: "/work/résumé final.md"})

	resp := f.get(t, "/api/v1/asks/"+response.GetAskId()+"/attachments/resume.md")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, `inline; filename*=utf-8''r%C3%A9sum%C3%A9%20final.md`, resp.Header.Get("Content-Disposition"))
}

// ---- errors --------------------------------------------------------------------

func TestAttachmentError_MapsEachFailureToItsConnectCode(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want connect.Code
	}{
		"no store":                   {hub.ErrAttachmentsUnavailable, connect.CodeUnavailable},
		"a blob over the limit":      {hub.ErrBlobTooLarge, connect.CodeInvalidArgument},
		"too many bytes are waiting": {fmt.Errorf("%w: 1000 MiB are waiting", hub.ErrBlobQuota), connect.CodeResourceExhausted},
		"a bad argument":             {fmt.Errorf("%w: sha256", hub.ErrInvalidArgument), connect.CodeInvalidArgument},
		"anything else":              {errors.New("disk failed"), connect.CodeInternal},
	} {
		require.Equal(t, tc.want, codeOf(t, conversationsH.AttachmentError(tc.err)), name)
	}
}
