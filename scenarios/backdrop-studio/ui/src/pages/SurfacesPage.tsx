import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.2.0";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";
import type { Surface } from "../api/studio";

/**
 * Every declared output target, with the citation its geometry came from.
 *
 * The authority column is the reason this is a page rather than a dropdown.
 * Some geometry is ours to choose and some is not: a hero is whatever width we
 * decide, while a Play feature graphic is whatever Google says it is this
 * quarter — and a wrong store dimension is rejected at submission, the most
 * expensive place to find it. Showing the citation beside the number is what
 * lets an operator tell the two kinds apart.
 */
export function SurfacesPage() {
  const { t } = useTranslation();
  const surfaces = useSurfaces();
  const rows = surfaces.data ?? [];

  const columns: Array<DataTableColumn<Surface>> = [
    {
      id: "id",
      header: t(strings.pages.surfaces.idHeading),
      accessor: (row) => <span className="font-mono text-sm">{row.id}</span>,
      sortValue: (row) => row.id,
    },
    {
      id: "kind",
      header: t(strings.pages.surfaces.kindHeading),
      accessor: (row) => <StatusBadge>{row.kind}</StatusBadge>,
      sortValue: (row) => row.kind,
    },
    {
      id: "geometry",
      header: t(strings.pages.surfaces.geometryHeading),
      accessor: (row) => `${row.width}×${row.height}`,
      sortValue: (row) => row.width * row.height,
    },
    {
      id: "placements",
      header: t(strings.pages.surfaces.placementsHeading),
      accessor: (row) => <span className="text-xs">{row.placements.join(", ")}</span>,
      sortValue: (row) => row.placements.join(","),
    },
    {
      id: "authority",
      header: t(strings.pages.surfaces.authorityHeading),
      accessor: (row) => <span className="text-xs">{row.authority}</span>,
      sortValue: (row) => row.authority,
    },
    {
      id: "confirmed",
      header: t(strings.pages.surfaces.confirmedHeading),
      accessor: (row) => <span className="font-mono text-xs">{row.confirmedOn}</span>,
      sortValue: (row) => row.confirmedOn,
    },
  ];

  const state = surfaces.isLoading
    ? "loading"
    : surfaces.isError
      ? "error"
      : rows.length === 0
        ? "empty"
        : "ready";

  return (
    <ExperienceSurface
      surfaceId="surfaces"
      state={state}
      statusMessage={state === "loading" ? t(strings.pages.surfaces.loading) : undefined}
      data-testid={selectors.pages.surfaces}
      aria-labelledby="surfaces-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="surfaces-heading" className="text-2xl font-semibold">
          {t(strings.pages.surfaces.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">
          {t(strings.pages.surfaces.description)}
        </p>
      </header>

      {state === "loading" ? <p role="status">{t(strings.pages.surfaces.loading)}</p> : null}
      {state === "error" ? <p role="alert">{t(strings.pages.surfaces.error)}</p> : null}
      {state === "empty" ? (
        <EmptyState
          title={t(strings.pages.surfaces.title)}
          description={t(strings.pages.surfaces.empty)}
        />
      ) : null}
      {state === "ready" ? (
        <DataTable
          rows={rows}
          columns={columns}
          getRowKey={(row) => row.id}
          caption={t(strings.pages.surfaces.caption)}
          emptyMessage={t(strings.pages.surfaces.empty)}
          tableTestId="surfaces-table"
        />
      ) : null}
    </ExperienceSurface>
  );
}
