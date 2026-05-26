import { Dialog } from "../../components/ui/dialog";
import { StatusChip } from "../../components/ui/status-chip";
import { VerifiedChip } from "../../components/ui/verified-chip";
import { AsyncSection } from "../../components/AsyncSection";
import { EmptyState } from "../../components/EmptyState";
import { useRestores } from "../../hooks/useRestores";
import type { CoverageRow } from "../../hooks/useCoverage";
import { restoreStatusMeta, sourceKindSlug } from "../../lib/status";
import { RESTORE_STATUS_STRINGS, SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { formatAge } from "../../lib/format";
import { tsToDate } from "../../lib/proto";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Read-only target inspector: the registration details plus the target's
 * restore/verify history, so an operator can see at a glance not just that a
 * target is backed up but whether — and when — it was last proven restorable.
 */
export function TargetInspector({ row, onClose }: { row: CoverageRow | null; onClose: () => void }) {
  const { t } = useTranslation();
  const targetId = row?.target.id ?? "";
  const { data, isLoading, isError, refetch } = useRestores(targetId);
  const restores = data ?? [];
  const never = t(strings.common.never);

  return (
    <Dialog
      open={row !== null}
      onClose={onClose}
      title={t(strings.targets.detailsTitle)}
      data-testid={selectors.targets.inspector}
    >
      {row && (
        <div className="flex flex-col gap-4">
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-app-muted-foreground">{t(strings.targets.owner)}</dt>
            <dd className="text-app-foreground">{row.target.owner}</dd>
            <dt className="text-app-muted-foreground">{t(strings.targets.name)}</dt>
            <dd className="text-app-foreground">{row.target.name}</dd>
            <dt className="text-app-muted-foreground">{t(strings.targets.kind)}</dt>
            <dd className="text-app-foreground">
              {t(SOURCE_KIND_STRINGS[sourceKindSlug(row.target.sourceKind)])}
            </dd>
            <dt className="text-app-muted-foreground">{t(strings.targets.locator)}</dt>
            <dd className="truncate font-mono text-xs text-app-foreground">{row.target.locator}</dd>
            <dt className="text-app-muted-foreground">{t(strings.overview.lastBackupLabel)}</dt>
            <dd className="text-app-foreground">{formatAge(row.lastSuccessAt, never)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.overview.verifiedLabel)}</dt>
            <dd>
              <VerifiedChip lastVerifiedAt={row.lastVerifiedAt} />
            </dd>
          </dl>

          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold text-app-foreground">{t(strings.targets.restoreHistory)}</h3>
            <AsyncSection
              isLoading={isLoading}
              isError={isError}
              isEmpty={restores.length === 0}
              onRetry={() => void refetch()}
              skeletonRows={2}
              emptyState={<EmptyState title={t(strings.targets.noRestores)} />}
            >
              <ul className="flex flex-col divide-y divide-app-border">
                {restores.map((r) => {
                  const meta = restoreStatusMeta(r.status);
                  return (
                    <li key={r.id} className="flex items-center justify-between gap-2 py-2 text-sm">
                      <StatusChip tone={meta.tone} labelKey={RESTORE_STATUS_STRINGS[meta.slug]} />
                      <span className="text-xs text-app-muted-foreground">
                        {formatAge(tsToDate(r.requestedAt), never)}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </AsyncSection>
          </div>
        </div>
      )}
    </Dialog>
  );
}
