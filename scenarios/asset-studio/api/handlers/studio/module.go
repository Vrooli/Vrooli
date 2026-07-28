package studio

import (
	"context"
	"fmt"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	"asset-studio/internal/module"
	core "asset-studio/internal/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

// Module mounts the P0 Studio Connect surface. Persistence is intentionally
// introduced behind the core domain seam; this adapter has no policy of its own.
func Module(db *database.RoutedDB) module.Module {
	store := core.NewSQLiteStore(db)
	state, err := store.Load(context.Background())
	if err != nil {
		log.Printf("studio state unavailable at startup: %v", err)
		state = core.New()
	}
	blobs, err := defaultBlobStore()
	if err != nil {
		log.Printf("studio blob store unavailable at startup: %v", err)
		blobs = blobstore.NewMemoryBlobStore()
	}
	h := NewConnectHandlerWithStore(state, store, blobs)
	path, handler := studioconnect.NewStudioServiceHandler(h)
	return module.Module{Name: "studio", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return core.Schema() }

func defaultBlobStore() (blobstore.BlobStore, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, err
	}
	path, err := resolver.Path(storage.Options{ScenarioID: "asset-studio"}, storage.ClassData, "assets")
	if err != nil {
		return nil, fmt.Errorf("resolve asset blobs: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(path), nil
}
