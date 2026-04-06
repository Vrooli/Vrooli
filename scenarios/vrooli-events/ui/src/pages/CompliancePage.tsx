// REQ-UI-011: Compliance & audit view
import { useQuery } from "@tanstack/react-query";
import { ClipboardCheck, RefreshCw } from "lucide-react";
import { Button } from "../components/ui/button";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { fetchViolations } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS } from "../lib/constants";
import { formatTimestamp } from "../lib/utils";

export function CompliancePage() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["violations"],
    queryFn: fetchViolations,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  return (
    <div className="space-y-4" data-testid="compliance-page">
      <PageHeader
        icon={ClipboardCheck}
        title="Compliance & Audit"
        actions={
          <Button size="sm" variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Refresh
          </Button>
        }
      />

      {isLoading && <Spinner label="Loading violations..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {data && data.length > 0 && (
        <Panel>
          <div className="overflow-x-auto">
            <table className="w-full text-sm" data-testid="compliance-table">
              <thead>
                <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-faint)]">
                  <th className="px-3 py-2">Timestamp</th>
                  <th className="px-3 py-2">Source</th>
                  <th className="px-3 py-2">Target</th>
                  <th className="px-3 py-2">Endpoint</th>
                  <th className="px-3 py-2">Rule Type</th>
                  <th className="px-3 py-2">Reason</th>
                </tr>
              </thead>
              <tbody>
                {data.map((v) => (
                  <tr
                    key={v.id}
                    className="border-b border-[var(--border-subtle)] hover:bg-white/5"
                  >
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-[var(--text-faint)]">
                      {formatTimestamp(v.timestamp)}
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{v.source_scenario}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{v.target_scenario}</td>
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-secondary)]">
                      {v.endpoint}
                    </td>
                    <td className="px-3 py-2">
                      <span className="rounded bg-[var(--surface-inset)] px-2 py-0.5 text-xs text-[var(--text-accent)]">
                        {v.rule_type}
                      </span>
                    </td>
                    <td className="max-w-[300px] truncate px-3 py-2 text-xs text-[var(--status-unhealthy)]">
                      {v.reason}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}

      {data && data.length === 0 && (
        <Panel>
          <div className="py-8 text-center text-sm text-[var(--text-muted)]">
            <ClipboardCheck className="mx-auto mb-3 h-8 w-8 text-[var(--text-faint)]" />
            <p>No policy violations recorded.</p>
            <p className="mt-1 text-xs text-[var(--text-faint)]">
              Violations appear here when events are blocked or rate-limited by policy rules.
            </p>
          </div>
        </Panel>
      )}
    </div>
  );
}
