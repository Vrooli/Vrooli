package campaign

import (
	"architecture-cartographer/internal/campaign"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign/campaign_v1connect"
)

// Module returns the campaign domain's contribution to the API router.
func Module(svc campaign.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := campaign_v1connect.NewCampaignServiceHandler(h)
	return module.Module{
		Name: "campaign",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the campaign domain's SQL contribution.
func Schema() string { return campaign.Schema() }
