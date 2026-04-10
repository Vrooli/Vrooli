// DOC: docs/reference/api-endpoints.md#event-query
import { useCallback, useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router-dom";
import { Activity, Search, RefreshCw, X } from "lucide-react";
import { Button } from "../components/ui/button";
import { EventTable } from "../components/EventTable";
import { EventDetail } from "../components/EventDetail";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { fetchEvents, type EventEnvelope } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS, INPUT_CLASS, QUERY_LIMIT_OPTIONS } from "../lib/constants";

export function EventLogPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [typeFilter, setTypeFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [correlationFilter, setCorrelationFilter] = useState("");
  const [limit, setLimit] = useState(50);
  const [selected, setSelected] = useState<EventEnvelope | null>(null);

  // Sync filters from URL query params on mount / navigation
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const type = params.get("type");
    const source = params.get("source");
    const cid = params.get("cid");
    const lim = params.get("limit");
    if (type) setTypeFilter(type);
    if (source) setSourceFilter(source);
    if (cid) setCorrelationFilter(cid);
    if (lim) { const n = Number(lim); if (!Number.isNaN(n) && n > 0) setLimit(n); }
  }, [location.search]);

  // Persist filters to URL query params
  const syncFiltersToUrl = useCallback(
    (type: string, source: string, cid: string, lim: number) => {
      const params = new URLSearchParams();
      if (type) params.set("type", type);
      if (source) params.set("source", source);
      if (cid) params.set("cid", cid);
      if (lim !== 50) params.set("limit", String(lim));
      const search = params.toString();
      navigate({ search: search ? `?${search}` : "" }, { replace: true });
    },
    [navigate],
  );

  const updateTypeFilter = useCallback(
    (v: string) => { setTypeFilter(v); syncFiltersToUrl(v, sourceFilter, correlationFilter, limit); },
    [sourceFilter, correlationFilter, limit, syncFiltersToUrl],
  );
  const updateSourceFilter = useCallback(
    (v: string) => { setSourceFilter(v); syncFiltersToUrl(typeFilter, v, correlationFilter, limit); },
    [typeFilter, correlationFilter, limit, syncFiltersToUrl],
  );
  const updateCorrelationFilter = useCallback(
    (v: string) => { setCorrelationFilter(v); syncFiltersToUrl(typeFilter, sourceFilter, v, limit); },
    [typeFilter, sourceFilter, limit, syncFiltersToUrl],
  );
  const updateLimit = useCallback(
    (v: number) => { setLimit(v); syncFiltersToUrl(typeFilter, sourceFilter, correlationFilter, v); },
    [typeFilter, sourceFilter, correlationFilter, syncFiltersToUrl],
  );

  const hasFilters = typeFilter !== "" || sourceFilter !== "" || correlationFilter !== "" || limit !== 50;
  const resetFilters = useCallback(() => {
    setTypeFilter("");
    setSourceFilter("");
    setCorrelationFilter("");
    setLimit(50);
    navigate({ search: "" }, { replace: true });
  }, [navigate]);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["events", typeFilter, sourceFilter, correlationFilter, limit],
    queryFn: () =>
      fetchEvents({
        type: typeFilter || undefined,
        source: sourceFilter || undefined,
        correlationId: correlationFilter || undefined,
        limit,
      }),
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  return (
    <div className="space-y-4">
      <PageHeader
        icon={Activity}
        title="Event History"
        actions={
          <Button size="sm" variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Refresh
          </Button>
        }
      />

      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-[var(--text-faint)]" />
          <input
            type="text"
            placeholder="Type glob (e.g. scenario.*.action.*)"
            value={typeFilter}
            onChange={(e) => updateTypeFilter(e.target.value)}
            className={`w-full pl-9 pr-3 ${INPUT_CLASS}`}
          />
        </div>
        <input
          type="text"
          placeholder="Source scenario"
          value={sourceFilter}
          onChange={(e) => updateSourceFilter(e.target.value)}
          className={`w-40 ${INPUT_CLASS}`}
        />
        <input
          type="text"
          placeholder="Correlation ID"
          value={correlationFilter}
          onChange={(e) => updateCorrelationFilter(e.target.value)}
          className={`w-40 ${INPUT_CLASS}`}
        />
        <select
          value={limit}
          onChange={(e) => updateLimit(Number(e.target.value))}
          className={INPUT_CLASS}
        >
          {QUERY_LIMIT_OPTIONS.map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
        {hasFilters && (
          <Button size="sm" variant="outline" onClick={resetFilters}>
            <X className="mr-1.5 h-3.5 w-3.5" />
            Reset
          </Button>
        )}
      </div>

      {isLoading && <Spinner label="Loading events…" />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {selected && <EventDetail event={selected} onClose={() => setSelected(null)} />}

      {data && (
        <>
          <div className="text-xs text-[var(--text-faint)]">{data.length} events returned</div>
          <EventTable events={data} onSelect={setSelected} />
        </>
      )}
    </div>
  );
}
