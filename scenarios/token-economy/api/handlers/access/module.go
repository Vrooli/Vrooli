package access

import (
	internalaccess "token-economy/internal/access"
	"token-economy/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
	earningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning/earningv1connect"
)

func Module(mints MintsDelegate, grants GrantsDelegate, holder HolderDelegate, earning EarningDelegate, catalog CatalogDelegate, redemption RedemptionDelegate, journal JournalDelegate, validator internalaccess.Validator) module.Module {
	handler := NewConnectHandler(mints, grants, holder, earning, catalog, redemption, journal)
	option := connect.WithInterceptors(internalaccess.Interceptor(validator))
	minterPath, minterHandler := accessconnect.NewMinterServiceHandler(handler, option)
	holderPath, holderHandler := accessconnect.NewHolderServiceHandler(handler, option)
	earningPath, earningHandler := earningconnect.NewEarningServiceHandler(handler, option)
	return module.Module{
		Name: "access",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router,
				connectx.ServiceMount{Path: minterPath, Handler: minterHandler},
				connectx.ServiceMount{Path: holderPath, Handler: holderHandler},
				connectx.ServiceMount{Path: earningPath, Handler: earningHandler},
			)
		},
		Endpoints: Endpoints,
	}
}
