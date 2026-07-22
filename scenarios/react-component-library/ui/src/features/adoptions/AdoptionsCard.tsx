import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { Button } from "../../components/ui/button";
import {
  DataTable,
  type DataTableColumn,
  type DataTableFilter,
} from "../../components/ui/data-table";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { StatusBadge } from "../../components/ui/status-badge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  adoptionsClient,
  LibraryVersionStatus,
  LocalStatus,
  type Adoption,
} from "../../api/adoptions";
import { errorMessage } from "../../lib/errorMessage";
import { CreateAdoptionDialog } from "./CreateAdoptionDialog";

const STATUS_KEY: Record<number, string> = {
  [LibraryVersionStatus.UNSPECIFIED]: "unspecified",
  [LibraryVersionStatus.CURRENT]: "current",
  [LibraryVersionStatus.BEHIND]: "behind",
  [LibraryVersionStatus.DEPRECATED]: "deprecated",
  [LibraryVersionStatus.MISSING]: "missing",
  [LibraryVersionStatus.UNKNOWN]: "unknown",
};

const EMPTY_ADOPTIONS: Adoption[] = [];

function statusLabelFor(
  status: LibraryVersionStatus,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  switch (status) {
    case LibraryVersionStatus.CURRENT:
      return t(strings.adoptions.status.current);
    case LibraryVersionStatus.BEHIND:
      return t(strings.adoptions.status.behind);
    case LibraryVersionStatus.UNKNOWN:
    case LibraryVersionStatus.MISSING:
      return t(strings.adoptions.status.unknown);
    default:
      return t(strings.adoptions.status.unspecified);
  }
}

function statusKey(s: LibraryVersionStatus): string {
  return STATUS_KEY[s] ?? "unspecified";
}

function localStatusLabelFor(status: LocalStatus, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (status) {
    case LocalStatus.MODIFIED:
      return t(strings.adoptions.status.modified);
    case LocalStatus.MISSING:
      return t(strings.adoptions.status.missing);
    case LocalStatus.UNKNOWN:
      return t(strings.adoptions.status.unknown);
    default:
      return t(strings.adoptions.status.clean);
  }
}

type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

function libraryTone(status: LibraryVersionStatus): StatusTone {
  switch (status) {
    case LibraryVersionStatus.CURRENT:
      return "success";
    case LibraryVersionStatus.BEHIND:
    case LibraryVersionStatus.DEPRECATED:
      return "warning";
    case LibraryVersionStatus.MISSING:
      return "danger";
    case LibraryVersionStatus.UNKNOWN:
      return "info";
    default:
      return "neutral";
  }
}

function localTone(status: LocalStatus): StatusTone {
  switch (status) {
    case LocalStatus.MODIFIED:
      return "warning";
    case LocalStatus.MISSING:
      return "danger";
    case LocalStatus.UNKNOWN:
      return "info";
    default:
      return "success";
  }
}

function adoptionSearchValue(adoption: Adoption) {
  return [
    adoption.scenario,
    adoption.libraryId,
    adoption.adoptedPath,
    adoption.adoptedVersion,
    adoption.statusDetail,
    statusKey(adoption.libraryVersionStatus),
  ]
    .filter(Boolean)
    .join(" ");
}

function refreshedAtLabel(adoption: Adoption, t: ReturnType<typeof useTranslation>["t"]) {
  if (!adoption.refreshedAt) {
    return t(strings.adoptions.neverRefreshed);
  }
  return t(strings.adoptions.refreshedAt, {
    when: adoption.refreshedAt.seconds
      ? new Date(Number(adoption.refreshedAt.seconds) * 1000).toLocaleString()
      : "",
  });
}

/**
 * AdoptionsCard renders the adoption registry: every soft-linked
 * library component → target scenario mapping plus its drift status.
 * Surface for req 08 (AD-001..AD-003).
 */
