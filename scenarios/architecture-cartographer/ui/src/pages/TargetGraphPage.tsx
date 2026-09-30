import * as React from "react";
import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { GraphAccessibleList } from "../features/graph/GraphAccessibleList";
import { GraphCanvas } from "../features/graph/GraphCanvas";
import { GraphFilterBar } from "../features/graph/GraphFilterBar";
import { GraphLegend } from "../features/graph/GraphLegend";
import { useGraphWorkspace } from "../features/graph/controllers/useGraphController";
import { buildGraphLayout } from "../features/graph/lib/graphAdapter";

export function TargetGraphPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const [domainFilter, setDomainFilter] = React.useState<ReadonlySet<string>>(
    () => new Set<string>(),
  );

  // Hooks must run unconditionally — pass an empty scenario when the URL is
  // malformed; the controllers gate themselves on `scenario.length > 0`.
  const workspaceScenario = scenario ?? "";
  const workspace = useGraphWorkspace(workspaceScenario);

  const snapshot = workspace.snapshot.data;
  const conflictsData = workspace.conflicts.data;
  const layout = React.useMemo(
    () =>
      buildGraphLayout(
        snapshot ?? undefined,
        conflictsData?.conflicts ?? [],
        domainFilter,
      ),
    [snapshot, conflictsData, domainFilter],
  );

  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetGraph}
      aria-labelledby="target-graph-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-graph-heading" className="text-xl font-semibold">
          {t(strings.pages.targetGraph.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetGraph.description)}
        </p>
      </header>

      <GraphFilterBar
        scenario={scenario}
        selected={domainFilter}
        onChange={setDomainFilter}
      />

      {workspace.snapshot.isPending ? (
        <LoadingState label={t(strings.pages.targetGraph.loading)} />
      ) : workspace.snapshot.isError ? (
        <ErrorState
          title={t(strings.pages.targetGraph.errorTitle)}
          message={workspace.snapshot.error instanceof Error
            ? workspace.snapshot.error.message
            : String(workspace.snapshot.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void workspace.snapshot.refetch();
          }}
        />
      ) : (
        <ErrorBoundary
          fallback={
            <div data-testid={selectors.features.graph.canvas.fallback}>
              <ErrorState
                title={t(strings.pages.targetGraph.fallbackTitle)}
                message={t(strings.pages.targetGraph.fallbackMessage)}
              />
            </div>
          }
        >
          <div className="grid gap-3 md:grid-cols-[1fr,minmax(0,18rem)]">
            <GraphCanvas layout={layout} scenario={scenario} />
            <div className="flex flex-col gap-3">
              <GraphLegend />
              {/* Always present in the DOM. The list is the AC-UI-GRAPH-A11Y
                  text alternative — sr-only on desktop (the SVG canvas is the
                  primary surface) and visible on mobile where the canvas
                  collapses. The `sr-only` Tailwind class strips visibility
                  while keeping the element accessible to screen readers. */}
              <GraphAccessibleList
                layout={layout}
                className="md:sr-only"
              />
            </div>
          </div>
        </ErrorBoundary>
      )}
    </section>
  );
}
