package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets/targets_v1connect"

	"research-desk/internal/capabilities"
	"research-desk/internal/courier"
	"research-desk/internal/home"
	"research-desk/internal/launch"
	"research-desk/internal/peer"
	"research-desk/internal/skillscripts"
	"research-desk/internal/wire"
	"research-desk/internal/writer"
)

// scenarioName is this scenario; scenarioRel its folder in the repository.
const (
	scenarioName = "research-desk"
	scenarioRel  = "scenarios/" + scenarioName
)

// wiring holds the production seams, resolved once at start.
type wiring struct {
	// scenarioDir is <repo>/scenarios/research-desk; empty when the
	// repository root cannot be resolved.
	scenarioDir string
	homeConfig  home.Config
	reader      *home.Reader
	writer      *writer.Writer
	courier     *courier.Courier
	launch      *launch.Service
	webConsole  *peer.Peer
	// entryChars is limits.json's entry_chars_max, read on each call.
	entryChars func() (int, error)
	// registry is the one table of whether each optional scenario can be
	// reached; nil when service.json cannot be read.
	registry *capabilities.Registry
	// resolver finds a scenario's port through its lifecycle record.
	resolver capabilities.PortResolver
	// version is service.version from .vrooli/service.json, the one place
	// the version is kept; "unknown" when it cannot be read.
	version string
}

// Locations a test or a second instance overrides. Unset means the
// production location under the user's home.
const (
	envResearchDir = "RESEARCH_DESK_RESEARCH_DIR"
	envStateDir    = "RESEARCH_DESK_STATE_DIR"
	envCodexDir    = "RESEARCH_DESK_CODEX_DIR"
	// envOtherCodexHomes is a path list of glob patterns, each naming a Codex
	// home outside Web Console's session folder.
	envOtherCodexHomes = "RESEARCH_DESK_OTHER_CODEX_HOMES"
)

// newWiring resolves the locations and the scenario folder and builds the
// one reader of the research home. A missing skills folder leaves mission
// checks and ceilings unknown; it never stops the projection.
// Only the lifecycle-managed live instance may write.
func newWiring(getenv func(string) string) *wiring {
	userHome, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("resolve the user's home directory for ~/.vrooli/research: %v", err)
	}
	w := &wiring{homeConfig: home.DefaultConfig(userHome), version: "unknown"}
	if dir := getenv(envResearchDir); dir != "" {
		w.homeConfig.ResearchDir = dir
	}
	if dir := getenv(envCodexDir); dir != "" {
		w.homeConfig.CodexDir = dir
	}
	if homes := getenv(envOtherCodexHomes); homes != "" {
		w.homeConfig.OtherCodexHomes = filepath.SplitList(homes)
	}
	stateDir := writer.DefaultStateDir(userHome)
	if dir := getenv(envStateDir); dir != "" {
		stateDir = dir
	}
	deps := home.Deps{Clock: schedule.System(), Procs: home.ProcTable{}}
	var paths skillscripts.Paths
	var repoRoot string
	if root, err := repocontract.ResolveRepoRoot(); err != nil {
		log.Printf("Vrooli repository root not resolved (set VROOLI_ROOT); mission checks, ceilings and capabilities will read unknown: %v", err)
	} else {
		repoRoot = root
		w.scenarioDir = filepath.Join(root, filepath.FromSlash(scenarioRel))
		paths = skillscripts.Paths{Skills: filepath.Join(w.scenarioDir, "skills")}
		w.version = serviceVersion(filepath.Join(w.scenarioDir, ".vrooli", "service.json"))
		deps.Missions = skillscripts.MissionChecker{Paths: paths, Runner: skillscripts.NewPython()}
		deps.Ceilings = paths.Ceilings
	}
	w.reader = home.NewReader(w.homeConfig, deps)
	limits := func() (*skillscripts.Limits, error) {
		if paths.Skills == "" {
			return nil, errors.New("the Vrooli repository root is not resolved (set VROOLI_ROOT), so limits.json cannot be read")
		}
		return skillscripts.LoadLimits(paths.Limits())
	}
	w.entryChars = func() (int, error) {
		l, err := limits()
		if err != nil {
			return 0, err
		}
		return l.EntryCharsMax, nil
	}
	w.writer = writer.New(w.homeConfig.ResearchDir, stateDir, w.entryChars)
	if err := liveInstance(getenv); err != nil {
		w.writer.ReadOnly(err.Error())
	}
	resolver := capabilities.DefaultResolver()
	w.resolver = resolver
	w.registry = w.loadRegistry(resolver)
	w.courier = &courier.Courier{
		Reader: w.reader,
		Writer: w.writer,
		Hub: &courier.EventHub{
			Events: capabilities.Reach(w.registry, resolver, "vrooli-events"),
			Hub:    &peer.Peer{Name: "notification-hub", BaseURL: capabilities.Reach(w.registry, resolver, "notification-hub")},
		},
		Clock:  deps.Clock,
		Logger: log.Default(),
	}
	w.webConsole = &peer.Peer{Name: "web-console", BaseURL: capabilities.Reach(w.registry, resolver, "web-console"), Credential: webConsoleCredential}
	w.launch = &launch.Service{
		Reader:  w.reader,
		Writer:  w.writer,
		Packets: skillscripts.Packets{Paths: paths, Runner: skillscripts.NewPython(), ResearchDir: w.homeConfig.ResearchDir},
		Limits:  limits,
		Launcher: &launch.Launcher{
			Terminal:   &launch.WebConsole{Peer: w.webConsole},
			Sessions:   w.reader,
			WorkingDir: repoRoot,
		},
		Access: w.webConsole.Access,
		Clock:  deps.Clock,
		Logger: log.Default(),
	}
	return w
}

