import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import { ExposureSchema, LeaseStatus, type Exposure, type Lease } from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";
import type { RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";
import { FailureClass, ProbeStatus } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { QueryState } from "../../components/ui/QueryState";
import { RouteInventoryTable, type RouteInventoryRow } from "../../components/ui/RouteInventoryTable";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { ApiError } from "../../api/client";
import { exposureClient } from "../../api/exposure";
import { probesClient } from "../../api/probes";
import { PublicExposure, routesClient, Tier } from "../../api/routes";
import { configClient } from "../../api/config";
import { failureClassLabel, failureClassTone, probeKindLabel, probeStatusLabel, probeStatusTone } from "../metrics/labels";
import { useEscapeKey } from "../../hooks/useEscapeKey";

const EXPOSURES_QUERY_KEY = ["exposures"] as const;
const CLASSIFY_QUERY_KEY = ["exposure-classify"] as const;

type TierFilter = "all" | "core" | "leased";
type HealthFilter = "all" | "healthy" | "attention" | "unknown";
type LeaseFilter = "all" | "active" | "none";
type PolicyFilter = "all" | "protected" | "public" | "unknown";
type ReadinessFilter = "all" | "ready" | "attention" | "unknown";

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

function hostnameFor(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return "—";
  }
}

function classificationFor(
  exposure: Exposure,
  classifications: RouteClassification[],
): RouteClassification | undefined {
  const keys = new Set([routeKey(exposure.subdomain), routeKey(exposure.scenario)]);
  return classifications.find((cls) => keys.has(routeKey(cls.subdomain)));
}

function latestExpiredLeases(leases: Lease[]): Lease[] {
  const latestByScenario = new Map<string, Lease>();
  for (const lease of leases) {
    const current = latestByScenario.get(lease.scenario);
    if (!current || (lease.createdAt && (!current.createdAt || timestampDate(lease.createdAt) > timestampDate(current.createdAt)))) {
      latestByScenario.set(lease.scenario, lease);
    }
  }
  return [...latestByScenario.values()].sort((a, b) => a.scenario.localeCompare(b.scenario));
}

