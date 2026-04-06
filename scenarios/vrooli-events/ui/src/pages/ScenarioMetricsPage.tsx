// DOC: docs/reference/api-endpoints.md#event-query
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Users, ArrowUpDown, RefreshCw } from "lucide-react";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/EmptyState";
import { ErrorAlert } from "../components/ErrorAlert";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { Spinner } from "../components/Spinner";
import { fetchEvents, type EventEnvelope } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS } from "../lib/constants";

interface ScenarioStats {
  scenario: string;
  outbound: number;
  inbound: number;
  errorRate: number;
  uniqueTypes: number;
}

type SortKey = "scenario" | "outbound" | "inbound" | "errorRate" | "uniqueTypes";

function aggregateByScenario(events: EventEnvelope[]): ScenarioStats[] {
  const outbound = new Map<string, number>();
  const inbound = new Map<string, number>();
  const errors = new Map<string, number>();
  const types = new Map<string, Set<string>>();

  for (const evt of events) {
    const src = evt.sourceScenario;
    outbound.set(src, (outbound.get(src) ?? 0) + 1);
    let srcTypes = types.get(src);
    if (!srcTypes) {
      srcTypes = new Set();
      types.set(src, srcTypes);
    }
    srcTypes.add(evt.eventType);

    const isError = evt.eventType.includes("error") || evt.eventType.includes("fail");
    if (isError) errors.set(src, (errors.get(src) ?? 0) + 1);

    if (evt.targetScenario) {
      const tgt = evt.targetScenario;
      inbound.set(tgt, (inbound.get(tgt) ?? 0) + 1);
      if (!types.has(tgt)) types.set(tgt, new Set());
    }
  }

  const allScenarios = new Set([...outbound.keys(), ...inbound.keys()]);
  return Array.from(allScenarios).map((scenario) => {
    const out = outbound.get(scenario) ?? 0;
    const err = errors.get(scenario) ?? 0;
    return {
      scenario,
      outbound: out,
      inbound: inbound.get(scenario) ?? 0,
      errorRate: out > 0 ? err / out : 0,
      uniqueTypes: types.get(scenario)?.size ?? 0,
    };
  });
}

export function ScenarioMetricsPage() {
  const navigate = useNavigate();
  const [sortKey, setSortKey] = useState<SortKey>("outbound");
  const [sortAsc, setSortAsc] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["events", "", "", "", 500],
    queryFn: () => fetchEvents({ limit: 500 }),
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const stats = useMemo(() => {
    if (!data) return [];
    const agg = aggregateByScenario(data);
    agg.sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (typeof av === "string" && typeof bv === "string") {
        return sortAsc ? av.localeCompare(bv) : bv.localeCompare(av);
      }
      const numA = Number(av);
      const numB = Number(bv);
      return sortAsc ? numA - numB : numB - numA;
    });
    return agg;
  }, [data, sortKey, sortAsc]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc((v) => !v);
    } else {
      setSortKey(key);
      setSortAsc(false);
    }
  };

  const columns: { key: SortKey; label: string }[] = [
    { key: "scenario", label: "Scenario" },
    { key: "outbound", label: "Outbound" },
    { key: "inbound", label: "Inbound" },
    { key: "errorRate", label: "Error Rate" },
    { key: "uniqueTypes", label: "Event Types" },
  ];

  return (
    <div className="space-y-4" data-testid="scenario-metrics-page">
      <PageHeader
        icon={Users}
        title="Per-Scenario Metrics"
        actions={
          <Button size="sm" variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Refresh
          </Button>
        }
      />

      {isLoading && <Spinner label="Loading scenario metrics..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {stats.length > 0 && (
        <Panel>
          <div className="overflow-x-auto">
            <table className="w-full text-sm" data-testid="scenario-metrics-table">
              <thead>
                <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-faint)]">
                  {columns.map((col) => (
                    <th key={col.key} className="px-3 py-2">
                      <button
                        className="inline-flex items-center gap-1 hover:text-[var(--text-secondary)]"
                        onClick={() => toggleSort(col.key)}
                        data-testid={`sort-${col.key}`}
                      >
                        {col.label}
                        <ArrowUpDown className="h-3 w-3" />
                      </button>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {stats.map((row) => (
                  <tr
                    key={row.scenario}
                    className="border-b border-[var(--border-subtle)] hover:bg-white/5"
                    data-testid={`scenario-row-${row.scenario}`}
                  >
                    <td className="px-3 py-2 font-mono">
                      <button
                        onClick={() => navigate(`/events?source=${encodeURIComponent(row.scenario)}`)}
                        className="text-[var(--text-accent)] hover:underline"
                        title="View events from this scenario"
                      >
                        {row.scenario}
                      </button>
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{row.outbound}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{row.inbound}</td>
                    <td className="px-3 py-2">
                      <span className={row.errorRate > 0.1 ? "text-[var(--status-unhealthy)]" : "text-[var(--text-secondary)]"}>
                        {(row.errorRate * 100).toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{row.uniqueTypes}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}

      {data && stats.length === 0 && (
        <EmptyState icon={Users} title="No scenario data yet" description="Scenario metrics will appear once events start flowing between scenarios." />
      )}
    </div>
  );
}
