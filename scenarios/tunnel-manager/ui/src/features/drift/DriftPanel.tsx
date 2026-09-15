import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { Button } from "../../components/ui/button";
import { IngressDriftTable } from "./IngressDriftTable";
import { QueryState } from "../../components/ui/QueryState";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { configClient } from "../../api/config";

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
  // A remote-capability failure leaves this page usable: the operator can
  // still understand the limitation and follow the setup/remediation action.
  // Model that as partial rather than making the required page surface fatal.
  const experienceState = driftQuery.isLoading ? "loading" : driftQuery.error ? "partial" : entries.length === 0 ? "empty" : "ready";

  return (
    <section data-testid={selectors.drift.panel} data-experience-surface="drift-results" data-experience-state={experienceState} className="flex flex-col gap-6">
      <div className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.drift.heading)}
            </p>
            <p data-testid={selectors.drift.summary} className="text-sm text-app-muted-foreground">
              {driftQuery.error
                ? t(strings.drift.summaryUnavailable)
                : t(strings.drift.summary, {
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
        errorLabel={t(strings.overview.driftUnavailable)}
        emptyLabel={t(strings.drift.empty)}
        onRetry={() => void driftQuery.refetch()}
        errorAction={
          <Link
            to="/settings"
            className="inline-flex min-h-11 items-center rounded-control border border-app-primary/40 px-4 text-sm font-medium text-app-primary transition-colors hover:bg-app-primary/10"
          >
            {t(strings.overview.configure)}
          </Link>
        }
      >
        <IngressDriftTable
          entries={entries}
          actionPending={actionPending}
          onAdopt={(hostname) => adoptMutation.mutate(hostname)}
          onIgnore={(hostname) => ignoreMutation.mutate(hostname)}
          onPrune={setPruneTarget}
        />
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
