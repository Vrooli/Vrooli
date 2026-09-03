package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"program-runtime/internal/bindings"
	"program-runtime/internal/budgets"
	"program-runtime/internal/capabilities"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/modules"
	"program-runtime/internal/programs"
	"program-runtime/internal/retention"
	"program-runtime/internal/server"
	"program-runtime/internal/sessions"
	"program-runtime/internal/shapes"
	"program-runtime/internal/telemetry"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	_ "modernc.org/sqlite"

	bindingsH "program-runtime/handlers/bindings"
	capsH "program-runtime/handlers/capabilities"
	healthH "program-runtime/handlers/health"
	libraryH "program-runtime/handlers/library"
	measuresH "program-runtime/handlers/measures"
	programsH "program-runtime/handlers/programs"
	sessionsH "program-runtime/handlers/sessions"
	shapesH "program-runtime/handlers/shapes"
	telemetryH "program-runtime/handlers/telemetry"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("program-runtime")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve program-runtime storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

// fileRootPath is the template's mandatory file-store seam. Domain stores
// compose their relative paths from it rather than retaining startup root
// strings, so X-Vrooli-Test-Mode is honored independently per request.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "program-runtime"}) {
		return
	}

	// A violated timeout ladder means some layer will be killed mid-write and
	// its failure will be untyped, which is worse than not starting: the
	// runtime would keep reporting healthy while returning `unexpected EOF`.
	if err := budgets.Validate(); err != nil {
		log.Fatalf("timeout budget ladder is invalid: %v", err)
	}

	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "program-runtime"})
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := sessions.EnsureCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("session schema compatibility failed: %v", err)
	}
	if err := bindings.EnsureCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("binding schema compatibility failed: %v", err)
	}
	if err := programs.EnsureCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("program schema compatibility failed: %v", err)
	}
	libraryRepository := library.NewRepository(db.Primary())
	if err := library.EnsureCompatibility(context.Background(), db.Primary()); err != nil {
		log.Fatalf("library schema compatibility failed: %v", err)
	}
	retentionDB, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("retention database connection failed: %v", err)
	}
	retentionWorker := retention.New(retention.Options{DB: retentionDB.Primary(), ProgramWindow: retention.ProgramWindow, RefusalWindow: retention.RefusalWindow, ReclaimWindow: retention.ReclaimWindow, Logger: log.Default()})
	retentionWorker.Start(context.Background())
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("repository root resolution failed: %v", err)
	}
	contractIndex := contracts.NewIndex()
	if err := contractIndex.Load(repoRoot); err != nil {
		log.Printf("program contract index unavailable: %v", err)
	} else {
		invalid := 0
		for _, contract := range contractIndex.List() {
			if contract.ValidationError != "" {
				invalid++
			}
		}
		log.Printf("program contract index loaded=%d validation_errors=%d", len(contractIndex.List()), invalid)
	}
	shapeRepository := shapes.NewRepository(db.Primary(), contractIndex)
	if resolved, resolveErr := shapeRepository.ResolveCoverage(context.Background(), contractIndex); resolveErr != nil {
		log.Printf("program shape coverage unavailable: %v", resolveErr)
	} else {
		log.Printf("program shape coverage resolved=%d", resolved)
	}
	bindingRegistry, err := bindings.LoadWithRetry(repoRoot, 5, 100*time.Millisecond)
	if err != nil {
		log.Fatalf("binding registry initialization failed: %v", err)
	}
	bindingRegistry.SetInvocationRecorder(bindings.NewInvocationRepository(db.Primary()))
	bindingRegistry.SetExerciseReader(bindings.NewReceiptExerciseReader(discovery.ResolveScenarioURLDefault, http.DefaultClient))
	refusalRepository := bindings.NewRefusalRepository(db.Primary())
	unresolvedRecorder, _ := refusalRepository.(bindings.UnresolvedRecorder)
	telemetryStore := telemetry.NewStoreWithDB(db.Primary(), telemetry.NewPublisher(os.Getenv("VROOLI_EVENTS_API_BASE")))
	shapeRepository.SetEventSink(telemetryStore)
	telemetryStore.Start(context.Background())
	registryBindings := bindingRegistry.List("", "")
	bindingSpecs := make([]programs.BindingSpec, 0, len(registryBindings))
	for _, binding := range registryBindings {
		bindingSpecs = append(bindingSpecs, programs.BindingSpec{ID: binding.GetId(), Namespace: strings.ReplaceAll(binding.GetScenario(), "-", "_"), Scenario: binding.GetScenario(), Group: binding.GetGroup(), Command: binding.GetCommand(), Effect: binding.GetEffect(), Reachable: binding.GetReachable(), ReachabilityReason: binding.GetReachabilityReason(), RowsField: binding.GetRowsField(), MetaFields: binding.GetMetaFields(), RowFieldCandidates: binding.GetRowFieldCandidates()})
	}
	bridgeURL := ""
	agentBridgeURL := ""
	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		bridgeURL = fmt.Sprintf("http://127.0.0.1:%s/internal/program-runtime/bindings/execute", port)
		agentBridgeURL = fmt.Sprintf("http://127.0.0.1:%s/internal/program-runtime/agent/execute", port)
	}
	runner := programs.NewSubprocessRunnerWithBindings(filepath.Join(repoRoot, "scenarios", "program-runtime", "kernel", "host", "engine.py"), bindingSpecs, bridgeURL, agentBridgeURL)
	runner.SetBindingProvider(func() []programs.BindingSpec {
		current := bindingRegistry.List("", "")
		out := make([]programs.BindingSpec, 0, len(current))
		for _, binding := range current {
			out = append(out, programs.BindingSpec{ID: binding.GetId(), Namespace: strings.ReplaceAll(binding.GetScenario(), "-", "_"), Scenario: binding.GetScenario(), Group: binding.GetGroup(), Command: binding.GetCommand(), Effect: binding.GetEffect(), Reachable: binding.GetReachable(), ReachabilityReason: binding.GetReachabilityReason(), RowsField: binding.GetRowsField(), MetaFields: binding.GetMetaFields(), RowFieldCandidates: binding.GetRowFieldCandidates()})
		}
		return out
	})
	runner.SetLibraryProvider(func() []programs.LibrarySpec {
		current, err := libraryRepository.ListCallable(context.Background())
		if err != nil {
			return nil
		}
		out := make([]programs.LibrarySpec, 0, len(current))
		for _, program := range current {
			if program == nil {
				continue
			}
			out = append(out, programs.LibrarySpec{Name: program.GetName(), Version: program.GetVersion(), Source: program.GetSource(), Description: program.GetDescription(), Current: program.GetCurrent()})
		}
		return out
	})
	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		runner.SetDiscoveryURL(fmt.Sprintf("http://127.0.0.1:%s/internal/program-runtime/bindings/resolve-intent", port))
	}
	workspaceResolver := sessions.NewTypedWorkspaceResolver(discovery.NewResolver(discovery.ResolverConfig{}), http.DefaultClient)
	sessionManager := sessions.NewManager(sessions.Options{Store: db.Primary(), WallBudget: envDurationMillis("PROGRAM_RUNTIME_WALL_BUDGET_MILLIS"), CPUBudget: envDurationMillis("PROGRAM_RUNTIME_CPU_BUDGET_MILLIS"), InferenceCeilingMicros: envInt64("PROGRAM_RUNTIME_INFERENCE_CEILING_MICROS"), DelegationCeilingMicros: envInt64("PROGRAM_RUNTIME_DELEGATION_CEILING_MICROS"), WorkspaceResolver: workspaceResolver, OnWorkspaceResolved: runner.SetSessionWorkspace, OnReclaimed: func(id string) { runner.KillSession(id); runner.ClearSessionWorkspace(id) }})
	var programService *programs.Service
	programService = programs.NewService(programs.Options{Store: db.Primary(), Runner: runner, Preflight: func(source string) []*programsv1.Diagnostic {
		current := bindingRegistry.List("", "")
		known := []string{"discover", "recall", "guide", "validate", "capture", "ai", "agent", "gather", "describe", "reachable", "lib", "vrooli", "__vrooli__", "Handle"}
		for _, binding := range current {
			name := strings.ReplaceAll(binding.GetScenario(), "-", "_")
			if name != "" && name != "vrooli" {
				known = append(known, name)
			}
		}
		return programs.ResolveSource(source, known, filepath.Join(repoRoot, "scenarios", "program-runtime", "kernel", "host", "analyze.py"))
	}, PreflightSession: func(ctx context.Context, sessionID, source string) []*programsv1.Diagnostic {
		current := bindingRegistry.List("", "")
		known := []string{"discover", "recall", "guide", "validate", "capture", "ai", "agent", "gather", "describe", "reachable", "lib", "vrooli", "__vrooli__", "Handle"}
		for _, binding := range current {
			name := strings.ReplaceAll(binding.GetScenario(), "-", "_")
			if name != "" && name != "vrooli" {
				known = append(known, name)
			}
		}
		for _, previous := range programService.List(ctx, sessionID, true) {
			known = append(known, programs.DeclaredNames(previous.GetSource(), filepath.Join(repoRoot, "scenarios", "program-runtime", "kernel", "host", "analyze.py"))...)
		}
		return programs.ResolveSource(source, known, filepath.Join(repoRoot, "scenarios", "program-runtime", "kernel", "host", "analyze.py"))
	}, RecordUnresolved: func(ctx context.Context, sessionID, attemptedName, provenance string) error {
		if unresolvedRecorder == nil {
			return nil
		}
		return unresolvedRecorder.RecordUnresolved(ctx, sessionID, provenance, attemptedName, time.Now().UTC())
	}, ShapeSink: shapeRepository, RecordMemory: func(id string, bytes int64) { _ = sessionManager.SetMemoryBytes(context.Background(), id, bytes) }, ExecutionBudget: func(id string) (programs.ExecutionLimits, error) {
		budget, err := sessionManager.ExecutionBudget(context.Background(), id)
		if err != nil {
			return programs.ExecutionLimits{}, err
		}
		return programs.ExecutionLimits{Wall: budget.WallBudget - budget.WallConsumed, CPU: budget.CPUBudget - budget.CPUConsumed}, nil
	}, ChargeExecution: func(id string, wall, cpu time.Duration) error {
		return sessionManager.ChargeExecution(context.Background(), id, wall, cpu)
	}, ValidateSession: func(id string) bool { _, err := sessionManager.Get(context.Background(), id); return err == nil }, LibraryVersion: func(string) string { return libraryRepository.CurrentStamp(context.Background()) }, Events: telemetryStore})
	reclamationStop := make(chan struct{})
	reclamationDone := make(chan struct{})
	go func() {
		defer close(reclamationDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				sessionManager.ReclaimIdle(context.Background(), now.UTC())
			case <-reclamationStop:
				return
			}
		}
	}()
	// The authoring eval composes the same governed inference binding a program
	// calls, so a measured score is attributable to a real route rather than to
	// a private client. Both seams are injected; when either cannot resolve, the
	// eval reports `unavailable` with the reason rather than a fabricated score.
	authoringDeps := programs.AuthoringDeps{
		SuitePath: programs.DefaultSuitePath(repoRoot),
		Author: func(ctx context.Context, instruction, task string) (string, string, error) {
			const authoringRole = "author.generator"
			// The role is schema-constrained, so the program is requested as a
			// field of a small object rather than as a bare string. A bare
			// `{"type":"string"}` grammar makes the model emit a fence and stop
			// after a few tokens; an object with one `source` field does not.
			const authoringSchema = `{"type":"object","properties":{"source":{"type":"string"}},"required":["source"]}`
			response, err := bindingRegistry.Execute(ctx, "ai-gateway/inference/run", map[string]any{
				"source":      task,
				"instruction": instruction,
				"role":        authoringRole,
				"schema":      authoringSchema,
			}, nil, false, bindings.InvocationMetadata{Provenance: "test"}, programs.AuthoringHTTPClient())
			if err != nil {
				return "", "", err
			}
			model, _ := response["model"].(string)
			raw, _ := response["valueJson"].(string)
			if strings.TrimSpace(raw) == "" {
				raw, _ = response["value_json"].(string)
			}
			if strings.TrimSpace(raw) == "" {
				return "", model, fmt.Errorf("authoring response carried no value")
			}
			var envelope struct {
				Source string `json:"source"`
			}
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				return "", model, fmt.Errorf("authoring response is not the declared object: %w", err)
			}
			if strings.TrimSpace(envelope.Source) == "" {
				return "", model, fmt.Errorf("authoring response carried an empty source field")
			}
			return envelope.Source, model, nil
		},
		RunCase: func(ctx context.Context, setup, source string) (string, string, int64, string, error) {
			session, err := sessionManager.Create(ctx, "", "", nil)
			if err != nil {
				return "", "", 0, "", err
			}
			defer func() { _, _ = sessionManager.Delete(context.Background(), session.ID, "authoring eval case complete") }()
			if strings.TrimSpace(setup) != "" {
				if _, err := programService.Submit(ctx, session.ID, setup, programsv1.Provenance_PROVENANCE_TEST, false); err != nil {
					return "", "", 0, "", fmt.Errorf("authoring setup failed: %w", err)
				}
			}
			p, err := programService.Submit(ctx, session.ID, source, programsv1.Provenance_PROVENANCE_TEST, false)
			if err != nil {
				return "", "", 0, "", err
			}
			return p.GetStdout(), p.GetFailureShape(), p.GetAgentBytes(), p.GetFailureDetail(), nil
		},
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.ModuleWithDescriptor(db, "program-runtime-api", "1.0.0", bindingRegistry.SkippedManifestCount, bindingRegistry.SnapshotMetadata),
		capsH.Module(capabilities.NewRegistry()),
		bindingsH.Module(bindingRegistry, libraryRepository),
		programsH.Module(programService, authoringDeps),
		libraryH.Module(libraryRepository, bindingRegistry, contractIndex),
		sessionsH.Module(sessionManager),
		telemetryH.Module(telemetryStore),
		shapesH.Module(shapeRepository),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/internal/program-runtime/bindings/execute", bindingsH.Bridge(bindingRegistry, sessionManager, refusalRepository))
	rootMux.Handle("/internal/program-runtime/bindings/describe", bindingsH.DescribeBridge(bindingRegistry, sessionManager))
	rootMux.Handle("/internal/program-runtime/bindings/reachability", bindingsH.ReachabilityBridge(bindingRegistry, sessionManager))
	rootMux.Handle("/internal/program-runtime/bindings/unresolved", bindingsH.UnresolvedBridge(sessionManager, unresolvedRecorder))
	rootMux.Handle("/internal/program-runtime/bindings/projection/", bindingsH.ProjectionBridge(sessionManager, bindingRegistry))
	rootMux.Handle("/internal/program-runtime/bindings/resolve-intent", bindingsH.IntentBridge(bindingRegistry, libraryRepository))
	rootMux.Handle("/internal/program-runtime/bindings/search", bindingsH.BindingCorpusHandler(bindingRegistry))
	rootMux.Handle("/internal/program-runtime/library/search", bindingsH.LibraryCorpusHandler(libraryRepository, contractIndex))
	rootMux.Handle("/internal/program-runtime/agent/execute", bindingsH.AgentBridge(sessionManager, programs.NewDiscoveryDelegator(nil)))
	rootMux.Handle("/internal/program-runtime/agent/start", bindingsH.AgentStartBridge(sessionManager, programs.NewDiscoveryDelegator(nil)))
	rootMux.Handle("/internal/program-runtime/agent/collect", bindingsH.AgentCollectBridge(sessionManager, programs.NewDiscoveryDelegator(nil)))

	// /measures is the measures-go serve substrate consumed by measures-health.
	// Declarations and execution stay typed and owned by this scenario.
	runtimeMeasures, err := measuresH.Handler(schedule.System(), func() int {
		return len(sessionManager.List(context.Background()))
	}, func() int {
		return len(programService.MineFailures(context.Background(), false))
	}, func() int {
		return sessionManager.CountDelegations(context.Background())
	}, func() int {
		total, _, _ := bindingRegistry.ExerciseMeasures(context.Background())
		return total
	}, func() int {
		_, unattributed, _ := bindingRegistry.ExerciseMeasures(context.Background())
		return unattributed
	}, func() int {
		_, failureRate, _ := bindingRegistry.InvocationMeasures(context.Background())
		return failureRate
	}, func() int {
		_, _, dormant := bindingRegistry.ExerciseMeasures(context.Background())
		return dormant
	}, func() int {
		return bindingRegistry.SustainedDegradedCount(context.Background())
	}, func() int {
		return bindingsH.LibraryDiscoveryUsage()
	}, func() int {
		return bindingsH.DiscoveryNullVerdictRatePercent()
	})
	if err != nil {
		log.Fatalf("measures registry: %v", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", runtimeMeasures))

	rootMux.Handle("/", srv.Handler())
	bindingsH.RegisterSearchHubProvider(context.Background(), libraryRepository)

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	// The write deadline is the outermost rung of the budget ladder, not a
	// taste call. Leaving it unset inherited api-core's 30s default, which
	// capped every synchronous call in a scenario whose sessions advertise a
	// four-hour wall budget; the ladder and its startup assertion live in
	// internal/budgets.
	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		ReadTimeout:  budgets.ServerRead,
		WriteTimeout: budgets.ServerWrite,
		Cleanup: func(ctx context.Context) error {
			_ = runner.Close()
			close(reclamationStop)
			<-reclamationDone
			telemetryStore.Stop()
			retentionWorker.Stop()
			_ = retentionDB.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func envInt64(name string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(name)), 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func envDurationMillis(name string) time.Duration {
	value := envInt64(name)
	if value == 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}
