package conversations_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	"google.golang.org/protobuf/proto"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

type attachmentFixture struct {
	// client is the owner's: the fixture wires no verifier, so the owner is the
	// legacy subject header, which the interceptor sets.
	client conversationsconnect.ConversationsServiceClient
	// anonymous presents nothing, like a host process with no credential.
	anonymous conversationsconnect.ConversationsServiceClient
	service   *hub.Service
	url       string
	http      *http.Client
}

// withHeader makes a client send the header on every call.
func withHeader(name, value string) connect.Option {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set(name, value)
			return next(ctx, req)
		}
	}))
}

// asSubject makes a client send the given owner subject on every call.
func asSubject(subject string) connect.Option {
	return withHeader("X-Vrooli-Identity-Subject", subject)
}

func newAttachmentFixture(t *testing.T) *attachmentFixture {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	service.SetAttachmentStore(hub.StaticBlobStore(blobstore.NewFilesystemBlobStore(t.TempDir())))
	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return &attachmentFixture{
		client:    conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL, asSubject("alice")),
		anonymous: conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL),
		service:   service, url: server.URL, http: server.Client(),
	}
}

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (f *attachmentFixture) put(t *testing.T, content string) string {
	t.Helper()
	hash := hashOf([]byte(content))
	_, err := f.client.PutAttachmentBlob(context.Background(), connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hash, Content: []byte(content)}))
	require.NoError(t, err)
	return hash
}

func (f *attachmentFixture) ask(t *testing.T, key string, attachments ...*v1.AskAttachment) *v1.AskResponse {
	t.Helper()
	response, err := f.client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
		Question: "Approve the plan?", AllowedAnswers: []string{"yes", "no"}, BigPicture: proto.String("The plan changes retention."),
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: key,
		Attachments: attachments,
	}))
	require.NoError(t, err)
	return response.Msg
}

func fileRef(name, hash string) *v1.AskAttachment {
	return &v1.AskAttachment{Name: name, Kind: v1.AskAttachment_KIND_FILE, Sha256: hash}
}

func (f *attachmentFixture) get(t *testing.T, path string, headers ...string) *http.Response {
	t.Helper()
	return f.do(t, http.MethodGet, path, headers...)
}

func (f *attachmentFixture) do(t *testing.T, method, path string, headers ...string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, f.url+path, nil)
	require.NoError(t, err)
	// Serving bytes needs a verified owner. The fixture wires no verifier, so
	// the owner is the legacy subject header; tests that probe identity use
	// getAs instead.
	req.Header.Set("X-Vrooli-Identity-Subject", "alice")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := f.http.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(data)
}

func codeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	return connectErr.Code()
}

