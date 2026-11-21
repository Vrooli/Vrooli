package diff

import (
	"log"
	"net/http"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	diffconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff/diff_v1connect"
)

// Module returns the diff domain's contribution: the DiffService discovery
// handler (Connect-RPC, ListDiffModes) plus the REST multipart compare edge
// (POST /api/v1/diff/compare). Comparison runs synchronously (pure-Go pixel +
// perceptual metrics) and is recorded as a terminal durable job; the produced
// heat-map is stored via the blob store.
func Module(blobs BlobStore, jobs *internaljobs.Manager, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	rest := &Deps{
		Store:  blobs,
		Jobs:   jobs,
		Guard:  storage.DefaultGuard(),
		Logger: logger,
	}
	connectPath, connectHandler := diffconnect.NewDiffServiceHandler(NewConnectHandler())
	return module.Module{
		Name: "diff",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/diff/compare", rest.compareHandler).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the diff domain owns no tables; recorded jobs live in the
// jobs domain's schema.
func Schema() string { return "" }
