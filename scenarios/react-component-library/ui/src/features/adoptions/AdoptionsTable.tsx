import { Button } from "@vrooli/react-component-library/Button/2";
import {
  DataTable,
  type DataTableColumn,
  type DataTableFilter,
} from "@vrooli/react-component-library/DataTable/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { LibraryVersionStatus, LocalStatus, type Adoption } from "../../api/adoptions";

const STATUS_KEY: Record<number, string> = {
  [LibraryVersionStatus.UNSPECIFIED]: "unspecified",
  [LibraryVersionStatus.CURRENT]: "current",
  [LibraryVersionStatus.BEHIND]: "behind",
  [LibraryVersionStatus.DEPRECATED]: "deprecated",
  [LibraryVersionStatus.MISSING]: "missing",
  [LibraryVersionStatus.UNKNOWN]: "unknown",
};

export function adoptionStatusKey(status: LibraryVersionStatus): string {
  return STATUS_KEY[status] ?? "unspecified";
}

type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

function statusLabel(status: LibraryVersionStatus, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === LibraryVersionStatus.CURRENT) return t(strings.adoptions.status.current);
  if (status === LibraryVersionStatus.BEHIND) return t(strings.adoptions.status.behind);
  if (status === LibraryVersionStatus.UNKNOWN || status === LibraryVersionStatus.MISSING) {
    return t(strings.adoptions.status.unknown);
  }
  return t(strings.adoptions.status.unspecified);
}

function localLabel(status: LocalStatus, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === LocalStatus.MODIFIED) return t(strings.adoptions.status.modified);
  if (status === LocalStatus.MISSING) return t(strings.adoptions.status.missing);
  if (status === LocalStatus.UNKNOWN) return t(strings.adoptions.status.unknown);
  return t(strings.adoptions.status.clean);
}

function libraryTone(status: LibraryVersionStatus): StatusTone {
  if (status === LibraryVersionStatus.CURRENT) return "success";
  if (status === LibraryVersionStatus.BEHIND || status === LibraryVersionStatus.DEPRECATED)
    return "warning";
  if (status === LibraryVersionStatus.MISSING) return "danger";
  if (status === LibraryVersionStatus.UNKNOWN) return "info";
  return "neutral";
}

function localTone(status: LocalStatus): StatusTone {
  if (status === LocalStatus.MODIFIED) return "warning";
  if (status === LocalStatus.MISSING) return "danger";
  if (status === LocalStatus.UNKNOWN) return "info";
  return "success";
}

function refreshedAtLabel(adoption: Adoption, t: ReturnType<typeof useTranslation>["t"]) {
  return adoption.refreshedAt
    ? t(strings.adoptions.refreshedAt, {
        when: adoption.refreshedAt.seconds
          ? new Date(Number(adoption.refreshedAt.seconds) * 1000).toLocaleString()
          : "",
      })
    : t(strings.adoptions.neverRefreshed);
}

