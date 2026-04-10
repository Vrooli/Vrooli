// DOC: docs/reference/configuration.md
import { useQuery } from "@tanstack/react-query";
import { Settings, Info } from "lucide-react";
import { fetchHealth } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS } from "../lib/constants";
import { formatBytes } from "../lib/utils";
import { ErrorAlert } from "../components/ErrorAlert";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";

export function SettingsPage() {
  const { data, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  return (
    <div className="space-y-6">
      <PageHeader icon={Settings} title="Settings" />

      <Panel>
        <h3 className="mb-4 text-sm font-medium text-[var(--text-secondary)]">Retention Configuration</h3>
        <div className="space-y-4 text-sm">
          <div className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-inset)] px-4 py-3">
            <div>
              <p className="text-[var(--text-secondary)]">Time-based retention</p>
              <p className="text-xs text-[var(--text-faint)]">Events older than this are pruned</p>
            </div>
            <span className="font-mono text-[var(--text-secondary)]">30 days</span>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-inset)] px-4 py-3">
            <div>
              <p className="text-[var(--text-secondary)]">Size-based retention</p>
              <p className="text-xs text-[var(--text-faint)]">Oldest events pruned when exceeded</p>
            </div>
            <span className="font-mono text-[var(--text-secondary)]">2 GB</span>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-inset)] px-4 py-3">
            <div>
              <p className="text-[var(--text-secondary)]">Pruning interval</p>
              <p className="text-xs text-[var(--text-faint)]">Background cleanup frequency</p>
            </div>
            <span className="font-mono text-[var(--text-secondary)]">6 hours</span>
          </div>
        </div>
        <div className="mt-4 flex items-start gap-2 rounded-lg bg-indigo-500/10 p-3 text-xs text-[var(--text-accent-light)]">
          <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>Retention settings are currently configured at the server level. API-configurable retention (REQ-ES-004) is planned for a future release.</span>
        </div>
      </Panel>

      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {data && (
        <Panel>
          <h3 className="mb-4 text-sm font-medium text-[var(--text-secondary)]">Current Store Status</h3>
          <dl className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <dt className="text-[var(--text-faint)]">Total Events</dt>
              <dd className="text-[var(--text-secondary)]">{data.store?.totalEvents?.toLocaleString() ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-[var(--text-faint)]">Payload Size</dt>
              <dd className="text-[var(--text-secondary)]">
                {data.store ? formatBytes(data.store.totalPayloadBytes) : "0 B"}
              </dd>
            </div>
            <div>
              <dt className="text-[var(--text-faint)]">Active Subscribers</dt>
              <dd className="text-[var(--text-secondary)]">{data.subscribers ?? 0}</dd>
            </div>
            <div>
              <dt className="text-[var(--text-faint)]">System Status</dt>
              <dd className={data.status === "healthy" ? "text-[var(--status-healthy)]" : "text-[var(--status-degraded)]"}>
                {data.status ?? "unknown"}
              </dd>
            </div>
          </dl>
        </Panel>
      )}
    </div>
  );
}
