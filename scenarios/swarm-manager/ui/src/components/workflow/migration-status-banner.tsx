/**
 * MigrationStatusBanner
 *
 * Partial-migration awareness: a lightweight inline notice consuming
 * GetMigrationStatus, rendered ONLY while the persisted-state migration is
 * `staged` or `quarantined`. Invisible in steady state (not-started /
 * promoted), so it costs nothing until Phase-8 tooling actually runs.
 */

import { AlertTriangle, Layers } from "lucide-react";
import { useMigrationStatusQuery } from "../../hooks/useAgentOpsQueries";
import { cn } from "../../lib/utils";

export function MigrationStatusBanner({ className }: { className?: string }) {
  const { data } = useMigrationStatusQuery();

  if (!data || (data.state !== "staged" && data.state !== "quarantined")) return null;

  const quarantined = data.state === "quarantined";

  return (
    <div
      className={cn(
        "mb-3 flex items-start gap-2 rounded-lg border px-3 py-2 text-xs",
        quarantined
          ? "border-red-500/30 bg-red-500/10 text-red-300"
          : "border-amber-500/30 bg-amber-500/10 text-amber-300",
        className,
      )}
      role={quarantined ? "alert" : "status"}
      data-testid="workflow-migration-banner"
      data-state={data.state}
    >
      {quarantined ? (
        <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      ) : (
        <Layers className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      )}
      <div className="min-w-0">
        <p className="font-medium">
          {quarantined
            ? "Workflow migration quarantined"
            : "Workflow migration staged"}
        </p>
        <p className="mt-0.5 text-[11px] opacity-80">
          {quarantined
            ? `Epoch ${data.epoch}: ${data.quarantinedCount} document${data.quarantinedCount === 1 ? "" : "s"} quarantined — some items may still use legacy records.`
            : `Epoch ${data.epoch}: ${data.stagedCount} document${data.stagedCount === 1 ? "" : "s"} staged, not yet promoted — canonical and legacy records coexist.`}
        </p>
      </div>
    </div>
  );
}
