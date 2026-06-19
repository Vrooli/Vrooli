import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { tunnelClient } from "../../api/tunnel";
import { exposureClient } from "../../api/exposure";
import { recoveryClient } from "../../api/recovery";
import { recoveryStatusLabel, recoveryStatusTone } from "../recovery/labels";

function tunnelTone(status: string): BadgeTone {
  if (status === "healthy") return "success";
  if (status === "degraded") return "warning";
  if (status === "unhealthy") return "danger";
  return "neutral";
}

function Card({
  testId,
  heading,
  children,
}: {
  testId: string;
  heading: string;
  children: React.ReactNode;
}) {
  return (
    <div data-testid={testId} className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      <h2 className="text-sm font-semibold uppercase text-app-muted-foreground">{heading}</h2>
      {children}
    </div>
  );
}

/**
 * OverviewPanel is the one-glance operations summary: tunnel health, the core
 * vs leased exposure split, and the live recovery state (with a circuit-breaker
 * warning when automatic recovery is paused). Each card links to its full
 * surface. Independent queries so one failing domain doesn't blank the page.
 */
export function OverviewPanel() {
  const { t } = useTranslation();

  const tunnelQuery = useQuery({ queryKey: ["tunnel-status"], queryFn: () => tunnelClient.getStatus({}) });
  const exposuresQuery = useQuery({ queryKey: ["exposures"], queryFn: () => exposureClient.listExposures({}) });
  const recoveryQuery = useQuery({ queryKey: ["recovery-state"], queryFn: () => recoveryClient.getState({}) });

  const exposures = exposuresQuery.data?.exposures ?? [];
  const coreCount = exposures.filter((e) => e.tier === "core").length;
  const leasedCount = exposures.filter((e) => e.tier === "leased").length;
  const recoveryState = recoveryQuery.data?.state;

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card testId={selectors.overview.tunnelCard} heading={t(strings.overview.tunnelHeading)}>
        <QueryState isLoading={tunnelQuery.isLoading} error={tunnelQuery.error}>
          {tunnelQuery.data?.status && (
            <div className="flex flex-col gap-2">
              <StatusBadge
                tone={tunnelTone(tunnelQuery.data.status.status)}
                data-testid={selectors.overview.tunnelStatus}
              >
                {tunnelQuery.data.status.status}
              </StatusBadge>
              <p className="text-sm text-app-muted-foreground">
                {t(strings.overview.scoreLabel)}:{" "}
                <span data-testid={selectors.overview.tunnelScore} className="font-semibold text-app-foreground">
                  {tunnelQuery.data.status.score}
                </span>
              </p>
            </div>
          )}
        </QueryState>
      </Card>

      <Card testId={selectors.overview.exposureCard} heading={t(strings.overview.exposureHeading)}>
        <QueryState isLoading={exposuresQuery.isLoading} error={exposuresQuery.error}>
          <dl className="flex flex-col gap-1 text-sm">
            <div className="flex justify-between">
              <dt className="text-app-muted-foreground">{t(strings.overview.coreLabel)}</dt>
              <dd data-testid={selectors.overview.coreCount} className="font-semibold tabular-nums">
                {coreCount}
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-app-muted-foreground">{t(strings.overview.leasedLabel)}</dt>
              <dd data-testid={selectors.overview.leasedCount} className="font-semibold tabular-nums">
                {leasedCount}
              </dd>
            </div>
          </dl>
          <Link to="/exposure" className="text-sm text-app-primary underline-offset-2 hover:underline">
            {t(strings.overview.viewExposure)}
          </Link>
        </QueryState>
      </Card>

      <Card testId={selectors.overview.recoveryCard} heading={t(strings.overview.recoveryHeading)}>
        <QueryState isLoading={recoveryQuery.isLoading} error={recoveryQuery.error}>
          {recoveryState && (
            <div className="flex flex-col gap-2">
              <StatusBadge
                tone={recoveryStatusTone(recoveryState.status)}
                data-testid={selectors.overview.recoveryStatus}
              >
                {t(recoveryStatusLabel(recoveryState.status))}
              </StatusBadge>
              {recoveryState.circuitOpen && (
                <p data-testid={selectors.overview.circuitWarning} role="alert" className="text-sm text-app-danger">
                  {t(strings.overview.circuitOpenWarning)}
                </p>
              )}
              <Link to="/recovery" className="text-sm text-app-primary underline-offset-2 hover:underline">
                {t(strings.overview.viewRecovery)}
              </Link>
            </div>
          )}
        </QueryState>
      </Card>
    </div>
  );
}
