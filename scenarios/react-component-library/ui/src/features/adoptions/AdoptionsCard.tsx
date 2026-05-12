import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  adoptionsClient,
  AdoptionStatus,
  type Adoption,
} from "../../api/adoptions";
import { errorMessage } from "../../lib/errorMessage";

const STATUS_KEY: Record<number, string> = {
  [AdoptionStatus.UNSPECIFIED]: "unspecified",
  [AdoptionStatus.CURRENT]: "current",
  [AdoptionStatus.BEHIND]: "behind",
  [AdoptionStatus.MODIFIED]: "modified",
  [AdoptionStatus.UNKNOWN]: "unknown",
};

function statusLabelFor(status: AdoptionStatus, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (status) {
    case AdoptionStatus.CURRENT:
      return t(strings.adoptions.status.current);
    case AdoptionStatus.BEHIND:
      return t(strings.adoptions.status.behind);
    case AdoptionStatus.MODIFIED:
      return t(strings.adoptions.status.modified);
    case AdoptionStatus.UNKNOWN:
      return t(strings.adoptions.status.unknown);
    default:
      return t(strings.adoptions.status.unspecified);
  }
}

// STATUS_CLASS pairs each status with both a visual color and an
// explicit textual label. Critical for AD-003 a11y: status must convey
// non-color information, so the badge also renders the localized word.
const STATUS_CLASS: Record<number, string> = {
  [AdoptionStatus.UNSPECIFIED]: "bg-slate-700 text-slate-200",
  [AdoptionStatus.CURRENT]: "bg-emerald-700 text-emerald-100",
  [AdoptionStatus.BEHIND]: "bg-amber-700 text-amber-100",
  [AdoptionStatus.MODIFIED]: "bg-orange-700 text-orange-100",
  [AdoptionStatus.UNKNOWN]: "bg-slate-600 text-slate-100",
};

function statusKey(s: AdoptionStatus): string {
  return STATUS_KEY[s] ?? "unspecified";
}

function statusClass(s: AdoptionStatus): string {
  return STATUS_CLASS[s] ?? STATUS_CLASS[AdoptionStatus.UNSPECIFIED]!;
}

/**
 * AdoptionsCard renders the adoption registry: every soft-linked
 * library component → target scenario mapping plus its drift status.
 * Surface for req 08 (AD-001..AD-003).
 */
