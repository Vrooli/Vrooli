import { AlertTriangle, Loader2, Search as SearchIcon, SearchX } from "lucide-react";

import { Card, CardBody } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { SearchInput } from "../../components/ui/SearchInput";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SURFACE_KIND_FILTERS, type SurfaceKindFilter } from "../../api/search";

import { ResultCard } from "./ResultCard";
import { useSearch } from "./useSearch";

const KIND_LABEL_KEY = {
  all: strings.pages.search.kind.all,
  component: strings.pages.search.kind.component,
  page: strings.pages.search.kind.page,
  feature: strings.pages.search.kind.feature,
  hook: strings.pages.search.kind.hook,
  layout: strings.pages.search.kind.layout,
  other: strings.pages.search.kind.other,
} as const satisfies Record<SurfaceKindFilter, string>;

export function SearchPage() {
  const { t } = useTranslation();
  const {
    query,
    setQuery,
    effectiveQuery,
    kind,
    setKind,
    clear,
    filteredHits,
    countByKind,
    isShortQuery,
    query_,
  } = useSearch();

  const showResults = query_.isSuccess && effectiveQuery.length > 0;
  const noHits = showResults && query_.data.hits.length === 0;
  const noHitsForFilter = showResults && !noHits && filteredHits.length === 0;
  const isLoading = query_.isFetching && effectiveQuery.length > 0;

  return (
    <section
      data-testid={selectors.pages.search}
      aria-labelledby="search-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="search-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.search.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.search.description)}
        </p>
      </header>

      <Card>
        <CardBody className="flex flex-col gap-4">
          <SearchInput
            value={query}
            onChange={setQuery}
            onClear={clear}
            ariaLabel={t(strings.pages.search.input.label)}
            clearLabel={t(strings.pages.search.input.clear)}
            placeholder={t(strings.pages.search.placeholder)}
            data-testid={selectors.search.input}
          />

          <fieldset
            className="flex flex-wrap items-center gap-2"
            data-testid={selectors.search.filters}
          >
            <legend className="sr-only">{t(strings.pages.search.filters.heading)}</legend>
            {SURFACE_KIND_FILTERS.map((k) => {
              const active = kind === k;
              const count = countByKind[k];
              return (
                <button
                  key={k}
                  type="button"
                  onClick={() => setKind(k)}
                  aria-pressed={active}
                  data-testid={selectors.search.kindFilter({ kind: k })}
                  className={
                    active
                      ? "rounded-pill bg-app-primary px-3 py-1 text-xs font-medium text-app-primary-foreground min-h-touch md:min-h-0"
                      : "rounded-pill border border-app-border bg-app-surface px-3 py-1 text-xs font-medium text-app-foreground hover:bg-app-surface-muted min-h-touch md:min-h-0"
                  }
                >
                  <span>{t(KIND_LABEL_KEY[k])}</span>
                  {showResults ? (
                    <span className="ml-2 tabular-nums text-app-muted-foreground">
                      {count}
                    </span>
                  ) : null}
                </button>
              );
            })}
          </fieldset>
        </CardBody>
      </Card>

      {isLoading ? (
        <p
          className="flex items-center gap-2 text-sm text-app-muted-foreground"
          role="status"
          aria-live="polite"
          data-testid={selectors.search.loading}
        >
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.search.loading)}
        </p>
      ) : null}

      {query_.error ? (
        <div
          role="alert"
          data-testid={selectors.search.error}
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          {t(strings.pages.search.error, {
            message:
              query_.error instanceof Error ? query_.error.message : String(query_.error),
          })}
        </div>
      ) : null}

      {/* Empty pre-query state */}
      {effectiveQuery.length === 0 && !isShortQuery ? (
        <EmptyState
          icon={SearchIcon}
          title={t(strings.pages.search.empty.title)}
          description={t(strings.pages.search.empty.description)}
          data-testid={selectors.search.empty}
        />
      ) : null}

      {/* Short query hint */}
      {isShortQuery ? (
        <p
          className="text-sm text-app-muted-foreground"
          data-testid={selectors.search.shortQuery}
          role="status"
        >
          {t(strings.pages.search.shortQuery)}
        </p>
      ) : null}

      {/* No hits returned */}
      {noHits ? (
        <EmptyState
          icon={SearchX}
          title={t(strings.pages.search.noResults.title, { query: effectiveQuery })}
          description={t(strings.pages.search.noResults.description)}
          data-testid={selectors.search.noResults}
        />
      ) : null}

      {/* Hits returned but filter narrows to zero */}
      {noHitsForFilter ? (
        <EmptyState
          icon={AlertTriangle}
          title={t(strings.pages.search.noResultsForFilter)}
          data-testid={selectors.search.noResultsForFilter}
        />
      ) : null}

      {showResults && filteredHits.length > 0 ? (
        <div className="flex flex-col gap-3">
          <p
            className="text-sm text-app-muted-foreground"
            data-testid={selectors.search.resultsSummary}
            aria-live="polite"
          >
            {t(strings.pages.search.results.summary, {
              count: filteredHits.length,
              query: effectiveQuery,
            })}
          </p>
          <ol
            className="flex flex-col gap-3"
            data-testid={selectors.search.resultsList}
            aria-label={t(strings.pages.search.results.listLabel)}
          >
            {filteredHits.map((hit, idx) => (
              <li key={`${hit.scenario}-${hit.slot}-${idx}`}>
                <ResultCard hit={hit} index={idx} query={effectiveQuery} />
              </li>
            ))}
          </ol>
        </div>
      ) : null}
    </section>
  );
}
