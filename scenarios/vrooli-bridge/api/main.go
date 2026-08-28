package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	internalaudit "vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/channelsign"
	"vrooli-bridge/internal/cpkeys"
	"vrooli-bridge/internal/cprev"
	internalcredentialgrant "vrooli-bridge/internal/credentialgrant"
	"vrooli-bridge/internal/hostbroker"
	internalmachines "vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/modules"
	"vrooli-bridge/internal/nodeauth"
	internalonboard "vrooli-bridge/internal/onboard"
	onboardssh "vrooli-bridge/internal/onboard/ssh"
	internalonboarding "vrooli-bridge/internal/onboarding"
	internaloperatorsession "vrooli-bridge/internal/operatorsession"
	internalpairing "vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	internalprovision "vrooli-bridge/internal/provision"
	internalqueue "vrooli-bridge/internal/queue"
	internalreadiness "vrooli-bridge/internal/readiness"
	internalregistry "vrooli-bridge/internal/registry"
	internalrelay "vrooli-bridge/internal/relay"
	internalruns "vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/server"
	internalsession "vrooli-bridge/internal/session"

	"github.com/vrooli/api-core/schedule"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/scopecatalog"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/api-core/trustposture"
	mdns "github.com/vrooli/mdns-go"
	repocontract "github.com/vrooli/repo-contract-go"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	_ "modernc.org/sqlite"
	"vrooli-bridge/pairingwords"

	artifactsH "vrooli-bridge/handlers/artifacts"
	attachedH "vrooli-bridge/handlers/attached"
	auditH "vrooli-bridge/handlers/audit"
	channelH "vrooli-bridge/handlers/channel"
	cleanupH "vrooli-bridge/handlers/cleanup"
	credentialgrantH "vrooli-bridge/handlers/credentialgrant"
	dispatchH "vrooli-bridge/handlers/dispatch"
	fleetH "vrooli-bridge/handlers/fleet"
	gateH "vrooli-bridge/handlers/gate"
	healthH "vrooli-bridge/handlers/health"
	identityH "vrooli-bridge/handlers/identity"
	machinesH "vrooli-bridge/handlers/machines"
	onboardH "vrooli-bridge/handlers/onboard"
	pairingH "vrooli-bridge/handlers/pairing"
	provisionH "vrooli-bridge/handlers/provision"
	queueH "vrooli-bridge/handlers/queue"
	readinessH "vrooli-bridge/handlers/readiness"
	registryH "vrooli-bridge/handlers/registry"
	relayH "vrooli-bridge/handlers/relay"
	runsH "vrooli-bridge/handlers/runs"
)

// registrarAdapter bridges the registry service to the pairing domain's
// NodeRegistrar seam so pairing can create durable node records on redeem/
// approve without importing registry's proto-facing handler.
type registrarAdapter struct {
	svc internalregistry.Service
}

type registryNodeKindResolver struct{ svc internalregistry.Service }

func (r registryNodeKindResolver) NodeKind(ctx context.Context, nodeID string) (string, error) {
	node, err := r.svc.Get(ctx, nodeID)
	if err != nil {
		return "", err
	}
	return node.Kind, nil
}

func (a registrarAdapter) RegisterNode(ctx context.Context, facts internalpairing.NodeFacts) (string, error) {
	return a.registerNode(ctx, facts, "")
}

func (a registrarAdapter) RegisterNodeWithCorrelation(ctx context.Context, facts internalpairing.NodeFacts, correlationID string) (string, error) {
	return a.registerNode(ctx, facts, correlationID)
}

func (a registrarAdapter) UpdateNodeScopes(ctx context.Context, nodeID string, scopes []string) error {
	node, err := a.svc.Get(ctx, nodeID)
	if err != nil {
		return err
	}
	_, err = a.svc.Update(ctx, internalregistry.UpdateInput{
		ID:           node.ID,
		Name:         node.Name,
		Kind:         node.Kind,
		Endpoint:     node.Endpoint,
		Capabilities: node.Capabilities,
		Scopes:       scopes,
		Revision:     node.Revision,
	})
	return err
}

