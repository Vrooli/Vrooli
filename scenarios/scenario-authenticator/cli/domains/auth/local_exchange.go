package auth

import (
	"context"

	"github.com/vrooli/api-core/authn"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
)

func exchangeLocal(ctx context.Context) (*accountsv1.LoginResponse, error) {
	return authn.ExchangeLocalMachinePrincipal(ctx)
}

func defaultAuthSocket() string {
	return authn.DefaultLocalAuthenticatorSocket()
}
