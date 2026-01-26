// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { GitGraph } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";

export type GraphPageProps = {
  onNavigate: (route: Route) => void;
};

export function GraphPage({ onNavigate }: GraphPageProps) {
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
            description="Visualize how concepts connect and where semantic clusters emerge."
            icon={<GitGraph className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />
          <div className="ko-card text-center p-12" data-testid={selectors.graph.emptyState}>
            <GitGraph className="h-16 w-16 ko-icon-strong mx-auto mb-4" />
            <p className="ko-muted">Graph visualization UI is not implemented yet</p>
            <p className="ko-text-sm ko-subtle mt-2">
              This page is reserved for exploring semantic relationships once the graph API + UI are wired.
            </p>
          </div>
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
