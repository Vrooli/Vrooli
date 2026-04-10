import { Activity } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { ResponsiveList, ResponsiveListItem } from "../ui/responsive-list";
import { ExecutionCard } from "./execution-card";
import { selectors } from "../../consts/selectors";
import { canCancelExecution, canRetryExecution, canStartExecution } from "../../lib";
import type { ExecutionRecord, PromptTrace } from "../../types";

export interface ExecutionListViewProps {
  status: string;
  hasLoaded: boolean;
  error: Error | null;
  tabItems: ExecutionRecord[];
  filteredItems: ExecutionRecord[];
  activeRuns: ExecutionRecord[];
  activeTabConfig: { id: string; label: string; emptyTitle: string; emptyDescription: string };
  busyId: string | null;
  traceByExecutionId: Record<string, PromptTrace>;
  traceLoadingId: string | null;
  agentManagerUiUrl: string | null;
  onStart: (executionId: string) => void;
  onCancel: (executionId: string) => void;
  onRetry: (executionId: string) => void;
  onViewTrace: (executionId: string) => void;
  onViewBacklog: (kind: string, name: string) => void;
  onFollowUp: (executionId: string) => void;
  onOpenReviewSandbox: (executionId: string) => void;
  onTriggerReview: (executionId: string) => void;
  onFetchRetry: () => void;
  onClearFilters: () => void;
}

export function ExecutionListView({
  status,
  hasLoaded,
  error,
  tabItems,
  filteredItems,
  activeRuns,
  activeTabConfig,
  busyId,
  traceByExecutionId,
  traceLoadingId,
  agentManagerUiUrl,
  onStart,
  onCancel,
  onRetry,
  onViewTrace,
  onViewBacklog,
  onFollowUp,
  onOpenReviewSandbox,
  onTriggerReview,
  onFetchRetry,
  onClearFilters,
}: ExecutionListViewProps) {
  const renderCard = (item: ExecutionRecord, _extraClasses?: string) => (
    <ExecutionCard
      item={item}
      isBusy={busyId === item.executionId}
      canStart={canStartExecution(item.status)}
      canCancel={canCancelExecution(item.status)}
      canRetry={canRetryExecution(item.status)}
      onStart={onStart}
      onCancel={onCancel}
      onRetry={onRetry}
      onViewTrace={onViewTrace}
      onViewBacklog={onViewBacklog}
      trace={traceByExecutionId[item.executionId]}
      traceLoading={traceLoadingId === item.executionId}
      agentManagerUiUrl={agentManagerUiUrl}
      onFollowUp={onFollowUp}
      onOpenReviewSandbox={onOpenReviewSandbox}
      onTriggerReview={onTriggerReview}
    />
  );

  return (
    <div className="space-y-4">
      {(status === "loading" || !hasLoaded) && tabItems.length === 0 ? (
        <PageLoadingState
          label="Loading execution runs..."
          variant="list"
          testId="execution-loading-state"
        />
      ) : null}

      {error && tabItems.length === 0 && hasLoaded ? (
        <ErrorState
          error={error}
          title="Unable to load execution runs"
          onRetry={onFetchRetry}
        />
      ) : null}

      {!error && tabItems.length === 0 && hasLoaded ? (
        <Card padding="lg" centered data-testid={selectors.execution.empty}>
          <Activity className="mx-auto h-12 w-12 text-slate-600" />
          <h3 className="mt-4 text-lg font-medium text-slate-300">{activeTabConfig.emptyTitle}</h3>
          <p className="mt-2 max-w-md text-sm text-slate-400">{activeTabConfig.emptyDescription}</p>
        </Card>
      ) : null}

      {tabItems.length > 0 && filteredItems.length === 0 ? (
        <Card padding="lg" centered data-testid={selectors.execution.noResults}>
          <Activity className="mx-auto h-12 w-12 text-slate-600" />
          <h3 className="mt-4 text-lg font-medium text-slate-300">No matching runs</h3>
          <p className="mt-2 text-sm text-slate-400">Try adjusting your search or filter criteria.</p>
          <Button variant="outline" size="sm" className="mt-4" onClick={onClearFilters}>
            Clear filters
          </Button>
        </Card>
      ) : null}

      {activeRuns.length > 0 ? (
        <div className="space-y-3" data-testid={selectors.execution.activeSection}>
          <div className="flex items-center gap-2 text-sm font-medium text-slate-300">
            <Activity className="h-4 w-4 text-cyan-400" />
            <span>Active Runs</span>
          </div>
          <ResponsiveList data-testid={selectors.execution.activeList} columns="md:grid-cols-1 lg:grid-cols-2">
            {activeRuns.map((item) => (
              <ResponsiveListItem
                key={`active-${item.executionId}`}
                className="md:border-cyan-500/30 md:bg-cyan-500/5 md:hover:border-cyan-500/50 md:hover:bg-cyan-500/10"
              >
                {renderCard(item)}
              </ResponsiveListItem>
            ))}
          </ResponsiveList>
        </div>
      ) : null}

      {filteredItems.length > 0 ? (
        <div className="space-y-3">
          {activeRuns.length > 0 ? (
            <div className="text-sm font-medium text-slate-400">{activeTabConfig.id === "all" ? "All Runs" : `All ${activeTabConfig.label}`}</div>
          ) : null}
          <ResponsiveList data-testid={selectors.execution.grid} columns="md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3">
            {filteredItems.map((item) => (
              <ResponsiveListItem key={item.executionId} interactive>
                {renderCard(item)}
              </ResponsiveListItem>
            ))}
          </ResponsiveList>
        </div>
      ) : null}
    </div>
  );
}
