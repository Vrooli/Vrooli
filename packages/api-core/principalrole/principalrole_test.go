package principalrole

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	apiidentity "github.com/vrooli/api-core/identity"
)

var testRole = NewRole("scenario-authenticator:hub-test", "hub-test:use", "hub-test", "-hub-test", "test role")

type verifierFunc func(string) (apiidentity.Principal, error)

func (f verifierFunc) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	return f(token)
}

func principal(subject string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: subject, Verified: true,
		Audience: testRole.Audience, Scopes: []string{testRole.Scope},
	}
}

func bearer(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	return h
}

func TestNamesRoundTripAndAddressTheSourcesOwnCredential(t *testing.T) {
	for _, source := range []string{"treasury", "vrooli-autoheal", "a1"} {
		if got, ok := testRole.SourceOf(testRole.Principal(source)); !ok || got != source {
			t.Fatalf("SourceOf(Principal(%q)) = %q, %v", source, got, ok)
		}
	}
	if got := testRole.Principal(" treasury "); got != "treasury-hub-test" {
		t.Fatalf("Principal must trim the source: %q", got)
	}
	for _, subject := range []string{"", "treasury", "-hub-test", "Treasury-hub-test", "1x-hub-test", "a b-hub-test", "treasury-hub-test-x", "treasury-hub-test\n", "treasuryXhub-test"} {
		if got, ok := testRole.SourceOf(subject); ok {
			t.Fatalf("SourceOf(%q) = %q, want it refused", subject, got)
		}
	}
	want := "scenario-authenticator/service-principal/treasury-hub-test/hub-test"
	if got := testRole.CredentialIdentity("treasury"); got != want {
		t.Fatalf("CredentialIdentity = %q, want %q", got, want)
	}
	if _, ok := (Role{}).SourceOf("x-hub-test"); ok {
		t.Fatal("a zero Role holds no principal")
	}
}

func TestSourceNamesWhatIsMissingAndWrapsErrRefused(t *testing.T) {
	good := principal("treasury-hub-test")
	if got, err := testRole.Source(good); err != nil || got != "treasury" {
		t.Fatalf("Source = %q, %v", got, err)
	}
	cases := map[string]struct {
		mutate func(*apiidentity.Principal)
		want   string
	}{
		"unverified":       {func(p *apiidentity.Principal) { p.Verified = false }, "verified service token"},
		"a human":          {func(p *apiidentity.Principal) { p.Kind = apiidentity.ActorHuman }, "verified service token"},
		"another audience": {func(p *apiidentity.Principal) { p.Audience = "scenario-authenticator:agent-manager" }, `not "scenario-authenticator:hub-test"`},
		"no scope":         {func(p *apiidentity.Principal) { p.Scopes = nil }, `lacks scope "hub-test:use"`},
		"other scope":      {func(p *apiidentity.Principal) { p.Scopes = []string{"hub-test:other"} }, `lacks scope "hub-test:use"`},
		"not a role name":  {func(p *apiidentity.Principal) { p.Subject = "treasury" }, "is not named <source>-hub-test"},
	}
	for name, c := range cases {
		p := good
		c.mutate(&p)
		got, err := testRole.Source(p)
		if got != "" || !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: Source = %q, %v, want ErrRefused naming %q", name, got, err, c.want)
		}
	}
}

func TestVerifiedSourceReadsTheBearerOnly(t *testing.T) {
	verifier := verifierFunc(func(token string) (apiidentity.Principal, error) {
		switch token {
		case "ok":
			return principal("treasury-hub-test"), nil
		case "wrong-role":
			return principal("treasury"), nil
		}
		return apiidentity.Principal{}, errors.New("invalid")
	})
	ctx := context.Background()
	if got, err := testRole.VerifiedSource(ctx, verifier, bearer("ok")); err != nil || got != "treasury" {
		t.Fatalf("VerifiedSource = %q, %v", got, err)
	}
	if _, err := testRole.VerifiedSource(ctx, verifier, bearer("wrong-role")); !errors.Is(err, ErrRefused) {
		t.Fatalf("a verified token without the role is refused with the reason: %v", err)
	}
	// An unverifiable token is not ErrRefused: it proves no service identity.
	if _, err := testRole.VerifiedSource(ctx, verifier, bearer("bad")); err == nil || errors.Is(err, ErrRefused) {
		t.Fatalf("an unverifiable token is a plain failure: %v", err)
	}
	cookie := http.Header{}
	cookie.Set("Cookie", "token=ok")
	for name, h := range map[string]http.Header{
		"no header": {}, "cookie only": cookie, "not bearer": {"Authorization": {"Basic ok"}},
		"two tokens": {"Authorization": {"Bearer ok bad"}}, "empty": {"Authorization": {"Bearer  "}},
	} {
		if got, err := testRole.VerifiedSource(ctx, verifier, h); err == nil || got != "" {
			t.Fatalf("%s: VerifiedSource = %q, %v, want an error", name, got, err)
		}
	}
	if _, err := testRole.VerifiedSource(ctx, nil, bearer("ok")); err == nil {
		t.Fatal("no verifier must fail closed")
	}
}

func TestNewTokensExpectsExactlyTheSourcesPrincipal(t *testing.T) {
	tokens := testRole.NewTokens("treasury", func(context.Context) (string, error) { return "cred", nil })
	if tokens == nil {
		t.Fatal("NewTokens returned nil")
	}
	if _, err := testRole.NewTokens("", func(context.Context) (string, error) { return "cred", nil }).Token(context.Background()); err == nil {
		t.Fatal("an empty source builds a client that refuses")
	}
}
