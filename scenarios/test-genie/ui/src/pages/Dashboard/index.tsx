import { useMemo } from "react";
import { ContinueSection } from "./ContinueSection";
import { HeaderSection } from "./HeaderSection";
import { GuidedFlows } from "./GuidedFlows";
import { StatsSection } from "./StatsSection";
import { useHealth } from "../../hooks/useHealth";
import { useRequests } from "../../hooks/useRequests";
import { useExecutions } from "../../hooks/useExecutions";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import { formatRelative } from "../../lib/formatters";

export function DashboardPage() {
  const { setActiveTab, applyFocusScenario, setExecutionForm } = useUIStore();

  const { queueMetrics, heroExecution } = useHealth();
  const { actionableRequest, queuePendingCount } = useRequests();
  const { executions, lastFailedExecution } = useExecutions();
  const { scenarioDirectoryEntries, catalogStats } = useScenarios();

  // Get most recent scenario for "continue" feature
  const recentScenario = scenarioDirectoryEntries[0]?.scenarioName;
  const hasHistory = scenarioDirectoryEntries.length > 0 || executions.length > 0;

  // Navigation handlers
  const handleResumeDebugging = () => {
    if (lastFailedExecution) {
      applyFocusScenario(lastFailedExecution.scenarioName);
      setExecutionForm({
        scenarioName: lastFailedExecution.scenarioName,
        preset: lastFailedExecution.preset ?? "quick"
      });
    }
    setActiveTab("runs");
  };

  const handleRunQueuedTests = () => {
    if (actionableRequest) {
      applyFocusScenario(actionableRequest.scenarioName);
    }
    setActiveTab("runs");
  };

  const handleContinueScenario = () => {
    if (recentScenario) {
      applyFocusScenario(recentScenario);
    }
    setActiveTab("runs");
  };

  const handleGetStarted = () => {
    setActiveTab("runs");
  };

  const handleViewQueue = () => setActiveTab("runs");
  const handleViewHistory = () => setActiveTab("runs");
  const handleViewScenarios = () => setActiveTab("runs");

  const handleRerunExecution = () => {
    if (heroExecution) {
      applyFocusScenario(heroExecution.scenarioName);
      setExecutionForm({
        scenarioName: heroExecution.scenarioName,
        preset: heroExecution.preset ?? "quick"
      });
    }
    setActiveTab("runs");
  };

  // Build guided flow items
  const guidedFlows = useMemo(() => [
    {
      key: "browse",
      step: "Step 1",
      title: "Browse scenarios",
      description: "View all tracked scenarios and their test status.",
      stat: `${scenarioDirectoryEntries.length} tracked`,
      statLabel: scenarioDirectoryEntries.length > 0 ? "Scenarios" : "Start tracking",
      action: "View scenarios",
      onClick: () => setActiveTab("runs")
    },
    {
      key: "queue",
      step: "Step 2",
      title: "Request tests",
      description: "Generate new tests for your scenarios.",
      stat: `${queuePendingCount} pending`,
      statLabel: "In queue",
      action: "Request tests",
      onClick: () => setActiveTab("generate")
    },
    {
      key: "run",
      step: "Step 3",
      title: "Run tests",
      description: "Execute test suites with configurable presets.",
      stat: heroExecution
        ? heroExecution.success ? "Passed" : "Failed"
        : "No runs",
      statLabel: heroExecution
        ? formatRelative(heroExecution.completedAt)
        : "Run your first test",
      action: "Run tests",
      onClick: () => setActiveTab("runs")
    },
    {
      key: "docs",
      step: "Learn",
      title: "Read docs",
      description: "Learn about test phases and best practices.",
      stat: "Available",
      statLabel: "Documentation",
      action: "Open docs",
      onClick: () => setActiveTab("docs")
    }
  ], [scenarioDirectoryEntries.length, queuePendingCount, heroExecution, setActiveTab]);

  return (
    <div className="flex flex-col gap-6">
      <ContinueSection
        lastFailedExecution={lastFailedExecution}
        actionableRequest={actionableRequest}
        recentScenario={recentScenario}
        hasHistory={hasHistory}
        onResumeDebugging={handleResumeDebugging}
        onRunQueuedTests={handleRunQueuedTests}
        onContinueScenario={handleContinueScenario}
        onGetStarted={handleGetStarted}
      />

      <HeaderSection />

      <GuidedFlows items={guidedFlows} />

      <StatsSection
        queueMetrics={queueMetrics}
        catalogStats={catalogStats}
        heroExecution={heroExecution}
        actionableRequest={actionableRequest}
        lastFailedExecution={lastFailedExecution}
        onViewQueue={handleViewQueue}
        onViewHistory={handleViewHistory}
        onViewScenarios={handleViewScenarios}
        onRerunExecution={handleRerunExecution}
      />
    </div>
  );
}

export { ContinueSection, HeaderSection, GuidedFlows, StatsSection };
