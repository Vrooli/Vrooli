import type { VaultResourceStatus } from "../../lib/api";
import { percentage } from "../../lib/formatters";
import type { Intent } from "../../lib/constants";
import { SeverityBadge } from "./SeverityBadge";

export const SecretsRow = ({ status }: { status: VaultResourceStatus }) => {
  const configuredPercent = percentage(status.secrets_found, status.secrets_total);
  const tone: Intent =
    status.health_status === "healthy"
      ? "good"
      : status.health_status === "critical"
      ? "danger"
      : status.health_status === "degraded"
      ? "warn"
      : "info";
  const progressColor =
    tone === "good"
      ? "bg-emerald-400"
      : tone === "warn"
      ? "bg-amber-400"
      : tone === "danger"
      ? "bg-red-500"
      : "bg-cyan-400";

  return (
    <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/5 px-4 py-3">
      <div>
        <p className="text-sm font-semibold text-white">{status.resource_name}</p>
        <p className="text-xs text-white/70">{status.secrets_found}/{status.secrets_total} configured</p>
      </div>
      <div className="flex w-1/2 flex-col items-end gap-2">
        <div className="h-2 w-full rounded-full bg-white/10">
          <div className={`h-2 rounded-full ${progressColor}`} style={{ width: `${configuredPercent}%` }} />
        </div>
        <div className="flex items-center gap-2 text-sm">
          <span>{configuredPercent}% ready</span>
          <SeverityBadge severity={status.health_status} />
        </div>
      </div>
    </div>
  );
};
