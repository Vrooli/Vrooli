import { useMemo, useState } from "react";

import type { StyleFilter } from "../api/studio";
import { AxisFilters } from "../components/studio/AxisFilters";
import { StyleSpecimen } from "../components/studio/StyleSpecimen";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/ui/empty-state";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { axisValues, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/** The seed every catalog specimen renders at, so tiles are comparable. */
const SPECIMEN_SEED = 7n;

/**
 * The catalog browser.
 *
 * This is the entry point and the discovery surface, and the audit found about
 * ten percent of it built: eleven routes pointed at one component holding four
 * hardcoded style rows. The replacement reads the real catalog and renders a
 * real specimen per style, because "what does this look like" is the only
 * question a style catalog exists to answer.
 */
export function CatalogPage() {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<StyleFilter>({});

  // The unfiltered catalog is fetched alongside the filtered one so the facets
  // list every value the catalog has rather than only those surviving the
  // current filter — otherwise choosing one facet silently empties the others
  // and the operator cannot widen without clearing everything.
  const all = useStyles({});
  const filtered = useStyles(filter);
  const surfaces = useSurfaces();

  const values = useMemo(() => axisValues(all.data ?? []), [all.data]);
  const styles = filtered.data ?? [];
  const isFiltered = Object.values(filter).some(Boolean);

  const state = filtered.isLoading || surfaces.isLoading
    ? "loading"
    : filtered.isError
      ? "error"
      : styles.length === 0
        ? "empty"
        : "ready";

  return (
    <ExperienceSurface
      surfaceId="catalog"
      state={state}
      statusMessage={
        state === "loading"
          ? t(strings.pages.catalog.loading)
          : state === "error"
            ? t(strings.pages.catalog.error)
            : undefined
      }
      data-testid={selectors.pages.catalog}
      aria-labelledby="catalog-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="catalog-heading" className="text-2xl font-semibold">
          {t(strings.pages.catalog.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">
          {t(strings.pages.catalog.description)}
        </p>
      </header>

      {all.data ? (
        <AxisFilters
          values={values}
          filter={filter}
          onChange={setFilter}
          resultCount={styles.length}
        />
      ) : null}

      {state === "loading" ? (
        <p role="status" className="rounded-panel border border-app-border p-6">
          {t(strings.pages.catalog.loading)}
        </p>
      ) : null}

      {state === "error" ? (
        <section role="alert" className="flex flex-col gap-3 rounded-panel border border-app-border p-6">
          <p>{t(strings.pages.catalog.error)}</p>
          <Button variant="secondary" onClick={() => void filtered.refetch()}>
            {t(strings.pages.catalog.retry)}
          </Button>
        </section>
      ) : null}

      {state === "empty" ? (
        <EmptyState
          title={t(
            isFiltered ? strings.pages.catalog.filteredEmptyTitle : strings.pages.catalog.emptyTitle,
          )}
          description={t(
            isFiltered
              ? strings.pages.catalog.filteredEmptyDescription
              : strings.pages.catalog.emptyDescription,
          )}
          action={
            isFiltered ? (
              <Button variant="secondary" onClick={() => setFilter({})}>
                {t(strings.pages.catalog.clearFilters)}
              </Button>
            ) : undefined
          }
        />
      ) : null}

      {state === "ready" ? (
        <div
          className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="catalog-grid"
        >
          {styles.map((style) => (
            <StyleSpecimen
              key={style.id}
              style={style}
              surfaces={surfaces.data ?? []}
              seed={SPECIMEN_SEED}
            />
          ))}
        </div>
      ) : null}
    </ExperienceSurface>
  );
}
