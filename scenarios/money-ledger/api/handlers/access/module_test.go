package access_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/browsersession"
	accesspb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"

	"money-ledger/internal/testutil/authfixture"
	"money-ledger/internal/testutil/ledgertest"
)

// The console asks before offering a write; the answer is the gate's own
// decision for the credential the request carries.
func TestWriteAccessReportsTheGatesDecision(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	for _, test := range []struct {
		name    string
		headers map[string]string
		actor   string
		refusal accesspb.WriteRefusal
	}{
		{"local browser with no sign-in", nil, "", accesspb.WriteRefusal_WRITE_REFUSAL_SIGNED_OUT},
		{"forged credential", map[string]string{"Authorization": "Bearer forged"}, "", accesspb.WriteRefusal_WRITE_REFUSAL_UNVERIFIED},
		{"another signed-in person", map[string]string{"Authorization": "Bearer " + l.Identity.Person("someone-else", authfixture.WriteScope)}, "", accesspb.WriteRefusal_WRITE_REFUSAL_NOT_OWNER},
		{"undeclared service", map[string]string{"Authorization": "Bearer " + l.Identity.Service("someone-else", authfixture.WriteScope)}, "", accesspb.WriteRefusal_WRITE_REFUSAL_UNDECLARED_SERVICE},
		{"owner through the tunnel", map[string]string{"Cf-Access-Jwt-Assertion": l.Identity.OwnerAccess()}, "operator", accesspb.WriteRefusal_WRITE_REFUSAL_UNSPECIFIED},
		{"owner session", map[string]string{"Authorization": "Bearer " + l.Identity.Owner()}, "operator", accesspb.WriteRefusal_WRITE_REFUSAL_UNSPECIFIED},
		{"treasury", map[string]string{"Authorization": "Bearer " + l.Identity.Service(authfixture.TreasuryWriter, authfixture.WriteScope)}, "treasury", accesspb.WriteRefusal_WRITE_REFUSAL_UNSPECIFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := connect.NewRequest(&accesspb.GetWriteAccessRequest{})
			for key, value := range test.headers {
				request.Header().Set(key, value)
			}
			out, err := l.Access.GetWriteAccess(ctx, request)
			require.NoError(t, err, "the question itself is an open read")
			require.Equal(t, test.actor != "", out.Msg.CanWrite)
			require.Equal(t, test.actor, out.Msg.Actor)
			require.Equal(t, test.refusal, out.Msg.Refusal)
			require.NotEmpty(t, out.Msg.Reason)
		})
	}
}

func signIn(t *testing.T, browser *ledgertest.Browser, email, password string) *accesspb.SignInResponse {
	t.Helper()
	out, err := browser.Access.SignIn(context.Background(), connect.NewRequest(&accesspb.SignInRequest{Email: email, Password: password}))
	require.NoError(t, err)
	return out.Msg
}

func access(t *testing.T, browser *ledgertest.Browser) *accesspb.GetWriteAccessResponse {
	t.Helper()
	out, err := browser.Access.GetWriteAccess(context.Background(), connect.NewRequest(&accesspb.GetWriteAccessRequest{}))
	require.NoError(t, err)
	return out.Msg
}

func manualEntry(l *ledgertest.Ledger, id string) *connect.Request[ingestpb.IngestEventRequest] {
	return connect.NewRequest(&ingestpb.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		ExternalId: id, AdapterId: "manual", AccountId: l.Account.Id, BookId: l.Book.Id, AmountMinor: 1200, Currency: "USD",
		OccurredAt: timestamppb.Now(), FetchedAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
	}})
}

