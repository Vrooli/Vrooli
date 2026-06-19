import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Exposure } from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";

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

const EXPOSURES_QUERY_KEY = ["exposures"] as const;

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

  const exposuresQuery = useQuery({
    queryKey: EXPOSURES_QUERY_KEY,
    queryFn: () => exposureClient.listExposures({}),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: EXPOSURES_QUERY_KEY });

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

  const revokeMutation = useMutation({
    mutationFn: (leaseId: string) => exposureClient.revokeLease({ leaseId }),
    onSuccess: () => void invalidate(),
  });

  const actionError = extendMutation.error ?? revokeMutation.error;
  const exposures = exposuresQuery.data?.exposures ?? [];

  const handleExpose = (e: React.FormEvent) => {
    e.preventDefault();
    const name = scenario.trim();
    if (name) exposeMutation.mutate(name);
  };

  return (
    <section data-testid={selectors.exposure.panel} className="flex flex-col gap-6">
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

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t(strings.exposure.heading)}</h2>
        <Button
          variant="outline"
          data-testid={selectors.exposure.refreshButton}
          onClick={() => void exposuresQuery.refetch()}
        >
          {t(strings.common.refresh)}
        </Button>
      </div>

      {actionError && (
        <p data-testid={selectors.exposure.actionError} role="alert" className="text-sm text-app-danger">
          {t(strings.exposure.actionError)}
        </p>
      )}

      <QueryState
        isLoading={exposuresQuery.isLoading}
        error={exposuresQuery.error}
        isEmpty={exposures.length === 0}
        loadingLabel={t(strings.exposure.loading)}
        errorLabel={t(strings.exposure.error)}
        emptyLabel={t(strings.exposure.empty)}
      >
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.exposure.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.exposure.colScenario)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colTier)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colUrl)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colPort)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colLease)}</th>
                <th className="px-3 py-2">{t(strings.exposure.colActions)}</th>
              </tr>
            </thead>
            <tbody>
              {exposures.map((exposure: Exposure) => {
                const lease = exposure.lease;
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
