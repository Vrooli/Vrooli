package analysis

import (
	"log"
	"net/http"

	internalanalysis "image-tools/internal/analysis"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis/analysis_v1connect"
)

// Module returns the analysis domain's contribution: the AnalysisService
// discovery handler (Connect-RPC) plus the REST multipart analyze edge
// (POST /api/v1/analysis/{operation}). Analysis ops run synchronously and are
// recorded as terminal durable jobs.
func Module(service *internalanalysis.Service, jobs *internaljobs.Manager, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	rest := &Deps{
		Service: service,
		Jobs:    jobs,
		Guard:   storage.DefaultGuard(),
		Logger:  logger,
	}
	connectPath, connectHandler := analysisconnect.NewAnalysisServiceHandler(NewConnectHandler())
	return module.Module{
		Name: "analysis",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/analysis/{operation}", rest.analyzeHandler).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the analysis domain owns no tables; recorded jobs live in
// the jobs domain's schema.
func Schema() string { return "" }
