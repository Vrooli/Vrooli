package impact

import (
	"log"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"proto-health/internal/baselines"
	"proto-health/internal/impact"
	"proto-health/internal/module"
	"proto-health/internal/protosurface"

	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
	impactconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact/impact_v1connect"
)

var ProtoFile = impactv1.File_proto_health_v1_impact_impact_proto

func Module(logger *log.Logger, repoRoot string) module.Module {
	loader, err := protosurface.NewDescriptorLoaderFromFile(
		repoRoot,
		filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"),
	)
	if err != nil {
		logger.Fatalf("impact descriptor loader: %v", err)
	}
	reporter := impact.New(repoRoot, loader)
	reporter.Baselines = baselines.NewClient(nil, nil)
	handler := NewConnectHandler(Deps{
		Logger:   logger,
		Reporter: reporter,
	})
	protoPath, protoHandler := impactconnect.NewImpactServiceHandler(handler)
	return module.Module{
		Name: "impact",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: protoPath, Handler: protoHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "impact_get",
		Path:        impactconnect.ImpactServiceGetImpactProcedure,
		Method:      "POST",
		Summary:     "Report proto contract impact",
		Description: "Compares the current scenario proto surface against a git baseline descriptor with buf breaking and returns classified compatibility findings.",
		Category:    "impact",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required, scenario id)",
				"against":  "string (optional, scope/ref; empty uses newest git-control-tower baseline with merge-base fallback)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"report": "ImpactReport"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id, invalid git ref, missing baseline descriptor, or buf breaking failure"},
		},
		Examples: []module.Example{
			{Name: "Impact default scope", Curl: "curl http://localhost:${API_PORT}/vrooli.proto_health.v1.impact.ImpactService/GetImpact -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
		},
	},
}
