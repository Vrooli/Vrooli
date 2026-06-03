import { useState } from "react";
import { Archive, Plus, Trash2 } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { AsyncSection } from "../components/AsyncSection";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { VerifiedChip } from "../components/ui/verified-chip";
import { RegisterTargetDialog } from "../features/targets/RegisterTargetDialog";
import { TargetInspector } from "../features/targets/TargetInspector";
import { CoverageBanner } from "../features/backup-coverage/CoverageBanner";
import { useCoverage, type CoverageRow } from "../hooks/useCoverage";
import { useDeregisterTarget } from "../hooks/useTargets";
import { RunStatus } from "../api/runs";
import { runStatusMeta, sourceKindSlug } from "../lib/status";
import { RUN_STATUS_STRINGS, SOURCE_KIND_STRINGS } from "../consts/statusStrings";
import { formatAge } from "../lib/format";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

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

/**
 * Targets surface — full co-equal CRUD over the backup catalog. Rows are
 * grouped by owner; each shows the source kind, the last backup age, the last
 * run status, and the verified chip (the proven-restorable signal). Register
 * and deregister are first-class actions; clicking a row opens the inspector.
 */
export function TargetsPage() {
  const { t } = useTranslation();
  const { rows, isLoading, isError, refetch } = useCoverage();
  const deregister = useDeregisterTarget();

  const [registerOpen, setRegisterOpen] = useState(false);
  const [inspecting, setInspecting] = useState<CoverageRow | null>(null);
  const [deregistering, setDeregistering] = useState<CoverageRow | null>(null);

  const never = t(strings.common.never);
  const groups = groupByOwner(rows, t(strings.overview.ownerFallback));

  const confirmDeregister = () => {
    if (!deregistering) return;
    deregister.mutate(
      { owner: deregistering.target.owner, name: deregistering.target.name },
      { onSuccess: () => setDeregistering(null) },
    );
  };

  return (
    <section
      data-testid={selectors.pages.targets}
      aria-labelledby="targets-heading"
      className="flex flex-col gap-6"
    >
      <div id="targets-heading">
        <PageHeader
          title={t(strings.layout.nav.targets)}
          subtitle={t(strings.targets.subtitle)}
          actions={
            <Button
              size="sm"
              data-testid={selectors.targets.registerButton}
              onClick={() => setRegisterOpen(true)}
            >
              <Plus aria-hidden="true" className="me-1.5 h-4 w-4" />
              {t(strings.targets.register)}
            </Button>
          }
        />
      </div>

      <CoverageBanner detailed />

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={rows.length === 0}
        onRetry={refetch}
        emptyState={
          <EmptyState
            icon={Archive}
            title={t(strings.targets.empty)}
            description={t(strings.targets.emptyHint)}
            action={
              <Button size="sm" onClick={() => setRegisterOpen(true)}>
                {t(strings.targets.register)}
              </Button>
            }
          />
        }
      >
        <div data-testid={selectors.targets.table} className="flex flex-col gap-5">
          {[...groups.entries()].map(([owner, ownerRows]) => (
            <div key={owner} className="flex flex-col gap-1.5">
              <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{owner}</p>
              <ul className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
                {ownerRows.map((row) => {
                  const runMeta = runStatusMeta(row.lastRunStatus);
                  return (
                    <li
                      key={row.target.id}
                      data-testid={selectors.targets.row({ id: row.target.id })}
                      className="flex flex-wrap items-center gap-x-3 gap-y-2 p-3"
                    >
                      <button
                        type="button"
                        onClick={() => setInspecting(row)}
                        className="min-w-0 flex-1 text-start"
                      >
                        <span className="block truncate text-sm font-medium text-app-foreground">
                          {row.target.name}
                        </span>
                        <span className="flex min-w-0 gap-2 text-xs text-app-muted-foreground">
                          <span className="shrink-0">
                            {t(SOURCE_KIND_STRINGS[sourceKindSlug(row.target.sourceKind)])}
                          </span>
                          <span className="truncate font-mono">{row.target.locator}</span>
                        </span>
                      </button>
                      <span className="text-xs text-app-muted-foreground">
                        {formatAge(row.lastSuccessAt, never)}
                      </span>
                      {row.lastRunStatus !== RunStatus.UNSPECIFIED && (
                        <StatusChip tone={runMeta.tone} labelKey={RUN_STATUS_STRINGS[runMeta.slug]} />
                      )}
                      <VerifiedChip lastVerifiedAt={row.lastVerifiedAt} />
                      <Button
                        variant="outline"
                        size="sm"
                        aria-label={t(strings.targets.deregister)}
                        data-testid={selectors.targets.deregisterButton}
                        onClick={() => setDeregistering(row)}
                      >
                        <Trash2 aria-hidden="true" className="h-4 w-4" />
                      </Button>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </div>
      </AsyncSection>

      <RegisterTargetDialog open={registerOpen} onClose={() => setRegisterOpen(false)} />
      <TargetInspector row={inspecting} onClose={() => setInspecting(null)} />
      <ConfirmDialog
        open={deregistering !== null}
        onClose={() => setDeregistering(null)}
        onConfirm={confirmDeregister}
        title={t(strings.targets.deregisterTitle)}
        body={t(strings.targets.deregisterBody, { name: deregistering?.target.name ?? "" })}
        confirmLabel={t(strings.targets.deregister)}
        danger
        busy={deregister.isPending}
        confirmTestId={selectors.targets.deregisterConfirm}
      />
    </section>
  );
}
