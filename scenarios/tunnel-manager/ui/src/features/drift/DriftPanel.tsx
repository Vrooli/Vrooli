import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IngressEntry } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { configClient } from "../../api/config";
import {
  ownershipStateLabel,
  ownershipStateTone,
  ingressSourceLabel,
} from "./labels";

const DRIFT_QUERY_KEY = ["drift"] as const;

/**
 * DriftPanel renders the live ingress drift report: every hostname classified
 * against the manifest and ownership ledger, with per-row Adopt / Ignore /
 * Prune actions. Adopt and Ignore are additive ledger writes; Prune is
 * destructive and gated behind a confirm dialog. All mutations invalidate the
 * drift query so the table reflects the reconciled state immediately.
 */
export function DriftPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [pruneTarget, setPruneTarget] = useState<string | null>(null);

  const driftQuery = useQuery({ queryKey: DRIFT_QUERY_KEY, queryFn: () => configClient.getDrift({}) });
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: DRIFT_QUERY_KEY });

  const adoptMutation = useMutation({
    mutationFn: (hostname: string) => configClient.adoptIngress({ hostname }),
    onSuccess: invalidate,
  });
  const ignoreMutation = useMutation({
    mutationFn: (hostname: string) => configClient.ignoreIngress({ hostname }),
    onSuccess: invalidate,
  });
  const pruneMutation = useMutation({
    mutationFn: (hostname: string) => configClient.pruneIngress({ hostname }),
    onSuccess: () => {
      setPruneTarget(null);
      invalidate();
    },
  });

  const actionError = adoptMutation.error ?? ignoreMutation.error ?? pruneMutation.error;
  const actionPending = adoptMutation.isPending || ignoreMutation.isPending || pruneMutation.isPending;
  const entries = driftQuery.data?.entries ?? [];
  const counts = driftQuery.data?.counts;
  const externalTracked = (counts?.externalOk ?? 0) + (counts?.ignored ?? 0);

  return (
    <section data-testid={selectors.drift.panel} className="flex flex-col gap-6">
      <div className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.drift.heading)}
            </p>
            <p data-testid={selectors.drift.summary} className="text-sm text-app-muted-foreground">
              {t(strings.drift.summary, {
                total: entries.length,
                managed: counts?.managed ?? 0,
                external: externalTracked,
                drift: counts?.unmanaged ?? 0,
              })}
            </p>
          </div>
          <Button
            variant="outline"
            data-testid={selectors.drift.refreshButton}
            onClick={() => void driftQuery.refetch()}
          >
            {t(strings.common.refresh)}
          </Button>
        </div>
        <p data-testid={selectors.drift.modeHint} className="text-sm text-app-muted-foreground">
          {t(strings.drift.modeHint)}
        </p>
      </div>

      {actionError && (
        <p data-testid={selectors.drift.actionError} role="alert" className="text-sm text-app-danger">
          {t(strings.drift.actionError)}
        </p>
      )}

      <QueryState
        isLoading={driftQuery.isLoading}
        error={driftQuery.error}
        isEmpty={entries.length === 0}
        loadingLabel={t(strings.drift.loading)}
        errorLabel={t(strings.drift.error)}
        emptyLabel={t(strings.drift.empty)}
      >
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.drift.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.drift.colHostname)}</th>
                <th className="px-3 py-2">{t(strings.drift.colTarget)}</th>
                <th className="px-3 py-2">{t(strings.drift.colState)}</th>
                <th className="px-3 py-2">{t(strings.drift.colSource)}</th>
                <th className="px-3 py-2">{t(strings.drift.colActions)}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry: IngressEntry) => (
                <tr
                  key={entry.hostname}
                  data-testid={selectors.drift.row}
                  className="border-b border-app-border last:border-0"
                >
                  <td data-testid={selectors.drift.hostname} className="px-3 py-2 font-medium">
                    {entry.hostname}
                  </td>
                  <td data-testid={selectors.drift.target} className="px-3 py-2 text-app-muted-foreground">
                    {entry.serviceTarget || "—"}
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge tone={ownershipStateTone(entry.state)} data-testid={selectors.drift.stateBadge}>
                      {t(ownershipStateLabel(entry.state))}
                    </StatusBadge>
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge tone="neutral" data-testid={selectors.drift.sourceBadge}>
                      {t(ingressSourceLabel(entry.source))}
                    </StatusBadge>
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        data-testid={selectors.drift.adoptButton({ hostname: entry.hostname })}
                        disabled={actionPending}
                        onClick={() => adoptMutation.mutate(entry.hostname)}
                      >
                        {t(strings.drift.adoptButton)}
                      </Button>
                      <Button
                        variant="outline"
                        data-testid={selectors.drift.ignoreButton({ hostname: entry.hostname })}
                        disabled={actionPending}
                        onClick={() => ignoreMutation.mutate(entry.hostname)}
                      >
                        {t(strings.drift.ignoreButton)}
                      </Button>
                      <Button
                        variant="outline"
                        data-testid={selectors.drift.pruneButton({ hostname: entry.hostname })}
                        disabled={actionPending}
                        onClick={() => setPruneTarget(entry.hostname)}
                      >
                        {t(strings.drift.pruneButton)}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </QueryState>

      {pruneTarget && (
        <div
          data-testid={selectors.drift.confirmDialog}
          role="alertdialog"
          aria-modal="true"
          aria-label={t(strings.drift.pruneConfirmTitle)}
          className="flex flex-col gap-3 rounded-panel border border-app-warning/40 bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.drift.pruneConfirmTitle)}</h3>
          <p className="text-sm text-app-muted-foreground">
            {t(strings.drift.pruneConfirmBody, { hostname: pruneTarget })}
          </p>
          <div className="flex gap-2">
            <Button
              data-testid={selectors.drift.confirmButton}
              disabled={pruneMutation.isPending}
              onClick={() => pruneMutation.mutate(pruneTarget)}
            >
              {t(strings.drift.pruneConfirm)}
            </Button>
            <Button
              variant="outline"
              data-testid={selectors.drift.cancelButton}
              onClick={() => setPruneTarget(null)}
            >
              {t(strings.drift.pruneCancel)}
            </Button>
          </div>
        </div>
      )}
    </section>
  );
}
