package transfer

import (
	"fmt"
	"log"

	"device-sync-hub/internal/clock"
	"device-sync-hub/internal/devices"
	"device-sync-hub/internal/module"
	"device-sync-hub/internal/realtime"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	transferconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer/transfer_v1connect"
)

// Wiring is the transfer domain's contribution plus the live handles main.go
// needs beyond the HTTP module: the Service (the retention purge loop sweeps
// through it) and the BlobStore. Returned together so the wiring is built once.
type Wiring struct {
	Module  module.Module
	Service internaltransfer.Service
	Store   blobstore.BlobStore
}

// New builds the transfer domain: repository, service (retention/quota/ACL +
// blob and realtime side effects), the Connect handler, and the two REST byte
// edges (multipart upload, streaming download). devSvc backs the directed-item
// trust check; hub (optional) backs realtime item events. Production blobs are
// the filesystem store under the storage-steer data class.
func New(db *database.RoutedDB, clk clock.Clock, devSvc devices.Service, hub *realtime.Hub, logger *log.Logger) (Wiring, error) {
	store, err := defaultBlobStore()
	if err != nil {
		return Wiring{}, err
	}
	return NewWithBlobStore(db, clk, devSvc, hub, store, logger), nil
}

// NewWithBlobStore is the explicit-injection variant used by tests (typically
// with blobstore.NewMemoryBlobStore()) and callers that swap the blob backend.
func NewWithBlobStore(db *database.RoutedDB, clk clock.Clock, devSvc devices.Service, hub *realtime.Hub, store blobstore.BlobStore, logger *log.Logger) Wiring {
	repo := internaltransfer.NewSQLiteRepository(db, clk)
	cfg := internaltransfer.Config{
		Repo:  repo,
		Blobs: store,
		Clock: clk,
		Log:   logger,
	}
	if devSvc != nil {
		cfg.Trust = deviceTrustChecker{svc: devSvc}
	}
	if hub != nil {
		cfg.Notif = hubNotifier{hub: hub}
	}
	svc := internaltransfer.NewService(cfg)

	connectPath, connectHandler := transferconnect.NewTransferServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	upload := newUploadHandler(UploadDeps{Service: svc, Store: store, Logger: logger})
	download := newDownloadHandler(DownloadDeps{Service: svc, Store: store})

	mod := module.Module{
		Name: "transfer",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/transfer/items", upload.handleUpload).Methods("POST")
			r.HandleFunc("/api/v1/transfer/items/{id}/content", download.handleDownload).Methods("GET")
		},
		Endpoints: Endpoints,
	}
	return Wiring{Module: mod, Service: svc, Store: store}
}

// defaultBlobStore resolves the storage-steer-mandated transfer blobs directory
// and returns a filesystem-backed blobstore rooted there. Lives in this package
// so transfer's file storage travels with the domain.
func defaultBlobStore() (blobstore.BlobStore, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("device-sync-hub")
	if err != nil {
		return nil, fmt.Errorf("resolve device-sync-hub storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"transfer-blobs",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve transfer blobs path: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(path), nil
}

// Schema re-exports the transfer domain schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaltransfer.Schema() }
