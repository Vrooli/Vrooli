import { Search } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { formatStandardRelativeTime } from "../../../lib/dateTime";
import type { RunnerHealthRow } from "../api/types";
import { HealthStatusBadge } from "./HealthStatusBadge";

interface RunnerHealthTableProps {
  rows: RunnerHealthRow[];
  onShowAudit: (runner: string) => void;
}

export function RunnerHealthTable({ rows, onShowAudit }: RunnerHealthTableProps) {
  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground" data-testid="runner-health-empty">
        No runner health observations yet.
      </p>
    );
  }
  const sorted = [...rows].sort((a, b) => {
    const sa = statusRank(a.status);
    const sb = statusRank(b.status);
    if (sa !== sb) return sa - sb;
    return a.runner.localeCompare(b.runner);
  });
  return (
    <div className="rounded-lg border border-border bg-card/40 overflow-hidden" data-testid="runner-health-table">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-xs uppercase tracking-wide text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-left">Status</th>
            <th className="px-3 py-2 text-left">Runner</th>
            <th className="px-3 py-2 text-left">Reason</th>
            <th className="px-3 py-2 text-left">Catalog</th>
            <th className="px-3 py-2 text-left">Last checked</th>
            <th className="px-3 py-2 text-right">Audit</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => (
            <tr
              key={row.runner}
              className="border-t border-border/60"
              data-testid={`runner-health-row-${row.runner}`}
            >
              <td className="px-3 py-2"><HealthStatusBadge status={row.status} /></td>
              <td className="px-3 py-2 font-mono text-xs">{row.runner}</td>
              <td className="px-3 py-2 text-muted-foreground">{row.reason ?? "—"}</td>
              <td className="px-3 py-2 text-xs text-muted-foreground">
                {row.catalog ? `${row.catalog.status} (${row.catalog.age_days}d / ${row.catalog.budget_days}d)` : "—"}
              </td>
              <td className="px-3 py-2 text-muted-foreground">
                {formatStandardRelativeTime(row.last_checked)}
              </td>
              <td className="px-3 py-2 text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onShowAudit(row.runner)}
                  aria-label={`Show audit history for runner ${row.runner}`}
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

function statusRank(s: RunnerHealthRow["status"]): number {
  return s === "failed" ? 0 : s === "unknown" ? 1 : 2;
}
