import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Mode, type ConfigReadiness, type DriftCounts } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { configClient } from "../../api/config";
import { tunnelClient } from "../../api/tunnel";
import { exposureClient } from "../../api/exposure";
import { recoveryClient } from "../../api/recovery";
import { recoveryStatusLabel, recoveryStatusTone } from "../recovery/labels";

type ModeCalloutKey =
  | typeof strings.overview.modeLocal
  | typeof strings.overview.modeLocalKeysConfigured
  | typeof strings.overview.modeRemote;

/**
 * Pick the mode-callout copy. Remote mode reports the managed/external/drift
 * tally; local mode describes the local cloudflared config. Callers pass the
 * EFFECTIVE mode (readiness.desiredMode): when Cloudflare credentials are
 * complete the tunnel is remote-managed, so the callout shows the remote tally
 * even if the persisted toggle was never flipped off the local default. The
 * keys-configured-but-unused branch is retained for back-compat but is no
 * longer reached, since complete credentials now make the mode effectively
 * remote.
 */
function modeCalloutKey(mode: Mode, readiness?: ConfigReadiness): ModeCalloutKey {
  if (mode === Mode.REMOTE) return strings.overview.modeRemote;
  if (readiness?.remoteAvailable) return strings.overview.modeLocalKeysConfigured;
  return strings.overview.modeLocal;
}

function tunnelTone(status: string): BadgeTone {
  if (status === "healthy") return "success";
  if (status === "degraded") return "warning";
  if (status === "unhealthy") return "danger";
  return "neutral";
}

function modeLabelKey(mode: Mode) {
  if (mode === Mode.REMOTE) return strings.config.mode.remote;
  if (mode === Mode.LOCAL) return strings.config.mode.local;
  return strings.config.mode.unspecified;
}

function readinessTone(remoteAvailable: boolean, syncReady: boolean): BadgeTone {
  if (syncReady && remoteAvailable) return "success";
  if (syncReady) return "info";
  return "warning";
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

  const configQuery = useQuery({ queryKey: ["config-state"], queryFn: () => configClient.getConfig({}) });
  const tunnelQuery = useQuery({ queryKey: ["tunnel-status"], queryFn: () => tunnelClient.getStatus({}) });
  const exposuresQuery = useQuery({ queryKey: ["exposures"], queryFn: () => exposureClient.listExposures({}) });
  const recoveryQuery = useQuery({ queryKey: ["recovery-state"], queryFn: () => recoveryClient.getState({}) });
  const driftQuery = useQuery({ queryKey: ["drift"], queryFn: () => configClient.getDrift({}) });

  const config = configQuery.data?.config;
  const readiness = configQuery.data?.readiness;
  const exposures = exposuresQuery.data?.exposures ?? [];
  const coreCount = exposures.filter((e) => e.tier === "core").length;
  const leasedCount = exposures.filter((e) => e.tier === "leased").length;
  const recoveryState = recoveryQuery.data?.state;
  const driftCounts: DriftCounts | undefined = driftQuery.data?.counts;
  const externalTracked = (driftCounts?.externalOk ?? 0) + (driftCounts?.ignored ?? 0);

  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <Card testId={selectors.overview.readinessCard} heading={t(strings.overview.readinessHeading)}>
        <QueryState isLoading={configQuery.isLoading} error={configQuery.error}>
          {config && readiness && (
            <div className="flex flex-col gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge
                  tone={readinessTone(readiness.remoteAvailable, readiness.syncReady)}
                  data-testid={selectors.overview.readinessStatus}
                >
                  {readiness.syncReady
                    ? t(strings.overview.readinessReady)
                    : t(strings.overview.readinessSetupRequired)}
                </StatusBadge>
                <StatusBadge tone="neutral" data-testid={selectors.overview.modeBadge}>
                  {t(modeLabelKey(readiness.desiredMode))}
                </StatusBadge>
              </div>
              <p data-testid={selectors.overview.modeCallout} className="text-sm text-app-foreground">
                {t(modeCalloutKey(readiness.desiredMode, readiness), {
                  managed: driftCounts?.managed ?? 0,
                  external: externalTracked,
                  drift: driftCounts?.unmanaged ?? 0,
                })}
              </p>
              <p className="text-sm text-app-muted-foreground">
                {readiness.modeReason || t(strings.overview.readinessFallback)}
              </p>
              {readiness.missingFields.length > 0 && (
                <p data-testid={selectors.overview.missingFields} className="text-sm text-app-warning">
                  {t(strings.config.missingFields, { fields: readiness.missingFields.join(", ") })}
                </p>
              )}
              <Link to="/settings" className="text-sm text-app-primary underline-offset-2 hover:underline">
                {t(strings.overview.configure)}
              </Link>
            </div>
          )}
        </QueryState>
      </Card>

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

      <Card testId={selectors.overview.driftCard} heading={t(strings.overview.driftHeading)}>
        <QueryState isLoading={driftQuery.isLoading} error={driftQuery.error}>
          <dl className="flex flex-col gap-1 text-sm">
            <div className="flex justify-between">
              <dt className="text-app-muted-foreground">
                {t(strings.overview.driftManaged)} ({t(strings.overview.urlStateLive)})
              </dt>
              <dd data-testid={selectors.overview.driftManaged} className="font-semibold tabular-nums">
                {driftCounts?.managed ?? 0}
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-app-muted-foreground">{t(strings.overview.driftExternal)}</dt>
              <dd data-testid={selectors.overview.driftExternal} className="font-semibold tabular-nums">
                {externalTracked}
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-app-muted-foreground">
                {t(strings.overview.driftUnmanaged)} ({t(strings.overview.urlStateDesired)})
              </dt>
              <dd data-testid={selectors.overview.driftUnmanaged} className="font-semibold tabular-nums">
                {(driftCounts?.unmanaged ?? 0) + (driftCounts?.missing ?? 0)}
              </dd>
            </div>
          </dl>
          <Link to="/drift" className="text-sm text-app-primary underline-offset-2 hover:underline">
            {t(strings.overview.viewDrift)}
          </Link>
        </QueryState>
      </Card>
    </div>
  );
}
