// Package consumerdeclarations exposes BAS's generic consumer declaration validator.
package consumerdeclarations

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	consumerdeclarationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations/consumer_declarationsconnect"
)

type Deps struct{ Logger *logrus.Logger }

func Module(deps Deps) connectx.ServiceMount {
	if deps.Logger == nil {
		panic("consumerdeclarations.Module requires Logger")
	}
	path, handler := consumerdeclarationsconnect.NewConsumerDeclarationsServiceHandler(&service{})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ consumerdeclarationsconnect.ConsumerDeclarationsServiceHandler = (*service)(nil)
