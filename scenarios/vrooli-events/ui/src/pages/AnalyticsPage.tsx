// DOC: docs/reference/api-endpoints.md#health
import { useQuery } from "@tanstack/react-query";
import { BarChart3, Database, HardDrive, Users, Clock } from "lucide-react";
import { fetchHealth } from "../lib/api";
import { METRICS_POLL_INTERVAL_MS } from "../lib/constants";
import { formatBytes } from "../lib/utils";
import { StatCard } from "../components/StatCard";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";

export function AnalyticsPage() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: METRICS_POLL_INTERVAL_MS,
  });

  return (
    <div className="space-y-6">
      <PageHeader icon={BarChart3} title="Analytics Overview" />

      {isLoading && <Spinner label="Loading metrics…" />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} />}

      {data && (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              label="Total Events"
              value={data.store?.totalEvents?.toLocaleString() ?? "—"}
              icon={Database}
            />
            <StatCard
              label="Store Size"
              value={data.store ? formatBytes(data.store.totalPayloadBytes) : "—"}
              icon={HardDrive}
              detail="Payload bytes stored"
            />
            <StatCard
              label="Active Subscribers"
              value={data.subscribers ?? 0}
              icon={Users}
              detail="SSE connections"
            />
            <StatCard
              label="System Status"
              value={data.status ?? "unknown"}
              icon={Clock}
              detail={`Last check: ${data.timestamp ? new Date(data.timestamp).toLocaleTimeString() : "—"}`}
            />
          </div>

          <Panel>
            <h3 className="mb-4 text-sm font-medium text-[var(--text-muted)]">System Information</h3>
            <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
              <div>
                <dt className="text-[var(--text-faint)]">Service</dt>
                <dd className="text-[var(--text-secondary)]">{data.service}</dd>
              </div>
              <div>
                <dt className="text-[var(--text-faint)]">Status</dt>
                <dd className={data.status === "healthy" ? "text-[var(--status-healthy)]" : "text-[var(--status-degraded)]"}>
                  {data.status}
                </dd>
              </div>
              <div>
                <dt className="text-[var(--text-faint)]">Readiness</dt>
                <dd className={data.readiness ? "text-[var(--status-healthy)]" : "text-[var(--status-unhealthy)]"}>
                  {data.readiness ? "Ready" : "Not Ready"}
                </dd>
              </div>
              <div>
                <dt className="text-[var(--text-faint)]">Retention Policy</dt>
                <dd className="text-[var(--text-secondary)]">30 days / 2 GB cap</dd>
              </div>
            </dl>
          </Panel>
        </>
      )}
    </div>
  );
}
