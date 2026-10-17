import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowUpRight, Boxes, Loader2, ScanLine, SearchX } from "lucide-react";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card, CardBody, CardDescription, CardHeader, CardTitle } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { Table, type ColumnDef } from "../../components/ui/Table";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";
import {
  SURFACE_KIND_FILTERS,
  encodeSurfaceId,
  type SurfaceKindFilter,
  type SurfaceRecord,
} from "../../api/inventory";

import { useInventory } from "./useInventory";

const SCENARIO_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

const KIND_LABEL_KEY = {
  all: strings.pages.search.kind.all,
  component: strings.pages.search.kind.component,
  page: strings.pages.search.kind.page,
  feature: strings.pages.search.kind.feature,
  hook: strings.pages.search.kind.hook,
  layout: strings.pages.search.kind.layout,
  other: strings.pages.search.kind.other,
} as const satisfies Record<SurfaceKindFilter, string>;

function formatTimestamp(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

export function InventoryPage() {
  const { t } = useTranslation();
  const {
    scenario,
    setScenario,
    activeScenario,
    submit,
    kind,
    setKind,
    filteredSurfaces,
    countByKind,
    query_,
  } = useInventory();

  const trimmed = scenario.trim();
  const inputInvalid = trimmed.length > 0 && !SCENARIO_NAME_PATTERN.test(trimmed);
  const [formError, setFormError] = useState<string | null>(null);

  const isLoading = query_.isFetching && activeScenario.length > 0;
  const hasData = query_.isSuccess && activeScenario.length > 0;
  const surfacesCount = hasData ? query_.data.surfaces.length : 0;
  const scannedAt = hasData ? query_.data.scannedAt : "";
  const noSurfaces = hasData && surfacesCount === 0;
  const noFilterMatches =
    hasData && !noSurfaces && filteredSurfaces.length === 0;

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setFormError(null);
    if (trimmed.length === 0 || !SCENARIO_NAME_PATTERN.test(trimmed)) {
      setFormError(t(strings.pages.inventory.form.scenarioHelp));
      return;
    }
    submit(trimmed);
  }

  const columns: ColumnDef<SurfaceRecord>[] = [
    {
      key: "displayName",
      header: t(strings.pages.inventory.columns.displayName),
      cell: (row, idx) => (
        <Link
          to={ROUTES.surfaceDetail(encodeSurfaceId(row.scenario, row.slot))}
          className="font-medium text-app-primary hover:underline break-all"
          data-testid={selectors.inventory.surfaceOpen({ index: idx })}
        >
          {row.displayName || row.slot || "—"}
        </Link>
      ),
    },
    {
      key: "kind",
      header: t(strings.pages.inventory.columns.kind),
      cell: (row) => (
        <Badge tone="neutral">{t(KIND_LABEL_KEY[row.kind === "unspecified" ? "all" : row.kind])}</Badge>
      ),
    },
    {
      key: "slot",
      header: t(strings.pages.inventory.columns.slot),
      cell: (row) => <span className="font-mono text-xs">{row.slot || "—"}</span>,
    },
    {
      key: "filePath",
      header: t(strings.pages.inventory.columns.filePath),
      cell: (row) => (
        <span className="font-mono text-xs text-app-muted-foreground break-all">
          {row.filePath || "—"}
        </span>
      ),
    },
  ];

  return (
    <section
      data-testid={selectors.pages.inventory}
      aria-labelledby="inventory-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="inventory-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.inventory.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.inventory.description)}
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.inventory.form.heading)}</CardTitle>
          <CardDescription>{t(strings.pages.inventory.form.scenarioHelp)}</CardDescription>
        </CardHeader>
        <CardBody>
          <form
            onSubmit={onSubmit}
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
            data-testid={selectors.inventory.form}
          >
            <div className="flex-1 min-w-0 flex flex-col gap-1">
              <label
                htmlFor="inventory-scenario"
                className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground"
              >
                {t(strings.pages.inventory.form.scenarioLabel)}
              </label>
              <input
                id="inventory-scenario"
                type="text"
                value={scenario}
                onChange={(e) => setScenario(e.target.value)}
                aria-invalid={inputInvalid || undefined}
                aria-describedby={formError ? "inventory-form-error" : undefined}
                placeholder={t(strings.pages.inventory.form.scenarioPlaceholder)}
                data-testid={selectors.inventory.scenarioInput}
                className="h-11 min-h-touch w-full rounded-control border border-app-border bg-app-background px-3 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus"
              />
            </div>
            <Button
              type="submit"
              loading={isLoading}
              disabled={isLoading || trimmed.length === 0}
              data-testid={selectors.inventory.submit}
            >
              {isLoading
                ? t(strings.pages.inventory.form.submitting)
                : t(strings.pages.inventory.form.submit)}
            </Button>
          </form>
          {formError ? (
            <p
              id="inventory-form-error"
              role="alert"
              className="mt-2 text-sm text-app-danger"
            >
              {formError}
            </p>
          ) : null}
        </CardBody>
      </Card>

      {hasData && !noSurfaces ? (
        <fieldset
          className="flex flex-wrap items-center gap-2"
          data-testid={selectors.inventory.filters}
        >
          <legend className="sr-only">{t(strings.pages.inventory.filters.heading)}</legend>
          {SURFACE_KIND_FILTERS.map((k) => {
            const active = kind === k;
            const count = countByKind[k];
            return (
              <button
                key={k}
                type="button"
                onClick={() => setKind(k)}
                aria-pressed={active}
                data-testid={selectors.inventory.kindFilter({ kind: k })}
                className={
                  active
                    ? "rounded-pill bg-app-primary px-3 py-1 text-xs font-medium text-app-primary-foreground min-h-touch md:min-h-0"
                    : "rounded-pill border border-app-border bg-app-surface px-3 py-1 text-xs font-medium text-app-foreground hover:bg-app-surface-muted min-h-touch md:min-h-0"
                }
              >
                <span>{t(KIND_LABEL_KEY[k])}</span>
                <span className="ml-2 tabular-nums text-app-muted-foreground">{count}</span>
              </button>
            );
          })}
        </fieldset>
      ) : null}

      {isLoading ? (
        <p
          className="flex items-center gap-2 text-sm text-app-muted-foreground"
          role="status"
          aria-live="polite"
          data-testid={selectors.inventory.loading}
        >
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.inventory.loading)}
        </p>
      ) : null}

      {query_.error ? (
        <div
          role="alert"
          data-testid={selectors.inventory.error}
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          {t(strings.pages.inventory.error, {
            message:
              query_.error instanceof Error ? query_.error.message : String(query_.error),
          })}
        </div>
      ) : null}

      {activeScenario.length === 0 ? (
        <EmptyState
          icon={ScanLine}
          title={t(strings.pages.inventory.empty.title)}
          description={t(strings.pages.inventory.empty.description)}
          data-testid={selectors.inventory.empty}
        />
      ) : null}

      {noSurfaces ? (
        <EmptyState
          icon={Boxes}
          title={t(strings.pages.inventory.noSurfaces.title, { scenario: activeScenario })}
          description={t(strings.pages.inventory.noSurfaces.description)}
          data-testid={selectors.inventory.noSurfaces}
        />
      ) : null}

      {noFilterMatches ? (
        <EmptyState
          icon={SearchX}
          title={t(strings.pages.inventory.noResultsForFilter)}
          data-testid={selectors.inventory.noResultsForFilter}
        />
      ) : null}

      {hasData && !noSurfaces && filteredSurfaces.length > 0 ? (
        <div className="flex flex-col gap-3">
          <p
            className="text-sm text-app-muted-foreground"
            data-testid={selectors.inventory.summary}
            aria-live="polite"
          >
            {t(strings.pages.inventory.summary, {
              count: filteredSurfaces.length,
              scenario: activeScenario,
              scannedAt: formatTimestamp(scannedAt),
            })}
          </p>
          <Table
            columns={columns}
            rows={filteredSurfaces}
            rowKey={(row, idx) => `${row.scenario}__${row.slot || idx}`}
            caption={t(strings.pages.inventory.title)}
            data-testid={selectors.inventory.surfacesTable}
          />
          <p className="flex items-center gap-1 text-xs text-app-muted-foreground">
            <ArrowUpRight aria-hidden className="h-3 w-3" />
            {t(strings.pages.search.results.openInInventory)}
          </p>
        </div>
      ) : null}
    </section>
  );
}
