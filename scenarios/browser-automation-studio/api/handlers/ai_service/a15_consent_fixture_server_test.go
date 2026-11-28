package ai_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/config"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	entitlementhandlers "github.com/vrooli/browser-automation-studio/handlers/entitlement"
	"github.com/vrooli/browser-automation-studio/handlers/vision_navigation"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/vision"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// TestA15ConsentCallerFixtureServer exposes the real BAS AIService consent
// caller for an external cross-module integration test. The fixture is opt-in
// because it intentionally blocks until its supervising process stops it.
func TestA15ConsentCallerFixtureServer(t *testing.T) {
	if os.Getenv("A15_CONSENT_CALLER_SERVER") != "1" {
		t.Skip("set A15_CONSENT_CALLER_SERVER=1 to run the BAS consent caller fixture server")
	}

	gatewayURL := strings.TrimSpace(os.Getenv("AI_GATEWAY_URL"))
	if gatewayURL == "" {
		t.Fatal("AI_GATEWAY_URL is required when A15_CONSENT_CALLER_SERVER=1")
	}
	if !strings.HasPrefix(gatewayURL, "http://") && !strings.HasPrefix(gatewayURL, "https://") {
		t.Fatalf("AI_GATEWAY_URL must be an http(s) URL, got %q", gatewayURL)
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	automation := &a15FixtureAutomationRunner{}
	screenshot := aihandlers.NewScreenshotHandler(log, aihandlers.WithScreenshotRunner(automation))
	dom := aihandlers.NewDOMHandler(log, aihandlers.WithDOMRunner(automation))
	elementAnalysis := aihandlers.NewElementAnalysisHandler(log, aihandlers.WithElementRunner(automation))
	modelClient := aihandlers.NewGatewayRolePromptClientWithResolver(
		func(context.Context) (string, error) { return gatewayURL, nil },
		&http.Client{Timeout: 90 * time.Second},
	)
	aiAnalysis := aihandlers.NewAIAnalysisHandler(log, dom,
		aihandlers.WithDOMExtractor(consentFixtureDOM{}),
		aihandlers.WithAIAnalysisModelClient(modelClient),
	)
	mount := Module(Deps{
		Screenshot:      screenshot,
		DOM:             dom,
		ElementAnalysis: elementAnalysis,
		AIAnalysis:      aiAnalysis,
		LinkPreview:     func(context.Context, string) (*LinkPreviewData, bool, error) { return nil, false, nil },
		Logger:          log,
	})

	router := chi.NewRouter()
	var closeVisionFixture func()
	if os.Getenv("A15_CONSENT_VISION") == "1" {
		var mountErr error
		closeVisionFixture, mountErr = mountA15VisionFixture(router, log)
		if mountErr != nil {
			t.Fatalf("mount BAS vision sidecar fixture: %v", mountErr)
		}
		defer closeVisionFixture()
	}
	connectx.RegisterChi(router, mount)
	var handler http.Handler = router
	handler = aihandlers.PrivacyConsentContextMiddleware(handler)
	handler = middleware.NewBYOKMiddleware().InjectBYOKKey(handler)
	if os.Getenv("A15_CONSENT_ENTITLEMENT") == "1" {
		lpbsURL := strings.TrimRight(strings.TrimSpace(os.Getenv("A15_CONSENT_LPBS_URL")), "/")
		if lpbsURL == "" || !strings.HasPrefix(lpbsURL, "http://127.0.0.1:") {
			t.Fatal("A15_CONSENT_LPBS_URL must name the task-owned loopback LPBS fixture")
		}
		credentials, credentialErr := (&testutil.CredentialStore{}).Authority()
		if credentialErr != nil {
			t.Fatalf("create fixture credential authority: %v", credentialErr)
		}
		credentialClient, clientErr := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: credentials})
		if clientErr != nil {
			t.Fatalf("create fixture credential client: %v", clientErr)
		}
		service := entitlement.NewService(config.EntitlementConfig{ServiceURL: lpbsURL, RequestTimeout: 5 * time.Second, CacheTTL: 0}, log)
		service.SetSessionResolver(&credentialclient.ConsumerSessionResolver{
			Credentials: credentialClient, LPBSBaseURL: lpbsURL, HTTPClient: &http.Client{Timeout: 5 * time.Second},
		})
		connectx.RegisterChi(router, entitlementhandlers.Module(entitlementhandlers.Deps{Provider: service, Logger: log}))
		handler = middleware.NewEntitlementMiddleware(service, log, attackJourneyIdentitySettings{}).InjectEntitlement(handler)
	} else {
		handler = monetization.InjectEntitlement(handler)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for BAS fixture: %v", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())
	fmt.Fprintf(os.Stdout, "A15_BAS_FIXTURE_URL=%s\n", baseURL)

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("BAS fixture server stopped unexpectedly: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		t.Fatalf("shutdown BAS fixture server: %v", err)
	}
}

