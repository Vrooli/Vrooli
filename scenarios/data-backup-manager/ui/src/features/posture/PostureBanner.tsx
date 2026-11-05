import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, ShieldAlert } from "lucide-react";

import { fetchHealth } from "../../api/health";
import { useTargetStatus } from "../../hooks/useTargetStatus";
import { queryKeys } from "../../hooks/keys";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { tsToDate } from "../../lib/proto";
import { isOlderThan } from "../../lib/format";
import { STALE_VERIFY_MS } from "../../lib/status";

/**
 * The Overview's headline band. Composes liveness (the REST /health readiness)
 * with backup posture derived from the owner-scoped status rollup: how many
 * targets are overdue/failed, and — the metric this product is built around —
 * how many are backed up but not (recently) verified. Calm when all-clear;
 * amber, never alarmist red, when attention is needed.
 *
 * Overdue is read from the server's `overdue` field (computed against
 * DBM_OVERDUE_AFTER) so the banner, `runs status`, and /health share one rule
 * rather than each re-deriving a threshold client-side.
 */
export function PostureBanner() {
  const { t } = useTranslation();
  const health = useQuery({ queryKey: queryKeys.health, queryFn: fetchHealth });
  const status = useTargetStatus();

  const statuses = status.data ?? [];
  const now = new Date();

  const overdue = statuses.filter((s) => s.overdue).length;

  const unverified = statuses.filter((s) =>
    isOlderThan(tsToDate(s.lastVerifiedAt), STALE_VERIFY_MS, now),
  ).length;

  const attention = overdue + unverified;
  const allClear = statuses.length > 0 && attention === 0;
  const Icon = allClear ? CheckCircle2 : ShieldAlert;
  const tone = allClear ? "text-app-success" : "text-app-warning";

  const loading = status.isLoading;

  return (
    <section
      data-testid={selectors.overview.posture}
      aria-label={t(strings.posture.title)}
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="flex items-center gap-3">
        <Icon aria-hidden="true" className={`h-6 w-6 shrink-0 ${tone}`} />
        <div className="flex flex-col">
          <p data-testid={selectors.overview.postureHeadline} className="text-sm font-semibold text-app-foreground">
            {loading
              ? t(strings.posture.loading)
              : allClear
                ? t(strings.posture.allClear)
                : t(strings.posture.attention, { count: attention })}
          </p>
          {!loading && attention > 0 && (
            <p className="flex flex-wrap gap-x-3 text-xs text-app-muted-foreground">
              <span>{t(strings.posture.overdue, { count: overdue })}</span>
              <span>{t(strings.posture.unverified, { count: unverified })}</span>
            </p>
          )}
        </div>
      </div>

      <dl className="flex items-center gap-4 text-xs">
        <div className="flex flex-col">
          <dt className="text-app-muted-foreground">{t(strings.posture.readinessLabel)}</dt>
          <dd className="font-medium text-app-foreground">
            {health.isError ? t(strings.posture.error) : (health.data?.readiness ?? "—")}
          </dd>
        </div>
        <div className="flex flex-col">
          <dt className="text-app-muted-foreground">{t(strings.posture.serviceLabel)}</dt>
          <dd className="font-medium text-app-foreground">{health.data?.service ?? "—"}</dd>
        </div>
      </dl>
    </section>
  );
}
