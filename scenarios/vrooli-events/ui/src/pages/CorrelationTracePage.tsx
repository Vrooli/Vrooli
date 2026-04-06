// DOC: docs/reference/api-endpoints.md#event-query
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";
import { GitBranch, Search } from "lucide-react";
import { Button } from "../components/ui/button";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { fetchEvents, type EventEnvelope } from "../lib/api";
import { INPUT_CLASS } from "../lib/constants";
import { formatTimestamp } from "../lib/utils";

function TraceTimeline({ events }: { events: EventEnvelope[] }) {
  const sorted = [...events].sort((a, b) => {
    const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return ta - tb;
  });

  return (
    <div className="space-y-0" data-testid="trace-timeline">
      {sorted.map((evt, i) => {
        const isError = evt.eventType.includes("error") || evt.eventType.includes("fail");
        return (
          <div key={evt.eventId} className="flex gap-4" data-testid={`trace-node-${i}`}>
            <div className="flex flex-col items-center">
              <div
                className={`h-3 w-3 rounded-full border-2 ${
                  isError
                    ? "border-[var(--status-unhealthy)] bg-[var(--status-unhealthy)]"
                    : "border-[var(--text-accent)] bg-[var(--text-accent)]"
                }`}
              />
              {i < sorted.length - 1 && (
                <div className="w-0.5 flex-1 bg-[var(--border-subtle)]" />
              )}
            </div>
            <div className="pb-4">
              <div className="flex items-center gap-2">
                <span className="font-mono text-xs text-[var(--text-accent)]">{evt.sourceScenario}</span>
                {evt.targetScenario && (
                  <>
                    <span className="text-xs text-[var(--text-faint)]">&rarr;</span>
                    <span className="font-mono text-xs text-[var(--text-secondary)]">{evt.targetScenario}</span>
                  </>
                )}
              </div>
              <p className="text-sm text-[var(--text-secondary)]">{evt.eventType}</p>
              {evt.createdAt && (
                <p className="text-xs text-[var(--text-faint)]">
                  {formatTimestamp(evt.createdAt)}
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function CorrelationTracePage() {
  const location = useLocation();
  const [correlationId, setCorrelationId] = useState("");
  const [activeSearch, setActiveSearch] = useState("");

  // Accept cross-navigation via ?cid= query param
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const cid = params.get("cid");
    if (cid && cid !== activeSearch) {
      setCorrelationId(cid);
      setActiveSearch(cid);
    }
  }, [location.search]); // eslint-disable-line react-hooks/exhaustive-deps

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["trace", activeSearch],
    queryFn: () => fetchEvents({ correlationId: activeSearch, limit: 100 }),
    enabled: activeSearch.length > 0,
  });

  const handleSearch = () => {
    if (correlationId.trim()) {
      setActiveSearch(correlationId.trim());
    }
  };

  return (
    <div className="space-y-4" data-testid="correlation-trace-page">
      <PageHeader icon={GitBranch} title="Correlation Traces" />

      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-[var(--text-faint)]" />
          <input
            type="text"
            placeholder="Enter correlation ID to trace..."
            value={correlationId}
            onChange={(e) => setCorrelationId(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className={`w-full pl-9 pr-3 ${INPUT_CLASS}`}
            data-testid="trace-correlation-input"
          />
        </div>
        <Button size="sm" onClick={handleSearch} data-testid="trace-search-button">
          Trace
        </Button>
      </div>

      {isLoading && <Spinner label="Tracing events..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {data && data.length > 0 && (
        <Panel>
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-medium text-[var(--text-secondary)]">
              Trace: <span className="font-mono text-[var(--text-accent)]">{activeSearch}</span>
            </h3>
            <span className="text-xs text-[var(--text-faint)]">{data.length} events</span>
          </div>
          <TraceTimeline events={data} />
        </Panel>
      )}

      {data && data.length === 0 && activeSearch && (
        <p className="text-sm text-[var(--text-muted)]">
          No events found for correlation ID "{activeSearch}".
        </p>
      )}

      {!activeSearch && (
        <Panel>
          <div className="py-8 text-center text-sm text-[var(--text-muted)]">
            <GitBranch className="mx-auto mb-3 h-8 w-8 text-[var(--text-faint)]" />
            <p>Enter a correlation ID to visualize the event chain.</p>
            <p className="mt-1 text-xs text-[var(--text-faint)]">
              Correlation traces show all events sharing the same correlation ID as a timeline.
            </p>
          </div>
        </Panel>
      )}
    </div>
  );
}
