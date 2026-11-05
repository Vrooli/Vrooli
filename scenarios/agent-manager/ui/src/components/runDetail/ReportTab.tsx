import { useRunReport } from "../../hooks/useApi";
import { MISSING, formatCount, formatDuration, formatTokens } from "../../lib/format";

function ms(value?: string): string {
  return value === undefined ? MISSING : formatDuration(Number(value));
}

export function ReportTab({ runId }: { runId: string }) {
  const report = useRunReport(runId);
  if (report.loading) return <div className="py-8 text-center text-muted-foreground">Loading run report…</div>;
  if (report.error) return <div className="py-8 text-center text-destructive">Report unavailable: {report.error}</div>;
  const data = report.data;
  if (!data) return <div className="py-8 text-center text-muted-foreground">Report unavailable</div>;
  return (
    <div className="space-y-3 text-sm" data-testid="run-report">
      <div><strong>Status:</strong> {data.status}{data.exit_code !== undefined ? ` (exit ${data.exit_code})` : ""}{data.error ? ` — ${data.error}` : ""}</div>
      <div><strong>Timing:</strong> duration={ms(data.duration_ms)}; heartbeat gap={ms(data.heartbeat_gap_ms)}; requests={formatCount(data.turns)}; tokens={formatTokens(Number(data.tokens))}; cost=${data.cost_usd}</div>
      <div><strong>Final output:</strong> {data.result.selection_status} ({data.result.selection_rule || MISSING}), candidates={data.result.candidate_count}</div>
      <div><strong>Structured:</strong> {data.result.structured_status || MISSING} {data.result.diagnostic_codes?.join(", ")}</div>
      <div><strong>Tools:</strong> project-owned={data.project_owned_tool_calls} external={data.external_tool_calls}; repeated={data.repeated_tool_calls}; files reread={data.files_read_more_than_once}</div>
      <div><strong>Model:</strong> requested={data.requested_model || MISSING} actual={data.actual_model || MISSING} fallbacks={data.fallback_count}</div>
      <div><strong>Diff:</strong> {data.diff.files} files, {data.diff.bytes} bytes ({data.diff.available.state})</div>
      <div><strong>Events:</strong> {data.events_availability.state}</div>
      <ul className="list-disc pl-5">{Object.entries(data.event_counts).sort(([left], [right]) => left.localeCompare(right)).map(([type, count]) => <li key={type}>{type}: {count}</li>)}</ul>
      <div><strong>Receipts:</strong> {data.receipts_availability.state} ({data.receipt_count})</div>
      <ul className="list-disc pl-5">{data.tools.map((tool) => <li key={tool.name}>{tool.name}: {tool.calls} calls, {tool.returned ?? 0} returned, {tool.failures} failed, {tool.unknown_outcome ?? 0} outcome unknown, {tool.unresolved ?? 0} unresolved</li>)}</ul>
    </div>
  );
}
