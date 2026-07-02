import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { AlertTriangle, ChevronRight } from "lucide-react";

import { ScenarioPicker } from "../../components/ScenarioPicker";
import { StatusChip } from "../../components/StatusChip";
import { EvidenceSummary } from "./EvidenceSummary";
import { RequirementDrawer } from "./RequirementDrawer";
import { groupMatrixRows, countUnproven, UNLINKED_GROUP_ID } from "./matrixModel";
import { useMatrix } from "./useMatrix";
import { useRecentScenarios } from "../../hooks/useRecentScenarios";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";

const statusTone = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized === "complete" || normalized === "implemented") return "success" as const;
  if (normalized === "failing") return "danger" as const;
  return "neutral" as const;
};

/**
 * Traceability matrix — the flagship surface. Operational targets grouped with
 * their requirements, each requirement showing status, criticality, and live
 * evidence, with unproven claims emphasized and a drill-in drawer for detail
 * and manual attestation.
 */
export function MatrixPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [scenario, setScenario] = useState(() => searchParams.get("scenario") ?? "");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { recents, remember, clear } = useRecentScenarios();

  const query = useMatrix(scenario);
  const response = query.data;

  const groups = useMemo(
    () => (response ? groupMatrixRows(response.matrix) : []),
    [response],
  );
  const unproven = useMemo(
    () => (response ? countUnproven(response.matrix) : 0),
    [response],
  );

  const selectedRow = useMemo(() => {
    if (!selectedId || !response) return null;
    return response.matrix.find((row) => row.requirementId === selectedId) ?? null;
  }, [selectedId, response]);

  const choose = (slug: string) => {
    setScenario(slug);
    setSelectedId(null);
    remember(slug);
    // Keep the URL shareable / deep-linkable (fleet rows navigate here).
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("scenario", slug);
      return next;
    });
  };

  const registry = response?.registry;

  return (
    <section
      data-testid={selectors.pages.matrix}
      aria-labelledby="matrix-heading"
      className="flex min-h-0 flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="matrix-heading" className="text-2xl font-semibold text-app-foreground">
          {t(strings.matrix.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.matrix.description)}</p>
      </header>

      <ScenarioPicker
        onSelect={choose}
        recents={recents}
        onClearRecents={clear}
        initialValue={scenario}
      />

      <div className="flex min-h-0 flex-1 gap-4">
        <div className="min-w-0 flex-1">
          {scenario === "" && (
            <p className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
              {t(strings.common.chooseScenario)}
            </p>
          )}

          {scenario !== "" && query.isLoading && (
            <p data-testid={selectors.matrix.loading} className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
              {t(strings.matrix.loading)}
            </p>
          )}

          {scenario !== "" && query.isError && (
            <div
              data-testid={selectors.matrix.error}
              role="alert"
              className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
            >
              <p>{t(strings.matrix.error)}</p>
              <p className="mt-1 text-xs opacity-80">{errorMessage(query.error, t)}</p>
            </div>
          )}

          {response && (
            <div className="flex flex-col gap-4">
              {registry && (
                <div
                  data-testid={selectors.matrix.registrySummary}
                  className="grid grid-cols-2 gap-2 sm:grid-cols-4"
                >
                  <RegistryTile label={t(strings.matrix.registry.modules)} value={registry.moduleCount} />
                  <RegistryTile label={t(strings.matrix.registry.requirements)} value={registry.requirementCount} />
                  <RegistryTile label={t(strings.matrix.registry.targets)} value={registry.operationalTargetCount} />
                  <div className="rounded-panel border border-app-border bg-app-surface p-3">
                    <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
                      {t(strings.matrix.registry.starter)}
                    </p>
                    <div className="mt-1">
                      {registry.starterTemplate ? (
                        <StatusChip tone="warning">{t(strings.matrix.unproven)}</StatusChip>
                      ) : (
                        <span className="text-sm text-app-foreground">—</span>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {unproven > 0 && (
                <p className="flex items-center gap-2 text-xs text-app-danger">
                  <AlertTriangle aria-hidden="true" className="h-4 w-4" />
                  <StatusChip tone="danger">{t(strings.matrix.unproven)}</StatusChip>
                </p>
              )}

              {response.degradedReason && (
                <p
                  data-testid={selectors.matrix.degradedBanner}
                  role="status"
                  className="rounded-panel border border-app-warning/40 bg-app-warning/10 p-3 text-xs text-app-warning"
                >
                  {t(strings.matrix.degraded, { reason: response.degradedReason })}
                </p>
              )}

              {groups.length === 0 ? (
                <p data-testid={selectors.matrix.empty} className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
                  {t(strings.matrix.empty)}
                </p>
              ) : (
                <div data-testid={selectors.matrix.grid} className="flex flex-col gap-4">
                  {groups.map((group) => (
                    <section
                      key={group.otId}
                      data-testid={selectors.matrix.otGroup({ otId: group.otId })}
                      className="overflow-hidden rounded-panel border border-app-border bg-app-surface"
                    >
                      <header className="flex flex-wrap items-center gap-2 border-b border-app-border bg-app-surface-muted px-3 py-2">
                        {group.otPriority && (
                          <span className="rounded-pill bg-app-primary/10 px-2 py-0.5 text-xs font-semibold text-app-primary">
                            {group.otPriority}
                          </span>
                        )}
                        <span className="font-mono text-xs text-app-muted-foreground">{group.otId}</span>
                        <span className="min-w-0 flex-1 truncate text-sm font-medium text-app-foreground">
                          {group.otTitle}
                        </span>
                        {group.otId !== UNLINKED_GROUP_ID && (
                          <StatusChip tone={group.otChecked ? "success" : "neutral"}>
                            {group.otChecked ? t(strings.matrix.checked) : t(strings.matrix.unchecked)}
                          </StatusChip>
                        )}
                      </header>

                      {group.isOrphanTarget && (
                        <p className="px-3 py-2 text-xs text-app-warning">
                          {t(strings.matrix.orphanTarget)}
                        </p>
                      )}
                      {group.otId === UNLINKED_GROUP_ID && (
                        <p className="px-3 py-2 text-xs text-app-warning">
                          {t(strings.matrix.orphanRequirement)}
                        </p>
                      )}

                      <ul className="divide-y divide-app-border">
                        {group.rows.map((row) => (
                          <li
                            key={row.requirementId}
                            data-testid={selectors.matrix.requirementRow({ requirementId: row.requirementId })}
                            className={row.unproven ? "border-l-2 border-app-danger" : undefined}
                          >
                            <button
                              type="button"
                              data-testid={selectors.matrix.drillButton({ requirementId: row.requirementId })}
                              onClick={() => setSelectedId(row.requirementId)}
                              aria-expanded={selectedId === row.requirementId}
                              className="flex w-full flex-wrap items-center gap-2 px-3 py-2 text-start hover:bg-app-surface-muted"
                            >
                              <span className="font-mono text-xs text-app-muted-foreground">{row.requirementId}</span>
                              <span className="min-w-0 flex-1 truncate text-sm text-app-foreground">
                                {row.requirementTitle}
                              </span>
                              {row.unproven && (
                                <StatusChip tone="danger">{t(strings.matrix.unproven)}</StatusChip>
                              )}
                              <StatusChip tone={statusTone(row.requirementStatus)}>
                                {row.requirementStatus}
                              </StatusChip>
                              <EvidenceSummary evidence={row.evidence} />
                              <ChevronRight aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
                            </button>
                          </li>
                        ))}
                      </ul>
                    </section>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {selectedRow && (
          <div className="fixed inset-0 z-20 bg-app-shell/40 md:static md:z-auto md:bg-transparent">
            <div className="absolute inset-y-0 end-0 w-full max-w-md md:static md:w-96">
              <RequirementDrawer
                scenario={scenario}
                row={selectedRow}
                onClose={() => setSelectedId(null)}
              />
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

function RegistryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-3">
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-xl font-semibold text-app-foreground">{value}</p>
    </div>
  );
}
