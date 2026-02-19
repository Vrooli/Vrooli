import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Clock, ShieldCheck } from "lucide-react";
import { fetchRecoveryEvents, type RecoveryEvent } from "../lib/api";
import { timeAgo } from "../lib/utils";
import { Tooltip } from "./ui/tooltip";
import { StatusBadge, statusToVariant } from "./ui/status-badge";
import { Pagination } from "./ui/pagination";
import { RefreshButton } from "./ui/refresh-button";
import { QueryState } from "./ui/query-state";
import { EmptyState } from "./ui/empty-state";

function triggerLabel(trigger: string): string {
  const labels: Record<string, string> = {
    ready_failure: "Ready Failure",
    ha_connection_loss: "HA Connection Loss",
    manual: "Manual",
  };
  return labels[trigger] ?? trigger;
}

const PAGE_SIZE = 10;

export function RecoveryTimeline() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["recovery-events"],
    queryFn: fetchRecoveryEvents,
    refetchInterval: 30000,
  });

  const [page, setPage] = useState(0);
  const pageSlice = data ? data.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE) : [];

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="recovery-events-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Recovery Events</h2>
        </div>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh recovery events" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading recovery events..."
        errorLabel="Failed to load recovery events."
        skeleton={
          <div className="space-y-2">
            {[1, 2].map((i) => <div key={i} className="h-16 rounded-lg bg-white/5" />)}
          </div>
        }
      >
        {data && data.length === 0 && (
          <EmptyState
            icon={ShieldCheck}
            title="No recovery events"
            description="Recovery events appear here when the system detects and responds to tunnel failures."
            className="mt-4"
          />
        )}

        {data && data.length > 0 && (
          <>
            <div className="mt-4 space-y-2" data-testid="recovery-events-timeline">
              {pageSlice.map((evt: RecoveryEvent) => (
                <div
                  key={evt.id}
                  className="flex items-start gap-3 rounded-lg bg-black/20 p-3"
                  data-testid={`recovery-event-${evt.id}`}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium text-slate-200">
                        {triggerLabel(evt.trigger_type)}
                      </span>
                      <StatusBadge variant={statusToVariant(evt.outcome)} label={evt.outcome} />
                    </div>
                    <p className="text-xs text-slate-300 break-words">
                      {evt.action}
                      {evt.details && ` — ${evt.details}`}
                    </p>
                    <Tooltip content={new Date(evt.created_at).toLocaleString()}>
                      <p className="text-xs text-slate-500 cursor-help">{timeAgo(evt.created_at)}</p>
                    </Tooltip>
                  </div>
                </div>
              ))}
            </div>
            <Pagination page={page} total={data.length} pageSize={PAGE_SIZE} onPageChange={setPage} className="mt-3" />
          </>
        )}
      </QueryState>
    </div>
  );
}
