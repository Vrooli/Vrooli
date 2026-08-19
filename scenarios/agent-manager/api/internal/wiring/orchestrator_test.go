package wiring

import (
	"testing"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/sirupsen/logrus"
)

func TestNewOrchestratorRejectsMissingRequiredCompositionDependencies(t *testing.T) {
	if _, err := NewOrchestrator(nil, nil, logrus.New(), nil, nil); err == nil {
		t.Fatal("nil database accepted")
	}
	if _, err := NewOrchestrator(&database.DB{}, nil, nil, nil, nil); err == nil {
		t.Fatal("nil logger accepted")
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

	deps, err := NewOrchestrator(db, nil, logrus.New(), nil, nil)
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
	Shutdown(nil, deps.Reconciler, deps.AwaitRegistry, deps.WorkflowNudger, deps.TranscriptImporter)
}

func TestNewOrchestratorHonorsExplicitLevers(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	levers := config.DefaultLevers()
	levers.Workflow.NudgeWorkers = 1
	deps, err := NewOrchestrator(db, nil, logrus.New(), nil, &levers)
	if err != nil {
		t.Fatalf("build orchestrator graph with levers: %v", err)
	}
	if deps.Orchestrator == nil {
		t.Fatal("orchestrator is nil")
	}
	Shutdown(nil, deps.Reconciler, deps.AwaitRegistry, deps.WorkflowNudger, deps.TranscriptImporter)
}
