import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Exposure } from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";
import type { RouteClassification } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";
import { FailureClass } from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

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
import { failureClassLabel, failureClassTone } from "../metrics/labels";

const EXPOSURES_QUERY_KEY = ["exposures"] as const;
const CLASSIFY_QUERY_KEY = ["exposure-classify"] as const;

type TierFilter = "all" | "core" | "leased";

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
  const [search, setSearch] = useState("");
  const [tierFilter, setTierFilter] = useState<TierFilter>("all");
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

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: EXPOSURES_QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: CLASSIFY_QUERY_KEY });
  };

  const exposeMutation = useMutation({
    mutationFn: (name: string) => exposureClient.expose({ scenario: name }),
    onSuccess: () => {
      setScenario("");
      void invalidate();
    },
  });

  const extendMutation = useMutation({
    mutationFn: (leaseId: string) => exposureClient.extendLease({ leaseId }),
    onSuccess: () => void invalidate(),
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
    onSuccess: () => void invalidate(),
  });

  const actionError = extendMutation.error ?? revokeMutation.error ?? reconcileMutation.error;
  const exposures = exposuresQuery.data?.exposures ?? [];
  const classifications = classifyQuery.data?.classifications ?? [];
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
    if (!normalizedSearch) return true;
    return [exposure.scenario, exposure.subdomain, exposure.publicUrl]
      .some((value) => value.toLowerCase().includes(normalizedSearch));
  });

  const handleExpose = (e: React.FormEvent) => {
    e.preventDefault();
    const name = scenario.trim();
    if (name) exposeMutation.mutate(name);
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
          />
        </label>
        <Button
          type="submit"
          data-testid={selectors.exposure.exposeButton}
          disabled={exposeMutation.isPending || scenario.trim() === ""}
        >
          {t(strings.exposure.exposeButton)}
        </Button>
      </form>
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

      <QueryState
        isLoading={exposuresQuery.isLoading}
        error={exposuresQuery.error}
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
                    <td className="px-3 py-2 font-medium">{exposure.scenario}</td>
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
                            onClick={() => revokeMutation.mutate(lease.id)}
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
    </section>
  );
}
