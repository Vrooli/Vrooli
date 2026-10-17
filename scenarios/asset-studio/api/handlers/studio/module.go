package studio

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

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
	dispatcher := core.RenderDispatcher(core.ProducerDispatchers{
		core.ProducerImage:   imageToolsDispatcherFromEnvironment(os.LookupEnv),
		core.ProducerRefine:  imageToolsDispatcherFromEnvironment(os.LookupEnv),
		core.ProducerVideo:   gatewayVideoDispatcherFromEnvironment(os.LookupEnv),
		core.ProducerCapture: browserCaptureDispatcherFromEnvironment(os.LookupEnv),
	})
	h := NewConnectHandlerWithDispatcher(state, store, blobs, dispatcher)
	h.SetAdvisoryAnalyzer(imageToolsAdvisoryAnalyzerFromEnvironment(os.LookupEnv))
	h.SetAgentCommissioner(agentManagerCommissionerFromEnvironment(os.LookupEnv))
	h.ReconcileRenders()
	path, handler := studioconnect.NewStudioServiceHandler(h)
	return module.Module{Name: "studio", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func agentManagerCommissionerFromEnvironment(lookup func(string) (string, bool)) AgentCommissioner {
	origin, err := configuredOrigin(lookup, "AGENT_MANAGER_API_URL")
	if err != nil {
		return unavailableCommissioner{}
	}
	return &agentManagerCommissioner{BaseURL: origin}
}

func imageToolsAdvisoryAnalyzerFromEnvironment(lookup func(string) (string, bool)) core.AdvisoryAnalyzer {
	origin, err := configuredOrigin(lookup, "IMAGE_TOOLS_API_URL")
	if err != nil {
		return core.UnavailableAdvisoryAnalyzer{Reason: err.Error()}
	}
	return &core.ImageToolsAdvisoryAnalyzer{BaseURL: origin}
}

func browserCaptureDispatcherFromEnvironment(lookup func(string) (string, bool)) core.RenderDispatcher {
	origin, err := configuredOrigin(lookup, "BROWSER_AUTOMATION_STUDIO_API_URL")
	if err != nil {
		return core.UnavailableRenderDispatcher{Reason: err.Error()}
	}
	return &core.BrowserCaptureDispatcher{BaseURL: origin}
}

func imageToolsDispatcherFromEnvironment(lookup func(string) (string, bool)) core.RenderDispatcher {
	origin, err := configuredOrigin(lookup, "IMAGE_TOOLS_API_URL")
	if err != nil {
		return core.UnavailableRenderDispatcher{Reason: err.Error()}
	}
	return &core.ImageToolsDispatcher{BaseURL: origin}
}

func gatewayVideoDispatcherFromEnvironment(lookup func(string) (string, bool)) core.RenderDispatcher {
	origin, err := configuredOrigin(lookup, "AI_GATEWAY_API_URL")
	if err != nil {
		return core.UnavailableRenderDispatcher{Reason: err.Error()}
	}
	return &core.GatewayVideoDispatcher{BaseURL: origin}
}

func configuredOrigin(lookup func(string) (string, bool), variable string) (string, error) {
	raw, configured := lookup(variable)
	if !configured || strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("%s is not configured", variable)
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%s must be an absolute http(s) origin without credentials, query, or fragment", variable)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
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
