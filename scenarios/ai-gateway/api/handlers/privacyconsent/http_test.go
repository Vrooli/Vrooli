package privacyconsent

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	store "ai-gateway/internal/privacyconsent"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

func consentHTTPFixture(t *testing.T) (*store.Store, http.Handler) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:consent-http?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	authority, err := store.New(db, func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) })
	require.NoError(t, err)
	return authority, New(authority)
}

func TestConsentHTTPRequiresVerifiedHumanAndIgnoresCallerIdentity(t *testing.T) {
	_, handler := consentHTTPFixture(t)
	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodPost, "/api/v1/ai/privacy-consent", strings.NewReader(`{"source":"own-key"}`)))
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)

	serviceContext := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "web-console", Verified: true})
	serviceRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ai/privacy-consent", strings.NewReader(`{"source":"own-key"}`)).WithContext(serviceContext)
	serviceRecorder := httptest.NewRecorder()
	handler.ServeHTTP(serviceRecorder, serviceRequest)
	require.Equal(t, http.StatusUnauthorized, serviceRecorder.Code)

	spoof := httptest.NewRecorder()
	spoofRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ai/privacy-consent", strings.NewReader(`{"source":"own-key","user_subject":"victim"}`)).WithContext(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "verified-user", Verified: true}))
	handler.ServeHTTP(spoof, spoofRequest)
	require.Equal(t, http.StatusBadRequest, spoof.Code, "unexpected identity fields are rejected")
}

func TestConsentHTTPIssuesOnceListsAndRevokesOnlyOwnerGrant(t *testing.T) {
	authority, handler := consentHTTPFixture(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "alice", Verified: true})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/privacy-consent", bytes.NewBufferString(`{"source":"own-key","lifetime_days":2}`)).WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)
	var body struct {
		Grant   store.Grant `json:"grant"`
		Receipt string      `json:"receipt"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.NotEmpty(t, body.Receipt)
	require.NoError(t, authority.Verify(ctx, body.Receipt, "own-key"))
	other := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "bob", Verified: true})
	require.ErrorIs(t, authority.Verify(other, body.Receipt, "own-key"), store.ErrInvalidGrant)

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/ai/privacy-consent", nil).WithContext(ctx))
	require.Equal(t, http.StatusOK, list.Code)
	require.Contains(t, list.Body.String(), body.Grant.ID)
	require.NotContains(t, list.Body.String(), body.Receipt, "list never re-reveals a bearer receipt")

	wrongOwner := httptest.NewRecorder()
	handler.ServeHTTP(wrongOwner, httptest.NewRequest(http.MethodDelete, "/api/v1/ai/privacy-consent/"+body.Grant.ID, nil).WithContext(other))
	require.Equal(t, http.StatusNotFound, wrongOwner.Code)
	require.NoError(t, authority.Verify(ctx, body.Receipt, "own-key"))

	revoke := httptest.NewRecorder()
	handler.ServeHTTP(revoke, httptest.NewRequest(http.MethodDelete, "/api/v1/ai/privacy-consent/"+body.Grant.ID, nil).WithContext(ctx))
	require.Equal(t, http.StatusNoContent, revoke.Code)
	require.ErrorIs(t, authority.Verify(ctx, body.Receipt, "own-key"), store.ErrInvalidGrant)
}

func TestConsentHTTPVerifyRequiresExactOwnerAndSourceWithoutEchoingReceipt(t *testing.T) {
	authority, handler := consentHTTPFixture(t)
	alice := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "alice", Verified: true})
	receipt, grant, err := authority.Grant(alice, "alice", "audio-tools-openai-gpt-live", time.Hour)
	require.NoError(t, err)
	verify := func(ctx context.Context, r, source string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"receipt": r, "source": source})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/ai/privacy-consent/verify", bytes.NewReader(body)).WithContext(ctx))
		return w
	}
	w := verify(alice, receipt, "audio-tools-openai-gpt-live")
	require.Equal(t, http.StatusOK, w.Code)
	require.NotContains(t, w.Body.String(), receipt)
	bob := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "bob", Verified: true})
	require.Equal(t, http.StatusForbidden, verify(bob, receipt, "audio-tools-openai-gpt-live").Code)
	require.Equal(t, http.StatusForbidden, verify(alice, receipt, "own-key").Code)
	require.NoError(t, authority.Revoke(alice, "alice", grant.ID))
	require.Equal(t, http.StatusForbidden, verify(alice, receipt, "audio-tools-openai-gpt-live").Code)
}
