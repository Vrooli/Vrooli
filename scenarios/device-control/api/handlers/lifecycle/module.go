package lifecycle

import (
	"device-control/internal/module"
	"github.com/gorilla/mux"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func Module(provider *Provider) module.Module {
	return module.Module{Name: "lifecycle-maintenance", Mount: func(r *mux.Router) {
		path, handler := commonconnect.NewLifecycleMaintenanceServiceHandler(provider)
		r.PathPrefix(path).Handler(apilifecycle.LoopbackOnly(handler))
	}}
}
