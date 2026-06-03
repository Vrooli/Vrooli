import { Archive } from "lucide-react";

import { StatusChip } from "../../components/ui/status-chip";
import { VerifiedChip } from "../../components/ui/verified-chip";
import { AsyncSection } from "../../components/AsyncSection";
import { EmptyState } from "../../components/EmptyState";
import { useCoverage, type CoverageRow } from "../../hooks/useCoverage";
import { RunStatus } from "../../api/runs";
import { runStatusMeta, sourceKindSlug } from "../../lib/status";
import { RUN_STATUS_STRINGS, SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { formatAge } from "../../lib/format";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Owner-grouped coverage grid — the heart of the Overview. Each registered
 * target shows when it was last backed up and, co-equally, when it was last
 * *verified*. The verified chip is what an operator scans for: a green
 * "Verified" vs. an amber "Unverified" makes the backed-up / proven-restorable
 * distinction impossible to miss.
 */
function groupByOwner(rows: CoverageRow[], fallback: string): Map<string, CoverageRow[]> {
  const groups = new Map<string, CoverageRow[]>();
  for (const row of rows) {
    const owner = row.target.owner || fallback;
    const list = groups.get(owner) ?? [];
    list.push(row);
    groups.set(owner, list);
  }
  return groups;
}

export function CoverageGrid() {
  const { t } = useTranslation();
  const { rows, isLoading, isError, refetch } = useCoverage();
  const never = t(strings.common.never);
  const groups = groupByOwner(rows, t(strings.overview.ownerFallback));

  return (
    <section data-testid={selectors.overview.coverage} className="flex flex-col gap-3">
      <h2 className="text-sm font-semibold text-app-foreground">{t(strings.overview.coverageHeading)}</h2>
      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={rows.length === 0}
        onRetry={refetch}
        emptyState={
          <EmptyState
            icon={Archive}
            title={t(strings.overview.coverageEmpty)}
            data-testid={selectors.overview.coverageEmpty}
          />
        }
      >
        <div className="flex flex-col gap-4">
          {[...groups.entries()].map(([owner, ownerRows]) => (
            <div key={owner} className="flex flex-col gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{owner}</p>
              <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {ownerRows.map((row) => {
                  const runMeta = runStatusMeta(row.lastRunStatus);
                  return (
                    <li
                      key={row.target.id}
                      data-testid={selectors.overview.coverageRow({ targetId: row.target.id })}
                      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <p className="truncate text-sm font-medium text-app-foreground">{row.target.name}</p>
                        <div className="flex shrink-0 items-center gap-1">
                          {row.overdue && (
                            <span className="rounded-full bg-app-warning/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-app-warning">
                              {t(strings.overview.overdueChip)}
                            </span>
                          )}
                          <VerifiedChip lastVerifiedAt={row.lastVerifiedAt} />
                        </div>
                      </div>
                      <p className="truncate text-xs text-app-muted-foreground">
                        {t(SOURCE_KIND_STRINGS[sourceKindSlug(row.target.sourceKind)])}
                      </p>
                      <div className="flex items-center justify-between gap-2 text-xs text-app-muted-foreground">
                        <span>
                          {t(strings.overview.lastBackupLabel)} {formatAge(row.lastSuccessAt, never)}
                        </span>
                        {row.lastRunStatus !== RunStatus.UNSPECIFIED && (
                          <StatusChip tone={runMeta.tone} labelKey={RUN_STATUS_STRINGS[runMeta.slug]} />
                        )}
                      </div>
                      {row.nextScheduledAt && (
                        <p className="text-xs text-app-muted-foreground">
                          {t(strings.overview.nextBackupLabel)} {formatAge(row.nextScheduledAt, never)}
                        </p>
                      )}
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </div>
      </AsyncSection>
    </section>
  );
}
