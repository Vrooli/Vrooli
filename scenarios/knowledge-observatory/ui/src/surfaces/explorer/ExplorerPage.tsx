// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/reference/api-endpoints.md#scenario-list
import { ClipboardList, FolderTree, ShieldCheck } from "lucide-react";
import { useMemo } from "react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { useScenarioExplorer } from "../../shared/hooks/explorerHooks";
import { Button } from "../../shared/ui/button";
import { DocTree } from "./components/DocTree";
import { HealthPanel } from "./components/HealthPanel";
import { ScenarioList } from "./components/ScenarioList";

export type ExplorerPageProps = {
  onNavigate: (route: Route) => void;
};

export function ExplorerPage({ onNavigate }: ExplorerPageProps) {
  const {
    filter,
    setFilter,
    scenarios,
    selectedScenario,
    setSelectedScenario,
    selectedDocPath,
    setSelectedDocPath,
    docTree,
    healthViewModel,
    scenariosState,
    docTreeState,
    docHealthState,
  } = useScenarioExplorer();

  const selectedNode = useMemo(() => {
    if (!docTree || !selectedDocPath) return null;
    const stack = [docTree];
    while (stack.length > 0) {
      const node = stack.pop();
      if (!node) continue;
      if (node.path === selectedDocPath) return node;
      if (node.children) {
        stack.push(...node.children);
      }
    }
    return null;
  }, [docTree, selectedDocPath]);

  const canOpenViewer = selectedNode?.type === "file";
  const openViewer = () => {
    if (!selectedDocPath || !canOpenViewer) return;
    if (typeof window !== "undefined") {
      window.location.hash = `#/viewer?path=${encodeURIComponent(selectedDocPath)}`;
    }
  };

  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Scenario Explorer Unavailable"
            description="The explorer UI encountered an unexpected error. You can retry or return to the dashboard."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell className="ko-explorer-shell">
        <div className="ko-explorer-grid">
          <Panel className="ko-explorer-panel">
            <PanelHeader
              title="Scenarios"
              description="Browse scenario documentation health at a glance."
              icon={<ClipboardList className="h-5 w-5 ko-icon" />}
            />
            <ScenarioList
              scenarios={scenarios}
              filter={filter}
              onFilterChange={setFilter}
              selectedScenario={selectedScenario}
              onSelectScenario={setSelectedScenario}
              isLoading={scenariosState.isLoading}
              hasError={scenariosState.hasError}
              errorMessage={scenariosState.errorMessage}
              onRefresh={scenariosState.refetch}
            />
          </Panel>

          <Panel className="ko-explorer-panel">
            <PanelHeader
              title="Documentation Tree"
              description="Inspect files, doc types, and warnings in context."
              icon={<FolderTree className="h-5 w-5 ko-icon" />}
            />
            <DocTree
              tree={docTree}
              selectedPath={selectedDocPath}
              onSelectPath={setSelectedDocPath}
              isLoading={docTreeState.isLoading}
              hasError={docTreeState.hasError}
              errorMessage={docTreeState.errorMessage}
              onRefresh={docTreeState.refetch}
            />
            <div className="mt-4 flex justify-end">
              <Button type="button" variant="outline" size="compact" onClick={openViewer} disabled={!canOpenViewer}>
                Open in Viewer
              </Button>
            </div>
          </Panel>

          <Panel className="ko-explorer-panel">
            <PanelHeader
              title="Documentation Health"
              description="Track missing, misplaced, and extra docs."
              icon={<ShieldCheck className="h-5 w-5 ko-icon" />}
            />
            <HealthPanel
              scenarioName={selectedScenario}
              healthViewModel={healthViewModel}
              isLoading={docHealthState.isLoading}
              hasError={docHealthState.hasError}
              errorMessage={docHealthState.errorMessage}
              onRefresh={docHealthState.refetch}
            />
          </Panel>
        </div>
      </PageShell>
    </ErrorBoundary>
  );
}
