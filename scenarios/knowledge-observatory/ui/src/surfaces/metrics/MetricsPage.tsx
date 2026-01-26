// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { Database } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { MetricsPanelContainer } from "./MetricsPanelContainer";

export type MetricsPageProps = {
  onNavigate: (route: Route) => void;
};

export function MetricsPage({ onNavigate }: MetricsPageProps) {
  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Metrics Panel Unavailable"
            description="The metrics UI failed to render. You can retry this section or return to the dashboard."
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
            title="Quality Metrics"
            description="Track coherence, freshness, redundancy, and coverage to spot drift or gaps."
            icon={<Database className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />
          <MetricsPanelContainer />
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