function provisioningRemediation(error: unknown, t: TFunction): string {
  const code = error instanceof ConnectError
    ? error.code
    : error instanceof ApiError
      ? error.code
      : "";
  if (code === Code.FailedPrecondition || code === "failed_precondition") return t(strings.exposure.provisioningRemediationPrecondition);
  if (code === Code.PermissionDenied || code === Code.Unauthenticated || code === "permission_denied" || code === "unauthenticated") return t(strings.exposure.provisioningRemediationPermission);
  if (code === Code.Unavailable || code === Code.DeadlineExceeded || code === "unavailable" || code === "deadline_exceeded") return t(strings.exposure.provisioningRemediationUnavailable);
  return t(strings.exposure.provisioningRemediationUnknown);
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [scenario, setScenario] = useState(() => searchParams.get("scenario") ?? "");
  const [duration, setDuration] = useState("604800");
  const [customHours, setCustomHours] = useState("24");
  const [reviewOpen, setReviewOpen] = useState(false);
  const [policyAcknowledged, setPolicyAcknowledged] = useState(false);
  const [revokeLeaseId, setRevokeLeaseId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [tierFilter, setTierFilter] = useState<TierFilter>("all");
  const [healthFilter, setHealthFilter] = useState<HealthFilter>("all");
  const [leaseFilter, setLeaseFilter] = useState<LeaseFilter>("all");
  const [policyFilter, setPolicyFilter] = useState<PolicyFilter>("all");
  const [readinessFilter, setReadinessFilter] = useState<ReadinessFilter>("all");
  const [showAllExpired, setShowAllExpired] = useState(false);
  const [selectedExposure, setSelectedExposure] = useState<Exposure | null>(null);
  const [selectedExpiredLease, setSelectedExpiredLease] = useState<Lease | null>(null);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState(false);
  const [verificationRequested, setVerificationRequested] = useState(false);
  const [exposeResult, setExposeResult] = useState<{
    scenario: string;
    publicUrl: string;
    localPort: number;
    expiresAt?: Date;
    portAssigned: boolean;
  } | null>(null);
  const [reconcileResult, setReconcileResult] = useState<{
    coreEnsured: number;
    leasesReaped: number;
  } | null>(null);
  const [leaseNotice, setLeaseNotice] = useState<{
    kind: "extend" | "revoke";
    when?: string;
    state?: string;
  } | null>(null);
  const reviewFocusRef = useRef<HTMLInputElement>(null);
  const detailFocusRef = useRef<HTMLButtonElement>(null);
  const revokeFocusRef = useRef<HTMLButtonElement>(null);

  const modalOpen = reviewOpen || Boolean(selectedExposure) || Boolean(selectedExpiredLease) || Boolean(revokeLeaseId);

  useEscapeKey(modalOpen, () => {
    if (revokeLeaseId) {
      setRevokeLeaseId(null);
    } else if (reviewOpen) {
      setReviewOpen(false);
    } else {
      setSelectedExposure(null);
      setSelectedExpiredLease(null);
    }
  });

  useEffect(() => {
    if (!modalOpen) return undefined;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [modalOpen]);

  useEffect(() => {
    if (revokeLeaseId) revokeFocusRef.current?.focus();
    else if (reviewOpen) reviewFocusRef.current?.focus();
    else if (selectedExposure || selectedExpiredLease) detailFocusRef.current?.focus();
  }, [reviewOpen, revokeLeaseId, selectedExposure, selectedExpiredLease]);

  const exposuresQuery = useQuery({
    queryKey: EXPOSURES_QUERY_KEY,
    queryFn: () => exposureClient.listExposures({}),
  });
  const expiredLeasesQuery = useQuery({
    queryKey: ["expired-leases"],
    queryFn: () => exposureClient.listLeases({ status: LeaseStatus.EXPIRED }),
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
  const runDetailProbesMutation = useMutation({
    mutationFn: () => probesClient.runProbes({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["exposure-detail-probes"] });
      void queryClient.invalidateQueries({ queryKey: CLASSIFY_QUERY_KEY });
    },
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: EXPOSURES_QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: CLASSIFY_QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: ["expired-leases"] });
  };

  const exposeMutation = useMutation({
    mutationFn: (request: { scenario: string; ttlSeconds: bigint }) => exposureClient.expose(request),
    onSuccess: (response) => {
      setExposeResult({
        scenario: scenario.trim(),
        publicUrl: response.publicUrl,
        localPort: response.localPort,
        expiresAt: response.lease?.expiresAt ? timestampDate(response.lease.expiresAt) : undefined,
        portAssigned: response.portAssigned,
      });
      setVerificationRequested(false);
      setScenario("");
      setReviewOpen(false);
      invalidate();
      void queryClient.invalidateQueries({ queryKey: ["routes"] });
      void queryClient.invalidateQueries({ queryKey: ["access-status"] });
    },
  });

  const extendMutation = useMutation({
    mutationFn: (leaseId: string) => exposureClient.extendLease({ leaseId }),
    onSuccess: (response) => {
      setLeaseNotice({
        kind: "extend",
        when: response.lease?.expiresAt ? formatDate(timestampDate(response.lease.expiresAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.common.never),
      });
      if (response.lease) {
        const updatedLease = response.lease;
        setSelectedExposure((current) => current && current.lease?.id === updatedLease.id
          ? create(ExposureSchema, {
              scenario: current.scenario,
              subdomain: current.subdomain,
              publicUrl: current.publicUrl,
              localPort: current.localPort,
              tier: current.tier,
              enabled: current.enabled,
              lease: updatedLease,
            })
          : current);
      }
      invalidate();
    },
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
    onSuccess: (response) => {
      setLeaseNotice({ kind: "revoke", state: response.retracted ? t(strings.config.publicExposureOff) : t(strings.exposure.tier.core) });
      if (response.retracted) setSelectedExposure((current) => current?.lease ? null : current);
      invalidate();
    },
  });

  const actionError = extendMutation.error ?? revokeMutation.error ?? reconcileMutation.error ?? runDetailProbesMutation.error;
  const copySelectedUrl = async () => {
    setCopyError(false);
    try {
      await navigator.clipboard.writeText(selectedExposure?.publicUrl ?? "");
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setCopyError(true);
    }
  };
  const exposures = useMemo(() => exposuresQuery.data?.exposures ?? [], [exposuresQuery.data?.exposures]);
  const expiredLeases = latestExpiredLeases(expiredLeasesQuery.data?.leases ?? []);
  const classifications = classifyQuery.data?.classifications ?? [];
  const routes = useMemo(() => routesQuery.data?.routes ?? [], [routesQuery.data?.routes]);
  const routeByScenario = useMemo(() => new Map(routes.map((route) => [route.scenario, route])), [routes]);
  const selectedRoute = routes.find((route) => route.scenario === scenario.trim());
  const selectedExpiredRoute = selectedExpiredLease
    ? routes.find((route) => route.scenario === selectedExpiredLease.scenario)
    : undefined;
  const routeCatalogReady = routesQuery.isSuccess && routes.length > 0;
  const accessStatus = accessQuery.data?.status;
  const selectedAccessHost = selectedRoute
    ? accessStatus?.hosts.find((host) => host.host === hostnameFor(selectedRoute.publicUrl))
    : undefined;
  const selectedExposureRoute = selectedExposure
    ? routes.find((route) => route.scenario === selectedExposure.scenario)
    : undefined;
  const selectedExposureClassification = selectedExposure
    ? classificationFor(selectedExposure, classifications)
    : undefined;
  const selectedExposureAccessHost = selectedExposureRoute
    ? accessStatus?.hosts.find((host) => host.host === hostnameFor(selectedExposureRoute.publicUrl))
    : undefined;
  const selectedExposureLease = selectedExposure?.lease;
  const resultClassification = exposeResult
    ? classifications.find((classification) => routeKey(classification.subdomain) === routeKey(exposeResult.scenario))
    : undefined;
  const resultRoute = exposeResult ? routes.find((route) => route.scenario === exposeResult.scenario) : undefined;
  const resultAccessHost = resultRoute
    ? accessStatus?.hosts.find((host) => host.host === hostnameFor(resultRoute.publicUrl))
    : undefined;
  const resultPolicyConfigured = Boolean(
    verificationRequested &&
    accessStatus &&
    (!accessStatus.enabled || resultAccessHost?.effectiveBypass),
  );
  const coreCount = exposures.filter((exposure) => exposure.tier === "core").length;
  const leasedCount = exposures.filter((exposure) => exposure.tier === "leased").length;
  const exposureLoading = exposuresQuery.isLoading || expiredLeasesQuery.isLoading || classifyQuery.isLoading || routesQuery.isLoading || accessQuery.isLoading;
  const exposureError = exposuresQuery.error || expiredLeasesQuery.error || classifyQuery.error || routesQuery.error || accessQuery.error;
  const experienceState = exposureLoading
    ? "loading"
    : exposureError && exposures.length === 0
      ? "error"
      : exposures.length === 0
        ? "empty"
        : exposureError
          ? "partial"
          : "ready";
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
    const route = routeByScenario.get(exposure.scenario);
    const accessHost = route
      ? accessStatus?.hosts.find((host) => host.host === hostnameFor(route.publicUrl))
      : undefined;
    const policy = !route
      ? "unknown"
      : route.publicExposure === PublicExposure.ENABLED
        ? "public"
        : route.publicExposure === PublicExposure.DISABLED
          ? "protected"
          : !accessStatus
            ? "unknown"
            : accessHost?.effectiveBypass
              ? "public"
              : "protected";
    const readiness = !route || !classification
      ? "unknown"
      : route.enabled && classification.classification === FailureClass.HEALTHY
        ? "ready"
        : "attention";
    if (policyFilter !== "all" && policy !== policyFilter) return false;
    if (readinessFilter !== "all" && readiness !== readinessFilter) return false;
    if (!normalizedSearch) return true;
    return [exposure.scenario, exposure.subdomain, exposure.publicUrl]
      .some((value) => value.toLowerCase().includes(normalizedSearch));
  });
  const inventoryRows: RouteInventoryRow[] = filteredExposures.map((exposure) => ({
    exposure,
    classification: classificationFor(exposure, classifications),
  }));
  const durationSeconds = duration === "custom" ? Math.round(Number(customHours) * 60 * 60) : Number(duration);
  const durationValid = Number.isInteger(durationSeconds) && durationSeconds >= 60 * 60 && durationSeconds <= 168 * 60 * 60;
  const projectedExpiry = durationValid ? formatDate(new Date(Date.now() + durationSeconds * 1000), { dateStyle: "medium", timeStyle: "short" }) : t(strings.exposure.healthUnknown);

  useEffect(() => {
    if (!searchParams.get("scenario") || exposuresQuery.isLoading) return;
    const requestedScenario = searchParams.get("scenario")?.trim() ?? "";
    const existingExposure = exposures.find((exposure) => routeKey(exposure.scenario) === routeKey(requestedScenario));
    if (existingExposure) {
      setSelectedExposure(existingExposure);
      setScenario("");
      setSearchParams((params) => {
        params.delete("scenario");
        return params;
      }, { replace: true });
    }
  }, [exposures, exposuresQuery.isLoading, searchParams, setSearchParams]);

  const handleExpose = (e: React.FormEvent) => {
    e.preventDefault();
    const name = scenario.trim();
    if (name && selectedRoute) {
      setPolicyAcknowledged(false);
      setReviewOpen(true);
    }
  };

  return (
    <section data-testid={selectors.exposure.panel} data-experience-surface="exposure-results" data-experience-state={experienceState} className="flex flex-col gap-6">
      <div
        data-testid={selectors.exposure.summary}
        className="grid grid-cols-2 gap-3 rounded-panel border border-app-border bg-app-surface p-4 sm:grid-cols-4"
      >
        <div className="col-span-2 sm:col-span-1">
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
            {t(strings.exposure.tier.leased)}
          </p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{leasedCount}</p>
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
          <select
            data-testid={selectors.exposure.exposeInput}
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            aria-label={t(strings.exposure.exposeHeading)}
            aria-describedby="exposure-route-selection-hint"
            disabled={!routeCatalogReady}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            <option value="">
              {routesQuery.isLoading ? t(strings.exposure.routeCatalogLoading) : t(strings.exposure.exposePlaceholder)}
            </option>
            {routes.filter((route) => route.scenario).map((route) => (
              <option key={route.scenario} value={route.scenario}>
                {route.scenario} · {route.publicUrl}
              </option>
            ))}
          </select>
          <span id="exposure-route-selection-hint" className="text-xs text-app-muted-foreground">
            {routesQuery.isLoading
              ? t(strings.exposure.routeCatalogLoading)
              : routesQuery.error
                ? t(strings.exposure.routeCatalogError)
                : scenario.trim() && !selectedRoute
                  ? t(strings.exposure.chooseKnownRoute)
                  : t(strings.exposure.reviewDescription)}
          </span>
          {routes.length > 0 ? <div className="mt-2" aria-label={t(strings.exposure.knownRoutesLabel)}><span className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.exposure.knownRoutesLabel)}</span><div className="mt-1 flex flex-wrap gap-1.5">{routes.filter((route) => route.scenario).slice(0, 6).map((route) => <button key={route.scenario} type="button" onClick={() => setScenario(route.scenario)} className={`min-h-11 rounded-pill border px-2.5 py-1 text-xs font-medium transition-colors ${route.scenario === scenario.trim() ? "border-app-primary bg-app-primary/10 text-app-primary" : "border-app-border bg-app-surface-muted text-app-muted-foreground hover:border-app-primary hover:text-app-primary"}`}>{route.scenario}</button>)}</div></div> : null}
          {routesQuery.error ? <Button type="button" variant="outline" size="sm" className="self-start" onClick={() => void routesQuery.refetch()}>{t(strings.common.refresh)}</Button> : null}
        </label>
        <Button
          type="submit"
          data-testid={selectors.exposure.exposeButton}
          disabled={exposeMutation.isPending || scenario.trim() === "" || !selectedRoute || !routeCatalogReady}
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
          <ReviewField label={t(strings.exposure.colTier)} value={t(selectedRoute.tier === Tier.CORE ? strings.exposure.tier.core : strings.exposure.tier.leased)} />
        </div>
      ) : null}
      {expiredLeases.length > 0 ? (
        <section data-testid={selectors.exposure.expiredPanel} className="rounded-panel border border-app-warning/30 bg-app-warning/5 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div><p className="text-xs font-semibold uppercase tracking-[0.16em] text-app-warning">{t(strings.exposure.expiredHeading)}</p><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.exposure.expiredDescription)}</p></div>
            <StatusBadge tone="warning">{expiredLeases.length}</StatusBadge>
          </div>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
          {expiredLeases.slice(0, showAllExpired ? expiredLeases.length : 6).map((lease) => <div key={lease.scenario} className="flex items-center justify-between gap-3 rounded-control border border-app-warning/20 bg-app-surface p-3"><button type="button" className="min-w-0 truncate text-left text-sm font-medium text-app-primary hover:underline" onClick={() => setSelectedExpiredLease(lease)}>{lease.scenario}</button><Button variant="outline" size="sm" className="min-h-11" data-testid={selectors.exposure.reExposeButton} onClick={() => { setScenario(lease.scenario); setPolicyAcknowledged(false); setReviewOpen(true); }}>{t(strings.exposure.reExposeButton)}</Button></div>)}
          </div>
          {expiredLeases.length > 6 ? <Button type="button" variant="outline" size="sm" className="mt-3" onClick={() => setShowAllExpired((current) => !current)}>{showAllExpired ? t(strings.exposure.showFewerExpired) : t(strings.exposure.showAllExpired, { count: expiredLeases.length })}</Button> : null}
        </section>
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
          <div className="mt-4 border-t border-app-success/20 pt-4">
            <div className="flex flex-wrap items-center justify-between gap-2"><p className="text-xs font-semibold uppercase tracking-[0.16em] text-app-success">{t(strings.exposure.verificationHeading)}</p><Button type="button" variant="outline" size="sm" data-testid={selectors.exposure.verifyResultButton} disabled={classifyQuery.isFetching || accessQuery.isFetching} onClick={() => { setVerificationRequested(true); void Promise.all([classifyQuery.refetch(), accessQuery.refetch(), routesQuery.refetch()]); }}>{classifyQuery.isFetching || accessQuery.isFetching ? t(strings.exposure.verificationRunning) : t(strings.exposure.verificationRun)}</Button></div>
            <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
              {[
                { label: strings.exposure.verificationLocal, status: verificationRequested ? resultClassification?.internal : undefined, configured: undefined, notApplicable: false },
                { label: strings.exposure.verificationTunnel, status: undefined, configured: verificationRequested && resultRoute?.enabled, notApplicable: false },
                { label: strings.exposure.verificationPublic, status: verificationRequested ? resultClassification?.external : undefined, configured: undefined, notApplicable: false },
                { label: strings.exposure.verificationPolicy, status: undefined, configured: resultPolicyConfigured, notApplicable: Boolean(verificationRequested && accessStatus && !accessStatus.enabled) },
              ].map((stage) => {
                const statusLabel = stage.notApplicable
                  ? t(strings.exposure.verificationNotRequired)
                  : stage.configured
                    ? t(strings.exposure.verificationConfigured)
                  : stage.status === undefined
                    ? t(strings.exposure.verificationUnknown)
                    : stage.status === ProbeStatus.UP
                      ? t(strings.exposure.verificationPassed)
                      : t(strings.exposure.verificationNeedsAttention);
                const tone = stage.notApplicable ? "neutral" : stage.configured ? "info" : stage.status === undefined ? "neutral" : probeStatusTone(stage.status);
                return <div key={stage.label} className="rounded-control border border-app-success/20 bg-app-surface/60 p-3"><p className="text-sm font-medium">{t(stage.label)}</p><StatusBadge tone={tone}>{classifyQuery.isFetching || accessQuery.isFetching ? t(strings.exposure.verificationPending) : statusLabel}</StatusBadge>{stage.status !== undefined && stage.status !== ProbeStatus.UP ? <p className="mt-1 text-xs text-app-muted-foreground">{t(probeStatusLabel(stage.status))}</p> : null}</div>;
              })}
            </div>
            <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.exposure.verificationHint)}{classifyQuery.error ? ` ${t(strings.exposure.error)}` : ""}</p>
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
              <div><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{t(strings.exposure.reviewTitle)}</p><h2 id="exposure-review-title" className="mt-1 text-2xl font-semibold">{scenario.trim()}</h2><p className="mt-2 text-sm text-app-muted-foreground">{t(strings.exposure.reviewDescription)}</p></div>
              <button type="button" aria-label={t(strings.common.close)} onClick={() => setReviewOpen(false)} className="touch-target rounded-full text-2xl text-app-muted-foreground hover:bg-app-surface-muted">×</button>
            </div>
            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              <ReviewField label={t(strings.exposure.targetLabel)} value={selectedRoute?.scenario || scenario.trim()} />
              <ReviewField label={t(strings.exposure.hostnameLabel)} value={selectedExpiredRoute ? hostnameFor(selectedExpiredRoute.publicUrl) : t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.colPort)} value={selectedRoute ? String(selectedRoute.localPort) : t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.routeLabel)} value={selectedRoute?.healthPath || t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.colUrl)} value={selectedRoute?.publicUrl || t(strings.exposure.healthUnknown)} />
              <ReviewField label={t(strings.exposure.colTier)} value={selectedRoute ? t(strings.exposure.tier.leased) : t(strings.exposure.healthUnknown)} />
              <label className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span className="block text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.exposure.colLease)}</span><select data-testid={selectors.exposure.durationSelect} value={duration} onChange={(event) => setDuration(event.target.value)} className="mt-2 w-full bg-transparent font-semibold text-app-foreground"><option value="3600">{t(strings.exposure.durationHour)}</option><option value="86400">{t(strings.exposure.durationDay)}</option><option value="604800">{t(strings.exposure.durationWeek)}</option><option value="custom">{t(strings.exposure.durationCustom)}</option></select>{duration === "custom" ? <><span className="mt-2 block text-xs text-app-muted-foreground">{t(strings.exposure.durationCustomHelp)}</span><input data-testid={selectors.exposure.customDurationInput} type="number" min="1" max="168" step="1" value={customHours} onChange={(event) => setCustomHours(event.target.value)} placeholder={t(strings.exposure.durationCustomPlaceholder)} className="mt-2 min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 text-app-foreground" aria-label={t(strings.exposure.durationCustomHours)} /></> : null}</label>
              <ReviewField label={t(strings.exposure.projectedExpiry)} value={projectedExpiry} />
            </div>
            <div className="mt-4 rounded-control border border-app-primary/30 bg-app-primary/5 p-4 text-sm"><p className="font-semibold">{t(strings.exposure.authLabel)}</p><p className="mt-1 font-medium text-app-foreground">{!accessStatus ? t(strings.exposure.authUnavailable) : selectedAccessHost?.effectiveBypass ? t(strings.config.publicExposureOn) : t(strings.exposure.authProtected)}</p><p className="mt-2 text-app-muted-foreground">{t(strings.config.publicExposureBody)}</p><label className="mt-4 flex cursor-pointer items-start gap-3 rounded-control border border-app-primary/20 bg-app-surface/70 p-3"><input ref={reviewFocusRef} data-testid={selectors.exposure.policyAcknowledgement} type="checkbox" checked={policyAcknowledged} onChange={(event) => setPolicyAcknowledged(event.target.checked)} className="mt-0.5 h-4 w-4 accent-app-primary" /><span className="text-sm font-medium">{t(strings.exposure.policyAcknowledgement)}</span></label></div>
            <div className="mt-4 rounded-control border border-app-border bg-app-surface-muted/60 p-4"><p className="text-xs font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">{t(strings.exposure.verificationHeading)}</p><div className="mt-3 grid gap-2 sm:grid-cols-2">{[strings.exposure.verificationLocal, strings.exposure.verificationTunnel, strings.exposure.verificationPublic, strings.exposure.verificationPolicy].map((label) => <div key={label} className="flex items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface p-3 text-sm"><span>{t(label)}</span><StatusBadge tone="neutral">{t(strings.exposure.verificationPending)}</StatusBadge></div>)}</div><p className="mt-3 text-xs text-app-muted-foreground">{t(strings.exposure.verificationHint)}</p></div>
            {exposeMutation.isPending ? <div className="mt-4 rounded-control border border-app-primary/30 bg-app-primary/5 p-4" role="status" aria-live="polite"><p className="text-sm font-semibold text-app-primary">{t(strings.exposure.provisioningTitle)}</p><div className="mt-3 grid gap-2 sm:grid-cols-2"><div className="rounded-control border border-app-primary/30 bg-app-surface p-3 text-sm"><span className="block font-medium">{t(strings.exposure.provisioningStageRequest)}</span><span className="mt-1 block text-xs text-app-primary">{t(strings.exposure.provisioningInProgress)}</span></div><div className="rounded-control border border-app-border bg-app-surface p-3 text-sm"><span className="block font-medium">{t(strings.exposure.provisioningStageVerification)}</span><span className="mt-1 block text-xs text-app-muted-foreground">{t(strings.exposure.provisioningPending)}</span></div></div></div> : null}
            {exposeMutation.error ? <div data-testid={selectors.exposure.exposeError} className="mt-4 rounded-control border border-app-danger/30 bg-app-danger/5 p-4" role="alert"><p className="text-sm font-semibold text-app-danger">{t(strings.exposure.provisioningFailure)}</p><p className="mt-1 text-sm text-app-danger">{errorMessage(exposeMutation.error, t)}</p><p className="mt-2 text-xs text-app-muted-foreground">{provisioningRemediation(exposeMutation.error, t)}</p></div> : null}
            <div className="mt-6 flex flex-col-reverse justify-end gap-2 sm:flex-row"><Button variant="outline" data-testid={selectors.exposure.cancelExposeButton} onClick={() => setReviewOpen(false)}>{t(strings.common.close)}</Button><Button data-testid={selectors.exposure.confirmExposeButton} disabled={exposeMutation.isPending || !durationValid || !policyAcknowledged} onClick={() => exposeMutation.mutate({ scenario: scenario.trim(), ttlSeconds: BigInt(durationSeconds) })}>{exposeMutation.isPending ? t(strings.common.loading) : exposeMutation.error ? t(strings.exposure.provisioningRetry) : t(strings.exposure.exposeButton)}</Button></div>
          </section>
        </div>
      )}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="grid flex-1 grid-cols-2 gap-3 sm:grid-cols-[minmax(0,1fr)_repeat(2,12rem)] lg:grid-cols-[minmax(0,1fr)_repeat(4,12rem)]">
          <label className="col-span-2 flex flex-col gap-1 text-sm sm:col-span-1">
            <span className="font-medium">{t(strings.exposure.searchLabel)}</span>
            <Input
              data-testid={selectors.exposure.searchInput}
              className="min-h-11"
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
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.policyFilterLabel)}</span>
            <select data-testid={selectors.exposure.policyFilter} value={policyFilter} onChange={(e) => setPolicyFilter(e.target.value as PolicyFilter)} className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground">
              <option value="all">{t(strings.exposure.policyFilterAll)}</option>
              <option value="protected">{t(strings.exposure.policyFilterProtected)}</option>
              <option value="public">{t(strings.exposure.policyFilterPublic)}</option>
              <option value="unknown">{t(strings.exposure.policyFilterUnknown)}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t(strings.exposure.readinessFilterLabel)}</span>
            <select data-testid={selectors.exposure.readinessFilter} value={readinessFilter} onChange={(e) => setReadinessFilter(e.target.value as ReadinessFilter)} className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground">
              <option value="all">{t(strings.exposure.readinessFilterAll)}</option>
              <option value="ready">{t(strings.exposure.readinessFilterReady)}</option>
              <option value="attention">{t(strings.exposure.readinessFilterAttention)}</option>
              <option value="unknown">{t(strings.exposure.readinessFilterUnknown)}</option>
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

      {leaseNotice ? (
        <p className="rounded-control border border-app-success/30 bg-app-success/5 p-3 text-sm text-app-success" role="status">
          {leaseNotice.kind === "extend"
            ? t(strings.exposure.extendResult, { when: leaseNotice.when ?? t(strings.common.never) })
            : t(strings.exposure.revokeResult, { state: leaseNotice.state ?? "" })}
        </p>
      ) : null}

      {classifyQuery.error && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-control border border-app-warning/30 bg-app-warning/5 p-3 text-sm" role="status">
          <span className="text-app-warning">{t(strings.exposure.error)}</span>
          <Button variant="outline" onClick={() => void classifyQuery.refetch()}>{t(strings.common.refresh)}</Button>
        </div>
      )}

      {revokeLeaseId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-6 backdrop-blur-sm" role="presentation">
          <section role="alertdialog" aria-modal="true" aria-labelledby="revoke-title" className="w-full max-w-md rounded-[1.25rem] border border-app-border bg-app-surface p-6 shadow-2xl">
            <h2 id="revoke-title" className="text-xl font-semibold">{t(strings.exposure.revokeButton)}</h2>
            <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.exposure.revokeConfirmDescription)}</p>
            <div className="mt-6 flex justify-end gap-2"><Button ref={revokeFocusRef} variant="outline" onClick={() => setRevokeLeaseId(null)}>{t(strings.common.close)}</Button><Button variant="outline" data-testid={selectors.exposure.revokeConfirmButton} className="border-app-danger text-app-danger hover:bg-app-danger/10" disabled={revokeMutation.isPending} onClick={() => { revokeMutation.mutate(revokeLeaseId); setRevokeLeaseId(null); }}>{t(strings.exposure.revokeButton)}</Button></div>
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
        <div className="grid gap-3 md:hidden">
          {filteredExposures.map((exposure: Exposure) => {
            const lease = exposure.lease;
            const classification = classificationFor(exposure, classifications);
            return (
              <article key={`mobile-${exposure.scenario}`} className="rounded-panel border border-app-border bg-app-surface p-4 shadow-sm">
                <div className="flex items-start justify-between gap-3">
                  <button type="button" className="min-w-0 truncate text-left text-base font-semibold text-app-primary hover:underline" onClick={() => setSelectedExposure(exposure)}>
                    {exposure.scenario}
                  </button>
                  {classification ? <StatusBadge tone={failureClassTone(classification.classification)}>{t(failureClassLabel(classification.classification))}</StatusBadge> : <StatusBadge tone="neutral">{t(strings.exposure.healthUnknown)}</StatusBadge>}
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <StatusBadge tone={tierTone(exposure.tier)}>{t(tierLabel(exposure.tier))}</StatusBadge>
                  <span className="text-xs text-app-muted-foreground">:{exposure.localPort}</span>
                  {lease?.expiresAt ? <span className="text-xs text-app-muted-foreground">{formatDate(timestampDate(lease.expiresAt), { dateStyle: "medium", timeStyle: "short" })}</span> : null}
                </div>
                <a href={exposure.publicUrl} target="_blank" rel="noreferrer" className="mt-3 block truncate text-sm text-app-primary hover:underline">{exposure.publicUrl}</a>
                {lease ? <div className="mt-3 flex flex-wrap gap-2"><Button variant="outline" size="sm" disabled={extendMutation.isPending} onClick={() => extendMutation.mutate(lease.id)}>{t(strings.exposure.extendButton)}</Button><Button variant="outline" size="sm" disabled={revokeMutation.isPending} onClick={() => setRevokeLeaseId(lease.id)}>{t(strings.exposure.revokeButton)}</Button></div> : null}
              </article>
            );
          })}
        </div>
        <RouteInventoryTable
          rows={inventoryRows}
          t={t}
          onSelect={setSelectedExposure}
          onExtend={(leaseId) => extendMutation.mutate(leaseId)}
          onRevoke={setRevokeLeaseId}
          extendPending={extendMutation.isPending}
          revokePending={revokeMutation.isPending}
        />
      </QueryState>

      {selectedExposure && (
        <div className="fixed inset-0 z-50 flex items-stretch justify-end bg-slate-950/60 backdrop-blur-sm" role="presentation">
          <aside data-testid={selectors.exposure.detailDialog} role="dialog" aria-modal="true" aria-labelledby="exposure-detail-title" className="h-full w-full max-w-xl overflow-y-auto border-l border-app-border bg-app-surface p-6 shadow-2xl sm:p-8">
            <div className="flex items-start justify-between gap-4"><div><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{t(strings.exposure.heading)}</p><h2 id="exposure-detail-title" className="mt-1 text-2xl font-semibold">{selectedExposure.scenario}</h2><p className="mt-1 text-sm text-app-muted-foreground">{selectedExposure.publicUrl}</p></div><button ref={detailFocusRef} type="button" data-testid={selectors.exposure.detailCloseButton} aria-label={t(strings.common.close)} onClick={() => setSelectedExposure(null)} className="touch-target rounded-full text-2xl text-app-muted-foreground">×</button></div>
            <div className="mt-6 grid gap-3 sm:grid-cols-2"><ReviewField label={t(strings.exposure.colPort)} value={String(selectedExposure.localPort)} /><ReviewField label={t(strings.exposure.hostnameLabel)} value={hostnameFor(selectedExposure.publicUrl)} /><ReviewField label={t(strings.exposure.colTier)} value={t(tierLabel(selectedExposure.tier))} /><ReviewField label={t(strings.exposure.routeLabel)} value={selectedExposureRoute?.healthPath ?? t(strings.exposure.healthUnknown)} /><ReviewField label={t(strings.exposure.colHealth)} value={selectedExposureClassification ? t(failureClassLabel(selectedExposureClassification.classification)) : t(strings.exposure.healthUnknown)} /><ReviewField label={t(strings.exposure.colLease)} value={selectedExposure.lease?.expiresAt ? formatDate(timestampDate(selectedExposure.lease.expiresAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.exposure.leaseNone)} /><ReviewField label={t(strings.exposure.authLabel)} value={selectedExposureAccessHost ? (selectedExposureAccessHost.effectiveBypass ? t(strings.config.publicExposureOn) : t(strings.exposure.authProtected)) : t(strings.exposure.authUnavailable)} /></div>
            <div className="mt-5 flex flex-wrap gap-2"><Button variant="outline" data-testid={selectors.exposure.detailCopyButton} onClick={() => void copySelectedUrl()}>{copied ? t(strings.common.copied) : t(strings.common.copyUrl)}</Button><Button data-testid={selectors.exposure.detailProbeButton} disabled={detailProbesQuery.isFetching || runDetailProbesMutation.isPending} onClick={() => runDetailProbesMutation.mutate()}>{runDetailProbesMutation.isPending ? t(strings.common.loading) : t(strings.metrics.runProbesButton)}</Button>{selectedExposureLease ? <><Button variant="outline" data-testid={selectors.exposure.detailExtendButton} disabled={extendMutation.isPending} onClick={() => extendMutation.mutate(selectedExposureLease.id)}>{t(strings.exposure.extendButton)}</Button><Button variant="outline" data-testid={selectors.exposure.detailRevokeButton} disabled={revokeMutation.isPending} onClick={() => setRevokeLeaseId(selectedExposureLease.id)}>{t(strings.exposure.revokeButton)}</Button></> : null}<a href={selectedExposure.publicUrl} target="_blank" rel="noreferrer" className="inline-flex h-11 items-center rounded-control border border-app-border px-5 text-sm font-medium text-app-foreground hover:bg-app-surface-muted">{t(strings.common.openUrl)}</a></div>
            {(actionError || leaseNotice || copyError) ? <div className="mt-4" role={actionError || copyError ? "alert" : "status"}>{copyError ? <p className="text-sm text-app-danger">{t(strings.exposure.copyError)}</p> : actionError ? <p className="text-sm text-app-danger">{t(strings.exposure.actionError)}</p> : <p className="text-sm text-app-success">{leaseNotice?.kind === "extend" ? t(strings.exposure.extendResult, { when: leaseNotice.when ?? t(strings.common.never) }) : t(strings.exposure.revokeResult, { state: leaseNotice?.state ?? "" })}</p>}</div> : null}
            <section className="mt-8"><h3 className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.metrics.probesHeading)}</h3><div className="mt-3"><QueryState isLoading={detailProbesQuery.isLoading} error={detailProbesQuery.error} onRetry={() => void detailProbesQuery.refetch()} isEmpty={detailProbesQuery.data?.results.length === 0} emptyLabel={t(strings.metrics.probesEmpty)}>{detailProbesQuery.data?.results.length ? <ul className="flex flex-col gap-2">{detailProbesQuery.data.results.map((probe) => <li key={probe.id} className="flex items-center justify-between rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span>{t(probeKindLabel(probe.kind))}</span><StatusBadge tone={probeStatusTone(probe.status)}>{t(probeStatusLabel(probe.status))}</StatusBadge></li>)}</ul> : null}</QueryState></div></section>
            <div className="mt-8 rounded-control border border-app-primary/30 bg-app-primary/5 p-4 text-sm"><p className="font-semibold">{t(strings.config.publicExposureHeading)}</p><p className="mt-1 text-app-muted-foreground">{t(strings.config.publicExposureBody)}</p></div>
          </aside>
        </div>
      )}
      {selectedExpiredLease && (
        <div className="fixed inset-0 z-50 flex items-stretch justify-end bg-slate-950/60 backdrop-blur-sm" role="presentation">
          <aside data-testid={selectors.exposure.detailDialog} role="dialog" aria-modal="true" aria-labelledby="expired-exposure-detail-title" className="h-full w-full max-w-xl overflow-y-auto border-l border-app-border bg-app-surface p-6 shadow-2xl sm:p-8">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-warning">{t(strings.exposure.expiredHeading)}</p>
                <h2 id="expired-exposure-detail-title" className="mt-1 text-2xl font-semibold">{selectedExpiredLease.scenario}</h2>
                <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.exposure.expiredDescription)}</p>
              </div>
              <button ref={detailFocusRef} type="button" data-testid={selectors.exposure.detailCloseButton} aria-label={t(strings.common.close)} onClick={() => setSelectedExpiredLease(null)} className="touch-target rounded-full text-2xl text-app-muted-foreground">×</button>
            </div>
            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              <ReviewField label={t(strings.exposure.colScenario)} value={selectedExpiredLease.scenario} />
              <ReviewField label={t(strings.exposure.colTier)} value={t(strings.exposure.tier.leased)} />
              <ReviewField label={t(strings.exposure.colLease)} value={selectedExpiredLease.expiresAt ? formatDate(timestampDate(selectedExpiredLease.expiresAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.exposure.leaseNone)} />
              <ReviewField label={t(strings.exposure.hostnameLabel)} value={selectedExpiredRoute ? hostnameFor(selectedExpiredRoute.publicUrl) : t(strings.exposure.healthUnknown)} />
            </div>
            <div className="mt-5 rounded-control border border-app-warning/30 bg-app-warning/5 p-4 text-sm">
              <p className="font-semibold text-app-warning">{t(strings.exposure.expiredHeading)}</p>
              <p className="mt-1 text-app-muted-foreground">{t(strings.exposure.expiredDescription)}</p>
            </div>
            <div className="mt-6 flex flex-wrap justify-end gap-2">
              <Button variant="outline" onClick={() => setSelectedExpiredLease(null)}>{t(strings.common.close)}</Button>
              <Button data-testid={selectors.exposure.reExposeButton} onClick={() => { setScenario(selectedExpiredLease.scenario); setPolicyAcknowledged(false); setSelectedExpiredLease(null); setReviewOpen(true); }}>{t(strings.exposure.reExposeButton)}</Button>
            </div>
          </aside>
        </div>
      )}
    </section>
  );
}

function ReviewField({ label, value }: { label: string; value: string }) {
  return <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"><span className="block text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{label}</span><span className="mt-1 block font-semibold text-app-foreground">{value}</span></div>;
}
