// DOC: docs/reference/api-endpoints.md#sse-subscribe
// DOC: docs/internal/TEMPORAL-FLOWS.md
import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Radio, Pause, Play, Trash2, X } from "lucide-react";
import { Button } from "../components/ui/button";
import { EventTable } from "../components/EventTable";
import { EventDetail } from "../components/EventDetail";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { subscribeSSE, type EventEnvelope } from "../lib/api";
import { INPUT_CLASS, STREAM_MAX_EVENTS } from "../lib/constants";

export function StreamPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [events, setEvents] = useState<EventEnvelope[]>([]);
  const [paused, setPaused] = useState(false);
  const [selected, setSelected] = useState<EventEnvelope | null>(null);
  const [connected, setConnected] = useState(false);
  const [typeFilter, setTypeFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const pausedRef = useRef(paused);

  // Sync filters from URL query params on mount / navigation
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const type = params.get("type");
    const source = params.get("source");
    if (type) setTypeFilter(type);
    if (source) setSourceFilter(source);
  }, [location.search]);

  // Persist filters to URL query params
  const syncFiltersToUrl = useCallback(
    (type: string, source: string) => {
      const params = new URLSearchParams();
      if (type) params.set("type", type);
      if (source) params.set("source", source);
      const search = params.toString();
      navigate({ search: search ? `?${search}` : "" }, { replace: true });
    },
    [navigate],
  );

  const updateTypeFilter = useCallback(
    (v: string) => { setTypeFilter(v); syncFiltersToUrl(v, sourceFilter); },
    [sourceFilter, syncFiltersToUrl],
  );
  const updateSourceFilter = useCallback(
    (v: string) => { setSourceFilter(v); syncFiltersToUrl(typeFilter, v); },
    [typeFilter, syncFiltersToUrl],
  );

  const hasFilters = typeFilter !== "" || sourceFilter !== "";
  const resetFilters = useCallback(() => {
    setTypeFilter("");
    setSourceFilter("");
    syncFiltersToUrl("", "");
  }, [syncFiltersToUrl]);

  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  useEffect(() => {
    const cleanup = subscribeSSE({
      type: typeFilter || undefined,
      source: sourceFilter || undefined,
      onEvent: (evt) => {
        if (!pausedRef.current) {
          setEvents((prev) => [evt, ...prev].slice(0, STREAM_MAX_EVENTS));
        }
        setConnected(true);
      },
      onError: () => setConnected(false),
    });
    setConnected(true);
    return cleanup;
  }, [typeFilter, sourceFilter]);

  const clear = useCallback(() => {
    setEvents([]);
    setSelected(null);
  }, []);

  return (
    <div className="space-y-4">
      <PageHeader
        icon={Radio}
        title="Live Event Stream"
        actions={
          <div className="flex items-center gap-2">
            <Button size="sm" variant="outline" onClick={() => setPaused((p) => !p)}>
              {paused ? <Play className="mr-1.5 h-3.5 w-3.5" /> : <Pause className="mr-1.5 h-3.5 w-3.5" />}
              {paused ? "Resume" : "Pause"}
            </Button>
            <Button size="sm" variant="outline" onClick={clear}>
              <Trash2 className="mr-1.5 h-3.5 w-3.5" />
              Clear
            </Button>
          </div>
        }
      />
      <div className="flex items-center gap-2">
        <span
          className={`inline-block h-2 w-2 rounded-full ${
            connected ? "animate-pulse bg-[var(--status-healthy-bright)]" : "bg-[var(--status-unhealthy-bright)]"
          }`}
        />
        <span className="text-xs text-[var(--text-muted)]">
          {connected ? "Connected" : "Disconnected — will auto-reconnect"}
        </span>
      </div>

      <div className="flex gap-3">
        <input
          type="text"
          placeholder="Filter by type glob (e.g. scenario.*)"
          value={typeFilter}
          onChange={(e) => updateTypeFilter(e.target.value)}
          className={`flex-1 ${INPUT_CLASS}`}
        />
        <input
          type="text"
          placeholder="Filter by source"
          value={sourceFilter}
          onChange={(e) => updateSourceFilter(e.target.value)}
          className={`w-48 ${INPUT_CLASS}`}
        />
        {hasFilters && (
          <Button size="sm" variant="outline" onClick={resetFilters}>
            <X className="mr-1.5 h-3.5 w-3.5" />
            Reset
          </Button>
        )}
      </div>

      <div className="text-xs text-[var(--text-faint)]">{events.length} events captured</div>

      {selected && <EventDetail event={selected} onClose={() => setSelected(null)} />}

      {events.length === 0 && connected && (
        <Panel>
          <div className="py-4 text-center text-sm text-[var(--text-muted)]">
            <Radio className="mx-auto mb-3 h-8 w-8 text-[var(--text-faint)]" />
            <p>Waiting for events...</p>
            <p className="mt-2 text-xs text-[var(--text-faint)]">
              Send a test event via the API:
            </p>
            <pre className="mx-auto mt-2 max-w-lg overflow-x-auto rounded-lg bg-[var(--surface-code)] p-3 text-left font-mono text-xs text-[var(--text-secondary)]">
{`curl -X POST http://localhost:<API_PORT>/api/v1/events/ingest \\
  -H "Content-Type: application/json" \\
  -d '{"source_scenario":"test","event_type":"ping","payload":{}}'`}
            </pre>
          </div>
        </Panel>
      )}

      <EventTable events={events} onSelect={setSelected} />
    </div>
  );
}
