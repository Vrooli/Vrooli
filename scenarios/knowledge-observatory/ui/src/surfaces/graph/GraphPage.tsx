// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { useEffect, useMemo, useState } from "react";
import { GitGraph, Loader2, AlertCircle, Network } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import { useKnowledgeGraphController } from "../../shared/hooks/knowledgeHooks";
import { Button } from "../../shared/ui/button";
import type { Route } from "../../shared/controllers/routeController";
import { GraphCanvas } from "./components/GraphCanvas";
import {
  buildCanvasRenderModel,
  fitViewportToRenderModel,
  mergeCanvasGraphs,
  toCanvasGraph,
  type CanvasGraph,
  type CanvasViewport,
  type GraphLayoutMode,
} from "./graphCanvasModel";

export type GraphPageProps = {
  onNavigate: (route: Route) => void;
};

const DEFAULT_MIN_WEIGHT = "0.25";

const parseWeight = (value: string) => {
  const parsed = Number.parseFloat(value);
  if (!Number.isFinite(parsed)) return 0;
  return Math.min(1, Math.max(0, parsed));
};

export function GraphPage({ onNavigate }: GraphPageProps) {
  const graph = useKnowledgeGraphController();

  const [layoutMode, setLayoutMode] = useState<GraphLayoutMode>("radial");
  const [minWeightInput, setMinWeightInput] = useState(DEFAULT_MIN_WEIGHT);
  const [highlightNeighbors, setHighlightNeighbors] = useState(true);
  const [expandEnabled, setExpandEnabled] = useState(true);
  const [canvasGraph, setCanvasGraph] = useState<CanvasGraph | null>(null);
  const [selectedNodeID, setSelectedNodeID] = useState<string | undefined>(undefined);
  const [viewport, setViewport] = useState<CanvasViewport>({ scale: 1, offsetX: 0, offsetY: 0 });
  const [shouldAutoFit, setShouldAutoFit] = useState(false);
  const [isExpanding, setIsExpanding] = useState(false);

  useEffect(() => {
    if (!graph.graphData) {
      setCanvasGraph(null);
      setSelectedNodeID(undefined);
      setViewport({ scale: 1, offsetX: 0, offsetY: 0 });
      return;
    }

    const parsed = toCanvasGraph(graph.graphData);
    setCanvasGraph(parsed);
    setSelectedNodeID(parsed.centerNodeID);
    setShouldAutoFit(true);
  }, [graph.graphData]);

  const renderModel = useMemo(() => {
    if (!canvasGraph) return null;
    return buildCanvasRenderModel({
      graph: canvasGraph,
      layoutMode,
      minWeight: parseWeight(minWeightInput),
      selectedNodeID,
      highlightNeighbors,
    });
  }, [canvasGraph, layoutMode, minWeightInput, selectedNodeID, highlightNeighbors]);

  useEffect(() => {
    if (!renderModel || !shouldAutoFit) return;
    setViewport(fitViewportToRenderModel(renderModel));
    setShouldAutoFit(false);
  }, [renderModel, shouldAutoFit]);

  const handleClear = () => {
    graph.clear();
    setLayoutMode("radial");
    setMinWeightInput(DEFAULT_MIN_WEIGHT);
    setHighlightNeighbors(true);
    setExpandEnabled(true);
    setCanvasGraph(null);
    setSelectedNodeID(undefined);
    setViewport({ scale: 1, offsetX: 0, offsetY: 0 });
    setShouldAutoFit(false);
  };

  const handleExpand = async () => {
    if (!canvasGraph || !selectedNodeID || !expandEnabled) return;
    const anchor = canvasGraph.nodes.find((node) => node.id === selectedNodeID);
    if (!anchor) return;

    setIsExpanding(true);
    try {
      const expanded = await graph.queryGraph(anchor.label);
      const merged = mergeCanvasGraphs({
        base: canvasGraph,
        incoming: toCanvasGraph(expanded),
        anchorNodeID: anchor.id,
      });
      setCanvasGraph(merged);
      setShouldAutoFit(true);
    } catch {
      // Expansion failures are surfaced through existing form error path on next submit.
    } finally {
      setIsExpanding(false);
    }
  };

  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Graph View Unavailable"
            description="The graph view hit an unexpected error. Retry or return to the dashboard."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell>
        <Panel>
          <PanelHeader
            title="Knowledge Graph"
            description="Generate a concept-centered graph from semantic similarity and inspect nodes, edges, and metadata."
            icon={<GitGraph className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />

          <div className="ko-stack-sm" data-testid={selectors.graph.container}>
            <form
              className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3"
              onSubmit={graph.submit}
              data-testid={selectors.graph.form}
            >
              <label className="ko-stack-xs">
                <span className="ko-text-xs ko-subtle">Center Concept</span>
                <input
                  className="ko-input"
                  placeholder="e.g. semantic drift"
                  value={graph.centerConcept}
                  onChange={(event) => graph.setCenterConcept(event.target.value)}
                  data-testid={selectors.graph.centerInput}
                />
              </label>

              <label className="ko-stack-xs">
                <span className="ko-text-xs ko-subtle">Collection (optional)</span>
                <input
                  className="ko-input"
                  placeholder="knowledge_chunks_v1"
                  value={graph.collection}
                  onChange={(event) => graph.setCollection(event.target.value)}
                  data-testid={selectors.graph.collectionInput}
                />
              </label>

              <label className="ko-stack-xs">
                <span className="ko-text-xs ko-subtle">Visibility (CSV)</span>
                <input
                  className="ko-input"
                  placeholder="shared,global"
                  value={graph.visibilityInput}
                  onChange={(event) => graph.setVisibilityInput(event.target.value)}
                  data-testid={selectors.graph.visibilityInput}
                />
              </label>

              <label className="ko-stack-xs">
                <span className="ko-text-xs ko-subtle">Namespaces (CSV)</span>
                <input
                  className="ko-input"
                  placeholder="knowledge-observatory"
                  value={graph.namespacesInput}
                  onChange={(event) => graph.setNamespacesInput(event.target.value)}
                  data-testid={selectors.graph.namespacesInput}
                />
              </label>

              <label className="ko-stack-xs">
                <span className="ko-text-xs ko-subtle">Tags (CSV)</span>
                <input
                  className="ko-input"
                  placeholder="docs,search"
                  value={graph.tagsInput}
                  onChange={(event) => graph.setTagsInput(event.target.value)}
                  data-testid={selectors.graph.tagsInput}
                />
              </label>

              <div className="grid grid-cols-3 gap-3">
                <label className="ko-stack-xs">
                  <span className="ko-text-xs ko-subtle">Depth</span>
                  <input
                    className="ko-input"
                    type="number"
                    min={1}
                    max={3}
                    value={graph.depthInput}
                    onChange={(event) => graph.setDepthInput(event.target.value)}
                    data-testid={selectors.graph.depthInput}
                  />
                </label>
                <label className="ko-stack-xs">
                  <span className="ko-text-xs ko-subtle">Limit</span>
                  <input
                    className="ko-input"
                    type="number"
                    min={1}
                    max={200}
                    value={graph.limitInput}
                    onChange={(event) => graph.setLimitInput(event.target.value)}
                    data-testid={selectors.graph.limitInput}
                  />
                </label>
                <label className="ko-stack-xs">
                  <span className="ko-text-xs ko-subtle">Threshold</span>
                  <input
                    className="ko-input"
                    type="number"
                    min={0.01}
                    max={1}
                    step={0.01}
                    value={graph.thresholdInput}
                    onChange={(event) => graph.setThresholdInput(event.target.value)}
                    data-testid={selectors.graph.thresholdInput}
                  />
                </label>
              </div>

              <div className="md:col-span-2 lg:col-span-3 flex flex-wrap gap-2">
                <Button type="submit" disabled={graph.isSubmitDisabled} data-testid={selectors.graph.submit}>
                  {graph.isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Network className="h-4 w-4" />}
                  <span className="ml-2">Generate Graph</span>
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleClear}
                  disabled={graph.isClearDisabled}
                  data-testid={selectors.graph.clear}
                >
                  Clear
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={graph.refetch}
                  disabled={!graph.hasData || graph.isLoading}
                  data-testid={selectors.graph.refresh}
                >
                  Refresh
                </Button>
              </div>
            </form>

            {graph.hasError && (
              <div className="ko-alert ko-alert-danger" data-testid={selectors.graph.error}>
                <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
                <div>
                  <p className="ko-alert-title ko-text-danger-strong">Graph Request Failed</p>
                  <p className="ko-text-sm ko-text-danger-muted mt-1">{graph.viewModel.errorMessage}</p>
                </div>
              </div>
            )}

            {graph.hasData && renderModel && (
              <div className="ko-stack-sm" data-testid={selectors.graph.results}>
                <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
                  <div className="ko-card p-3">
                    <p className="ko-text-xs ko-subtle">Center</p>
                    <p className="ko-text-sm ko-text-primary">{graph.viewModel.center}</p>
                  </div>
                  <div className="ko-card p-3">
                    <p className="ko-text-xs ko-subtle">Nodes</p>
                    <p className="ko-text-sm ko-text-primary">{graph.viewModel.nodeCount}</p>
                  </div>
                  <div className="ko-card p-3">
                    <p className="ko-text-xs ko-subtle">Edges</p>
                    <p className="ko-text-sm ko-text-primary">{graph.viewModel.edgeCount}</p>
                  </div>
                  <div className="ko-card p-3">
                    <p className="ko-text-xs ko-subtle">Generated</p>
                    <p className="ko-text-sm ko-text-primary">{graph.viewModel.tookMsLabel}</p>
                  </div>
                </div>

                <div className="ko-card p-4" data-testid={selectors.graph.legend}>
                  <div className="grid grid-cols-1 lg:grid-cols-5 gap-3 items-end">
                    <div className="ko-stack-xs">
                      <span className="ko-text-xs ko-subtle">Minimum edge weight</span>
                      <input
                        className="ko-input"
                        type="number"
                        min={0}
                        max={1}
                        step={0.01}
                        value={minWeightInput}
                        onChange={(event) => setMinWeightInput(event.target.value)}
                        data-testid={selectors.graph.minWeightInput}
                      />
                    </div>

                    <div className="ko-stack-xs">
                      <span className="ko-text-xs ko-subtle">Layout</span>
                      <div className="flex gap-2">
                        <Button
                          type="button"
                          size="compact"
                          variant={layoutMode === "radial" ? "primary" : "outline"}
                          onClick={() => {
                            setLayoutMode("radial");
                            setShouldAutoFit(true);
                          }}
                          data-testid={selectors.graph.layoutRadial}
                        >
                          Radial
                        </Button>
                        <Button
                          type="button"
                          size="compact"
                          variant={layoutMode === "force" ? "primary" : "outline"}
                          onClick={() => {
                            setLayoutMode("force");
                            setShouldAutoFit(true);
                          }}
                          data-testid={selectors.graph.layoutForce}
                        >
                          Force
                        </Button>
                        <Button
                          type="button"
                          size="compact"
                          variant={layoutMode === "column" ? "primary" : "outline"}
                          onClick={() => {
                            setLayoutMode("column");
                            setShouldAutoFit(true);
                          }}
                          data-testid={selectors.graph.layoutColumn}
                        >
                          Column
                        </Button>
                      </div>
                    </div>

                    <label className="ko-checkbox-row">
                      <input
                        type="checkbox"
                        checked={highlightNeighbors}
                        onChange={(event) => setHighlightNeighbors(event.target.checked)}
                        data-testid={selectors.graph.highlightNeighbors}
                      />
                      <span>Highlight selected node neighborhood</span>
                    </label>

                    <label className="ko-checkbox-row">
                      <input
                        type="checkbox"
                        checked={expandEnabled}
                        onChange={(event) => setExpandEnabled(event.target.checked)}
                        data-testid={selectors.graph.expandToggle}
                      />
                      <span>Enable incremental expand-on-select</span>
                    </label>

                    <div className="ko-text-xs ko-subtle">
                      <p>Center node: larger blue core.</p>
                      <p>Edge thickness: semantic weight.</p>
                    </div>
                  </div>
                </div>

                {!graph.viewModel.hasGraph ? (
                  <div className="ko-card p-6 text-center" data-testid={selectors.graph.emptyState}>
                    <p className="ko-muted">No graph nodes found for the current filters.</p>
                    <p className="ko-text-sm ko-subtle mt-1">Try a broader concept or lower threshold.</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 2xl:grid-cols-3 gap-4">
                    <div className="2xl:col-span-2">
                      <GraphCanvas
                        model={renderModel}
                        viewport={viewport}
                        onViewportChange={setViewport}
                        selectedNodeID={selectedNodeID}
                        onSelectNode={setSelectedNodeID}
                        onFit={() => setViewport(fitViewportToRenderModel(renderModel))}
                        onReset={() => {
                          setSelectedNodeID(canvasGraph?.centerNodeID);
                          setViewport({ scale: 1, offsetX: 0, offsetY: 0 });
                          setShouldAutoFit(true);
                        }}
                        canExpand={expandEnabled}
                        isExpanding={isExpanding}
                        onExpand={handleExpand}
                      />
                    </div>

                    <div className="ko-stack-sm">
                      <div className="ko-card p-4" data-testid={selectors.graph.nodes}>
                        <h3 className="ko-text-sm ko-text-strong mb-3">Nodes</h3>
                        <div className="ko-stack-xs max-h-96 overflow-auto pr-1">
                          {graph.viewModel.nodes.map((node) => (
                            <button
                              key={node.id}
                              type="button"
                              className="ko-panel-inset p-3 rounded w-full text-left"
                              onClick={() => setSelectedNodeID(node.id)}
                            >
                              <div className="flex items-center justify-between gap-2">
                                <p className="ko-text-sm ko-text-primary truncate">{node.label}</p>
                                <span className="ko-text-xs ko-subtle">Score: {node.scoreLabel}</span>
                              </div>
                              <p className="ko-text-xs ko-subtle mt-1 font-mono">ID: {node.id}</p>
                            </button>
                          ))}
                        </div>
                      </div>

                      <div className="ko-card p-4" data-testid={selectors.graph.edges}>
                        <h3 className="ko-text-sm ko-text-strong mb-3">Edges</h3>
                        <div className="ko-stack-xs max-h-96 overflow-auto pr-1">
                          {graph.viewModel.edges.map((edge, index) => (
                            <div key={`${edge.source}:${edge.target}:${index}`} className="ko-panel-inset p-3 rounded">
                              <p className="ko-text-sm ko-text-primary">
                                {edge.source} → {edge.target}
                              </p>
                              <p className="ko-text-xs ko-subtle mt-1">
                                Weight {edge.weightLabel} · {edge.relationship}
                              </p>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {!graph.hasData && !graph.hasError && !graph.isLoading && (
              <div className="ko-card text-center p-10" data-testid={selectors.graph.emptyState}>
                <GitGraph className="h-14 w-14 ko-icon-strong mx-auto mb-4" />
                <p className="ko-muted">Generate a graph to inspect semantic relationships.</p>
                <p className="ko-text-sm ko-subtle mt-2">
                  Choose a center concept, tune filters, and submit to query the graph API.
                </p>
              </div>
            )}
          </div>
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
