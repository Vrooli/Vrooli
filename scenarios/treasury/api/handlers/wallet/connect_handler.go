package wallet

import (
	"context"
	"errors"
	"net/http"

	"github.com/vrooli/api-core/receiptsigning"
	"treasury/internal/baseusdc"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
	domain "treasury/internal/wallet"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	walletconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet/wallet_v1connect"
)

type Wallets interface {
	Create(context.Context, domain.CreateInput) (domain.Wallet, error)
	Rotate(context.Context, string, string, string, string) (domain.Wallet, error)
	Get(context.Context, string, string, string) (domain.Wallet, error)
	Prove(context.Context, string, string, string, string) (domain.Proof, error)
	QualifyRecovery(context.Context, string, string, string) (domain.Wallet, error)
	Recover(context.Context, string, string, string) (domain.Wallet, error)
}

type PortableRecovery interface {
	PersistPortableRecoveryBundle(context.Context, string, string, string, string) (domain.PortableRecoveryArtifact, error)
	RestorePortableRecoveryBundle(context.Context, string, string, string, []byte, string) (domain.Wallet, error)
	RestorePortableRecoveryWithWitness(context.Context, string, string, string, []byte, []byte, string, ...[]byte) (domain.Wallet, error)
	RestorePersistedPortableRecoveryBundle(context.Context, string, string, string, string) (domain.Wallet, error)
}

type RotationPhases interface {
	PreparePortableRotation(context.Context, string, string, string, string) (domain.RotationPreparation, error)
	CommitPortableRotationWithAcknowledgement(context.Context, string, string, string, string, string, domain.RotationPublicationReceipt) (domain.Wallet, receiptsigning.SignatureEnvelope, error)
}

type ReceiptChecker interface {
	Verify(context.Context, string, domain.Wallet, string) (baseusdc.Check, error)
	Recheck(context.Context, string, domain.Wallet, string, string) (baseusdc.Check, error)
	List(context.Context, string, string, string, int) ([]baseusdc.Check, error)
}

type connectHandler struct {
	authorizer       operatorauth.Authorizer
	wallets          Wallets
	portableRecovery PortableRecovery
	rotationPhases   RotationPhases
	baseUSDC         ReceiptChecker
}

func NewConnectHandler(authorizer operatorauth.Authorizer, wallets Wallets, recovery PortableRecovery, receipts ReceiptChecker) walletconnect.WalletAdminServiceHandler {
	phases, _ := wallets.(RotationPhases)
	return &connectHandler{authorizer: authorizer, wallets: wallets, portableRecovery: recovery, rotationPhases: phases, baseUSDC: receipts}
}

func (h *connectHandler) operatorIdentity(ctx context.Context, request interface{ Header() http.Header }) (operatorauth.Identity, error) {
	if h.authorizer == nil {
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
	switch {
	case err == nil:
		return identity, nil
	case errors.Is(err, operatorauth.ErrRequired):
		return operatorauth.Identity{}, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operatorauth.ErrDenied):
		return operatorauth.Identity{}, connect.NewError(connect.CodePermissionDenied, err)
	default:
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
}

func Module(authorizer operatorauth.Authorizer, wallets Wallets, recovery PortableRecovery, receipts ReceiptChecker) module.Module {
	path, handler := walletconnect.NewWalletAdminServiceHandler(NewConnectHandler(authorizer, wallets, recovery, receipts))
	return module.Module{Name: "walletadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
