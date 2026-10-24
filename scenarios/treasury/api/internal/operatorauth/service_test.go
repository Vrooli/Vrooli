package operatorauth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"treasury/internal/operatorauth"

	"github.com/stretchr/testify/require"
)

func TestStaticTokenSeparatesAgentAndOperatorRealms(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)

	tests := []struct {
		name    string
		headers http.Header
		wantErr error
	}{
		{name: "missing", headers: http.Header{}, wantErr: operatorauth.ErrRequired},
		{name: "wrong", headers: http.Header{operatorauth.HeaderOperatorToken: []string{"wrong"}}, wantErr: operatorauth.ErrDenied},
		{name: "agent only", headers: http.Header{operatorauth.HeaderAgentToken: []string{"valid-agent-token"}}, wantErr: operatorauth.ErrDenied},
		{name: "mixed realm", headers: http.Header{operatorauth.HeaderAgentToken: []string{"valid-agent-token"}, operatorauth.HeaderOperatorToken: []string{"operator-secret"}}, wantErr: operatorauth.ErrDenied},
		{name: "operator", headers: http.Header{operatorauth.HeaderOperatorToken: []string{"operator-secret"}}},
		{name: "operator bearer", headers: http.Header{"Authorization": []string{"Bearer operator-secret"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := authorizer.Authorize(context.Background(), tt.headers)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, identity.Subject)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "local-operator", identity.Subject)
		})
	}
}

func TestStaticTokenFailsClosedWithoutConfiguration(t *testing.T) {
	_, err := operatorauth.NewStaticToken(" ")
	require.ErrorIs(t, err, operatorauth.ErrUnavailable)

	_, err = (operatorauth.Unavailable{Cause: errors.New("not configured")}).Authorize(context.Background(), http.Header{operatorauth.HeaderOperatorToken: []string{"anything"}})
	require.ErrorIs(t, err, operatorauth.ErrUnavailable)
}