func (a registrarAdapter) registerNode(ctx context.Context, facts internalpairing.NodeFacts, correlationID string) (string, error) {
	node, err := a.svc.Register(ctx, internalregistry.RegisterInput{
		Name:                 facts.Name,
		OS:                   facts.OS,
		Arch:                 facts.Arch,
		Endpoint:             facts.Endpoint,
		Capabilities:         facts.Capabilities,
		Scopes:               facts.Scopes,
		PairingCorrelationID: correlationID,
	})
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

func (a registrarAdapter) FindNodeByPairingCorrelation(ctx context.Context, correlationID string) (string, error) {
	node, err := a.svc.GetByPairingCorrelation(ctx, correlationID)
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

// cpKeyDir resolves the directory the control plane's long-lived Ed25519
// identity key is persisted in (internal/cpkeys). It mirrors sqliteDSN's
// resolution so the key lands in the same variant-aware namespace as the DB
// (shadow-safe). A BRIDGE_CP_KEY_DIR env override wins for tests/ops.
func cpKeyDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("BRIDGE_CP_KEY_DIR")); dir != "" {
		return dir, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-bridge")
	if err != nil {
		return "", fmt.Errorf("resolve vrooli-bridge storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		filepath.Join("control-plane-keys", ".keep"),
	)
	if err != nil {
		return "", fmt.Errorf("resolve control-plane key dir: %w", err)
	}
	return filepath.Dir(path), nil
}

// bootstrapScriptPath resolves the local path to the node bootstrap script the
// onboard orchestrator copies to each host. Resolution order: the explicit
// BRIDGE_BOOTSTRAP_SCRIPT override, scenario/repository roots supplied by the
// lifecycle, the executable location, and finally the working directory. The
// lifecycle deliberately starts API steps from the API module directory, so a
// working-directory-only lookup is not portable across supported launchers.
// A missing script is a hard configuration error surfaced at boot.
func bootstrapScriptPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("BRIDGE_BOOTSTRAP_SCRIPT")); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			abs, err := filepath.Abs(p)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
		return "", fmt.Errorf("configured BRIDGE_BOOTSTRAP_SCRIPT is not a file: %s", p)
	}
	workDir, _ := os.Getwd()
	exe, _ := os.Executable()
	candidates := bootstrapScriptCandidates(
		workDir,
		strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR")),
		strings.TrimSpace(os.Getenv("SCENARIO_PATH")),
		strings.TrimSpace(os.Getenv("VROOLI_ROOT")),
		exe,
	)
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			abs, err := filepath.Abs(c)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
	}
	return "", fmt.Errorf("node bootstrap script not found (set BRIDGE_BOOTSTRAP_SCRIPT); looked in %v", candidates)
}

// bootstrapScriptCandidates keeps path discovery deterministic and testable.
// Environment-provided roots are preferred over process cwd because lifecycle
// and service managers may launch the same binary from different directories.
func bootstrapScriptCandidates(workDir, scenarioDir, scenarioPath, repoRoot, executable string) []string {
	var candidates []string
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		candidates = append(candidates, filepath.Clean(path))
	}
	for _, dir := range []string{scenarioDir, scenarioPath} {
		add(filepath.Join(dir, "bootstrap", "bootstrap.sh"))
	}
	add(filepath.Join(repoRoot, "scenarios", "vrooli-bridge", "bootstrap", "bootstrap.sh"))
	if strings.TrimSpace(executable) != "" {
		exeDir := filepath.Dir(executable)
		add(filepath.Join(exeDir, "..", "bootstrap", "bootstrap.sh"))
		add(filepath.Join(exeDir, "bootstrap", "bootstrap.sh"))
	}
	add(filepath.Join(workDir, "bootstrap", "bootstrap.sh"))
	add(filepath.Join(workDir, "scenarios", "vrooli-bridge", "bootstrap", "bootstrap.sh"))
	return candidates
}

// deriveControlPlaneURL builds the dial-back URL nodes use to reach this
// control plane when neither the request nor BRIDGE_CONTROL_PLANE_URL names
// one. It pairs the primary outbound IP (reachable from LAN nodes, unlike
// localhost) with the port this process serves on — the same API_PORT
// resolution api-core/server uses. The Bridge manifest reserves 18767; retain
// the same value here for direct binary runs outside lifecycle injection.
func deriveControlPlaneURL() string {
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		port = "18767"
	}
	return "http://" + net.JoinHostPort(outboundIP(), port)
}

func canonicalControlPlaneEndpoint() (string, string) {
	if configured := strings.TrimSpace(os.Getenv("BRIDGE_CONTROL_PLANE_URL")); configured != "" {
		return configured, "configured"
	}
	if tunnel := strings.TrimSpace(os.Getenv("BRIDGE_TUNNEL_URL")); tunnel != "" {
		return tunnel, "tunnel"
	}
	return deriveControlPlaneURL(), "derived"
}

