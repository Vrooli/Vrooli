package signals

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	signalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/signals/signals_v1connect"
	"signal-inbox/internal/clock"
	"signal-inbox/internal/enrichment"
	"signal-inbox/internal/module"
	internal "signal-inbox/internal/signals"
)

func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	store, err := defaultBlobStore()
	if err != nil {
		logger.Fatalf("signals BlobStore: %v", err)
	}
	return ModuleWithBlobStore(db, clk, store, logger)
}

func ModuleWithBlobStore(db *database.RoutedDB, clk clock.Clock, store blobstore.BlobStore, logger *log.Logger) module.Module {
	enricher := enrichment.NewService(
		enrichment.NewSQLiteRepository(db),
		clk,
		enrichment.NewHTMLExtractor(&http.Client{Timeout: 15 * time.Second}),
		enrichment.NewImageExtractor(store, enrichment.OSCommandRunner{}),
	)
	svc := internal.NewService(internal.NewSQLiteRepository(db, clk), clk, enricher)
	path, handler := signalsconnect.NewSignalsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	upload := NewImageUploadHandler(store, logger)
	return module.Module{Name: "signals", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		router.Handle("/api/v1/signals/images", upload).Methods("POST")
	}, Endpoints: Endpoints}
}

func defaultBlobStore() (blobstore.BlobStore, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(storage.Options{ScenarioID: "signal-inbox"}, storage.ClassData, "signals")
	if err != nil {
		return nil, fmt.Errorf("resolve signals storage path: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(path), nil
}

func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "signals_capture", Path: signalsconnect.SignalsServiceCaptureSignalProcedure, Method: "POST", Summary: "Capture a signal", Description: "Appends a URL, text, or image reference to the immutable signal journal. The source kind is inferred from the populated payload.", Category: "signals", Request: &module.Schema{Type: "CaptureSignalRequest"}, Response: &module.Schema{Type: "CaptureSignalResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Exactly one capture source is required"}, {Status: 500, Code: "internal", Description: "Journal append failed"}}},
	{ID: "signals_get", Path: signalsconnect.SignalsServiceGetSignalProcedure, Method: "POST", Summary: "Get a signal", Description: "Reads one immutable signal by id.", Category: "signals", Request: &module.Schema{Type: "GetSignalRequest"}, Response: &module.Schema{Type: "GetSignalResponse"}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Signal does not exist"}}},
	{ID: "signals_list", Path: signalsconnect.SignalsServiceListSignalsProcedure, Method: "POST", Summary: "List signals", Description: "Lists the journal newest first. Categories and disposition never filter this corpus read.", Category: "signals", Request: &module.Schema{Type: "ListSignalsRequest"}, Response: &module.Schema{Type: "ListSignalsResponse"}},
	{ID: "signals_upload_image", Path: "/api/v1/signals/images", Method: "POST", Summary: "Upload image for capture", Description: "Stores opaque image bytes in BlobStore and returns a proto-typed payload reference for CaptureSignal.", Category: "signals", Request: &module.Schema{Type: "multipart/form-data"}, Response: &module.Schema{Type: "UploadImageResponse"}, RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "multipart file bytes cannot be represented in a Connect request; response remains proto-typed"}},
}