export function AdoptionsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", { scenario }],
    queryFn: () => adoptionsClient.listAdoptions({ scenario, limit: 0 }),
  });

  const refreshMutation = useMutation({
    mutationFn: () => adoptionsClient.refreshAdoptions({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["adoptions"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adoptionsClient.deleteAdoption({ id }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["adoptions"] });
    },
  });

  const adoptions: Adoption[] = adoptionsQuery.data?.adoptions ?? [];

  const summary = useMemo(() => {
    const acc = { current: 0, behind: 0, modified: 0, unknown: 0 };
    for (const a of adoptions) {
      const k = statusKey(a.status as AdoptionStatus);
      if (k === "current") acc.current++;
      else if (k === "behind") acc.behind++;
      else if (k === "modified") acc.modified++;
      else if (k === "unknown") acc.unknown++;
    }
    return acc;
  }, [adoptions]);

  return (
    <section
      data-testid={selectors.adoptions.card}
      aria-label={t(strings.adoptions.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4 backdrop-blur-sm"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-slate-400">{t(strings.adoptions.title)}</h2>
        <Button
          data-testid={selectors.adoptions.refreshButton}
          onClick={() => refreshMutation.mutate()}
          disabled={refreshMutation.isPending}
        >
          {refreshMutation.isPending
            ? t(strings.adoptions.refreshing)
            : t(strings.adoptions.refreshAction)}
        </Button>
      </div>

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
        <label className="block text-xs text-slate-400">
          {t(strings.adoptions.scenarioFilterLabel)}
          <Input
            data-testid={selectors.adoptions.scenarioFilter}
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            placeholder={t(strings.adoptions.scenarioFilterPlaceholder)}
            className="mt-1"
          />
        </label>
      </div>

      {adoptionsQuery.isLoading && (
        <p data-testid={selectors.adoptions.loading} className="mt-3 text-slate-200">
          {t(strings.adoptions.loading)}
        </p>
      )}
      {adoptionsQuery.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-red-400">
          {errorMessage(adoptionsQuery.error, t)}
        </p>
      )}
      {refreshMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-red-400">
          {errorMessage(refreshMutation.error, t)}
        </p>
      )}
      {deleteMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-red-400">
          {errorMessage(deleteMutation.error, t)}
        </p>
      )}

      {adoptionsQuery.data && adoptions.length === 0 && !adoptionsQuery.isLoading && (
        <p data-testid={selectors.adoptions.empty} className="mt-3 text-slate-200">
          {t(strings.adoptions.empty)}
        </p>
      )}

      {adoptions.length > 0 && (
        <>
          <p data-testid={selectors.adoptions.summary} className="mt-3 text-xs text-slate-400">
            {t(strings.adoptions.summary, {
              count: adoptions.length,
              current: summary.current,
              behind: summary.behind,
              modified: summary.modified,
              unknown: summary.unknown,
            })}
          </p>
          <ul data-testid={selectors.adoptions.list} className="mt-2 space-y-2 text-sm text-slate-200">
            {adoptions.map((a) => {
              const status = a.status as AdoptionStatus;
              const statusLabel = statusLabelFor(status, t);
              return (
                <li
                  key={a.id}
                  data-testid={selectors.adoptions.item}
                  className="rounded-lg border border-white/10 p-3"
                >
                  <div className="flex items-baseline justify-between gap-3">
                    <span
                      data-testid={selectors.adoptions.itemScenario}
                      className="font-medium"
                    >
                      {a.scenario}
                    </span>
                    <span
                      data-testid={selectors.adoptions.itemStatus}
                      role="status"
                      aria-label={statusLabel}
                      className={
                        "rounded-pill px-2 py-0.5 text-[0.65rem] uppercase tracking-wide " +
                        statusClass(status)
                      }
                    >
                      {statusLabel}
                    </span>
                  </div>
                  <div
                    data-testid={selectors.adoptions.itemPath}
                    className="mt-1 font-mono text-xs text-slate-400"
                  >
                    {a.adoptedPath}
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                    <span data-testid={selectors.adoptions.itemLibraryId}>{a.libraryId}</span>
                    {a.adoptedVersion && (
                      <span data-testid={selectors.adoptions.itemVersion}>
                        {t(strings.adoptions.versionLabel, { version: a.adoptedVersion })}
                      </span>
                    )}
                    <span data-testid={selectors.adoptions.itemRefreshedAt}>
                      {a.refreshedAt
                        ? t(strings.adoptions.refreshedAt, {
                            when: a.refreshedAt.seconds
                              ? new Date(Number(a.refreshedAt.seconds) * 1000).toLocaleString()
                              : "",
                          })
                        : t(strings.adoptions.neverRefreshed)}
                    </span>
                  </div>
                  {a.statusDetail && (
                    <div
                      data-testid={selectors.adoptions.itemStatusDetail}
                      className="mt-1 text-xs text-slate-400"
                    >
                      {a.statusDetail}
                    </div>
                  )}
                  <div className="mt-2 flex items-center justify-end">
                    <Button
                      data-testid={selectors.adoptions.itemDeleteButton}
                      onClick={() => deleteMutation.mutate(a.id)}
                      disabled={deleteMutation.isPending}
                      className="h-7 px-3 text-xs"
                    >
                      {deleteMutation.isPending
                        ? t(strings.adoptions.deleting)
                        : t(strings.adoptions.deleteAction)}
                    </Button>
                  </div>
                  <span data-testid={selectors.adoptions.itemId} className="sr-only">
                    {a.id}
                  </span>
                </li>
              );
            })}
          </ul>
        </>
      )}
    </section>
  );
}