export function AdoptionsCard({ componentId, suggestionsOnly = false }: { componentId?: string; suggestionsOnly?: boolean }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", { scenario, componentId }],
    queryFn: () => adoptionsClient.listAdoptions({ scenario, componentId, limit: 0 }),
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

  const adoptions = adoptionsQuery.data?.adoptions ?? EMPTY_ADOPTIONS;
  const suggestionsQuery = useQuery({
    queryKey: ["adoptions", "suggestions", scenario],
    queryFn: () => adoptionsClient.suggestAdoptions({ scenario, limit: 8 }),
  });

  const summary = useMemo(() => {
    const acc = { current: 0, behind: 0, modified: 0, unknown: 0 };
    for (const a of adoptions) {
      const k = statusKey(a.libraryVersionStatus);
      if (k === "current") acc.current++;
      else if (k === "behind") acc.behind++;
      else if (k === "unknown") acc.unknown++;
      if (a.localStatus === LocalStatus.MODIFIED) acc.modified++;
    }
    return acc;
  }, [adoptions]);

  const columns: Array<DataTableColumn<Adoption>> = [
    {
      id: "scenario",
      header: "Scenario",
      sortValue: (adoption) => adoption.scenario,
      searchValue: adoptionSearchValue,
      accessor: (adoption) => (
        <div className="min-w-56">
          <div data-testid={selectors.adoptions.itemScenario} className="font-medium">
            {adoption.scenario}
          </div>
          <div
            data-testid={selectors.adoptions.itemPath}
            className="mt-1 font-mono text-xs text-app-muted-foreground"
          >
            {adoption.adoptedPath}
          </div>
          <span data-testid={selectors.adoptions.itemId} className="sr-only">
            {adoption.id}
          </span>
        </div>
      ),
    },
    {
      id: "component",
      header: "Component",
      sortValue: (adoption) => adoption.libraryId,
      searchValue: (adoption) => `${adoption.libraryId} ${adoption.adoptedVersion}`,
      accessor: (adoption) => (
        <div className="min-w-48 text-xs">
          <div data-testid={selectors.adoptions.itemLibraryId}>{adoption.libraryId}</div>
          {adoption.adoptedVersion && (
            <div
              data-testid={selectors.adoptions.itemVersion}
              className="mt-1 text-app-muted-foreground"
            >
              {t(strings.adoptions.versionLabel, { version: adoption.adoptedVersion })}
            </div>
          )}
        </div>
      ),
    },
    {
      id: "status",
      header: "Status",
      sortValue: (adoption) => statusKey(adoption.libraryVersionStatus),
      searchValue: (adoption) =>
        `${statusKey(adoption.libraryVersionStatus)} ${localStatusLabelFor(adoption.localStatus, t)}`,
      accessor: (adoption) => {
        const statusLabel = statusLabelFor(adoption.libraryVersionStatus, t);
        const localLabel = localStatusLabelFor(adoption.localStatus, t);
        return (
          <div className="flex min-w-48 flex-col gap-2">
            <div className="flex flex-wrap gap-2">
              <StatusBadge
                data-testid={selectors.adoptions.itemStatus}
                role="status"
                aria-label={`${statusLabel} / ${localLabel}`}
                tone={libraryTone(adoption.libraryVersionStatus)}
              >
                {statusLabel} / {localLabel}
              </StatusBadge>
              <StatusBadge tone={localTone(adoption.localStatus)}>{localLabel}</StatusBadge>
            </div>
            {adoption.statusDetail && (
              <div
                data-testid={selectors.adoptions.itemStatusDetail}
                className="text-xs text-app-muted-foreground"
              >
                {adoption.statusDetail}
              </div>
            )}
          </div>
        );
      },
    },
    {
      id: "refreshed",
      header: "Refreshed",
      sortValue: (adoption) => Number(adoption.refreshedAt?.seconds ?? 0),
      searchValue: (adoption) => refreshedAtLabel(adoption, t),
      accessor: (adoption) => (
        <span data-testid={selectors.adoptions.itemRefreshedAt} className="text-xs">
          {refreshedAtLabel(adoption, t)}
        </span>
      ),
    },
    {
      id: "actions",
      header: "Actions",
      className: "text-right",
      accessor: (adoption) => (
        <Button
          data-testid={selectors.adoptions.itemDeleteButton}
          onClick={() => deleteMutation.mutate(adoption.id)}
          disabled={deleteMutation.isPending}
          className="h-8 px-3 text-xs"
        >
          {deleteMutation.isPending
            ? t(strings.adoptions.deleting)
            : t(strings.adoptions.deleteAction)}
        </Button>
      ),
    },
  ];

  if (suggestionsOnly) {
    const suggestions = suggestionsQuery.data?.suggestions ?? [];
    return <section data-testid="adoption-suggestions" className="rounded-panel border border-app-border bg-app-surface p-4"><h3 className="text-sm font-semibold text-app-foreground">{t(strings.adoptions.suggestions.title)}</h3><p className="mt-1 text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.subtitle)}</p><div className="mt-3 space-y-2">{suggestions.map((item) => <article key={`${item.scenario}-${item.componentId}`} className="flex items-start justify-between gap-3 rounded-control border border-app-border p-3 text-xs"><div><div className="font-medium text-app-foreground">{item.displayName} <span className="text-app-muted-foreground">→ {item.scenario}</span></div><StatusBadge tone="neutral">{item.classification === 1 ? t("adoptions.suggestions.heuristic", { defaultValue: "Heuristic candidate — review before adopting" }) : t("adoptions.suggestions.unavailable", { defaultValue: "Unavailable candidate" })}</StatusBadge><ul className="mt-1 list-disc space-y-0.5 ps-4 text-app-muted-foreground">{item.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div><Button size="sm" onClick={() => void navigate(`/?action=adopt&assetId=${encodeURIComponent(item.componentId)}&targetScenario=${encodeURIComponent(item.scenario)}`)}>{t(strings.adoptions.suggestions.adoptAction)}</Button></article>)}{!suggestionsQuery.isLoading && suggestions.length === 0 && <p className="text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.empty)}</p>}</div></section>;
  }

  const filters: Array<DataTableFilter<Adoption>> = [
    {
      id: "all",
      label: "All",
      predicate: () => true,
    },
    {
      id: "current",
      label: t(strings.adoptions.status.current),
      predicate: (adoption) => adoption.libraryVersionStatus === LibraryVersionStatus.CURRENT,
    },
    {
      id: "behind",
      label: t(strings.adoptions.status.behind),
      predicate: (adoption) => adoption.libraryVersionStatus === LibraryVersionStatus.BEHIND,
    },
    {
      id: "modified",
      label: t(strings.adoptions.status.modified),
      predicate: (adoption) => adoption.localStatus === LocalStatus.MODIFIED,
    },
  ];

  return (
    <section
      data-testid={selectors.adoptions.card}
      aria-label={t(strings.adoptions.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4 backdrop-blur-sm"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.adoptions.title)}</h2>
        <div className="flex items-center gap-2">
          <Button
            data-testid={selectors.adoptions.createButton}
            variant="secondary"
            onClick={() => void navigate("/?action=adopt")}
          >
            {t(strings.adoptions.createAction)}
          </Button>
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
      </div>

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
        <label className="block text-xs text-app-muted-foreground">
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
        <p data-testid={selectors.adoptions.loading} className="mt-3 text-app-foreground">
          {t(strings.adoptions.loading)}
        </p>
      )}
      {adoptionsQuery.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-app-danger">
          {errorMessage(adoptionsQuery.error, t)}
        </p>
      )}
      {refreshMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-app-danger">
          {errorMessage(refreshMutation.error, t)}
        </p>
      )}
      {deleteMutation.error && (
        <p data-testid={selectors.adoptions.error} className="mt-3 text-app-danger">
          {errorMessage(deleteMutation.error, t)}
        </p>
      )}

      {adoptions.length === 0 && !adoptionsQuery.isLoading && !adoptionsQuery.error && (
        <div data-testid={selectors.adoptions.empty} className="mt-3">
          <EmptyState
            title={t(strings.adoptions.empty)}
            action={(
              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" onClick={() => void navigate("/?action=adopt")}>
                  {t(strings.adoptions.createAction)}
                </Button>
                <Button
                  onClick={() => refreshMutation.mutate()}
                  disabled={refreshMutation.isPending}
                >
                  {refreshMutation.isPending
                    ? t(strings.adoptions.refreshing)
                    : t(strings.adoptions.refreshAction)}
                </Button>
              </div>
            )}
          />
        </div>
      )}

      {adoptions.length > 0 && (
        <>
          <p data-testid={selectors.adoptions.summary} className="mt-3 text-xs text-app-muted-foreground">
            {t(strings.adoptions.summary, {
              count: adoptions.length,
              current: summary.current,
              behind: summary.behind,
              modified: summary.modified,
              unknown: summary.unknown,
            })}
          </p>
          <div data-testid={selectors.adoptions.list} className="mt-2">
            <DataTable
              rows={adoptions}
              columns={columns}
              getRowKey={(adoption) => adoption.id}
              caption={t(strings.adoptions.title)}
              searchLabel={t(strings.adoptions.scenarioFilterLabel)}
              searchPlaceholder={t(strings.adoptions.scenarioFilterPlaceholder)}
              emptyMessage={t(strings.adoptions.empty)}
              filters={filters}
            />
          </div>
        </>
      )}

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <h2 className="text-sm font-semibold text-app-foreground">{t(strings.adoptions.suggestions.title)}</h2>
        <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.subtitle)}</p>
        <div className="mt-3 space-y-2">
          {(suggestionsQuery.data?.suggestions ?? []).map((item) => (
            <div key={`${item.scenario}-${item.componentId}`} className="flex items-start justify-between gap-3 rounded-control border border-app-border p-3 text-xs">
              <div>
                <div className="font-medium text-app-foreground">{item.displayName} <span className="text-app-muted-foreground">→ {item.scenario}</span></div>
                <p className="mt-1 text-app-muted-foreground">{item.classification === 1 ? t("adoptions.suggestions.heuristic", { defaultValue: "Heuristic candidate — review before adopting" }) : t("adoptions.suggestions.unavailable", { defaultValue: "Unavailable candidate" })}</p>
                <ul className="mt-1 list-disc space-y-0.5 ps-4 text-app-muted-foreground">{item.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul>
              </div>
              <Button size="sm" onClick={() => void navigate(`/?action=adopt&assetId=${encodeURIComponent(item.componentId)}&targetScenario=${encodeURIComponent(item.scenario)}`)}>{t(strings.adoptions.suggestions.adoptAction)}</Button>
            </div>
          ))}
          {!suggestionsQuery.isLoading && (suggestionsQuery.data?.suggestions.length ?? 0) === 0 && <p className="text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.empty)}</p>}
        </div>
      </section>
      <details className="mt-3 text-xs text-app-muted-foreground">
        <summary className="cursor-pointer">Advanced local re-link</summary>
        <p className="mt-2">Use this only to record an existing local copy; normal adoption starts the guided adopt-assist workflow.</p>
        <Button variant="secondary" size="sm" className="mt-2" onClick={() => setCreateOpen(true)}>Open local re-link</Button>
      </details>
      <CreateAdoptionDialog open={createOpen} initial={null} onClose={() => setCreateOpen(false)} />
    </section>
  );
}