func TestAttachmentSendFlow_StatPutAskThenGetAsk(t *testing.T) {
	f := newAttachmentFixture(t)
	ctx := context.Background()
	plan := []byte("# The plan\nKeep logs 30 days.\n")
	chart := []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")

	stat, err := f.client.StatAttachmentBlobs(ctx, connect.NewRequest(&v1.StatAttachmentBlobsRequest{Sha256: []string{hashOf(plan), hashOf(chart)}}))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{hashOf(plan), hashOf(chart)}, stat.Msg.GetMissing(), "nothing is stored yet")

	put, err := f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf(plan), MediaType: "text/markdown", Content: plan}))
	require.NoError(t, err)
	require.Equal(t, int64(len(plan)), put.Msg.GetSizeBytes())
	require.False(t, put.Msg.GetExisted())
	again, err := f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf(plan), Content: plan}))
	require.NoError(t, err)
	require.True(t, again.Msg.GetExisted())
	f.put(t, string(chart))

	stat, err = f.client.StatAttachmentBlobs(ctx, connect.NewRequest(&v1.StatAttachmentBlobsRequest{Sha256: []string{hashOf(plan), hashOf(chart)}}))
	require.NoError(t, err)
	require.Empty(t, stat.Msg.GetMissing())

	response := f.ask(t, "send-flow",
		&v1.AskAttachment{Name: "plan.md", Label: "The plan", Role: v1.AskAttachment_ROLE_EXPLAINER, Kind: v1.AskAttachment_KIND_FILE, Sha256: hashOf(plan), SourcePath: "/work/plan.md", SourceHost: "elsewhere"},
		&v1.AskAttachment{Name: "report", Kind: v1.AskAttachment_KIND_FOLDER, Entries: []*v1.AskAttachmentEntry{{Path: "img/chart.svg", Sha256: hashOf(chart)}}},
		fileRef("lost.pdf", hashOf([]byte("never uploaded"))),
	)

	var dropped *v1.PacketWarning
	for _, w := range response.GetWarnings() {
		if w.GetCode() == "attachment_blob_missing" {
			dropped = w
		}
	}
	require.NotNil(t, dropped, "warnings: %v", response.GetWarnings())
	require.Equal(t, "attachments[2]", dropped.GetField())
	require.Equal(t, "dropped", dropped.GetSeverity())
	require.NotEmpty(t, response.GetAskId(), "a dropped attachment never fails the ask")

	got, err := f.client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: response.GetAskId()}))
	require.NoError(t, err)
	ask := got.Msg.GetAsk()
	require.Equal(t, int32(2), ask.GetAttachmentCount())
	require.Len(t, ask.GetAttachments(), 2)
	planOut := ask.GetAttachments()[0]
	require.Equal(t, "plan.md", planOut.GetName())
	require.Equal(t, "The plan", planOut.GetLabel())
	require.Equal(t, v1.AskAttachment_ROLE_EXPLAINER, planOut.GetRole())
	require.Equal(t, v1.AskAttachment_KIND_FILE, planOut.GetKind())
	require.Equal(t, "markdown", planOut.GetPreviewKind())
	require.Equal(t, int64(len(plan)), planOut.GetSizeBytes())
	require.Equal(t, "/work/plan.md", planOut.GetSourcePath())
	require.Equal(t, "unknown", planOut.GetSourceStatus(), "another host's paths are not checked")
	folderOut := ask.GetAttachments()[1]
	require.Equal(t, v1.AskAttachment_KIND_FOLDER, folderOut.GetKind())
	require.Equal(t, "directory", folderOut.GetPreviewKind())
	require.Equal(t, "svg", folderOut.GetEntries()[0].GetPreviewKind())

	list, err := f.client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(2), list.Msg.GetAsks()[0].GetAttachmentCount(), "a list row carries the count, not the attachments")
	require.Empty(t, list.Msg.GetAsks()[0].GetAttachments())
}

func TestPutAttachmentBlob_RefusesAMismatchedHashAndOversizeFiles(t *testing.T) {
	f := newAttachmentFixture(t)
	ctx := context.Background()

	_, err := f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf([]byte("a")), Content: []byte("b")}))
	require.Equal(t, connect.CodeInvalidArgument, codeOf(t, err), "the Hub recomputes the hash")

	over := make([]byte, hub.MaxAttachmentBlobBytes+1)
	_, err = f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf(over), Content: over}))
	require.Equal(t, connect.CodeInvalidArgument, codeOf(t, err), "a file is at most 64 MiB")

	exact := bytes.Repeat([]byte("x"), hub.MaxAttachmentBlobBytes)
	_, err = f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf(exact), Content: exact}))
	require.NoError(t, err, "the upload procedure accepts a 64 MiB message")
}

func TestOtherProceduresDoNotAcceptAFileSizedMessage(t *testing.T) {
	f := newAttachmentFixture(t)
	huge := strings.Repeat("q", 9<<20)
	_, err := f.client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
		Question: huge, AllowedAnswers: []string{"yes"}, Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		SensitivityLabel: "private", IdempotencyKey: "huge",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, codeOf(t, err))
}

func TestPutAttachmentBlob_WithoutAStoreIsUnavailable(t *testing.T) {
	f := newAttachmentFixture(t)
	f.service.SetAttachmentStore(nil)
	_, err := f.client.PutAttachmentBlob(context.Background(), connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hashOf([]byte("a")), Content: []byte("a")}))
	require.Equal(t, connect.CodeUnavailable, codeOf(t, err))
}

// ---- content route -----------------------------------------------------------

