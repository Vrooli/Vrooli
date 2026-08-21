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
import { probesClient } from "../../api/probes";
import { FailureClass } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

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

function arcPath(cx: number, cy: number, radius: number, start: number, end: number): string {
  const startPoint = { x: cx + Math.cos(start) * radius, y: cy + Math.sin(start) * radius };
  const endPoint = { x: cx + Math.cos(end) * radius, y: cy + Math.sin(end) * radius };
  const largeArcFlag = end - start <= Math.PI ? 0 : 1;
  return ["M", startPoint.x, startPoint.y, "A", radius, radius, 0, largeArcFlag, 1, endPoint.x, endPoint.y].join(" ");
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

function Finding({ testId, label, detail, href }: { testId: string; label: string; detail: string; href: string }) {
  return <Link data-testid={testId} to={href} className="rounded-control border border-app-border bg-app-surface-muted p-3 transition-colors hover:border-app-primary hover:bg-app-surface">
    <span className="block text-sm font-semibold">{label}</span><span className="mt-1 block text-xs text-app-muted-foreground">{detail}</span>
  </Link>;
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
  const classificationQuery = useQuery({ queryKey: ["exposure-classify"], queryFn: () => probesClient.classify({}) });

  const config = configQuery.data?.config;
  const readiness = configQuery.data?.readiness;
  const exposures = exposuresQuery.data?.exposures ?? [];
  const coreCount = exposures.filter((e) => e.tier === "core").length;
  const leasedCount = exposures.filter((e) => e.tier === "leased").length;
  const recoveryState = recoveryQuery.data?.state;
  const driftCounts: DriftCounts | undefined = driftQuery.data?.counts;
  const externalTracked = (driftCounts?.externalOk ?? 0) + (driftCounts?.ignored ?? 0);
  const classifications = classificationQuery.data?.classifications ?? [];
  const unhealthyCount = classifications.filter((item) => item.classification !== FailureClass.HEALTHY).length;
  const findingCount = (readiness?.missingFields.length ?? 0) + unhealthyCount + (recoveryState?.circuitOpen ? 1 : 0) + (driftCounts?.unmanaged ?? 0);
  const constellationStatus = tunnelQuery.data?.status?.status ?? "unknown";
  const constellationLoading = exposuresQuery.isLoading || tunnelQuery.isLoading;

  return (
    <div className="flex flex-col gap-5">
      <section
        data-testid={selectors.overview.constellation}
        aria-label={t(strings.overview.exposureHeading)}
        className="relative overflow-hidden rounded-[1.25rem] border border-app-border bg-[radial-gradient(circle_at_50%_38%,rgba(37,99,235,0.16),transparent_42%),var(--color-surface)] p-5 shadow-sm sm:p-7"
      >
        <div className="relative z-10 flex flex-col gap-1">
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-app-primary">{t(strings.pages.overview.title)}</p>
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{t(strings.overview.exposureHeading)}</h2>
          <p className="max-w-xl text-sm text-app-muted-foreground">{t(strings.overview.readinessFallback)}</p>
        </div>
        <div className="relative mx-auto mt-3 aspect-[2.35/1] max-w-4xl min-h-[18rem]">
          {constellationLoading ? <div className="absolute inset-8 flex flex-col items-center justify-center gap-4 rounded-full border border-app-border/70 bg-app-surface-muted/30" role="status" aria-label={t(strings.common.loading)}><span className="h-16 w-16 animate-pulse rounded-full border-4 border-app-primary/20 border-t-app-primary" /><span className="h-3 w-48 animate-pulse rounded-full bg-app-border" /><span className="h-3 w-32 animate-pulse rounded-full bg-app-border/70" /></div> : null}
          <svg className={`absolute inset-0 h-full w-full transition-opacity ${constellationLoading ? "opacity-25" : "opacity-100"}`} viewBox="0 0 720 400" role="img" aria-label={`Tunnel ${constellationStatus} with ${exposures.length} exposed routes`}>
            <defs>
              <radialGradient id="constellationGlow"><stop stopColor="var(--color-primary)" stopOpacity=".45"/><stop offset="1" stopColor="var(--color-primary)" stopOpacity="0"/></radialGradient>
            </defs>
            <circle cx="360" cy="200" r="120" fill="url(#constellationGlow)" />
            {[0, 1, 2, 3].map((ring) => <circle key={ring} cx="360" cy="200" r={78 + ring * 28} fill="none" stroke="var(--color-border)" strokeOpacity=".65" strokeDasharray={ring === 3 ? "3 10" : "1 0"} />)}
            <path d={arcPath(360, 200, 106, -Math.PI * 0.82, Math.PI * 0.05)} fill="none" stroke="var(--color-primary)" strokeWidth="7" strokeLinecap="round" opacity=".9" />
            <path d={arcPath(360, 200, 106, Math.PI * 0.12, Math.PI * 0.58)} fill="none" stroke="var(--color-success)" strokeWidth="7" strokeLinecap="round" opacity=".9" />
            <path d={arcPath(360, 200, 106, Math.PI * 0.65, Math.PI * 0.96)} fill="none" stroke="var(--color-warning)" strokeWidth="7" strokeLinecap="round" opacity=".9" />
            {exposures.slice(0, 6).map((exposure, index) => {
              const angle = (-Math.PI / 2) + (index * Math.PI * 2) / Math.max(Math.min(exposures.length, 6), 1);
              const x = 360 + Math.cos(angle) * 132;
              const y = 200 + Math.sin(angle) * 132;
              const healthy = classifications.find((item) => item.subdomain === exposure.subdomain)?.classification === FailureClass.HEALTHY;
              const nodeColor = healthy ? "var(--color-success)" : "var(--color-warning)";
              return <g key={exposure.scenario} tabIndex={0} role="img" aria-label={exposure.scenario + ", " + (healthy ? "healthy" : "needs attention")}><line x1="360" y1="200" x2={x} y2={y} stroke={nodeColor} strokeOpacity=".55" strokeWidth="2"/><circle cx={x} cy={y} r="21" fill="none" stroke={nodeColor} strokeOpacity=".22" strokeWidth="4" strokeDasharray={exposure.lease ? "18 7" : "0 0"} /><circle cx={x} cy={y} r="15" fill="var(--color-surface)" stroke={nodeColor} strokeWidth="3"/><circle cx={x} cy={y} r="4" fill={nodeColor} /><text x={x} y={y + 31} textAnchor="middle" fill="var(--color-muted-foreground)" fontSize="12">{exposure.scenario.slice(0, 16)}</text></g>;
            })}
            <circle cx="360" cy="200" r="42" fill="var(--color-surface-raised)" stroke="var(--color-primary)" strokeWidth="3" />
            <circle cx="360" cy="200" r="8" fill={constellationStatus === "healthy" ? "var(--color-success)" : "var(--color-warning)"} />
            <text x="360" y="245" textAnchor="middle" fill="var(--color-foreground)" fontSize="14" fontWeight="600">{t(strings.overview.tunnelHeading)}</text>
          </svg>
          <div className="absolute bottom-0 left-0 right-0 flex flex-wrap justify-center gap-2 text-xs text-app-muted-foreground">
            <span className="rounded-full border border-app-border bg-app-surface/80 px-3 py-1">{exposures.length} {t(strings.exposure.colUrl)}</span>
            <span className="rounded-full border border-app-border bg-app-surface/80 px-3 py-1">{coreCount} {t(strings.overview.coreLabel)}</span>
            <span className="rounded-full border border-app-border bg-app-surface/80 px-3 py-1">{leasedCount} {t(strings.overview.leasedLabel)}</span>
            <span className="rounded-full border border-app-border bg-app-surface/80 px-3 py-1">{constellationStatus}</span>
          </div>
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-center gap-x-5 gap-y-2 text-xs text-app-muted-foreground" aria-label={t(strings.overview.exposureHeading)}>
          <span className="inline-flex items-center gap-2"><i className="h-2 w-2 rounded-full bg-app-primary" aria-hidden="true" />{t(strings.overview.tunnelHeading)}</span>
          <span className="inline-flex items-center gap-2"><i className="h-2 w-2 rounded-full bg-app-success" aria-hidden="true" />{t(strings.overview.readinessReady)}</span>
          <span className="inline-flex items-center gap-2"><i className="h-2 w-2 rounded-full bg-app-warning" aria-hidden="true" />{t(strings.overview.readinessSetupRequired)}</span>
          <span className="inline-flex items-center gap-2"><i className="h-2 w-2 rounded-full border border-app-warning" aria-hidden="true" />{t(strings.overview.leasedLabel)}</span>
        </div>
      </section>

      <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-label={t(strings.overview.driftHeading)}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-muted-foreground">{t(strings.overview.driftHeading)}</p><h2 className="mt-1 text-lg font-semibold">{findingCount ? t(strings.overview.configure) : t(strings.overview.readinessReady)}</h2></div>
          <StatusBadge tone={findingCount ? "warning" : "success"}>{findingCount ? t(strings.overview.readinessSetupRequired) : t(strings.overview.readinessReady)}</StatusBadge>
        </div>
        {findingCount > 0 && <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {readiness?.missingFields.length ? <Finding testId={selectors.overview.finding} label={t(strings.config.remoteAvailability)} detail={readiness.missingFields.join(", ")} href="/settings" /> : null}
          {unhealthyCount ? <Finding testId={selectors.overview.finding} label={t(strings.metrics.heading)} detail={t(strings.metrics.error)} href="/metrics" /> : null}
          {recoveryState?.circuitOpen ? <Finding testId={selectors.overview.finding} label={t(strings.recovery.circuitOpenLabel)} detail={t(strings.overview.circuitOpenWarning)} href="/recovery" /> : null}
          {(driftCounts?.unmanaged ?? 0) > 0 ? <Finding testId={selectors.overview.finding} label={t(strings.overview.driftHeading)} detail={t(strings.overview.viewDrift)} href="/drift" /> : null}
        </div>}
      </section>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
      <Card testId={selectors.overview.readinessCard} heading={t(strings.overview.readinessHeading)}>
        <QueryState isLoading={configQuery.isLoading} error={configQuery.error} onRetry={() => void configQuery.refetch()}>
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
        <QueryState isLoading={tunnelQuery.isLoading} error={tunnelQuery.error} onRetry={() => void tunnelQuery.refetch()}>
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
        <QueryState isLoading={exposuresQuery.isLoading} error={exposuresQuery.error} onRetry={() => void exposuresQuery.refetch()}>
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
        <QueryState isLoading={recoveryQuery.isLoading} error={recoveryQuery.error} onRetry={() => void recoveryQuery.refetch()}>
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
        <QueryState isLoading={driftQuery.isLoading} error={driftQuery.error} onRetry={() => void driftQuery.refetch()}>
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
    </div>
  );
}
