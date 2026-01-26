import { Activity, AlertCircle, Database, GitGraph, Search } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { routeToHash, type Route } from "../../shared/controllers/routeController";
import type { HealthViewModel } from "../../shared/controllers/knowledgeController";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import { Button } from "../../shared/ui/button";
import { FeatureCardLink } from "./components/FeatureCardLink";

export type DashboardHealthState = {
  viewModel: HealthViewModel;
  isLoading: boolean;
  hasError: boolean;
  hasData: boolean;
  refetch: () => void;
};

export type DashboardPageProps = {
  health: DashboardHealthState;
  onNavigate: (route: Route) => void;
};

export function DashboardPage({ health, onNavigate }: DashboardPageProps) {
  const { viewModel, isLoading, hasError, hasData, refetch } = health;
  const { status: healthStatus, service: serviceName, lastUpdated: lastUpdate } = viewModel;

  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Dashboard Unavailable"
            description="The dashboard failed to render. You can retry or jump to another page."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              {
                label: "Go to Search",
                onClick: () => onNavigate("search"),
                variant: "secondary",
              },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell>
        <Panel testId={selectors.dashboard.quickActions}>
          <div className="flex items-center justify-between gap-4 flex-wrap">
            <div>
              <h2 className="ko-text-lg font-semibold">Start a Knowledge Check</h2>
              <p className="ko-text-sm ko-muted mt-1">
                Jump into the workflows operators use most: search, assess, and explore.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button asChild variant="primary" data-testid={selectors.dashboard.quickSearch}>
                <a href={routeToHash("search")}>Run a Search</a>
              </Button>
              <Button asChild variant="secondary" data-testid={selectors.dashboard.quickMetrics}>
                <a href={routeToHash("metrics")}>Review Metrics</a>
              </Button>
              <Button asChild variant="secondary" data-testid={selectors.dashboard.quickGraph}>
                <a href={routeToHash("graph")}>Explore Graph</a>
              </Button>
            </div>
          </div>
        </Panel>

        <Panel testId={selectors.dashboard.healthSection}>
          <PanelHeader
            title="System Health"
            icon={<Activity className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />

          {isLoading && (
            <div className="ko-stack-xs">
              <div className="ko-loading-bar"></div>
              <p className="ko-text-sm ko-muted">Querying knowledge base status...</p>
            </div>
          )}

          {hasError && (
            <div className="ko-alert ko-alert-danger" data-testid={selectors.dashboard.healthError}>
              <AlertCircle className="h-5 w-5 ko-text-danger mt-0.5" />
              <div>
                <p className="ko-alert-title ko-text-danger-strong">Connection Error</p>
                <p className="ko-text-sm ko-text-danger-muted mt-1">
                  Unable to reach the API. Confirm the scenario is running, then refresh.
                </p>
              </div>
            </div>
          )}

          {hasData && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              <div className="ko-card p-4">
                <p className="ko-meta">Status</p>
                <p className="text-xl font-bold mt-1">{healthStatus}</p>
              </div>
              <div className="ko-card p-4">
                <p className="ko-meta">Service</p>
                <p className="text-xl font-bold mt-1">{serviceName}</p>
              </div>
              <div className="ko-card p-4">
                <p className="ko-meta">Last Update</p>
                <p className="ko-text-sm font-semibold mt-1">{lastUpdate}</p>
              </div>
            </div>
          )}

          <Button
            className="mt-4"
            variant="primary"
            onClick={() => refetch()}
            data-testid={selectors.dashboard.healthRefresh}
          >
            Refresh Status
          </Button>
        </Panel>

        <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <FeatureCardLink
            route="search"
            title="Semantic Search"
            description="Query knowledge base using natural language across all collections"
            icon={<Search className="h-8 w-8 ko-icon" />}
            testId={selectors.dashboard.featureSearch}
          />
          <FeatureCardLink
            route="graph"
            title="Knowledge Graph"
            description="Explore semantic relationships and concept connections"
            icon={<GitGraph className="h-8 w-8 ko-icon" />}
            badge="Preview"
            testId={selectors.dashboard.featureGraph}
          />
          <FeatureCardLink
            route="metrics"
            title="Quality Metrics"
            description="Monitor coherence, freshness, and redundancy scores"
            icon={<Database className="h-8 w-8 ko-icon" />}
            testId={selectors.dashboard.featureMetrics}
          />
        </section>

        <Panel testId={selectors.dashboard.cliSection}>
          <PanelHeader title="CLI Commands" className="mb-4" />
          <div className="ko-stack-xs ko-text-sm">
            <div className="ko-card p-3">
              <code className="ko-code">knowledge-observatory search "your query"</code>
              <p className="ko-subtle ko-text-xs mt-1">Semantic search across knowledge base</p>
            </div>
            <div className="ko-card p-3">
              <code className="ko-code">knowledge-observatory health --watch</code>
              <p className="ko-subtle ko-text-xs mt-1">Real-time health monitoring</p>
            </div>
            <div className="ko-card p-3">
              <code className="ko-code">knowledge-observatory graph --center "concept"</code>
              <p className="ko-subtle ko-text-xs mt-1">Generate knowledge relationship graph</p>
            </div>
          </div>
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
