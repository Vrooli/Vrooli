import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody } from "../../../components/ui/dialog";
import { formatStandardDateTime } from "../../../lib/dateTime";
import type { HealthAuditFilters } from "../api/types";
import { useHealthAudit } from "../hooks/useHealth";
import { HealthStatusBadge } from "./HealthStatusBadge";

interface HealthAuditDrawerProps {
  open: boolean;
  filters: HealthAuditFilters | null;
  onOpenChange: (open: boolean) => void;
}

export function HealthAuditDrawer({ open, filters, onOpenChange }: HealthAuditDrawerProps) {
  const enabled = open && filters !== null;
  const { data, isLoading, error } = useHealthAudit(filters ?? { scope: "model" }, { enabled });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {filters
              ? filters.scope === "runner"
                ? `Audit history — runner ${filters.runner ?? "all"}`
                : `Audit history — ${filters.runner ?? "all"} / ${filters.model ?? "all"}`
              : "Audit history"}
          </DialogTitle>
        </DialogHeader>
        <DialogBody>
          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : error ? (
            <p className="text-sm text-destructive" data-testid="audit-error">
              Failed to load audit: {error.message}
            </p>
          ) : !data || data.rows.length === 0 ? (
            <p className="text-sm text-muted-foreground" data-testid="audit-empty">
              No audit rows for this filter.
            </p>
          ) : (
            <ul className="space-y-2" data-testid="audit-rows">
              {data.rows.map((row) => (
                <li
                  key={row.id}
                  className="rounded-md border border-border/60 bg-card/40 p-2 text-sm"
                  data-testid={`audit-row-${row.id}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <HealthStatusBadge status={row.status} />
                    <span className="text-xs text-muted-foreground">
                      {formatStandardDateTime(row.timestamp)}
                    </span>
                  </div>
                  <div className="mt-1 font-mono text-xs text-muted-foreground">
                    {row.runnerType}
                    {row.modelId ? ` / ${row.modelId}` : ""} · triggered by {row.triggeredBy}
                  </div>
                  {row.reason ? (
                    <div className="mt-1 text-xs">
                      <span className="text-muted-foreground">Reason:</span> {row.reason}
                    </div>
                  ) : null}
                  {row.message ? (
                    <div className="mt-0.5 text-xs text-muted-foreground">{row.message}</div>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
