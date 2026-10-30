import { Download } from "lucide-react";
import type { ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Card } from "@vrooli/react-component-library/Card/1";
import { Stat } from "@vrooli/react-component-library/Stat/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { Toggle } from "@vrooli/react-component-library/Toggle/1";
import { RunScope, type MeasureValidity, type RunBreakdownRow } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import type { TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { formatUsd } from "../../lib/currency";
import { formatCount, formatDuration, formatPercent, formatPoints, formatTokens, MISSING } from "../../lib/format";
import { measuresClient } from "./measuresClient";
import { CALL_OPTIONS, useUsageForecast } from "./api";
import { windowKey, windowTitle } from "./format";

const RANGES = { "6h": 6, "24h": 24, "7d": 168, "30d": 720 } as const;
type Range = keyof typeof RANGES;
const DEFAULT_RANGE: Range = "7d";

const SCOPES = [
  { id: "agent_manager", label: "Agent Manager runs", value: RunScope.AGENT_MANAGER },
  { id: "operator_sessions", label: "Operator sessions", value: RunScope.OPERATOR_SESSIONS },
  { id: "all", label: "Everything", value: RunScope.ALL },
] as const;
type ScopeId = (typeof SCOPES)[number]["id"];

const REFRESH_MS = 60_000;

/** The range ending at the current minute, so a refetch within the minute asks the same question. */
function windowFor(range: Range): TimeWindow {
  const to = new Date(Math.floor(Date.now() / 60_000) * 60_000);
  const from = new Date(to.getTime() - RANGES[range] * 3600_000);
  return { window: { case: "custom", value: { from: timestampFromDate(from), to: timestampFromDate(to) } } } as TimeWindow;
}

/** The measure's own verdict on its sample: an unavailable measure reads "—" with its reason, never a made-up number. */
function usable(validity?: MeasureValidity): boolean {
  return !!validity && validity.state !== "unavailable";
}

function useDetails(range: Range, scope: ScopeId) {
  const scopeValue = SCOPES.find((item) => item.id === scope)?.value ?? RunScope.AGENT_MANAGER;
  const request = () => ({ window: windowFor(range), filter: { scope: scopeValue } });
  const options = { refetchInterval: REFRESH_MS, retry: false } as const;
  const key = (name: string) => ["usage", "details", name, range, scope] as const;
  return {
    success: useQuery({ queryKey: key("success"), queryFn: () => measuresClient.runSuccessRate(request(), CALL_OPTIONS), ...options }),
    cost: useQuery({ queryKey: key("cost"), queryFn: () => measuresClient.runCost(request(), CALL_OPTIONS), ...options }),
    cycle: useQuery({ queryKey: key("cycle"), queryFn: () => measuresClient.runCycleTime(request(), CALL_OPTIONS), ...options }),
    status: useQuery({ queryKey: key("status"), queryFn: () => measuresClient.runStatusDistribution(request(), CALL_OPTIONS), ...options }),
    models: useQuery({ queryKey: key("models"), queryFn: () => measuresClient.modelBreakdown(request(), CALL_OPTIONS), ...options }),
    profiles: useQuery({ queryKey: key("profiles"), queryFn: () => measuresClient.profileBreakdown(request(), CALL_OPTIONS), ...options }),
  };
}
type Details = ReturnType<typeof useDetails>;

/** One tile: its own query, its own error and retry. */
function QueryTile<T>({ label, query, render }: { label: string; query: UseQueryResult<T>; render: (data: T) => { value: ReactNode; caption?: string } }) {
  if (query.error && !query.data) {
    return (
      <Card className="flex h-full flex-col gap-1 p-3">
        <span className="text-xs text-muted-foreground">{label}</span>
        <span role="alert" className="text-sm text-destructive">Couldn't load</span>
        <button type="button" onClick={() => void query.refetch()} className="inline-flex min-h-11 items-center self-start text-sm text-primary hover:underline sm:min-h-8">Retry</button>
      </Card>
    );
  }
  const shown = query.data ? render(query.data) : undefined;
  return (
    <div data-testid="usage-details-tile" className="h-full min-w-0">
      <Stat className="h-full" label={label} value={shown?.value ?? MISSING} caption={shown?.caption} loading={query.isLoading} />
    </div>
  );
}

function QuotaTile() {
  const forecast = useUsageForecast();
  return (
    <QueryTile
      label="Quota used (current windows)"
      query={forecast}
      render={(data) => {
        // Each provider pool's primary window (the API's mark), with a percent reading in its period.
        const rows = data.windows.filter((window) => window.primary && !window.dormant && window.hasPercent && window.sampleCount > 0n && window.status !== "unknown");
        if (rows.length === 0) return { value: MISSING };
        // 1 pt is 1% of its own window, so windows are listed, never summed.
        return { value: <ul className="m-0 list-none space-y-0.5 p-0 text-base">{rows.map((window) => <li key={windowKey(window)}>{windowTitle(window, rows)} {formatPoints(window.usedPercent)}</li>)}</ul> };
      }}
    />
  );
}

function Tiles({ details }: { details: Details }) {
  return (
    <div className="grid grid-cols-2 items-stretch gap-3 lg:grid-cols-5" data-testid="usage-details-tiles">
      <QueryTile label="Runs finished" query={details.success} render={(data) => ({ value: formatCount(data.terminalRuns), caption: usable(data.validity) ? `${formatPercent(data.rate)} succeeded` : data.validity?.reason })} />
      <QueryTile
        label="Tokens"
        query={details.cost}
        render={(data) => {
          const reread = data.totalTokens > 0n ? Number(data.cacheReadTokens) / Number(data.totalTokens) : undefined;
          const billed = data.totalChargeMicroUsd > 0n ? ` · ${formatUsd(Number(data.totalChargeMicroUsd) / 1_000_000)} billed per token` : "";
          return { value: formatTokens(data.totalTokens), caption: reread === undefined ? undefined : `${formatPercent(reread, 0)} re-read from cache · ${formatTokens(data.totalTokens - data.cacheReadTokens)} new${billed}` };
        }}
      />
      <QuotaTile />
      <QueryTile label="Median run time" query={details.cycle} render={(data) => (usable(data.validity) && data.completedDurationRuns > 0n ? { value: formatDuration(data.p50DurationMs), caption: `p90 ${formatDuration(data.p90DurationMs)} · ${formatCount(data.completedDurationRuns)} runs` } : { value: MISSING, caption: data.validity?.reason })} />
      <QueryTile
        label="Queue"
        query={details.status}
        render={(data) => {
          const count = (...names: string[]) => data.rows.filter((row) => names.includes(row.status)).reduce((sum, row) => sum + row.count, 0n);
          return { value: formatCount(count("pending", "starting")), caption: `${formatCount(count("running"))} active` };
        }}
      />
    </div>
  );
}

interface TableColumn {
  header: string;
  cell: (row: RunBreakdownRow) => string;
  numeric?: boolean;
}

/** A breakdown table with its own loading, error and empty states. */
function BreakdownTable({ title, subject, query, label, columns }: { title: string; subject: string; query: UseQueryResult<{ rows: RunBreakdownRow[] }>; label: (row: RunBreakdownRow) => string; columns: TableColumn[] }) {
  const rows = [...(query.data?.rows ?? [])].sort((a, b) => Number(b.totalTokens - a.totalTokens) || Number(b.runCount - a.runCount)).slice(0, 10);
  return (
    <Card className="space-y-3 p-4">
      <h3 className="text-base font-semibold">{title}</h3>
      <DataState compact subject={subject} loading={query.isLoading} error={query.error} isEmpty={rows.length === 0} onRetry={() => void query.refetch()} empty={<p className="text-sm text-muted-foreground">No {subject.toLowerCase()} in this window.</p>}>
        <Table>
          <table>
            <thead><tr><th scope="col">{title.replace("By ", "")}</th>{columns.map((column) => <th key={column.header} scope="col" className={column.numeric ? "whitespace-nowrap text-right" : undefined}>{column.header}</th>)}</tr></thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.key || row.value}>
                  <td className="min-w-0 break-words">{label(row)}</td>
                  {columns.map((column) => <td key={column.header} className={column.numeric ? "whitespace-nowrap text-right tabular-nums" : undefined}>{column.cell(row)}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </Table>
      </DataState>
    </Card>
  );
}

const RUN_COLUMNS: TableColumn[] = [
  { header: "Runs", cell: (row) => formatCount(row.runCount), numeric: true },
  { header: "Success", cell: (row) => (row.runCount > 0n ? formatPercent(Number(row.successCount) / Number(row.runCount), 0) : MISSING), numeric: true },
  { header: "Tokens", cell: (row) => formatTokens(row.totalTokens), numeric: true },
];

const NO_PROFILE = "No profile (interactive)";

function csv(details: Details, range: Range, scope: ScopeId): string {
  const rows: string[][] = [["section", "name", "runs", "success", "tokens"], ["range", range, scope, "", ""]];
  for (const [section, query] of [["model", details.models], ["profile", details.profiles]] as const) {
    for (const row of query.data?.rows ?? []) rows.push([section, row.value || NO_PROFILE, String(row.runCount), String(row.successCount), String(row.totalTokens)]);
  }
  rows.push(["tokens", "", String(details.success.data?.terminalRuns ?? ""), String(details.success.data?.successfulRuns ?? ""), String(details.cost.data?.totalTokens ?? "")]);
  return rows.map((row) => row.map((cell) => `"${cell.replace(/"/g, '""')}"`).join(",")).join("\n");
}

function exportCsv(details: Details, range: Range, scope: ScopeId) {
  const url = URL.createObjectURL(new Blob([csv(details, range, scope)], { type: "text/csv" }));
  const link = Object.assign(document.createElement("a"), { href: url, download: `agent-manager-usage-${range}-${scope}.csv` });
  link.click();
  URL.revokeObjectURL(url);
}

/** Usage › Details: the run counts, tokens and run times behind the quota, for Agent Manager runs unless asked otherwise. */
export function UsageDetails() {
  const [params, setParams] = useSearchParams();
  const range = (Object.keys(RANGES) as Range[]).find((value) => value === params.get("range")) ?? DEFAULT_RANGE;
  const scope = SCOPES.find((item) => item.id === params.get("scope"))?.id ?? "agent_manager";
  const details = useDetails(range, scope);
  const setParam = (name: string, value: string, fallback: string) => setParams((current) => {
    const updated = new URLSearchParams(current);
    if (value === fallback) updated.delete(name);
    else updated.set(name, value);
    return updated;
  }, { replace: true });
  return (
    <div className="flex min-w-0 flex-col gap-4" data-testid="usage-details">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <ButtonGroup label="Runs counted" collapse="never" className="flex-wrap">
          {SCOPES.map((item) => <Toggle key={item.id} size="lg" pressed={scope === item.id} onPressedChange={() => setParam("scope", item.id, "agent_manager")}>{item.label}</Toggle>)}
        </ButtonGroup>
        <div className="flex flex-wrap items-center gap-2">
          <ButtonGroup label="Time range" collapse="never">
            {(Object.keys(RANGES) as Range[]).map((value) => <Toggle key={value} size="lg" pressed={range === value} onPressedChange={() => setParam("range", value, DEFAULT_RANGE)}>{value}</Toggle>)}
          </ButtonGroup>
          <button type="button" onClick={() => exportCsv(details, range, scope)} className="inline-flex min-h-11 items-center gap-2 rounded-md border border-border px-3 text-sm hover:bg-muted sm:min-h-8">
            <Download className="h-4 w-4" aria-hidden="true" />Export
          </button>
        </div>
      </div>
      <Tiles details={details} />
      <p className="text-xs text-muted-foreground">Tokens and quota count all work in the window. Success rate and run time count only the runs in the filter above. Subscription work is measured in quota points; dollars appear only for runs billed per token.</p>
      <div className="grid items-start gap-4 lg:grid-cols-2">
        <BreakdownTable title="By model" subject="Models" query={details.models} label={(row) => row.value || "No model recorded"} columns={RUN_COLUMNS} />
        <BreakdownTable title="By profile" subject="Profiles" query={details.profiles} label={(row) => (row.value && row.value !== "unknown" ? row.value : NO_PROFILE)} columns={RUN_COLUMNS} />
      </div>
      <p className="text-sm text-muted-foreground">Model-fallback and runner-failure insights moved to <Link to="/observability" className="inline-flex min-h-11 items-center px-1 text-primary hover:underline sm:min-h-8">Health</Link>.</p>
    </div>
  );
}
