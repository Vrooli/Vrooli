// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/reference/api-endpoints.md#scenario-list
import { ClipboardList, FolderTree } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { ResizeHandle } from "../../shared/components/ResizeHandle";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { buildDocMetaViewModel } from "../../shared/controllers/viewerController";
import { useScenarioExplorer } from "../../shared/hooks/explorerHooks";
import { useResizableLayout } from "../../shared/hooks/useResizableLayout";
import { useIsMobileOrTablet } from "../../shared/hooks/useViewportSize";
import type { DocViewMode } from "../../shared/hooks/viewerHooks";
import { fetchDocContent } from "../../shared/services/documentationApi";
import { DocTree } from "./components/DocTree";
import { DocViewer } from "./components/DocViewer";
import { HealthModal } from "./components/HealthModal";
import { MobileHeader } from "./components/MobileHeader";
import { ScenarioList } from "./components/ScenarioList";

export type ExplorerPageProps = {
  onNavigate: (route: Route) => void;
};

type MobileView = "scenarios" | "tree" | "viewer";

export function ExplorerPage({ onNavigate }: ExplorerPageProps) {
  const isMobileOrTablet = useIsMobileOrTablet();
  const { state: layoutState, handleScenarioResize, handleViewerResize } = useResizableLayout();

  // Mobile navigation state
  const [mobileView, setMobileView] = useState<MobileView>("scenarios");

  // Modal state
  const [isHealthModalOpen, setIsHealthModalOpen] = useState(false);

  // Viewer state
  const [viewerPath, setViewerPath] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<DocViewMode>("preview");

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

  // Fetch document content when viewing
  const docContentQuery = useQuery({
    queryKey: ["docContent", viewerPath],
    queryFn: () => fetchDocContent(viewerPath ?? "", "raw"),
    enabled: Boolean(viewerPath?.trim()),
  });

  const docMeta = useMemo(
    () => buildDocMetaViewModel(docContentQuery.data),
    [docContentQuery.data]
  );

  // Handle scenario selection
  const handleSelectScenario = useCallback(
    (name: string) => {
      setSelectedScenario(name);
      setViewerPath(null);
      if (isMobileOrTablet) {
        setMobileView("tree");
      }
    },
    [setSelectedScenario, isMobileOrTablet]
  );

  // Handle document selection (open in viewer)
  const handleSelectDoc = useCallback(
    (path: string) => {
      setSelectedDocPath(path);
      setViewerPath(path);
      if (isMobileOrTablet) {
        setMobileView("viewer");
      }
    },
    [setSelectedDocPath, isMobileOrTablet]
  );

  // Handle back navigation from viewer to tree
  const handleBackToTree = useCallback(() => {
    setViewerPath(null);
    if (isMobileOrTablet) {
      setMobileView("tree");
    }
  }, [isMobileOrTablet]);

  // Handle back navigation from tree to scenarios (mobile only)
  const handleBackToScenarios = useCallback(() => {
    setMobileView("scenarios");
  }, []);

  // Render mobile view
  if (isMobileOrTablet) {
    return (
      <ErrorBoundary
        fallback={({ error, reset }) => (
          <PageShell>
            <SectionErrorState
              title="Scenario Explorer Unavailable"
              description="The explorer UI encountered an unexpected error."
              errorMessage={error.message}
              actions={[
                { label: "Retry", onClick: reset },
                { label: "Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
              ]}
            />
          </PageShell>
        )}
      >
        <PageShell variant="full-viewport" className="ko-explorer-shell">
          {mobileView === "scenarios" && (
            <div className="ko-mobile-view">
              <MobileHeader title="Scenarios" showBack={false} />
              <div className="ko-scroll-container p-4">
                <ScenarioList
                  scenarios={scenarios}
                  filter={filter}
                  onFilterChange={setFilter}
                  selectedScenario={selectedScenario}
                  onSelectScenario={handleSelectScenario}
                  isLoading={scenariosState.isLoading}
                  hasError={scenariosState.hasError}
                  errorMessage={scenariosState.errorMessage}
                  onRefresh={scenariosState.refetch}
                />
              </div>
            </div>
          )}

          {mobileView === "tree" && (
            <div className="ko-mobile-view">
              <MobileHeader
                title={selectedScenario ?? "Documentation"}
                onBack={handleBackToScenarios}
              />
              <div className="ko-scroll-container p-4">
                <DocTree
                  tree={docTree}
                  selectedPath={selectedDocPath}
                  onSelectPath={handleSelectDoc}
                  isLoading={docTreeState.isLoading}
                  hasError={docTreeState.hasError}
                  errorMessage={docTreeState.errorMessage}
                  onRefresh={docTreeState.refetch}
                  healthScoreLabel={healthViewModel.healthScoreLabel}
                  healthTone={healthViewModel.healthTone}
                  onHealthClick={() => setIsHealthModalOpen(true)}
                />
              </div>
            </div>
          )}

          {mobileView === "viewer" && viewerPath && (
            <DocViewer
              path={viewerPath}
              content={docContentQuery.data?.content}
              isLoading={docContentQuery.isLoading}
              hasError={Boolean(docContentQuery.error)}
              errorMessage={
                docContentQuery.error instanceof Error
                  ? docContentQuery.error.message
                  : "Unable to load document."
              }
              onRefresh={() => docContentQuery.refetch()}
              onBack={handleBackToTree}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              showBackButton
              meta={docMeta}
            />
          )}

          <HealthModal
            isOpen={isHealthModalOpen}
            onClose={() => setIsHealthModalOpen(false)}
            scenarioName={selectedScenario}
            healthViewModel={healthViewModel}
            isLoading={docHealthState.isLoading}
            hasError={docHealthState.hasError}
            errorMessage={docHealthState.errorMessage}
            onRefresh={docHealthState.refetch}
          />
        </PageShell>
      </ErrorBoundary>
    );
  }

  // Render desktop 2-column layout
  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Scenario Explorer Unavailable"
            description="The explorer UI encountered an unexpected error."
            errorMessage={error.message}
            actions={[
              { label: "Retry", onClick: reset },
              { label: "Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell variant="full-viewport" className="ko-explorer-shell">
        <div className="ko-explorer-columns">
          {/* Left column: Scenarios */}
          <div
            className="ko-explorer-column"
            style={{ width: layoutState.scenarioColumnWidth }}
          >
            <Panel className="flex flex-col h-full">
              <PanelHeader
                title="Scenarios"
                description="Browse scenario documentation"
                icon={<ClipboardList className="h-5 w-5 ko-icon" />}
              />
              <div className="ko-scroll-container p-4">
                <ScenarioList
                  scenarios={scenarios}
                  filter={filter}
                  onFilterChange={setFilter}
                  selectedScenario={selectedScenario}
                  onSelectScenario={handleSelectScenario}
                  isLoading={scenariosState.isLoading}
                  hasError={scenariosState.hasError}
                  errorMessage={scenariosState.errorMessage}
                  onRefresh={scenariosState.refetch}
                />
              </div>
            </Panel>
          </div>

          <ResizeHandle direction="vertical" onResize={handleScenarioResize} />

          {/* Right column: Tree or Viewer */}
          <div className="ko-explorer-column" style={{ flex: 1 }}>
            <Panel className="flex flex-col h-full">
              {viewerPath ? (
                // Show viewer when a document is selected
                <DocViewer
                  path={viewerPath}
                  content={docContentQuery.data?.content}
                  isLoading={docContentQuery.isLoading}
                  hasError={Boolean(docContentQuery.error)}
                  errorMessage={
                    docContentQuery.error instanceof Error
                      ? docContentQuery.error.message
                      : "Unable to load document."
                  }
                  onRefresh={() => docContentQuery.refetch()}
                  onBack={handleBackToTree}
                  viewMode={viewMode}
                  onViewModeChange={setViewMode}
                  splitRatio={layoutState.viewerSplitRatio}
                  onSplitResize={handleViewerResize}
                  showBackButton
                  meta={docMeta}
                />
              ) : (
                // Show tree when no document is selected
                <>
                  <PanelHeader
                    title="Documentation Tree"
                    description="Inspect files, doc types, and warnings"
                    icon={<FolderTree className="h-5 w-5 ko-icon" />}
                  />
                  <div className="ko-scroll-container p-4">
                    <DocTree
                      tree={docTree}
                      selectedPath={selectedDocPath}
                      onSelectPath={handleSelectDoc}
                      isLoading={docTreeState.isLoading}
                      hasError={docTreeState.hasError}
                      errorMessage={docTreeState.errorMessage}
                      onRefresh={docTreeState.refetch}
                      healthScoreLabel={healthViewModel.healthScoreLabel}
                      healthTone={healthViewModel.healthTone}
                      onHealthClick={() => setIsHealthModalOpen(true)}
                    />
                  </div>
                </>
              )}
            </Panel>
          </div>
        </div>

        <HealthModal
          isOpen={isHealthModalOpen}
          onClose={() => setIsHealthModalOpen(false)}
          scenarioName={selectedScenario}
          healthViewModel={healthViewModel}
          isLoading={docHealthState.isLoading}
          hasError={docHealthState.hasError}
          errorMessage={docHealthState.errorMessage}
          onRefresh={docHealthState.refetch}
        />
      </PageShell>
    </ErrorBoundary>
  );
}
