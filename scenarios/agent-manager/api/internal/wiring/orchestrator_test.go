package wiring

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
)

func testDurableRoots(t *testing.T) *filerouting.RoutedRoots {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve user home: %v", err)
	}
	base, err := os.MkdirTemp(home, "agent-manager-wiring-")
	if err != nil {
		t.Fatalf("create durable test root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	return filerouting.New(corestorage.Paths{StateDir: filepath.Join(base, "state")})
}

func TestNewOrchestratorRejectsMissingRequiredCompositionDependencies(t *testing.T) {
	if _, err := NewOrchestrator(nil, nil, logrus.New(), nil, nil, testDurableRoots(t)); err == nil {
		t.Fatal("nil database accepted")
	}
	if _, err := NewOrchestrator(&database.DB{}, nil, nil, nil, nil, testDurableRoots(t)); err == nil {
		t.Fatal("nil logger accepted")
	}
	if _, err := NewOrchestrator(&database.DB{}, nil, logrus.New(), nil, nil); err == nil {
		t.Fatal("missing durable file roots accepted")
	}
}

func TestResolveWorkspaceSandboxURLHonorsExplicitConfiguration(t *testing.T) {
	t.Setenv("WORKSPACE_SANDBOX_URL", "http://sandbox.example:15427")
	if got := resolveWorkspaceSandboxURL(); got != "http://sandbox.example:15427" {
		t.Fatalf("resolved URL=%q", got)
	}
}

func TestNewRunnersRegistersEverySupportedType(t *testing.T) {
	runners := NewRunners()
	if runners.Registry == nil {
		t.Fatal("runner registry is nil")
	}
	if got, want := len(runners.Registry.List()), len(domain.ValidRunnerTypes()); got != want {
		t.Fatalf("registered runners=%d, want %d", got, want)
	}
}

func TestNewOrchestratorBuildsCompleteGraphWithoutStartingWorkers(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	deps, err := NewOrchestrator(db, nil, logrus.New(), nil, nil, testDurableRoots(t))
	if err != nil {
		t.Fatalf("build orchestrator graph: %v", err)
	}
	if deps.Orchestrator == nil || deps.Reconciler == nil || deps.AwaitRegistry == nil || deps.WorkflowNudger == nil {
		t.Fatalf("runtime graph missing lifecycle dependencies: %+v", deps)
	}
	if deps.StatsService == nil || deps.StatsRepository == nil || deps.PricingService == nil || deps.ModelHealthProbe == nil || deps.HealthStore == nil || deps.EventRepository == nil {
		t.Fatalf("runtime graph missing operational dependencies: %+v", deps)
	}
	if deps.RolePolicyState == nil || deps.PermissionPolicyState == nil || deps.PermissionPolicy == nil || deps.StatsEngine == nil {
		t.Fatalf("runtime graph missing policy or stats dependencies: %+v", deps)
	}

	// Construction must not start workers: cleanup is safe before Server.Start
	// has established lifecycle ownership.
	Shutdown(nil, deps.Reconciler, deps.AwaitRegistry, deps.WorkflowNudger, deps.TranscriptImporter, deps.FrictionPublisher)
}

func TestNewOrchestratorHonorsExplicitLevers(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	levers := config.DefaultLevers()
	levers.Workflow.NudgeWorkers = 1
	deps, err := NewOrchestrator(db, nil, logrus.New(), nil, &levers, testDurableRoots(t))
	if err != nil {
		t.Fatalf("build orchestrator graph with levers: %v", err)
	}
	if deps.Orchestrator == nil {
		t.Fatal("orchestrator is nil")
	}
	Shutdown(nil, deps.Reconciler, deps.AwaitRegistry, deps.WorkflowNudger, deps.TranscriptImporter, deps.FrictionPublisher)
}