func TestAttachmentContentRoute_ServesTheStoredCopyWithSafeHeaders(t *testing.T) {
	f := newAttachmentFixture(t)
	content := "# The plan\nKeep logs 30 days.\n"
	response := f.ask(t, "content-1", &v1.AskAttachment{Name: "plan.md", Kind: v1.AskAttachment_KIND_FILE, Sha256: f.put(t, content), SourcePath: "/work/the plan.md"})
	path := "/api/v1/asks/" + response.GetAskId() + "/attachments/plan.md"

	resp := f.get(t, path)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, content, body(t, resp))
	require.Equal(t, "text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "private, no-cache", resp.Header.Get("Cache-Control"), "the URL does not name the bytes, so every use is revalidated")
	csp := resp.Header.Get("Content-Security-Policy")
	require.Contains(t, csp, "sandbox", "a file opened directly can never run as a page of the Hub's origin")
	require.Contains(t, csp, "default-src 'none'")
	require.Equal(t, `"sha256-`+hashOf([]byte(content))+`"`, resp.Header.Get("ETag"))
	require.Equal(t, "bytes", resp.Header.Get("Accept-Ranges"))
	require.Equal(t, `inline; filename="the plan.md"`, resp.Header.Get("Content-Disposition"))
}

func TestAttachmentContentRoute_SupportsByteRangesForMedia(t *testing.T) {
	f := newAttachmentFixture(t)
	content := "0123456789abcdefghij"
	response := f.ask(t, "range-1", fileRef("clip.mp4", f.put(t, content)))
	path := "/api/v1/asks/" + response.GetAskId() + "/attachments/clip.mp4"

	partial := f.get(t, path, "Range", "bytes=5-9")
	require.Equal(t, http.StatusPartialContent, partial.StatusCode)
	require.Equal(t, "56789", body(t, partial))
	require.Equal(t, "bytes 5-9/20", partial.Header.Get("Content-Range"))
	require.Equal(t, "video/mp4", partial.Header.Get("Content-Type"))

	tail := f.get(t, path, "Range", "bytes=-4")
	require.Equal(t, http.StatusPartialContent, tail.StatusCode)
	require.Equal(t, "ghij", body(t, tail))

	unsatisfiable := f.get(t, path, "Range", "bytes=100-200")
	require.Equal(t, http.StatusRequestedRangeNotSatisfiable, unsatisfiable.StatusCode)
}

func TestAttachmentContentRoute_ServesFolderEntriesByPath(t *testing.T) {
	f := newAttachmentFixture(t)
	response := f.ask(t, "folder-1", &v1.AskAttachment{Name: "report", Kind: v1.AskAttachment_KIND_FOLDER, Entries: []*v1.AskAttachmentEntry{
		{Path: "sub/index.md", Sha256: f.put(t, "# Index")},
		{Path: "data/rows.csv", Sha256: f.put(t, "a,b\n1,2\n")},
	}})
	base := "/api/v1/asks/" + response.GetAskId() + "/attachments/report"

	index := f.get(t, base+"?path=sub/index.md")
	require.Equal(t, http.StatusOK, index.StatusCode)
	require.Equal(t, "# Index", body(t, index))
	require.Equal(t, "text/markdown; charset=utf-8", index.Header.Get("Content-Type"))
	require.Equal(t, `inline; filename=index.md`, index.Header.Get("Content-Disposition"))

	csv := f.get(t, base+"?path=data%2Frows.csv")
	require.Equal(t, http.StatusOK, csv.StatusCode)
	require.Equal(t, "text/csv; charset=utf-8", csv.Header.Get("Content-Type"))

	for name, path := range map[string]string{
		"a folder with no path":    base,
		"a path that is not there": base + "?path=nope.md",
		"a path outside":           base + "?path=../../etc/passwd",
	} {
		resp := f.get(t, path)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, name)
	}
}

func TestAttachmentContentRoute_IsRecipientOnlyAndRefusesUniformly(t *testing.T) {
	f := newAttachmentFixture(t)
	hash := f.put(t, "secret")
	mine := f.ask(t, "mine", fileRef("a.md", hash))
	// An ask whose recipient is someone else: the route's recipient check, not
	// the ask's existence, is what refuses it.
	theirs, _, err := f.service.AskFrom(context.Background(), "test", hub.AskSpec{
		Recipient: "bob", Question: "Bob's question?", Options: []hub.AskOption{{Key: "y", Label: "y"}}, Deadline: time.Now().Add(time.Hour),
		SensitivityLabel: "private", IdempotencyKey: "bobs", Attachments: []hub.AskAttachment{{Name: "b.md", Kind: hub.AttachmentFile, SHA256: hash}},
	})
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, f.get(t, "/api/v1/asks/"+mine.GetAskId()+"/attachments/a.md").StatusCode)
	asBob := f.do(t, http.MethodGet, "/api/v1/asks/"+mine.GetAskId()+"/attachments/a.md", "X-Vrooli-Identity-Subject", "bob")
	require.Equal(t, http.StatusNotFound, asBob.StatusCode, "a verified owner who is not the ask's recipient")
	bodies := map[string]string{}
	for name, path := range map[string]string{
		"another recipient's ask": "/api/v1/asks/" + theirs.ID + "/attachments/b.md",
		"unknown ask":             "/api/v1/asks/no-such-ask/attachments/a.md",
		"unknown attachment":      "/api/v1/asks/" + mine.GetAskId() + "/attachments/nope.md",
		"a path on a file":        "/api/v1/asks/" + mine.GetAskId() + "/attachments/a.md?path=x",
	} {
		resp := f.get(t, path)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, name)
		bodies[name] = body(t, resp)
	}
	for name, got := range bodies {
		require.Equal(t, bodies["unknown ask"], got, "%s must be indistinguishable from an unknown ask", name)
	}
}

