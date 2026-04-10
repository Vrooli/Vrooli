// REQ-UI-010: Subscription health detail
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { HeartPulse, ArrowLeft, Send, AlertTriangle, CheckCircle, XCircle } from "lucide-react";
import { Button } from "../components/ui/button";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { StatCard } from "../components/StatCard";
import { fetchSubscription, fetchSubscriptionHealth, type SubscriptionData } from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS } from "../lib/constants";
import { formatTimestamp } from "../lib/utils";

export function SubscriptionHealthPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const subId = Number(id);

  const { data: sub, isLoading: subLoading, error: subError } = useQuery({
    queryKey: ["subscription", subId],
    queryFn: () => fetchSubscription(subId),
    enabled: !Number.isNaN(subId),
  });

  const { data: health, isLoading: healthLoading, error: healthError, refetch } = useQuery({
    queryKey: ["subscription-health", subId],
    queryFn: () => fetchSubscriptionHealth(subId),
    enabled: !Number.isNaN(subId),
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const isLoading = subLoading || healthLoading;
  const error = subError ?? healthError;

  const statusColor = health?.status === "active"
    ? "text-[var(--status-healthy)]"
    : health?.status === "circuit_broken"
      ? "text-[var(--error-text)]"
      : "text-[var(--text-muted)]";

  return (
    <div className="space-y-4">
      <PageHeader
        icon={HeartPulse}
        title={sub ? `Health: ${sub.name}` : `Subscription #${id} Health`}
        actions={
          <Button size="sm" variant="outline" onClick={() => navigate("/subscriptions")}>
            <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
            Back
          </Button>
        }
      />

      {isLoading && <Spinner label="Loading health data..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}

      {sub && (
        <Panel>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-xs text-[var(--text-faint)]">Owner</span>
              <p className="text-[var(--text-secondary)]">{sub.owner_scenario}</p>
            </div>
            <div>
              <span className="text-xs text-[var(--text-faint)]">Event Pattern</span>
              <p className="font-mono text-xs text-[var(--text-accent)]">{sub.event_pattern}</p>
            </div>
            <div>
              <span className="text-xs text-[var(--text-faint)]">Delivery Type</span>
              <p className="text-[var(--text-secondary)]">{sub.delivery_type}</p>
            </div>
            <div>
              <span className="text-xs text-[var(--text-faint)]">Delivery Target</span>
              <p className="truncate font-mono text-xs text-[var(--text-secondary)]">{sub.delivery_target}</p>
            </div>
          </div>
        </Panel>
      )}

      {health && (
        <>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <StatCard
              label="Total Delivered"
              value={health.total_delivered}
              icon={Send}
            />
            <StatCard
              label="Total Failed"
              value={health.total_failed}
              icon={XCircle}
              detail={health.total_failed > 0 ? "Check delivery target" : undefined}
            />
            <StatCard
              label="Consecutive Failures"
              value={health.consecutive_failures}
              icon={AlertTriangle}
            />
            <StatCard
              label="Status"
              value={health.status}
              icon={CheckCircle}
              detail={
                health.status === "circuit_broken"
                  ? "Delivery suspended"
                  : "Delivery active"
              }
            />
          </div>

          <Panel>
            <h3 className="mb-3 text-sm font-medium text-[var(--text-secondary)]">Details</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-[var(--text-faint)]">Status</span>
                <span className={statusColor}>{health.status}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-[var(--text-faint)]">Last Delivered</span>
                <span className="text-[var(--text-secondary)]">
                  {health.last_delivered_at
                    ? formatTimestamp(health.last_delivered_at)
                    : "Never"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-[var(--text-faint)]">Last Failed</span>
                <span className="text-[var(--text-secondary)]">
                  {health.last_failed_at
                    ? formatTimestamp(health.last_failed_at)
                    : "Never"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-[var(--text-faint)]">Success Rate</span>
                <span className="text-[var(--text-secondary)]">
                  {health.total_delivered + health.total_failed > 0
                    ? `${((health.total_delivered / (health.total_delivered + health.total_failed)) * 100).toFixed(1)}%`
                    : "N/A"}
                </span>
              </div>
            </div>
          </Panel>
        </>
      )}
    </div>
  );
}