func startMDNSResponder(logger *log.Logger) *mdns.Responder {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("BRIDGE_MDNS_ADVERTISE")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			enabled = parsed
		}
	}
	controlPlaneURL, _ := canonicalControlPlaneEndpoint()
	responder := mdns.NewResponder(mdns.ResponderConfig{
		Service:  "_vrooli-bridge._tcp.local",
		Instance: "vrooli-bridge",
		Host:     "vrooli-bridge.local",
		Port:     bridgeAPIPort(),
		Address:  net.ParseIP(outboundIP()),
		URL:      controlPlaneURL,
	})
	if !enabled {
		return responder
	}
	if err := responder.Start(context.Background()); err != nil {
		logger.Printf("mDNS advertisement disabled: %v", err)
	}
	return responder
}

func bridgeAPIPort() int {
	port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("API_PORT")))
	if err != nil || port <= 0 || port > 65535 {
		return 18767
	}
	return port
}

// outboundIP returns the IP of the interface holding the default route. The
// UDP "dial" never sends a packet — it only asks the kernel which source
// address it would pick for a non-routable TEST-NET-1 destination. Hosts with
// no default route (fully offline) fall back to loopback, which still serves
// the onboard-this-same-machine case.
func outboundIP() string {
	conn, err := net.Dial("udp", "192.0.2.1:9")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil && !addr.IP.IsUnspecified() {
		return addr.IP.String()
	}
	return "127.0.0.1"
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "vrooli-bridge"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "vrooli-bridge",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Column-evolution migrations run BEFORE EnsureSchemas so an existing DB is at
	// the declared shape before EnsureSchemas' drift check verifies it (adding a
	// column to a CREATE TABLE IF NOT EXISTS is a silent no-op on a DB that already
	// has the table). Guarded + idempotent: a fresh DB has no table yet and skips.
	if err := internalonboard.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("onboard schema migration failed: %v", err)
	}
	if err := internalregistry.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("registry schema migration failed: %v", err)
	}
	if err := internalpairing.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("pairing schema migration failed: %v", err)
	}
	if err := internalmachines.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("machine schema migration failed: %v", err)
	}
	if err := internalruns.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("runs schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := internalmachines.BackfillLegacy(context.Background(), db.Primary()); err != nil {
		log.Fatalf("machine legacy backfill failed: %v", err)
	}

	clk := schedule.System()
	logger := log.Default()
	mdnsResponder := startMDNSResponder(logger)

	// Owner identity is resolved against scenario-authenticator (the "Owner →
	// control plane" boundary, SECURITY.md). The resolver finds the
	// authenticator's URL by name via api-core/discovery (no env var); the
	// client verifies owner JWTs offline against its published RS256 key.
	authResolver := discovery.NewResolver(discovery.ResolverConfig{})
	posture, err := trustposture.LoadWorkingTree()
	if err != nil {
		log.Fatalf("load trust posture: %v", err)
	}
	postureDefaults, err := trustposture.DefaultsFor(posture.Posture)
	if err != nil {
		log.Fatalf("resolve trust posture defaults: %v", err)
	}
	log.Printf("trust posture %q selects %d default Bridge execution scope(s)", posture.Posture, len(postureDefaults.NodeExecutionScopes))
	operatorSessionStore := internaloperatorsession.NewSQLiteRepository(db)
	var breakGlassPublic []byte
	publicKeyPath := strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_PUBLIC_KEY"))
	if publicKeyPath == "" {
		if paths, pathErr := trustposture.ResolveKeyPaths(); pathErr == nil {
			publicKeyPath = paths.Public
		}
	}
	if publicKeyPath != "" {
		breakGlassPublic, err = os.ReadFile(publicKeyPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Fatalf("read pinned break-glass public key: %v", err)
			}
		}
	}
	authClient := auth.NewClient(auth.Config{
		Resolver:            authResolver,
		JWKSGrace:           postureDefaults.JWKSCacheGrace,
		BreakGlassPublicKey: breakGlassPublic,
		BreakGlassAudience:  strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_AUDIENCE")),
		BreakGlassTarget:    strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_TARGET")),
		LocalSessions:       operatorSessionStore,
	})

	// The presence hub is the in-memory view of which nodes hold a dial-out
	// channel and their self-reported health. It is shared with the registry
	// read path (which overlays live online/offline onto stored nodes) and the
	// channel handler (which opens a Conn per dial-out connection).
	watchdogConfig := internalqueue.WatchdogConfigFromEnv()
	presenceHub := presence.NewHub(clk, presence.WithHeartbeatStaleAfter(watchdogConfig.PresenceStaleAfter))

	// The channel handler persists last-seen onto the registry's nodes table via
	// the repository's TouchLastSeen seam. It shares the same db/table the
	// registry module reads, so a heartbeat's last-seen is visible immediately.
	nodeLastSeen := internalregistry.NewSQLiteRepository(db, clk)

	// Pairing (OT-P0-002): the pairing service mints/burns codes and stores node
	// credentials. It registers redeeming nodes through the registry service
	// (the NodeRegistrar seam), and its repository doubles as the nodeauth
	// credential store and the registry atomic-revoke's CredentialRevoker.
	cpKeyDir, err := cpKeyDir()
	if err != nil {
		log.Fatalf("resolve control-plane key dir: %v", err)
	}
	cpKeypair, err := cpkeys.LoadOrCreate(cpKeyDir)
	if err != nil {
		log.Fatalf("load control-plane identity key: %v", err)
	}
	pairingRepo := internalpairing.NewSQLiteRepository(db, clk)
	// One registry service instance is shared by the pairing registrar (creates
	// node records on redeem) and the dispatch handler (reads node scopes to
	// authorize a job). Both read/write the same `nodes` table.
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		log.Fatalf("locate repository for node grant validation: %v", err)
	}
	grantCatalog, err := scopecatalog.BuildResilient(repoRoot)
	var grantValidator func([]string) error
	pairingDefaultScopes := append([]string(nil), postureDefaults.NodeExecutionScopes...)
	if err != nil {
		// A malformed scenario manifest must not take the fleet control plane
		// down. Registry and health remain available for diagnosis, while all
		// scope-bearing writes and execution services fail closed below.
		log.Printf("degraded catalog: node grants and dispatch unavailable: %v", err)
	} else {
		grantValidator = internalregistry.NewCatalogGrantValidator(grantCatalog)
		if scopes, ok := internalpairing.ScopesForPreset(grantCatalog, internalpairing.PresetReadOnly); ok {
			pairingDefaultScopes = scopes
		}
	}
	registryOpts := make([]internalregistry.Option, 0, 1)
	if grantValidator != nil {
		registryOpts = append(registryOpts, internalregistry.WithGrantValidator(grantValidator))
	}
	registrySvc := internalregistry.NewService(nodeLastSeen, registryOpts...)
	registrar := registrarAdapter{svc: registrySvc}
	pairingOpts := make([]internalpairing.Option, 0, 1)
	if grantValidator != nil {
		pairingOpts = append(pairingOpts, internalpairing.WithGrantValidator(grantValidator))
	}
	pairingOpts = append(pairingOpts, internalpairing.WithConfirmationValidator(func(req internalpairing.PairingRequest, words []string) error {
		expected, err := pairingwords.Derive(cpKeypair.PublicKeyBase64(), req.PublicKey)
		if err != nil {
			return internalpairing.ErrInvalid{Field: "confirmation_words", Reason: "cannot derive confirmation"}
		}
		if len(words) != len(expected) {
			return internalpairing.ErrInvalid{Field: "confirmation_words", Reason: "must contain the three words shown by pair list"}
		}
		for i := range expected {
			if !strings.EqualFold(strings.TrimSpace(words[i]), expected[i]) {
				return internalpairing.ErrInvalid{Field: "confirmation_words", Reason: "do not match the request"}
			}
		}
		return nil
	}))
	pairingSvc := internalpairing.NewService(pairingRepo, registrar, clk, pairingOpts...)
	// The node mutual-auth verifier reads node public keys from the pairing
	// repository (a revoked credential reads as absent). Construct it before
	// the grant handler so owner and node-facing grant operations fail closed.
	nodeVerifier := nodeauth.NewVerifier(pairingRepo)
	grantSvc := internalcredentialgrant.NewService(
		internalcredentialgrant.NewSQLiteRepository(db, clk.Now),
		registryNodeKindResolver{svc: registrySvc},
		clk.Now,
	)
	grantHandler := credentialgrantH.NewHandler(credentialgrantH.ModuleDeps{
		Service: grantSvc, Presence: presenceHub, Signer: cpKeypair,
		SealingPublicKey: pairingSvc.SealingPublicKey, NodeVerifier: nodeVerifier, Logger: logger,
		ResolveValue: func(ctx context.Context, logicalID, field string) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity(logicalID)
			if err != nil {
				return "", err
			}
			return authority.Require(identity, field)
		},
	})
	if _, err := pairingSvc.ReconcileEnrollments(context.Background()); err != nil {
		log.Fatalf("pairing enrollment reconciliation failed: %v", err)
	}

	// Audit (OT-P0-008): the append-only accountability substrate. Construct it
	// before queue restoration so boot reconciliation can account for every run
	// it terminates.
	var auditStore internalaudit.Store = internalaudit.NewSQLiteStore(db, clk)
	if endpoint := strings.TrimSpace(os.Getenv("VROOLI_WORKSPACE_SANDBOX_AUDIT_URL")); endpoint != "" {
		auditStore = &internalaudit.HTTPStore{Endpoint: endpoint}
		log.Printf("audit: workspace-sandbox sink enabled at %s", endpoint)
	}
	sessionManager := internalsession.NewManager(clk, auditStore)
	presenceHub.SetOfflineHook(func(nodeID string) {
		// A dial-out channel can disappear briefly while the agent reconnects.
		// Keep the PTY and its bounded scrollback alive during that repair window;
		// only a node that remains offline past the grace period gets a typed,
		// auditable terminal outcome.
		go func() {
			time.Sleep(90 * time.Second)
			if !presenceHub.IsOnline(nodeID) {
				sessionManager.CloseByNode(context.Background(), nodeID, "node_channel_lost")
			}
		}()
	})
	pushSession := func(ctx context.Context, nodeID, sessionID string, frame *sessionv1.Frame) error {
		payload, marshalErr := channelsign.Marshal(cpKeypair, &channelv1.ServerFrame{
			Payload: &channelv1.ServerFrame_Session{Session: &sharedv1.SessionFrame{SessionId: sessionID, Frame: frame}},
		})
		if marshalErr != nil {
			return marshalErr
		}
		if delivered := presenceHub.Push(nodeID, payload); delivered == 0 {
			return fmt.Errorf("node %q has no live channel", nodeID)
		}
		return nil
	}

	// Runs (OT-P0-005): a single durable-run service instance is shared by the
	// runs handler (operator verbs + node-facing ReportRunEvent ingest) and the
	// dispatch handler (Create), so the in-memory block-once waiter and
	// live-event subscriber coordination is one coherent instance.
	//
	// queue (OT-P1-004): the per-node job scheduler sits on the dispatch → push
	// path (bounded concurrency + fair FIFO) and is shared with the runs terminal
	// hook (free a slot + promote the next queued job when a run finishes) and
	// AbortRun's node-cancel push. The hook closure captures the `scheduler` var,
	// assigned just below, so the runs service and the scheduler reference each
	// other without a construction cycle.
	var scheduler *internalqueue.Scheduler
	runsSvc := internalruns.NewService(
		internalruns.NewSQLiteRepository(db, clk), clk,
		internalruns.WithCanceller(queueH.NewChannelCanceller(presenceHub, cpKeypair)),
		internalruns.WithTerminalHook(func(ctx context.Context, run internalruns.Run) {
			if scheduler != nil {
				scheduler.Complete(ctx, run.NodeID, run.ID)
			}
		}),
	)
	durableQueue := queueH.NewDurableStore(runsSvc)
	if reconciled, reconcileErr := internalqueue.Reconcile(
		context.Background(), durableQueue, presenceHub,
		queueH.NewChannelPusher(presenceHub, cpKeypair), clk,
	); reconcileErr != nil {
		log.Fatalf("queue reconciliation failed: %v", reconcileErr)
	} else {
		for _, outcome := range reconciled {
			auditOutcome := internalaudit.OutcomeAccepted
			if outcome.Terminal {
				auditOutcome = internalaudit.OutcomeFailed
			}
			if _, auditErr := auditStore.Append(context.Background(), internalaudit.Record{
				Action: internalaudit.ActionDispatch, Actor: "system:reconcile", NodeID: outcome.NodeID,
				Outcome: auditOutcome, Detail: "delivery reconciliation: " + outcome.Reason,
				RunID: outcome.RunID,
			}); auditErr != nil {
				log.Printf("queue reconciliation audit for %q failed: %v", outcome.RunID, auditErr)
			}
		}
	}
	scheduler, err = internalqueue.NewSchedulerWithStore(
		queueH.NewChannelPusher(presenceHub, cpKeypair), queueH.NewAborter(runsSvc), clk, 0,
		durableQueue,
	)
	if err != nil {
		log.Fatalf("queue projection restore failed: %v", err)
	}
	presenceHub.SetOnlineHook(func(nodeID string) {
		if scheduler != nil {
			scheduler.Promote(context.Background(), nodeID)
		}
		if grantHandler != nil {
			go func() {
				if syncErr := grantHandler.SyncNode(context.Background(), nodeID); syncErr != nil {
					logger.Printf("credential grants: sync node %q: %v", nodeID, syncErr)
				}
			}()
		}
	})
	watchdog := internalqueue.NewWatchdog(durableQueue, scheduler, queueH.NewAborter(runsSvc), clk,
		watchdogConfig, func(outcome internalqueue.Reconciliation) {
			auditOutcome := internalaudit.OutcomeAccepted
			if outcome.Terminal {
				auditOutcome = internalaudit.OutcomeFailed
			}
			if _, auditErr := auditStore.Append(context.Background(), internalaudit.Record{
				Action: internalaudit.ActionDispatch, Actor: "system:watchdog", NodeID: outcome.NodeID,
				Outcome: auditOutcome, Detail: "delivery watchdog: " + outcome.Reason,
				RunID: outcome.RunID,
			}); auditErr != nil {
				log.Printf("queue watchdog audit for %q failed: %v", outcome.RunID, auditErr)
			}
		})
	watchdog.Start(context.Background())

	// revResolver (phase 6): resolves the revision an onboarding/provisioning op
	// pins to. By default that is the control plane's EXACT current commit
	// (`git rev-parse HEAD`), so nodes never drift from the control plane; the
	// "@cp" sentinel expands to it, a bad ref is rejected with a friendly boundary
	// error, and a commit that was never pushed fails preflight with push-first
	// guidance. It is shared by onboard, provision, and (via provision) fleet roll
	// so all three behave identically. git discovers the monorepo root from the
	// working directory; BRIDGE_CP_REPO_DIR / BRIDGE_CP_GIT_REMOTE override.
	revResolver := cprev.New()

	// provision (OT-P0-006): the PRIVILEGED tier. Built once here so the same
	// instance backs both the provision handler (operator verbs + node ingest)
	// and the fleet roll's provisioner adapter — the in-memory op coordination
	// stays coherent across both call sites.
	provisionSvc := provisionH.NewService(db, clk, registrySvc, presenceHub, auditStore, cpKeypair,
		internalprovision.WithRevisionResolver(revResolver))
	sshStateDir, err := onboardssh.ResolveStateDir()
	if err != nil {
		log.Fatalf("resolve onboarding SSH state dir: %v", err)
	}
	sshSvc := onboardssh.NewService(sshStateDir)
	cleanupSvc := cleanupH.NewService(db, clk, registrySvc, pairingSvc, presenceHub, auditStore, cpKeypair, sshSvc)

	// onboard (phase 5): the orchestration tier. It drives a raw SSH host to a
	// paired, ONLINE, auto-starting fleet agent as a durable, server-owned op
	// (SSH first-touch → push bootstrap script → remote bootstrap → verify
	// online). The owner SSH password is request-scoped and zeroed by the SSH
	// key install; the single-use pairing code is issued server-side (pairingSvc)
	// and injected into the remote bootstrap over stdin (never argv/logs). It
	// owns its durable op tables and reconciles ops orphaned by a restart at boot.
	bootstrapScript, err := bootstrapScriptPath()
	if err != nil {
		log.Fatalf("resolve node bootstrap script: %v", err)
	}
	// The onboard service shares the revision resolver (default/@cp/preflight) and
	// picks up an optional default control-plane URL from BRIDGE_CONTROL_PLANE_URL
	// so an operator can omit control_plane_url when the control plane knows its
	// own public URL.
	// Working-tree source mode ships the control plane's LOCAL tree to the node
	// over SSH (owner development/validation mode). The snapshotter enumerates the
	// same checkout the revision resolver reads (BRIDGE_CP_REPO_DIR or the process
	// working directory). The node-revision recorder stamps the node record with the
	// provenance the op brought it to so `nodes`/fleet UI render a dirty node loudly.
	readinessEndpoint, readinessSource := canonicalControlPlaneEndpoint()
	fallbackMode := "lan"
	if readinessSource == "tunnel" {
		fallbackMode = "tunnel"
	}
	endpointStore := internalreadiness.NewStore(db, internalreadiness.Endpoint{URL: readinessEndpoint, Mode: fallbackMode, Source: readinessSource})
	onboardOpts := []internalonboard.Option{
		internalonboard.WithRevisionResolver(revResolver),
		internalonboard.WithProtectionProvisioner(onboardH.NewProtectionProvisioner(cleanupSvc)),
		internalonboard.WithDefaultScopes(postureDefaults.NodeExecutionScopes),
		internalonboard.WithWorkingTreeSource(internalonboard.NewWorkingTreeSource(strings.TrimSpace(os.Getenv("BRIDGE_CP_REPO_DIR")))),
		internalonboard.WithArtifactBuilder(internalonboard.NewArtifactBuilder()),
		internalonboard.WithNodeRevisionRecorder(onboardH.NewNodeRevisionRecorder(registrySvc)),
		internalonboard.WithEndpointResolver(func(ctx context.Context) (string, string, error) {
			selected, err := endpointStore.Resolve(ctx)
			return selected.URL, selected.Mode, err
		}),
		internalonboard.WithFirewallAdmitter(internalonboard.FirewallAdmitterFunc(func(ctx context.Context, candidateIP string) (internalonboard.FirewallAdmissionResult, error) {
			result, err := hostbroker.NewSocketClient().Call(ctx, hostbroker.AdmissionRequest("bridge.ufw.allow", "onboard-auto-"+candidateIP, candidateIP))
			return internalonboard.FirewallAdmissionResult{Status: result.Status, Code: result.Code, Changed: result.Changed, Managed: result.Evidence.Managed}, err
		})),
	}
	if handoffEndpoint, handoffErr := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-onboarding"); handoffErr == nil {
		handoffURL := internalonboarding.HandoffEndpoint(handoffEndpoint)
		onboardOpts = append(onboardOpts, internalonboard.WithOnboardingHandoff(internalonboarding.HTTPHandoffClient{Endpoint: handoffURL}))
		log.Printf("onboard: scenario selection handoff enabled at %s", handoffURL)
	} else {
		log.Printf("onboard: optional onboarding scenario unavailable: %v", handoffErr)
	}
	if cpURL, source := canonicalControlPlaneEndpoint(); source == "configured" {
		onboardOpts = append(onboardOpts, internalonboard.WithDefaultControlPlaneURL(cpURL))
	} else {
		// Zero-config default: the control plane knows its own address, so a
		// bare start (no env, no per-request control_plane_url) must still
		// onboard instead of failing validation. Per-request and env values
		// always win over this derivation.
		derived := cpURL
		log.Printf("onboard: default control-plane URL derived as %s (override per request or with BRIDGE_CONTROL_PLANE_URL)", derived)
		onboardOpts = append(onboardOpts, internalonboard.WithDefaultControlPlaneURL(derived))
	}
	onboardSvc := onboardH.NewService(db, clk, pairingSvc, presenceHub, sshSvc, bootstrapScript, onboardOpts...)
	if n, rerr := onboardSvc.ResumeInterrupted(context.Background()); rerr != nil {
		log.Printf("onboard: reconcile interrupted ops failed: %v", rerr)
	} else if n > 0 {
		log.Printf("onboard: marked %d interrupted onboarding op(s) FAILED (safe to retry)", n)
	}

	// dispatch (OT-P0-004): the allowlist gate, built once here so the SAME
	// instance backs both the dispatch handler and the gate domain's runner
	// adapter — every cross-OS gate validation run flows through the same
	// allowlist + per-node scopes + audit gate as any other job.
	dispatchSvc := dispatchH.NewService(registrySvc, runsSvc, auditStore, presenceHub, scheduler, grantSvc)
	// Relay reuses the same node reader, presence gate, manifest-derived
	// admission, signed channel transport, and append-only audit sink. Its
	// response broker is wired into the node-facing Presence RPC below, so a
	// caller cancellation and a node terminal response share one correlation.
	relayBroker := internalrelay.NewBroker()
	relaySvc := relayH.NewService(
		registrySvc, presenceHub, auditStore,
		queueH.NewChannelRelayPusher(presenceHub, cpKeypair), relayBroker,
	)

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "vrooli-bridge-api", "1.0.0"),
		readinessH.Module(db, onboardSvc, endpointStore, true, internalonboard.NewUFWObserver(), hostbroker.NewSocketClient()),
		// identity: same-origin owner sign-in / registration facade. The browser
		// never calls scenario-authenticator cross-origin; it calls this bridge RPC,
		// which forwards to the authenticator (resolved by name via the shared
		// authResolver) and relays the issued owner JWT. Unauthenticated (it precedes
		// the caller holding a token); owns no credential logic and no tables.
		identityH.Module(authResolver, logger, operatorSessionStore),
		// machines: operator-intent identity and lifecycle. It references Node
		// lineage rather than copying Registry or live Presence state.
		machinesH.Module(db, clk, sshSvc, registrySvc, pairingSvc, presenceHub, onboardSvc, logger),
		// registry RevokeNode performs atomic revocation: durable revoke +
		// credential destruction (pairingSvc) + live-channel drop (presenceHub).
		registryH.Module(registrySvc, presenceHub, pairingSvc, presenceHub, watchdogConfig.PresenceStaleAfter, logger),
		attachedH.Module(db.Primary(), logger, presenceHub),
		channelH.Module(presenceHub, nodeLastSeen, nodeVerifier, logger,
			channelH.WithDeliveryAckRecorder(runsSvc), channelH.WithAuditSink(auditStore),
			channelH.WithSessionManager(sessionManager, authClient, registrySvc), channelH.WithSessionPush(pushSession),
			channelH.WithRelayResponseSink(relayBroker), channelH.WithCredentialReceiptRecorder(grantSvc)),
		cleanupH.Module(cleanupSvc, nodeVerifier, logger),
		credentialgrantH.Module(grantHandler),
		pairingH.Module(pairingSvc, cpKeypair.PublicKeyBase64(), pairingDefaultScopes, internalpairing.PermissionPresets(grantCatalog), nodeVerifier, logger),
		// dispatch (OT-P0-004): the allowlist gate. It reads node scopes
		// (registrySvc), checks presence + protocol compatibility, creates durable
		// runs (runsSvc), audits (auditStore), and submits typed jobs to the
		// per-node scheduler (bounded concurrency on the channel-push path).
		dispatchH.Module(dispatchSvc, logger),
		// relay (OT-P1): owner-gated short-lived command calls. Admission is
		// delegated to the same dispatch policy before the signed node frame is
		// emitted; the response is bounded and correlated to this call.
		relayH.Module(relaySvc, logger),
		// runs (OT-P0-005): durable run lifecycle + node-facing event ingest.
		runsH.Module(runsSvc, nodeVerifier, logger),
		// queue (OT-P1-004): read-only control-plane view over the per-node
		// scheduler (which jobs are running vs queued, per node).
		queueH.Module(scheduler, logger),
		// provision (OT-P0-006): the PRIVILEGED tier. Owns its durable op tables;
		// reads node revocation (registrySvc), checks presence + pushes the
		// privileged ProvisionCommand (presenceHub), audits (auditStore), and
		// gates the node-facing ReportProvisionEvent on mutual auth (nodeVerifier).
		provisionH.Module(provisionSvc, nodeVerifier, logger),
		// onboard (phase 5): durable, orchestrated one-shot node onboarding over
		// SSH. Owns its durable op tables; drives SSH first-touch + SCP + remote
		// bootstrap (streamed VBOOTSTRAP markers), issues the pairing code
		// server-side (pairingSvc), and confirms the node ONLINE (presenceHub).
		onboardH.Module(onboardSvc, db, clk, sshSvc, logger),
		// fleet (OT-P1-001): fleet-wide version roll. Enumerates nodes
		// (registrySvc), gates on presence + protocol compatibility (presenceHub),
		// and dispatches a privileged provisioning op per eligible node by
		// delegating to the shared provision service (provisionSvc).
		fleetH.Module(db, clk, registrySvc, presenceHub, provisionSvc, revResolver, logger),
		// gate (OT-P1-002): cross-OS deployment gate. Selects one eligible node
		// per target OS (registrySvc + presenceHub), dispatches a validation run to
		// each by delegating to the shared dispatch service (dispatchSvc) + runs
		// service (runsSvc), and aggregates per-OS verdicts into one cross-OS
		// result deployment-manager owns. Owns its durable gate tables.
		gateH.Module(db, clk, registrySvc, presenceHub, dispatchSvc, runsSvc, logger),
		// artifacts (OT-P1-003): non-git artifact distribution. Validates node
		// revocation (registrySvc) and delegates the byte move to device-sync-hub
		// directed delivery (bridge stores no blob).
		artifactsH.Module(db, clk, registrySvc, runsSvc, nodeVerifier, logger),
		// audit (OT-P0-008): owner-gated read of the append-only trail.
		auditH.Module(auditStore, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	// auth.Middleware best-effort-injects the owner Identity when a valid
	// bearer token is present; owner-gated RPCs fail closed via RequireOwner.
	rootMux.Handle("/", auth.MiddlewareWithAudit(authClient, logger, auth.BreakGlassAuditFunc(func(ctx context.Context, id auth.Identity) error {
		_, err := auditStore.Append(ctx, internalaudit.Record{
			Action: internalaudit.ActionBreakGlass,
			Actor:  id.OwnerID, NodeID: "local-control-plane",
			Scenario: "scenario-authenticator", Verb: "break-glass",
			Outcome: internalaudit.OutcomeAccepted,
		})
		return err
	}))(srv.Handler()))

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	innerHandler := apihttp.TestModeMiddleware(rootMux)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		innerHandler.ServeHTTP(w, r)
	})

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// The dial-out SSE channel and interactive WebSocket sessions are
		// intentionally long-lived. api-core's normal 30-second write
		// timeout is correct for request/response APIs but would silently
		// evict a healthy node channel before the reconnect contract can be
		// exercised. Keep the server bound while the handlers' own idle and
		// lifetime limits remain authoritative.
		WriteTimeout: 24 * time.Hour,
		Cleanup: func(ctx context.Context) error {
			_ = mdnsResponder.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