// mountA15VisionFixture mounts the production BAS VisionNavigation handler,
// PlaywrightVisionNavigator, and session broker around an explicitly selected
// loopback sidecar. The fixture creates one fresh browser session on demand.
func mountA15VisionFixture(router *chi.Mux, log *logrus.Logger) (func(), error) {
	driverURL := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL"))
	if driverURL == "" || !strings.HasPrefix(driverURL, "http://127.0.0.1:") {
		return nil, fmt.Errorf("PLAYWRIGHT_DRIVER_URL must name an isolated loopback sidecar")
	}
	client, err := driver.NewClientWithURL(driverURL, driver.WithoutCircuitBreaker(), driver.WithLogger(log))
	if err != nil {
		return nil, fmt.Errorf("create isolated Playwright driver client: %w", err)
	}
	manager := session.NewManagerWithClient(client, session.WithLogger(log))
	if err := os.Setenv("PLAYWRIGHT_DRIVER_URL", driverURL); err != nil {
		return nil, err
	}
	navigator := vision.NewPlaywrightVisionNavigator(log, vision.WithPlaywrightSessionBroker(manager))
	registry := vision.NewNavigatorRegistry()
	registry.Register(navigator)
	store := &testutil.CredentialStore{}
	credentialAuthority, err := store.Authority()
	if err != nil {
		return nil, err
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/openrouter")
	if err != nil {
		return nil, err
	}
	if err := credentialAuthority.Put(identity, "api-key", "fixture-only-openrouter-key"); err != nil {
		return nil, err
	}
	mount := vision_navigation.Module(vision_navigation.Deps{Logger: log, Registry: registry, Tracker: navigator, CredentialAuthority: credentialAuthority})
	connectx.RegisterChi(router, mount)
	callback := aihandlers.NewVisionNavigationHandler(log, aihandlers.WithPlaywrightNavigator(navigator))
	router.Post("/api/v1/internal/ai-navigate/callback", callback.HandleAINavigateCallback)

	var sessionID string
	router.Get("/__fixture/bas/session", func(w http.ResponseWriter, r *http.Request) {
		if sessionID == "" {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			created, createErr := manager.Create(ctx, session.Spec{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Mode: session.ModeExecution, ReuseMode: "fresh"})
			if createErr != nil {
				http.Error(w, "create fixture session: "+createErr.Error(), http.StatusBadGateway)
				return
			}
			if _, navigateErr := created.Navigate(ctx, "http://"+r.Host+"/__fixture/page", session.WithWaitUntil("domcontentloaded")); navigateErr != nil {
				http.Error(w, "navigate fixture session: "+navigateErr.Error(), http.StatusBadGateway)
				return
			}
			sessionID = created.ID()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID})
	})
	router.Get("/__fixture/page", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html><title>Fixture</title><main><h1>Local fixture page</h1><p>Nothing external is reachable from this journey.</p></main>`)
	})

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		manager.CloseAll(ctx)
	}, nil
}

// The runner keeps non-AI methods on the real module mountable without
// requiring a browser process. AI analysis uses the real production DOM
// handler and AIElementAnalyzer path.
type a15FixtureAutomationRunner struct{}

func (*a15FixtureAutomationRunner) Run(context.Context, int, int, []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	return []autocontracts.StepOutcome{{Success: true, StepIndex: 0, NodeID: "a15-fixture", StepType: "navigate"}}, nil, nil
}