// The owner signs in from the console's own page and records an entry in
// their name; the session lives in a cookie the page script cannot read.
func TestOwnerSignsInFromTheBrowserAndWrites(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	browser := l.Browser(t)
	_, err := browser.Ingest.IngestEvent(ctx, manualEntry(l, "before-sign-in"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	signedIn := signIn(t, browser, authfixture.OwnerEmail, authfixture.OwnerPassword)
	require.True(t, signedIn.Access.CanWrite)
	require.Equal(t, "operator", signedIn.Access.Actor)
	require.Equal(t, authfixture.OwnerEmail, signedIn.Access.SignedInAs)
	require.True(t, signedIn.Access.BrowserSession)
	for _, cookie := range browser.Jar.Cookies(mustURL(t, l.URL)) {
		require.NotEmpty(t, cookie.Value, cookie.Name)
	}

	_, err = browser.Ingest.RegisterAdapter(ctx, connect.NewRequest(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: "manual", Name: "Manual entry", Kind: ingestpb.AdapterKind_ADAPTER_KIND_MANUAL, Enabled: true}}))
	require.NoError(t, err)
	posted, err := browser.Ingest.IngestEvent(ctx, manualEntry(l, "after-sign-in"))
	require.NoError(t, err)
	require.Equal(t, "operator", posted.Msg.Posting.Actor)

	signedOut, err := browser.Access.SignOut(ctx, connect.NewRequest(&accesspb.SignOutRequest{}))
	require.NoError(t, err)
	require.True(t, signedOut.Msg.RevocationConfirmed)
	after := access(t, browser)
	require.False(t, after.CanWrite)
	require.False(t, after.BrowserSession)
	_, err = browser.Ingest.IngestEvent(ctx, manualEntry(l, "after-sign-out"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Equal(t, 1, l.Count(t, "postings"))
}

// Signing in proves who someone is, not that they own the records: another
// account signs in and is still refused, and the page can say who it is.
func TestAnotherPersonSigningInIsStillNotTheOwner(t *testing.T) {
	l := ledgertest.New(t)
	browser := l.Browser(t)
	signedIn := signIn(t, browser, authfixture.OtherEmail, authfixture.OtherPassword)
	require.False(t, signedIn.Access.CanWrite)
	require.Equal(t, accesspb.WriteRefusal_WRITE_REFUSAL_NOT_OWNER, signedIn.Access.Refusal)
	require.Equal(t, authfixture.OtherEmail, signedIn.Access.SignedInAs)
	_, err := browser.Ingest.IngestEvent(context.Background(), manualEntry(l, "other"))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Zero(t, l.Count(t, "postings"))
}

func TestRefusedSignInSetsNoSession(t *testing.T) {
	l := ledgertest.New(t)
	browser := l.Browser(t)
	_, err := browser.Access.SignIn(context.Background(), connect.NewRequest(&accesspb.SignInRequest{Email: authfixture.OwnerEmail, Password: "wrong"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = browser.Access.SignIn(context.Background(), connect.NewRequest(&accesspb.SignInRequest{Email: authfixture.OwnerEmail}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Empty(t, browser.Jar.Cookies(mustURL(t, l.URL)))
	require.Equal(t, accesspb.WriteRefusal_WRITE_REFUSAL_SIGNED_OUT, access(t, browser).Refusal)
}

// A page opened after the access cookie lapsed is renewed from the refresh
// cookie when it asks, and a dead refresh token is cleared, not retried.
func TestExpiredBrowserSessionIsRenewedWhenThePageAsks(t *testing.T) {
	l := ledgertest.New(t)
	browser := l.Browser(t)
	signIn(t, browser, authfixture.OwnerEmail, authfixture.OwnerPassword)
	site := mustURL(t, l.URL)
	browser.Jar.SetCookies(site, []*http.Cookie{{Name: browsersession.AccessCookie, Value: "", Path: "/", MaxAge: -1}})
	renewed := access(t, browser)
	require.True(t, renewed.CanWrite, renewed.Reason)
	require.True(t, renewed.BrowserSession)

	browser.Jar.SetCookies(site, []*http.Cookie{
		{Name: browsersession.AccessCookie, Value: "", Path: "/", MaxAge: -1},
		{Name: browsersession.RefreshCookie, Value: "revoked", Path: "/"},
	})
	dead := access(t, browser)
	require.False(t, dead.CanWrite)
	require.False(t, dead.BrowserSession, "a refused renewal clears the session")
	require.Empty(t, browser.Jar.Cookies(site))
}

// Another website's script cannot read the ledger or ride the owner's
// session, whatever the UI server's CORS policy allows.
func TestCrossSiteBrowserRequestsAreRefused(t *testing.T) {
	l := ledgertest.New(t)
	for _, site := range []string{"cross-site", "same-site"} {
		for _, path := range []string{"/vrooli.money_ledger.v1.ledger.BooksService/ListBooks", "/vrooli.money_ledger.v1.access.AccessService/SignIn", "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent"} {
			request, err := http.NewRequest(http.MethodPost, l.URL+path, strings.NewReader("{}"))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Sec-Fetch-Site", site)
			request.Header.Set("Cf-Access-Jwt-Assertion", l.Identity.OwnerAccess())
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			response.Body.Close()
			require.Equal(t, http.StatusForbidden, response.StatusCode, "%s %s", site, path)
		}
	}
	require.Zero(t, l.Count(t, "postings"))
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}