export function AdoptionsTable({
  adoptions,
  onDelete,
  deletePending,
}: {
  adoptions: Adoption[];
  onDelete: (id: string) => void;
  deletePending: boolean;
}) {
  const { t } = useTranslation();
  const searchValue = (adoption: Adoption) =>
    [
      adoption.scenario,
      adoption.libraryId,
      adoption.adoptedPath,
      adoption.adoptedVersion,
      adoption.statusDetail,
      adoptionStatusKey(adoption.libraryVersionStatus),
    ]
      .filter(Boolean)
      .join(" ");
  const columns: Array<DataTableColumn<Adoption>> = [
    {
      id: "scenario",
      header: "Scenario",
      sortValue: (a) => a.scenario,
      searchValue,
      accessor: (a) => (
        <div className="min-w-column-wide">
          <div data-testid={selectors.adoptions.itemScenario} className="font-medium">
            {a.scenario}
          </div>
          <div
            data-testid={selectors.adoptions.itemPath}
            className="mt-space-3xs font-mono text-xs text-app-muted-foreground"
          >
            {a.adoptedPath}
          </div>
          <span data-testid={selectors.adoptions.itemId} className="sr-only">
            {a.id}
          </span>
        </div>
      ),
    },
    {
      id: "component",
      header: "Component",
      sortValue: (a) => a.libraryId,
      searchValue: (a) => `${a.libraryId} ${a.adoptedVersion}`,
      accessor: (a) => (
        <div className="min-w-column text-xs">
          <div data-testid={selectors.adoptions.itemLibraryId}>{a.libraryId}</div>
          {a.adoptedVersion && (
            <div
              data-testid={selectors.adoptions.itemVersion}
              className="mt-space-3xs text-app-muted-foreground"
            >
              {t(strings.adoptions.versionLabel, { version: a.adoptedVersion })}
            </div>
          )}
        </div>
      ),
    },
    {
      id: "status",
      header: "Status",
      sortValue: (a) => adoptionStatusKey(a.libraryVersionStatus),
      searchValue: (a) =>
        `${adoptionStatusKey(a.libraryVersionStatus)} ${localLabel(a.localStatus, t)}`,
      accessor: (a) => {
        const library = statusLabel(a.libraryVersionStatus, t);
        const local = localLabel(a.localStatus, t);
        return (
          <div className="flex min-w-column flex-col gap-space-2xs">
            <div className="flex flex-wrap gap-space-2xs">
              <StatusBadge
                data-testid={selectors.adoptions.itemStatus}
                role="status"
                aria-label={`${library} / ${local}`}
                tone={libraryTone(a.libraryVersionStatus)}
              >
                {library} / {local}
              </StatusBadge>
              <StatusBadge tone={localTone(a.localStatus)}>{local}</StatusBadge>
            </div>
            {a.statusDetail && (
              <div
                data-testid={selectors.adoptions.itemStatusDetail}
                className="text-xs text-app-muted-foreground"
              >
                {a.statusDetail}
              </div>
            )}
          </div>
        );
      },
    },
    {
      id: "refreshed",
      header: "Refreshed",
      sortValue: (a) => Number(a.refreshedAt?.seconds ?? 0),
      searchValue: (a) => refreshedAtLabel(a, t),
      accessor: (a) => (
        <span data-testid={selectors.adoptions.itemRefreshedAt} className="text-xs">
          {refreshedAtLabel(a, t)}
        </span>
      ),
    },
    {
      id: "actions",
      header: "Actions",
      className: "text-right",
      accessor: (a) => (
        <Button
          data-testid={selectors.adoptions.itemDeleteButton}
          onClick={() => onDelete(a.id)}
          disabled={deletePending}
          className="h-control-tight px-space-xs text-xs"
        >
          {deletePending ? t(strings.adoptions.deleting) : t(strings.adoptions.deleteAction)}
        </Button>
      ),
    },
  ];
  const filters: Array<DataTableFilter<Adoption>> = [
    { id: "all", label: "All", predicate: () => true },
    {
      id: "current",
      label: t(strings.adoptions.status.current),
      predicate: (a) => a.libraryVersionStatus === LibraryVersionStatus.CURRENT,
    },
    {
      id: "behind",
      label: t(strings.adoptions.status.behind),
      predicate: (a) => a.libraryVersionStatus === LibraryVersionStatus.BEHIND,
    },
    {
      id: "modified",
      label: t(strings.adoptions.status.modified),
      predicate: (a) => a.localStatus === LocalStatus.MODIFIED,
    },
  ];
  return (
    <DataTable
      rows={adoptions}
      columns={columns}
      getRowKey={(a) => a.id}
      caption={t(strings.adoptions.title)}
      searchLabel={t(strings.adoptions.scenarioFilterLabel)}
      searchPlaceholder={t(strings.adoptions.scenarioFilterPlaceholder)}
      emptyMessage={t(strings.adoptions.empty)}
      filters={filters}
    />
  );
}
