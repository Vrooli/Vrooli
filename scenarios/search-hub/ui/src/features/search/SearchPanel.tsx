import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import type { QueryResponse } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { listActiveProviders, runQuery, type SearchInput } from "../../api/search";
import { FederationStatusBar } from "./FederationStatusBar";
import { SearchResults } from "./SearchResults";

/**
 * SearchPanel is the federated-search surface: a query box, type facets built
 * from the live registry, an "expand search" affordance (fan out to every
 * active provider), and the result area (unified ranked list when the reranker
 * runs, honest by-provider grouping otherwise). It wires to RoutingService.Query
 * + the registry's ListProviders; FederationStatusBar adds live model/provider
 * health above the results.
 *
 * Search is submit-driven: the form composes a SearchInput and a react-query
 * key, so loading/error/empty states fall out of the query lifecycle and
 * repeated identical searches are cached.
 */
export function SearchPanel() {
  const { t } = useTranslation();

  const [queryText, setQueryText] = useState("");
  const [selectedTypes, setSelectedTypes] = useState<readonly string[]>([]);
  const [expandAll, setExpandAll] = useState(false);
  const [submitted, setSubmitted] = useState<SearchInput | null>(null);

  const providersQuery = useQuery({
    queryKey: ["active-providers"],
    queryFn: listActiveProviders,
  });

  const availableTypes = useMemo(() => {
    const set = new Set<string>();
    for (const p of providersQuery.data ?? []) {
      if (p.type) set.add(p.type);
    }
    return [...set].sort();
  }, [providersQuery.data]);

  const searchQuery = useQuery({
    queryKey: ["search", submitted],
    // `enabled` guarantees submitted is non-null when this runs; the cast keeps
    // the type checker happy without a banned non-null assertion.
    queryFn: () => runQuery(submitted as SearchInput),
    enabled: submitted !== null && submitted.query.trim() !== "",
  });

  const toggleType = (type: string) => {
    setSelectedTypes((prev) =>
      prev.includes(type) ? prev.filter((x) => x !== type) : [...prev, type],
    );
  };

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = queryText.trim();
    if (!trimmed) return;
    setSubmitted({
      query: trimmed,
      types: expandAll ? [] : [...selectedTypes],
      all: expandAll,
    });
  };

  const data = searchQuery.data;
  const totalHits = data ? data.groups.reduce((sum, g) => sum + g.hits.length, 0) : 0;

  return (
    <div data-testid={selectors.search.panel} className="flex flex-col gap-4">
      <form onSubmit={onSubmit} className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            data-testid={selectors.search.input}
            value={queryText}
            onChange={(e) => setQueryText(e.target.value)}
            placeholder={t(strings.search.queryPlaceholder)}
            aria-label={t(strings.search.queryLabel)}
            className="flex-1"
          />
          <Button type="submit" data-testid={selectors.search.submit}>
            {searchQuery.isFetching ? t(strings.search.searching) : t(strings.search.submit)}
          </Button>
        </div>

        {availableTypes.length > 0 ? (
          <fieldset
            data-testid={selectors.search.typeFacets}
            className="flex flex-wrap items-center gap-2"
            disabled={expandAll}
          >
            <legend className="me-2 text-xs uppercase text-app-muted-foreground">
              {t(strings.search.typesLabel)}
            </legend>
            {availableTypes.map((type) => {
              const active = !expandAll && selectedTypes.includes(type);
              return (
                <button
                  key={type}
                  type="button"
                  data-testid={selectors.search.typeFacet({ type })}
                  aria-pressed={active}
                  onClick={() => toggleType(type)}
                  className={
                    "rounded-full border px-3 py-1 text-sm transition-colors disabled:opacity-50 " +
                    (active
                      ? "border-app-primary bg-app-primary text-app-primary-foreground"
                      : "border-app-border text-app-foreground hover:bg-app-surface-muted")
                  }
                >
                  {type}
                </button>
              );
            })}
          </fieldset>
        ) : null}

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            data-testid={selectors.search.expandToggle}
            checked={expandAll}
            onChange={(e) => setExpandAll(e.target.checked)}
          />
          {t(strings.search.expandLabel)}
        </label>
      </form>

      <FederationStatusBar />

      <ResultsArea
        hasSubmitted={submitted !== null}
        isLoading={searchQuery.isFetching}
        isError={searchQuery.isError}
        data={data}
        totalHits={totalHits}
      />
    </div>
  );
}

function ResultsArea({
  hasSubmitted,
  isLoading,
  isError,
  data,
  totalHits,
}: {
  hasSubmitted: boolean;
  isLoading: boolean;
  isError: boolean;
  data: QueryResponse | undefined;
  totalHits: number;
}) {
  const { t } = useTranslation();

  if (!hasSubmitted) {
    return (
      <p data-testid={selectors.search.empty} className="text-sm text-app-muted-foreground">
        {t(strings.search.emptyState)}
      </p>
    );
  }
  if (isLoading) {
    return (
      <p data-testid={selectors.search.loading} className="text-sm text-app-muted-foreground">
        {t(strings.search.searching)}
      </p>
    );
  }
  if (isError || !data) {
    return (
      <p data-testid={selectors.search.error} className="text-sm text-app-destructive">
        {t(strings.search.error)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div data-testid={selectors.search.summary} className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-app-muted-foreground">
          {t(strings.search.summary, {
            corpora: data.groups.length,
            hits: totalHits,
            latency: Number(data.latencyMs),
          })}
        </span>
        <span className="rounded-full border border-app-border px-2 py-0.5 text-xs text-app-muted-foreground">
          {data.reranked ? t(strings.search.reranked) : t(strings.search.grouped)}
        </span>
        {data.degraded ? (
          <span className="rounded-full border border-app-destructive/40 px-2 py-0.5 text-xs text-app-destructive">
            {t(strings.search.degraded)}
          </span>
        ) : null}
      </div>

      {data.routingExplanation.length > 0 ? (
        <details data-testid={selectors.search.routing} className="text-xs text-app-muted-foreground">
          <summary className="cursor-pointer">{t(strings.search.routingHeading)}</summary>
          <ul className="mt-1 list-disc ps-5">
            {data.routingExplanation.map((line, i) => (
              <li key={i}>{line}</li>
            ))}
          </ul>
        </details>
      ) : null}

      {totalHits === 0 ? (
        <p data-testid={selectors.search.noResults} className="text-sm text-app-muted-foreground">
          {t(strings.search.noResults)}
        </p>
      ) : (
        <SearchResults data={data} />
      )}
    </div>
  );
}
