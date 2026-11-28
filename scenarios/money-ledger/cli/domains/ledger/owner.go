package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Money Ledger accepts a write only from a verified, authorised identity (see
// the API's access policy). Every write command here acts for the owner, so
// the owner says so with --local-owner and the CLI exchanges this machine's own
// credential for an owner session scoped to Money Ledger. It is never an
// automatic fallback, and it is unavailable inside an identified agent run: an
// agent does not write the owner's financial records.

const localOwnerFlag = "local-owner"

const moneyLedgerResource = "money-ledger"

// ownerEligible and ownerExchange are the seams the tests replace.
var (
	ownerEligible = cliutil.ConfirmLocalOwnerEligible
	ownerExchange = func(ctx context.Context) (string, error) {
		login, err := authn.ExchangeLocalMachinePrincipalForResource(ctx, moneyLedgerResource)
		if err != nil {
			return "", err
		}
		return login.GetTokens().GetAccessToken(), nil
	}
)

// call is a generated Connect client method.
type call[Req, Res any] func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error)

// read sends one read. Reads need no identity.
func read[Req, Res any](rpc call[Req, Res], msg *Req) (*Res, error) {
	response, err := rpc(context.Background(), connect.NewRequest(msg))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

// asOwner sends one request, presenting the owner session when the operator
// asked for it with --local-owner, and explains a refusal. Every write command
// goes through here, so they all authenticate and fail the same way.
func asOwner[Req, Res any](c cliapp.OperationContext, rpc call[Req, Res], msg *Req) (*Res, error) {
	request := connect.NewRequest(msg)
	if err := actAsOwner(c, request); err != nil {
		return nil, err
	}
	response, err := rpc(context.Background(), request)
	if err != nil {
		return nil, explainRefusal(err, c.BoolFlag(localOwnerFlag))
	}
	return response.Msg, nil
}

// actAsOwner presents the owner session on a write when the operator asked for
// it with --local-owner.
func actAsOwner[T any](c cliapp.OperationContext, request *connect.Request[T]) error {
	if !c.BoolFlag(localOwnerFlag) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ownerEligible(ctx); err != nil {
		return fmt.Errorf("--%s is unavailable here: %w; run it yourself, outside an agent run", localOwnerFlag, err)
	}
	token, err := ownerExchange(ctx)
	if err == nil && strings.TrimSpace(token) == "" {
		err = errors.New("it returned no session")
	}
	if err != nil {
		return fmt.Errorf("--%s could not get an owner session from the local authenticator: %w", localOwnerFlag, err)
	}
	request.Header().Set("Authorization", "Bearer "+token)
	return nil
}

// explainRefusal tells the operator what Money Ledger's refusal means and what
// to do, keeping the code for a caller that checks it.
func explainRefusal(err error, asOwner bool) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated:
		if asOwner {
			return fmt.Errorf("money-ledger could not verify the owner session from the local authenticator: %w", err)
		}
		return fmt.Errorf("money-ledger accepts a write only from a verified identity; to write it yourself, add --%s (not available inside an agent run): %w", localOwnerFlag, err)
	case connect.CodePermissionDenied:
		if asOwner {
			return fmt.Errorf("money-ledger did not accept this machine's owner account as the owner of these records; the host's owner-binding policy must name it with money-ledger:write (see Money Ledger's docs/internal/SECURITY.md): %w", err)
		}
		return fmt.Errorf("money-ledger refused this identity for this operation: %w", err)
	}
	return err
}