// liveInstance is nil when this process is the lifecycle-managed live
// instance: the control plane started it with a runtime instance identity and
// the live storage namespace. A test, a boot check or a shadow variant is not,
// and only reads.
func liveInstance(getenv func(string) string) error {
	switch {
	case getenv(preflight.LifecycleManagedEnvVar) != "true":
		return errors.New("not started by the Vrooli lifecycle")
	case getenv(runtimeInstanceEnv) == "":
		return errors.New("the lifecycle gave no runtime instance identity (" + runtimeInstanceEnv + ")")
	case getenv(storage.EnvStorageNamespace) != scenarioName:
		return fmt.Errorf("storage namespace %q is not the live instance's %q", getenv(storage.EnvStorageNamespace), scenarioName)
	}
	return nil
}

// runtimeInstanceEnv is the lifecycle identity the control plane installs.
const runtimeInstanceEnv = "VROOLI_RUNTIME_INSTANCE_ID"

// start runs serve in the background with a context of its own. The returned
// stop, given to the server's shutdown cleanup, cancels it, waits for the
// courier's current pass to end (bounded by ctx) and releases the write lock.
func (w *wiring) start() (stop func(ctx context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.serve(ctx)
	}()
	return func(shutdown context.Context) error {
		cancel()
		select {
		case <-done:
		case <-shutdown.Done():
			return fmt.Errorf("the courier did not stop before shutdown ended: %w", shutdown.Err())
		}
		return w.writer.Close()
	}
}

