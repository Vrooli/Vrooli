// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { useState, type FormEvent } from "react";
import { Activity, BookOpen, FileText, GitGraph, Target } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import { storeSearchIntent } from "../../shared/controllers/searchIntent";
import { DEFAULT_SEARCH_MODE, type SearchMode } from "../../shared/controllers/searchModes";
import type { Route } from "../../shared/controllers/routeController";
import type { HealthViewModel } from "../../shared/controllers/knowledgeController";
import { useActivityFeed } from "../../shared/hooks/activityHooks";
import { useDocumentationSummary } from "../../shared/hooks/documentationSummaryHooks";
import { Button } from "../../shared/ui/button";
import { FeatureCardLink } from "./components/FeatureCardLink";
import { ActivityFeed } from "./components/ActivityFeed";
import { HealthStatCard } from "./components/HealthStatCard";
import { QuickSearchPanel } from "./components/QuickSearchPanel";

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
  const { viewModel, isLoading, hasError, refetch } = health;
  const { status: healthStatus, service: serviceName, lastUpdated: lastUpdate } = viewModel;
  const docSummary = useDocumentationSummary();
  const activityItems = useActivityFeed();
  const [quickMode, setQuickMode] = useState<SearchMode>(DEFAULT_SEARCH_MODE);
  const [quickQuery, setQuickQuery] = useState("");
  const trimmedQuery = quickQuery.trim();
  const healthTone = hasError ? "poor" : isLoading ? "medium" : "good";

  const handleQuickSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!trimmedQuery) return;
    storeSearchIntent({ mode: quickMode, value: trimmedQuery });
    onNavigate("search");
  };

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
          <QuickSearchPanel
            mode={quickMode}
            query={quickQuery}
            onModeChange={setQuickMode}
            onQueryChange={setQuickQuery}
            onSubmit={handleQuickSubmit}
            isSubmitDisabled={!trimmedQuery}
          />
        </Panel>

        <Panel testId={selectors.dashboard.healthSection}>
          <PanelHeader title="System Health" icon={<Activity className="h-5 w-5 ko-icon" />} className="mb-4" />
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <HealthStatCard
              title="Knowledge Health"
              value={healthStatus}
              subtitle={`Service: ${serviceName} · Updated ${lastUpdate}`}
              tone={healthTone}
              icon={<Activity className="h-5 w-5 ko-icon" />}
              isLoading={isLoading}
              hasError={hasError}
              errorMessage="Unable to reach the API. Confirm the scenario is running."
              testId={hasError ? selectors.dashboard.healthError : undefined}
            />
            <HealthStatCard
              title="Documentation Health"
              value={docSummary.viewModel.averageHealthLabel}
              subtitle={`${docSummary.viewModel.coverageLabel} · Updated ${docSummary.viewModel.lastModifiedLabel}`}
              tone={docSummary.viewModel.averageHealthTone}
              icon={<BookOpen className="h-5 w-5 ko-icon" />}
              isLoading={docSummary.isLoading}
              hasError={docSummary.hasError}
              errorMessage={docSummary.errorMessage}
            />
            <HealthStatCard
              title="Scenario Coverage"
              value={docSummary.viewModel.coveragePercentLabel}
              subtitle={docSummary.viewModel.manifestCoverageLabel}
              tone={docSummary.viewModel.coverageTone}
              icon={<Target className="h-5 w-5 ko-icon" />}
              isLoading={docSummary.isLoading}
              hasError={docSummary.hasError}
              errorMessage={docSummary.errorMessage}
            />
          </div>
          <Button
            className="mt-4"
            variant="primary"
            onClick={() => refetch()}
            data-testid={selectors.dashboard.healthRefresh}
          >
            Refresh Status
          </Button>
        </Panel>

        <section className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
          <FeatureCardLink
            route="search"
            title="Search Docs"
            description="Run semantic, file, text, unified, or deep searches from one workspace."
            icon={<FileText className="h-8 w-8 ko-icon" />}
            testId={selectors.dashboard.featureSearch}
          />
          <FeatureCardLink
            route="explorer"
            title="Browse Scenarios"
            description="Audit documentation structure, health, and file trees."
            icon={<BookOpen className="h-8 w-8 ko-icon" />}
            testId={selectors.dashboard.featureExplorer}
          />
          <FeatureCardLink
            route="graph"
            title="View Graph"
            description="Explore semantic relationships and concept connections."
            icon={<GitGraph className="h-8 w-8 ko-icon" />}
            badge="Preview"
            testId={selectors.dashboard.featureGraph}
          />
          <FeatureCardLink
            route="metrics"
            title="Run Health Audit"
            description="Review coherence, freshness, and redundancy scores."
            icon={<Activity className="h-8 w-8 ko-icon" />}
            testId={selectors.dashboard.featureMetrics}
          />
        </section>

        <section className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Panel testId={selectors.dashboard.activityFeed}>
            <PanelHeader title="Activity Feed" className="mb-4" />
            <ActivityFeed items={activityItems} />
          </Panel>
          <Panel testId={selectors.dashboard.cliSection}>
            <PanelHeader title="CLI Commands" className="mb-4" />
            <div className="ko-stack-xs ko-text-sm">
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory search "your query"</code>
                <p className="ko-subtle ko-text-xs mt-1">Semantic search across knowledge base</p>
              </div>
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory docs search-files "**/README.md"</code>
                <p className="ko-subtle ko-text-xs mt-1">Search documentation files by pattern</p>
              </div>
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory docs health knowledge-observatory</code>
                <p className="ko-subtle ko-text-xs mt-1">Audit documentation health</p>
              </div>
            </div>
          </Panel>
        </section>
      </PageShell>
    </ErrorBoundary>
  );
}
