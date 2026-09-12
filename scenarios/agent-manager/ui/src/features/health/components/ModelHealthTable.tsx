import { Search } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { formatStandardRelativeTime } from "../../../lib/dateTime";
import type { ModelHealthRow } from "../api/types";
import { HealthStatusBadge } from "./HealthStatusBadge";

interface ModelHealthTableProps {
  rows: ModelHealthRow[];
  onShowAudit: (runner: string, model: string) => void;
}

export function ModelHealthTable({ rows, onShowAudit }: ModelHealthTableProps) {
  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground" data-testid="model-health-empty">
        No model health observations yet. Once a run probes a model, it appears here.
      </p>
    );
  }
  const sorted = [...rows].sort((a, b) => {
    const sa = statusRank(a.status);
    const sb = statusRank(b.status);
    if (sa !== sb) return sa - sb;
    if (a.runner !== b.runner) return a.runner.localeCompare(b.runner);
    return a.model.localeCompare(b.model);
  });
  return (
    <div className="rounded-lg border border-border bg-card/40 overflow-hidden" data-testid="model-health-table">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-xs uppercase tracking-wide text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-left">Status</th>
            <th className="px-3 py-2 text-left">Runner</th>
            <th className="px-3 py-2 text-left">Model</th>
            <th className="px-3 py-2 text-left">Reason</th>
            <th className="px-3 py-2 text-left">Last checked</th>
            <th className="px-3 py-2 text-right">Audit</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => (
            <tr
              key={`${row.runner}|${row.model}`}
              className="border-t border-border/60"
              data-testid={`model-health-row-${row.runner}-${row.model}`}
            >
              <td className="px-3 py-2"><HealthStatusBadge status={row.status} /></td>
              <td className="px-3 py-2 font-mono text-xs">{row.runner}</td>
              <td className="px-3 py-2 font-mono text-xs">{row.model}</td>
              <td className="px-3 py-2 text-muted-foreground">{row.reason ?? "—"}</td>
              <td className="px-3 py-2 text-muted-foreground">
                {formatStandardRelativeTime(row.last_checked)}
              </td>
              <td className="px-3 py-2 text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onShowAudit(row.runner, row.model)}
                  aria-label={`Show audit history for ${row.runner} / ${row.model}`}
                >
                  <Search className="h-3.5 w-3.5" />
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function statusRank(s: ModelHealthRow["status"]): number {
  // Failed first (most actionable), unknown next, ok last.
  return s === "failed" ? 0 : s === "unknown" ? 1 : 2;
}