// serviceVersion reads service.version from service.json.
func serviceVersion(path string) string {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "unknown"
	}
	var manifest struct {
		Service struct {
			Version string `json:"version"`
		} `json:"service"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Service.Version == "" {
		return "unknown"
	}
	return manifest.Service.Version
}

// serve becomes the writer once the lock is free (a restart overlaps the old
// process briefly), marks launches the last stop interrupted, then runs the
// courier until ctx ends. An instance that is not the writer does nothing.
func (w *wiring) serve(ctx context.Context) {
	if err := w.writer.Role(); err != nil {
		log.Printf("research-desk reads only: %v", err)
		return
	}
	for {
		err := w.writer.Held()
		if err == nil {
			break
		}
		log.Printf("research-desk waits for the write lock: %v", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
	go w.registerResearchBackupTargetUntilAvailable(ctx)
	if err := w.launch.MarkInterrupted(ctx); err != nil {
		log.Printf("launches interrupted by the last stop were not marked: %v", err)
	}
	w.courier.Run(ctx, time.Minute)
}

func (w *wiring) registerResearchBackupTargetUntilAvailable(ctx context.Context) {
	for {
		attempt, cancel := context.WithTimeout(ctx, peer.CallTimeout+capabilities.CheckTimeout)
		err := w.registerResearchBackupTarget(attempt)
		cancel()
		if err == nil {
			log.Printf("research home backup target is registered")
			return
		}
		log.Printf("research home backup target registration will retry: %v", err)
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// registerResearchBackupTarget keeps the desk's externally located research
// home attached to the owner's lifecycle. The DBM dependency is optional: a
// temporarily unavailable backup service must not stop the desk from serving
// its research home. Only the live lifecycle writer reaches this call.
func (w *wiring) registerResearchBackupTarget(ctx context.Context) error {
	if w.registry == nil || w.resolver == nil {
		return errors.New("capability registry is unavailable")
	}
	port, err := w.resolver.ResolveScenarioPort(ctx, "data-backup-manager", "API_PORT")
	if err != nil {
		return fmt.Errorf("resolve Data Backup Manager API port: %w", err)
	}
	dbm := &peer.Peer{
		Name: "data-backup-manager",
		BaseURL: func(context.Context) (string, error) {
			return fmt.Sprintf("http://localhost:%d", port), nil
		},
	}
	httpClient, baseURL, option, err := dbm.Dial(ctx)
	if err != nil {
		return err
	}
	client := targetsconnect.NewTargetsServiceClient(httpClient, baseURL, option)
	_, err = client.RegisterTarget(ctx, connect.NewRequest(&targetsv1.RegisterTargetRequest{
		Owner:      scenarioName,
		Name:       "research-home",
		SourceKind: sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM,
		Locator:    w.homeConfig.ResearchDir,
	}))
	return err
}

// webConsoleCredential is the desk's own Web Console credential. The desk has
// none yet: the service principal is prepared by
// scripts/issue-service-principal.sh, which the operator runs, and the client
// that reads it waits on a dependency install. Until then the
// desk sends no credential and Web Console's documented availability policy
// decides; with agents refused at the desk that gives the
// desk nothing a local operator process does not already have.
func webConsoleCredential(context.Context) (string, error) { return "", nil }

// automaticLaunch says how an automatic launch reaches Web Console, or why it
// cannot: Web Console is not running, or refused the desk's last call.
func (w *wiring) automaticLaunch(ctx context.Context) (string, error) {
	access, err := w.webConsole.Access(ctx)
	if err != nil {
		return "", err
	}
	if _, _, _, err := w.webConsole.Dial(ctx); err != nil {
		return access, err
	}
	return access, w.webConsole.Refused()
}

// courierRoute is the courier's lens in the overview.
const courierRoute = "courier"

// webConsoleAccess is how the desk reaches Web Console, for health and the
// overview: the kind and a plain sentence.
func (w *wiring) webConsoleAccess(ctx context.Context) wire.Access {
	access, err := w.automaticLaunch(ctx)
	a := wire.Access{Kind: wire.AccessKind(access), Automatic: err == nil}
	switch {
	case errors.Is(err, capabilities.ErrNotReached):
		a.Said = "Web Console access: " + access + "; Web Console was not reached just now (timed out), so whether automatic launch works is unknown until it answers; manual launch works"
	case access == "":
		a.Said = "Web Console access: unknown (" + err.Error() + "); automatic launch is off and manual launch is the whole flow"
	case err != nil:
		a.Said = "Web Console access: " + access + "; automatic launch is off and manual launch is the whole flow: " + err.Error()
	default:
		a.Said = "Web Console access: " + access
	}
	return a
}

// routes are the overview's lenses for what lies outside the home: each
// optional scenario as the capability registry reports it, with the
// command that starts it for the CLI, and the courier's health.
// Web Console access is its own dependency of the overview.
func (w *wiring) routes(ctx context.Context) []home.Lens {
	var out []home.Lens
	if w.registry == nil {
		out = append(out, home.Lens{Name: "capability registry", Reason: "not loaded: .vrooli/service.json not found, so no optional scenario is reached", Command: "set VROOLI_ROOT to the Vrooli checkout and restart research-desk"})
	} else {
		for _, state := range w.registry.Resolve(ctx) {
			lens := home.Lens{Name: state.DependencySlug, Available: state.Status == capabilities.StatusAvailable, Reason: state.Message}
			if !lens.Available && state.Status != capabilities.StatusUnknown {
				lens.Command = capabilities.StartHint(ctx, w.resolver, state.DependencySlug)
			}
			out = append(out, lens)
		}
	}
	if err := w.writer.Role(); err != nil {
		return append(out, home.Lens{Name: courierRoute, Reason: "this instance only reads (" + err.Error() + "); the live instance runs the courier"})
	}
	ok, said := w.courier.Health().Say()
	return append(out, home.Lens{Name: courierRoute, Available: ok, Reason: said})
}

// loadRegistry projects .vrooli/service.json; nil when it cannot be read,
// which every peer and the capabilities route then report.
func (w *wiring) loadRegistry(resolver capabilities.PortResolver) *capabilities.Registry {
	if w.scenarioDir == "" {
		return nil
	}
	defs, err := capabilities.Load(filepath.Join(w.scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		log.Printf("capabilities unavailable: %v", err)
		return nil
	}
	return capabilities.NewRegistry(defs, resolver)
}

// researchHomeReadable is the health check: the research home is a folder.
func (w *wiring) researchHomeReadable(context.Context) error {
	info, err := os.Stat(w.homeConfig.ResearchDir)
	if err != nil {
		return fmt.Errorf("research home %s is not readable: %w", w.homeConfig.ResearchDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("research home %s is not a folder", w.homeConfig.ResearchDir)
	}
	return nil
}
