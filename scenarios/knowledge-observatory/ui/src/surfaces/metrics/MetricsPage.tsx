import { Database } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
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
        <section className="ko-panel ko-section">
          <div className="flex items-center gap-2 mb-2">
            <Database className="h-5 w-5 text-green-500" />
            <h2 className="ko-text-lg font-semibold">Quality Metrics</h2>
          </div>
          <p className="ko-text-sm ko-muted mb-4">
            Track coherence, freshness, redundancy, and coverage to spot drift or gaps.
          </p>
          <MetricsPanelContainer />
        </section>
      </PageShell>
    </ErrorBoundary>
  );
}
