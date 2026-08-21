import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Exposure } from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";
import type { RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";
import { FailureClass, ProbeStatus } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { exposureClient } from "../../api/exposure";
import { probesClient } from "../../api/probes";
import { routesClient } from "../../api/routes";
import { configClient } from "../../api/config";
import { failureClassLabel, failureClassTone } from "../metrics/labels";

const EXPOSURES_QUERY_KEY = ["exposures"] as const;
const CLASSIFY_QUERY_KEY = ["exposure-classify"] as const;

type TierFilter = "all" | "core" | "leased";
type HealthFilter = "all" | "healthy" | "attention" | "unknown";
type LeaseFilter = "all" | "active" | "none";

function tierTone(tier: string): BadgeTone {
  if (tier === "core") return "info";
  if (tier === "leased") return "success";
  return "neutral";
}

type TierKey = (typeof strings.exposure.tier)[keyof typeof strings.exposure.tier];

function tierLabel(tier: string): TierKey {
  if (tier === "core") return strings.exposure.tier.core;
  if (tier === "leased") return strings.exposure.tier.leased;
  return strings.exposure.tier.unknown;
}

function routeKey(value: string): string {
  return value.trim().toLowerCase();
}

function classificationFor(
  exposure: Exposure,
  classifications: RouteClassification[],
): RouteClassification | undefined {
  const keys = new Set([routeKey(exposure.subdomain), routeKey(exposure.scenario)]);
  return classifications.find((cls) => keys.has(routeKey(cls.subdomain)));
}

/**
 * ExposurePanel is the primary operations surface: the live table of every
 * exposed scenario (core + leased) plus the expose / extend / revoke actions.
 * Reads ListExposures; mutations invalidate the query so the table reflects the
 * reconciled state immediately.
 */
export function ExposurePanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");
  const [duration, setDuration] = useState("604800");
  const [reviewOpen, setReviewOpen] = useState(false);
  const [revokeLeaseId, setRevokeLeaseId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [tierFilter, setTierFilter] = useState<TierFilter>("all");
  const [healthFilter, setHealthFilter] = useState<HealthFilter>("all");
  const [leaseFilter, setLeaseFilter] = useState<LeaseFilter>("all");
  const [selectedExposure, setSelectedExposure] = useState<Exposure | null>(null);
  const [copied, setCopied] = useState(false);
  const [exposeResult, setExposeResult] = useState<{
    publicUrl: string;
    localPort: number;
    expiresAt?: Date;
    portAssigned: boolean;
  } | null>(null);
  const [reconcileResult, setReconcileResult] = useState<{
    coreEnsured: number;
    leasesReaped: number;
  } | null>(null);

  const exposuresQuery = useQuery({
    queryKey: EXPOSURES_QUERY_KEY,
    queryFn: () => exposureClient.listExposures({}),
  });

  const classifyQuery = useQuery({
    queryKey: CLASSIFY_QUERY_KEY,
    queryFn: () => probesClient.classify({}),
  });
  const routesQuery = useQuery({
    queryKey: ["routes"],
    queryFn: () => routesClient.listRoutes({}),
  });
  const accessQuery = useQuery({
    queryKey: ["access-status"],
    queryFn: () => configClient.getAccessStatus({}),
  });
  const detailProbesQuery = useQuery({
    queryKey: ["exposure-detail-probes", selectedExposure?.subdomain],
    queryFn: () => probesClient.listProbes({ subdomain: selectedExposure?.subdomain ?? "", limit: 8 }),
    enabled: Boolean(selectedExposure),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: EXPOSURES_QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: CLASSIFY_QUERY_KEY });
  };

  const exposeMutation = useMutation({
    mutationFn: (request: { scenario: string; ttlSeconds: bigint }) => exposureClient.expose(request),
    onSuccess: (response) => {
      setExposeResult({
        publicUrl: response.publicUrl,
        localPort: response.localPort,
        expiresAt: response.lease?.expiresAt ? timestampDate(response.lease.expiresAt) : undefined,
        portAssigned: response.portAssigned,
      });
      setScenario("");
      setReviewOpen(false);
      invalidate();
    },
  });

  const extendMutation = useMutation({
    mutationFn: (leaseId: string) => exposureClient.extendLease({ leaseId }),
    onSuccess: () => invalidate(),
  });

  const reconcileMutation = useMutation({
    mutationFn: () => exposureClient.reconcile({}),
    onSuccess: (resp) => {
      setReconcileResult({
        coreEnsured: resp.coreEnsured,
        leasesReaped: resp.leasesReaped,
      });
      invalidate();
    },
  });

  const revokeMutation = useMutation({
    mutationFn: (leaseId: string) => exposureClient.revokeLease({ leaseId }),
    onSuccess: () => invalidate(),
  });

  const actionError = extendMutation.error ?? revokeMutation.error ?? reconcileMutation.error;
  const exposures = exposuresQuery.data?.exposures ?? [];
  const classifications = classifyQuery.data?.classifications ?? [];
  const routes = routesQuery.data?.routes ?? [];
  const selectedRoute = routes.find((route) => route.scenario === scenario.trim());
  const accessStatus = accessQuery.data?.status;
  const coreCount = exposures.filter((exposure) => exposure.tier === "core").length;
  const leasedCount = exposures.filter((exposure) => exposure.tier === "leased").length;
  const unhealthyCount = exposures.filter((exposure) => {
    const classification = classificationFor(exposure, classifications);
    return classification && classification.classification !== FailureClass.HEALTHY;
  }).length;
  const normalizedSearch = search.trim().toLowerCase();
  const filteredExposures = exposures.filter((exposure) => {
    const matchesTier = tierFilter === "all" || exposure.tier === tierFilter;
    if (!matchesTier) return false;
    const classification = classificationFor(exposure, classifications);
    const matchesHealth = healthFilter === "all"
      || (healthFilter === "unknown" && !classification)
      || (healthFilter === "healthy" && classification?.classification === FailureClass.HEALTHY)
      || (healthFilter === "attention" && classification && classification.classification !== FailureClass.HEALTHY);
    if (!matchesHealth) return false;
    const matchesLease = leaseFilter === "all" || (leaseFilter === "active" && Boolean(exposure.lease)) || (leaseFilter === "none" && !exposure.lease);
    if (!matchesLease) return false;
    if (!normalizedSearch) return true;
    return [exposure.scenario, exposure.subdomain, exposure.publicUrl]
      .some((value) => value.toLowerCase().includes(normalizedSearch));
  });

  const handleExpose = (e: React.FormEvent) => {
    e.preventDefault();
    const name = scenario.trim();
    if (name) setReviewOpen(true);
  };

  return (
    <section data-testid={selectors.exposure.panel} className="flex flex-col gap-6">
      <div
        data-testid={selectors.exposure.summary}
        className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4 sm:grid-cols-3"
      >
        <div>
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">{t(strings.exposure.heading)}</p>
          <p className="mt-1 text-sm text-app-muted-foreground">
            {t(strings.exposure.summary, {
              total: exposures.length,
              core: coreCount,
              leased: leasedCount,
            })}
          </p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.exposure.tier.core)}
          </p>
          <p data-testid={selectors.exposure.coreCount} className="mt-1 text-2xl font-semibold tabular-nums">
            {coreCount}
          </p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.exposure.unhealthySummary, { count: unhealthyCount })}
          </p>
          <p
            data-testid={selectors.exposure.unhealthyCount}
            className="mt-1 text-2xl font-semibold tabular-nums"
          >
            {unhealthyCount}
          </p>
        </div>
      </div>

      <form
        data-testid={selectors.exposure.exposeForm}
        onSubmit={handleExpose}
        className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4 sm:flex-row sm:items-end"
      >
        <label className="flex flex-1 flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.exposure.exposeHeading)}</span>
          <Input
            data-testid={selectors.exposure.exposeInput}
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            placeholder={t(strings.exposure.exposePlaceholder)}
            aria-label={t(strings.exposure.exposeHeading)}
            list="known-tunnel-routes"
          />
          <datalist id="known-tunnel-routes">
            {routes.filter((route) => route.scenario).map((route) => <option key={route.scenario} value={route.scenario}>{route.publicUrl}</option>)}
          </datalist>
        </label>
        <Button
          type="submit"
          data-testid={selectors.exposure.exposeButton}
          disabled={exposeMutation.isPending || scenario.trim() === ""}
        >
          {t(strings.exposure.exposeButton)}
        </Button>
      </form>
      {scenario.trim() && selectedRoute ? (
        <div className="grid gap-3 rounded-panel border border-app-primary/25 bg-app-primary/5 p-4 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-center">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-app-primary">{t(strings.exposure.exposeHeading)}</p>
            <p className="mt-1 truncate text-base font-semibold text-app-foreground">{selectedRoute.scenario}</p>
            <p className="mt-1 truncate text-sm text-app-muted-foreground">{selectedRoute.publicUrl}</p>
          </div>
          <ReviewField label={t(strings.exposure.colPort)} value={String(selectedRoute.localPort)} />
          <ReviewField label={t(strings.exposure.colTier)} value={t(strings.exposure.tier.leased)} />
        </div>
      ) : null}
      {exposeResult && (
        <div className="rounded-panel border border-app-success/40 bg-app-success/5 p-4" role="status">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-app-success">{t(strings.exposure.exposeButton)}</p>
              <a href={exposeResult.publicUrl} target="_blank" rel="noreferrer" className="mt-1 block break-all text-sm text-app-primary underline-offset-2 hover:underline">{exposeResult.publicUrl}</a>
              <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.exposure.leaseActive, { when: exposeResult.expiresAt ? formatDate(exposeResult.expiresAt, { dateStyle: "medium", timeStyle: "short" }) : t(strings.common.never) })} · {exposeResult.localPort}</p>
            </div>
            {exposeResult.portAssigned && <StatusBadge tone="warning">{t(strings.exposure.colPort)}</StatusBadge>}
          </div>
        </div>
      )}
      {reviewOpen && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-950/60 p-0 backdrop-blur-sm sm:items-center sm:p-6" role="presentation">
          <section
            data-testid={selectors.exposure.reviewDialog}
            role="dialog"
            aria-modal="true"
            aria-labelledby="exposure-review-title"
            className="w-full max-w-2xl rounded-t-[1.25rem] border border-app-border bg-app-surface p-5 shadow-2xl sm:rounded-[1.25rem] sm:p-7"
          >
            <div className="flex items-start justify-between gap-4">
              <div><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{t(strings.exposure.exposeHeading)}</p><h2 id="exposure-review-title" className="mt-1 text-2xl font-semibold">{scenario.trim()}</h2><p className="mt-2 text-sm text-app-muted-foreground">{t(strings.config.setupDescription)}</p></div>
              <button type="button" aria-label={t(strings.common.refresh)} onClick={() => setReviewOpen(false)} className="touch-target rounded-full text-2xl text-app-muted-foreground hover:bg-app-surface-muted">×</button>
            </div>
            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              <ReviewField label={t(strings.config.localConfigPath)} value={selectedRoute ? `${selectedRoute.scenario} · ${selectedRoute.localPort}` : t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.colUrl)} value={selectedRoute?.publicUrl || t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.colTier)} value={selectedRoute ? t(strings.exposure.tier.leased) : t(strings.exposure.healthUnknown)} />
              <label className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span className="block text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.exposure.colLease)}</span><select data-testid={selectors.exposure.durationSelect} value={duration} onChange={(event) => setDuration(event.target.value)} className="mt-2 w-full bg-transparent font-semibold text-app-foreground"><option value="3600">{t(strings.exposure.durationHour)}</option><option value="86400">{t(strings.exposure.durationDay)}</option><option value="604800">{t(strings.exposure.durationWeek)}</option></select></label>
            </div>
            <div className="mt-4 rounded-control border border-app-primary/30 bg-app-primary/5 p-4 text-sm"><p className="font-semibold">{t(strings.config.publicExposureHeading)}</p><p className="mt-1 text-app-muted-foreground">{accessStatus?.enabled ? t(strings.config.publicExposureOn) : t(strings.config.publicExposureOff)}{accessStatus?.configured === false ? ` · ${t(strings.config.remoteAvailability)}` : ""}</p></div>
            <div className="mt-6 flex flex-col-reverse justify-end gap-2 sm:flex-row"><Button variant="outline" data-testid={selectors.exposure.cancelExposeButton} onClick={() => setReviewOpen(false)}>{t(strings.common.refresh)}</Button><Button data-testid={selectors.exposure.confirmExposeButton} disabled={exposeMutation.isPending} onClick={() => exposeMutation.mutate({ scenario: scenario.trim(), ttlSeconds: BigInt(duration) })}>{exposeMutation.isPending ? t(strings.common.loading) : t(strings.exposure.exposeButton)}</Button></div>
          </section>
        </div>
      )}
      {exposeMutation.error && (
        <p data-testid={selectors.exposure.exposeError} role="alert" className="text-sm text-app-danger">
          {errorMessage(exposeMutation.error, t)}
        </p>
      )}

      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="grid flex-1 gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.searchLabel)}</span>
            <Input
              data-testid={selectors.exposure.searchInput}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t(strings.exposure.searchPlaceholder)}
            />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.tierFilterLabel)}</span>
            <select
              data-testid={selectors.exposure.tierFilter}
              value={tierFilter}
              onChange={(e) => setTierFilter(e.target.value as TierFilter)}
              className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            >
              <option value="all">{t(strings.exposure.tierFilterAll)}</option>
              <option value="core">{t(strings.exposure.tierFilterCore)}</option>
              <option value="leased">{t(strings.exposure.tierFilterLeased)}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.colHealth)}</span>
            <select data-testid={selectors.exposure.healthFilter} value={healthFilter} onChange={(e) => setHealthFilter(e.target.value as HealthFilter)} className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground">
              <option value="all">{t(strings.exposure.tierFilterAll)}</option><option value="healthy">{t(strings.overview.readinessReady)}</option><option value="attention">{t(strings.overview.readinessSetupRequired)}</option><option value="unknown">{t(strings.exposure.healthUnknown)}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.colLease)}</span>
            <select data-testid={selectors.exposure.leaseFilter} value={leaseFilter} onChange={(e) => setLeaseFilter(e.target.value as LeaseFilter)} className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground">
              <option value="all">{t(strings.exposure.tierFilterAll)}</option><option value="active">{t(strings.exposure.leaseActive, { when: t(strings.common.never) })}</option><option value="none">{t(strings.exposure.leaseNone)}</option>
            </select>
          </label>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            data-testid={selectors.exposure.refreshButton}
            onClick={() => {
              void exposuresQuery.refetch();
              void classifyQuery.refetch();
            }}
          >
            {t(strings.common.refresh)}
          </Button>
          <Button
            data-testid={selectors.exposure.reconcileButton}
            disabled={reconcileMutation.isPending}
            onClick={() => reconcileMutation.mutate()}
          >
            {t(strings.exposure.reconcileButton)}
          </Button>
        </div>
      </div>

      {reconcileResult && (
        <p data-testid={selectors.exposure.reconcileResult} className="text-sm text-app-muted-foreground">
          {t(strings.exposure.reconcileResult, {
            core: reconcileResult.coreEnsured,
            leases: reconcileResult.leasesReaped,
          })}
        </p>
      )}

      {actionError && (
        <p data-testid={selectors.exposure.actionError} role="alert" className="text-sm text-app-danger">
          {t(strings.exposure.actionError)}
        </p>
      )}

      {revokeLeaseId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-6 backdrop-blur-sm" role="presentation">
          <section role="alertdialog" aria-modal="true" aria-labelledby="revoke-title" className="w-full max-w-md rounded-[1.25rem] border border-app-border bg-app-surface p-6 shadow-2xl">
            <h2 id="revoke-title" className="text-xl font-semibold">{t(strings.exposure.revokeButton)}</h2>
            <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.config.publicExposureBody)}</p>
            <div className="mt-6 flex justify-end gap-2"><Button variant="outline" onClick={() => setRevokeLeaseId(null)}>{t(strings.common.refresh)}</Button><Button variant="outline" data-testid={selectors.exposure.revokeConfirmButton} className="border-app-danger text-app-danger hover:bg-app-danger/10" disabled={revokeMutation.isPending} onClick={() => { revokeMutation.mutate(revokeLeaseId); setRevokeLeaseId(null); }}>{t(strings.exposure.revokeButton)}</Button></div>
          </section>
        </div>
      )}

      <QueryState
        isLoading={exposuresQuery.isLoading}
        error={exposuresQuery.error}
        onRetry={() => void exposuresQuery.refetch()}
        isEmpty={exposures.length === 0 || filteredExposures.length === 0}
        loadingLabel={t(strings.exposure.loading)}
        errorLabel={t(strings.exposure.error)}
        emptyLabel={exposures.length === 0 ? t(strings.exposure.empty) : t(strings.exposure.filteredEmpty)}
      >
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.exposure.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.exposure.colScenario)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colTier)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colHealth)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colUrl)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colPort)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colLease)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colActions)}</th>
              </tr>
            </thead>
            <tbody>
              {filteredExposures.map((exposure: Exposure) => {
                const lease = exposure.lease;
                const classification = classificationFor(exposure, classifications);
                return (
                  <tr
                    key={exposure.scenario}
                    data-testid={selectors.exposure.row}
                    className="border-b border-app-border last:border-0"
                  >
                    <td className="px-3 py-2 font-medium"><button type="button" className="text-left text-app-primary underline-offset-2 hover:underline" onClick={() => setSelectedExposure(exposure)}>{exposure.scenario}</button></td>
                    <td className="px-3 py-2">
                      <StatusBadge tone={tierTone(exposure.tier)} data-testid={selectors.exposure.tierBadge}>
                        {t(tierLabel(exposure.tier))}
                      </StatusBadge>
                    </td>
                    <td className="px-3 py-2">
                      {classification ? (
                        <StatusBadge
                          tone={failureClassTone(classification.classification)}
                          data-testid={selectors.exposure.healthBadge}
                        >
                          {t(failureClassLabel(classification.classification))}
                        </StatusBadge>
                      ) : (
                        <StatusBadge tone="neutral" data-testid={selectors.exposure.healthBadge}>
                          {t(strings.exposure.healthUnknown)}
                        </StatusBadge>
                      )}
                    </td>
                    <td className="px-3 py-2">
                      <a
                        data-testid={selectors.exposure.url}
                        href={exposure.publicUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="text-app-primary underline-offset-2 hover:underline"
                      >
                        {exposure.publicUrl}
                      </a>
                    </td>
                    <td className="px-3 py-2 tabular-nums">{exposure.localPort}</td>
                    <td data-testid={selectors.exposure.leaseExpiry} className="px-3 py-2">
                      {lease?.expiresAt
                        ? t(strings.exposure.leaseActive, {
                            when: formatDate(timestampDate(lease.expiresAt), {
                              dateStyle: "medium",
                              timeStyle: "short",
                            }),
                          })
                        : t(strings.exposure.leaseNone)}
                    </td>
                    <td className="px-3 py-2">
                      {lease && (
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            data-testid={selectors.exposure.extendButton}
                            disabled={extendMutation.isPending}
                            onClick={() => extendMutation.mutate(lease.id)}
                          >
                            {t(strings.exposure.extendButton)}
                          </Button>
                          <Button
                            variant="outline"
                            data-testid={selectors.exposure.revokeButton}
                            disabled={revokeMutation.isPending}
                            onClick={() => setRevokeLeaseId(lease.id)}
                          >
                            {t(strings.exposure.revokeButton)}
                          </Button>
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </QueryState>

      {selectedExposure && (
        <div className="fixed inset-0 z-50 flex items-stretch justify-end bg-slate-950/60 backdrop-blur-sm" role="presentation">
          <aside data-testid={selectors.exposure.detailDialog} role="dialog" aria-modal="true" aria-labelledby="exposure-detail-title" className="h-full w-full max-w-xl overflow-y-auto border-l border-app-border bg-app-surface p-6 shadow-2xl sm:p-8">
            <div className="flex items-start justify-between gap-4"><div><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{t(strings.exposure.heading)}</p><h2 id="exposure-detail-title" className="mt-1 text-2xl font-semibold">{selectedExposure.scenario}</h2><p className="mt-1 text-sm text-app-muted-foreground">{selectedExposure.publicUrl}</p></div><button type="button" data-testid={selectors.exposure.detailCloseButton} aria-label={t(strings.common.refresh)} onClick={() => setSelectedExposure(null)} className="touch-target rounded-full text-2xl text-app-muted-foreground">×</button></div>
            <div className="mt-6 grid gap-3 sm:grid-cols-2"><ReviewField label={t(strings.exposure.colPort)} value={String(selectedExposure.localPort)} /><ReviewField label={t(strings.exposure.colTier)} value={selectedExposure.tier} /><ReviewField label={t(strings.exposure.colLease)} value={selectedExposure.lease?.expiresAt ? formatDate(timestampDate(selectedExposure.lease.expiresAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.exposure.leaseNone)} /><ReviewField label={t(strings.config.publicExposureHeading)} value={t(strings.config.publicExposureOff)} /></div>
            <div className="mt-5 flex flex-wrap gap-2"><Button variant="outline" data-testid={selectors.exposure.detailCopyButton} onClick={() => { void navigator.clipboard.writeText(selectedExposure.publicUrl).then(() => { setCopied(true); window.setTimeout(() => setCopied(false), 1800); }); }}>{copied ? t(strings.common.refresh) : t(strings.exposure.colUrl)}</Button><Button data-testid={selectors.exposure.detailProbeButton} disabled={detailProbesQuery.isFetching} onClick={() => void detailProbesQuery.refetch()}>{t(strings.metrics.runProbesButton)}</Button><a href={selectedExposure.publicUrl} target="_blank" rel="noreferrer" className="inline-flex h-11 items-center rounded-control border border-app-border px-5 text-sm font-medium text-app-foreground hover:bg-app-surface-muted">{t(strings.exposure.colUrl)}</a></div>
            <section className="mt-8"><h3 className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.metrics.probesHeading)}</h3>{detailProbesQuery.isLoading ? <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.common.loading)}</p> : detailProbesQuery.data?.results.length ? <ul className="mt-3 flex flex-col gap-2">{detailProbesQuery.data.results.map((probe) => <li key={probe.id} className="flex items-center justify-between rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span>{probe.kind}</span><StatusBadge tone={probe.status === ProbeStatus.UP ? "success" : "warning"}>{String(probe.status)}</StatusBadge></li>)}</ul> : <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.metrics.probesEmpty)}</p>}</section>
            <div className="mt-8 rounded-control border border-app-primary/30 bg-app-primary/5 p-4 text-sm"><p className="font-semibold">{t(strings.config.publicExposureHeading)}</p><p className="mt-1 text-app-muted-foreground">{t(strings.config.publicExposureBody)}</p></div>
          </aside>
        </div>
      )}
    </section>
  );
}

function ReviewField({ label, value }: { label: string; value: string }) {
  return <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span className="block text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{label}</span><span className="mt-1 block font-semibold text-app-foreground">{value}</span></div>;
}
