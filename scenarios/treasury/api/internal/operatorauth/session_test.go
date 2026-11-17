package operatorauth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/browsersession"
	coreidentity "github.com/vrooli/api-core/identity"
	"treasury/internal/operatorauth"
)

type sessionVerifier struct {
	principal coreidentity.Principal
	err       error
}

func (v sessionVerifier) Verify(context.Context, string) (coreidentity.Principal, error) {
	return v.principal, v.err
}

func TestSessionOrStaticMapsOnlyVerifiedHumanAndNeverFallsBack(t *testing.T) {
	static, err := operatorauth.NewStaticToken("legacy-static")
	require.NoError(t, err)
	human := sessionVerifier{principal: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "person-42", Verified: true}}
	authorizer := operatorauth.SessionOrStatic{Static: static, Verifier: human, HumanSubjects: []string{"person-42"}, OperatorSubject: "local-operator"}
	headers := http.Header{}
	headers.Set("Cookie", browsersession.AccessCookie+"=signed-human-session")
	got, err := authorizer.Authorize(context.Background(), headers)
	require.NoError(t, err)
	require.Equal(t, "local-operator", got.Subject)

	for name, verifier := range map[string]sessionVerifier{
		"agent":         {principal: coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: "agent", Verified: true}},
		"unverified":    {principal: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "person-42"}},
		"bad signature": {err: context.Canceled},
	} {
		t.Run(name, func(t *testing.T) {
			bad := operatorauth.SessionOrStatic{Static: static, Verifier: verifier}
			got, err := bad.Authorize(context.Background(), headers)
			require.ErrorIs(t, err, operatorauth.ErrDenied)
			require.Empty(t, got.Subject, "an invalid cookie must not fall back to an ambient static token")
		})
	}
	unmapped := operatorauth.SessionOrStatic{
		Static: static, Verifier: sessionVerifier{principal: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "another-person", Verified: true}},
		HumanSubjects: []string{"person-42"}, OperatorSubject: "local-operator",
	}
	_, err = unmapped.Authorize(context.Background(), headers)
	require.ErrorIs(t, err, operatorauth.ErrDenied, "a verified but unmapped human must not inherit the operator realm")

	legacyHeaders := http.Header{operatorauth.HeaderOperatorToken: []string{"legacy-static"}}
	legacy, err := authorizer.Authorize(context.Background(), legacyHeaders)
	require.NoError(t, err)
	require.Equal(t, "local-operator", legacy.Subject)
}

func TestSessionOrStaticRejectsMixedRealmHeaders(t *testing.T) {
	authorizer := operatorauth.SessionOrStatic{Verifier: sessionVerifier{principal: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "person", Verified: true}}}
	headers := http.Header{"Cookie": []string{browsersession.AccessCookie + "=signed"}}
	headers.Set(operatorauth.HeaderAgentToken, "agent-token")
	_, err := authorizer.Authorize(context.Background(), headers)
	require.ErrorIs(t, err, operatorauth.ErrDenied)
}