func TestAttachmentContentRoute_HeadAndConditionalRequests(t *testing.T) {
	f := newAttachmentFixture(t)
	response := f.ask(t, "head-1", fileRef("a.png", f.put(t, "pixels")))
	path := "/api/v1/asks/" + response.GetAskId() + "/attachments/a.png"
	etag := f.get(t, path).Header.Get("ETag")
	require.NotEmpty(t, etag)

	head := f.do(t, http.MethodHead, path)
	require.Equal(t, http.StatusOK, head.StatusCode)
	require.Equal(t, "6", head.Header.Get("Content-Length"))
	require.Empty(t, body(t, head))

	cached := f.get(t, path, "If-None-Match", etag)
	require.Equal(t, http.StatusNotModified, cached.StatusCode)

	post := f.do(t, http.MethodPost, path)
	require.Equal(t, http.StatusMethodNotAllowed, post.StatusCode)
}

func TestAttachmentContentRoute_AMissingStoredFileIsNotFound(t *testing.T) {
	f := newAttachmentFixture(t)
	// The inventory says stored, the bytes are gone.
	store := blobstore.NewMemoryBlobStore()
	f.service.SetAttachmentStore(hub.StaticBlobStore(store))
	hash := f.put(t, "soon lost")
	response := f.ask(t, "lost-1", fileRef("a.md", hash))
	require.NoError(t, store.Delete(context.Background(), "sha256/"+hash[:2]+"/"+hash))

	resp := f.get(t, "/api/v1/asks/"+response.GetAskId()+"/attachments/a.md")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAttachmentContentEndpointIsDeclaredAsABrowserSurface(t *testing.T) {
	var found bool
	for _, endpoint := range conversationsH.Endpoints {
		if endpoint.ID != "conversations_attachment_content" {
			continue
		}
		found = true
		require.Equal(t, http.MethodGet, endpoint.Method)
		require.Equal(t, "/api/v1/asks/{ask_id}/attachments/{name}", endpoint.Path)
		require.NotNil(t, endpoint.RESTException)
		require.Equal(t, "browser_surface", string(endpoint.RESTException.Reason))
		require.NotEmpty(t, endpoint.RESTException.Note)
	}
	require.True(t, found)
}

// ---- identity ------------------------------------------------------------------

// A verifier-backed fixture: only the bearer token "owner-token" is an owner.
type tokenVerifier struct{}

func (tokenVerifier) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if token != "owner-token" {
		return owneridentity.Identity{}, errors.New("not an owner credential")
	}
	return owneridentity.Identity{Subject: "alice"}, nil
}

// refusingProvider stands for Cloudflare Access presenting an assertion that
// fails verification.
type refusingProvider struct{}

func (refusingProvider) Source() apiidentity.AuthSource { return apiidentity.SourceCloudflareAccess }
func (refusingProvider) VerifyRequest(context.Context, *http.Request) (apiidentity.Principal, error) {
	return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureInvalid, apiidentity.SourceCloudflareAccess)
}

// verifiedHumanProvider stands for a valid Cloudflare Access login.
type verifiedHumanProvider struct{}

func (verifiedHumanProvider) Source() apiidentity.AuthSource {
	return apiidentity.SourceCloudflareAccess
}

func (verifiedHumanProvider) VerifyRequest(_ context.Context, r *http.Request) (apiidentity.Principal, error) {
	if r.Header.Get("Cf-Access-Jwt-Assertion") == "" {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureMissing, apiidentity.SourceCloudflareAccess)
	}
	return apiidentity.Principal{
		Kind: apiidentity.ActorHuman, Subject: "access-user", Email: "owner@example.test", Verified: true,
		Source: apiidentity.SourceCloudflareAccess, Sources: []apiidentity.AuthSource{apiidentity.SourceCloudflareAccess},
	}, nil
}

// identityRoute serves the content route the way main wires it: the module
// behind the owner authenticator with the given providers and verifier, over a
// service that has one ask with one attachment.
func identityRoute(t *testing.T, providers []authn.Provider, verifier identity.Verifier) (server *httptest.Server, path string) {
	t.Helper()
	f := newAttachmentFixture(t)
	response := f.ask(t, "identity-1", fileRef("a.md", f.put(t, "secret contents")))
	router := mux.NewRouter()
	conversationsH.ModuleWithVerifier(f.service, verifier).Mount(router)
	auth := identity.OwnerAuthenticator{Providers: providers, OwnerSubject: f.service.ResolveRecipient}
	server = httptest.NewServer(auth.Middleware(router))
	t.Cleanup(server.Close)
	return server, "/api/v1/asks/" + response.GetAskId() + "/attachments/a.md"
}

func fetch(t *testing.T, server *httptest.Server, path string, headers map[string]string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	require.NoError(t, err)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(data)
}

func TestAttachmentContentRoute_RefusesARequestWithoutAVerifiedOwner(t *testing.T) {
	// No provider, no verifier: a loopback process that presents nothing.
	server, path := identityRoute(t, nil, nil)
	status, content := fetch(t, server, path, nil)
	require.Equal(t, http.StatusUnauthorized, status)
	require.NotContains(t, content, "secret contents")
}

func TestAttachmentContentRoute_AcceptsTheOwnersBearerAndRefusesOthers(t *testing.T) {
	server, path := identityRoute(t, nil, tokenVerifier{})

	status, content := fetch(t, server, path, map[string]string{"Authorization": "Bearer owner-token"})
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "secret contents", content)
	status, _ = fetch(t, server, path, map[string]string{"Authorization": "Bearer someone-else"})
	require.Equal(t, http.StatusUnauthorized, status, "a credential that is not the owner's")
	status, _ = fetch(t, server, path, nil)
	require.Equal(t, http.StatusUnauthorized, status, "no credential")
	// The legacy subject header is honored only when no verifier is wired.
	status, _ = fetch(t, server, path, map[string]string{"X-Vrooli-Identity-Subject": "alice"})
	require.Equal(t, http.StatusUnauthorized, status, "a self-declared subject is not an identity")
}

func TestAttachmentContentRoute_CloudflareAccessLogin(t *testing.T) {
	server, path := identityRoute(t, []authn.Provider{verifiedHumanProvider{}}, tokenVerifier{})

	status, content := fetch(t, server, path, map[string]string{"Cf-Access-Jwt-Assertion": "valid-assertion"})
	require.Equal(t, http.StatusOK, status, "the header Cloudflare injects on every request, including <img> and <video>")
	require.Equal(t, "secret contents", content)
	status, _ = fetch(t, server, path, nil)
	require.Equal(t, http.StatusUnauthorized, status, "a request that never passed Access")
}

func TestAttachmentContentRoute_ARefusedAccessAssertionDoesNotFallBackToAWeakerPath(t *testing.T) {
	server, path := identityRoute(t, []authn.Provider{refusingProvider{}}, tokenVerifier{})
	status, _ := fetch(t, server, path, map[string]string{"Cf-Access-Jwt-Assertion": "expired", "Authorization": "Bearer owner-token"})
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestAttachmentContentRoute_PersonalLocalTrustsOnlyDirectLoopback(t *testing.T) {
	server, path := identityRoute(t, []authn.Provider{identity.LoopbackOwnerProvider{}}, nil)

	status, content := fetch(t, server, path, nil)
	require.Equal(t, http.StatusOK, status, "personal_local: a loopback request is the current OS user, as for Answer")
	require.Equal(t, "secret contents", content)
	status, _ = fetch(t, server, path, map[string]string{"Cf-Connecting-Ip": "203.0.113.9"})
	require.Equal(t, http.StatusUnauthorized, status, "a request that came through the tunnel is not local")
	status, _ = fetch(t, server, path, map[string]string{"Cf-Access-Jwt-Assertion": "anything"})
	require.Equal(t, http.StatusUnauthorized, status)
}
