import type {
  QueryResponse,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import type {
  SearchHit,
  ProviderResultGroup,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * SearchResults renders a completed QueryResponse. When the reranker produced a
 * unified cross-provider ordering (`reranked` + a non-empty `ranked` list) that
 * is the primary view; otherwise the honest by-provider grouping is shown. Every
 * hit carries its provenance (provider group + path) so a result is traceable to
 * where it came from — the operator-friendliness requirement, mirrored from the
 * CLI.
 */
export function SearchResults({ data }: { data: QueryResponse }) {
  const { t } = useTranslation();
  const unified = data.reranked && data.ranked.length > 0;

  return (
    <div data-testid={selectors.search.results} className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground">
        {unified ? t(strings.search.rankedHeading) : t(strings.search.groupedHeading)}
      </h3>

      {unified ? (
        <ol className="flex flex-col gap-2">
          {data.ranked.map((hit, i) => (
            <li key={`${hit.providerId}-${hit.id}-${i}`}>
              <HitRow hit={hit} reranked />
            </li>
          ))}
        </ol>
      ) : (
        <div className="flex flex-col gap-4">
          {data.groups.map((group) => (
            <GroupBlock key={group.providerId} group={group} />
          ))}
        </div>
      )}
    </div>
  );
}

function GroupBlock({ group }: { group: ProviderResultGroup }) {
  const { t } = useTranslation();
  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-3">
      <header className="mb-2 flex items-center gap-2">
        <span className="font-medium">{group.providerId}</span>
        {group.degraded ? (
          <span className="rounded-full border border-app-destructive/40 px-2 py-0.5 text-xs text-app-destructive">
            {t(strings.search.degraded)}
          </span>
        ) : (
          <span className="text-xs text-app-muted-foreground">({group.count})</span>
        )}
      </header>
      {group.degraded ? (
        <p className="text-xs text-app-destructive">{group.note}</p>
      ) : group.hits.length === 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.search.groupEmpty)}</p>
      ) : (
        <ol className="flex flex-col gap-2">
          {group.hits.map((hit, i) => (
            <li key={`${hit.id}-${i}`}>
              <HitRow hit={hit} reranked={false} />
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function HitRow({ hit, reranked }: { hit: SearchHit; reranked: boolean }) {
  const { t } = useTranslation();
  const title = hit.title.trim() || hit.id;
  const provenancePath = hit.path.trim() || hit.id;
  const scoreText = reranked
    ? t(strings.search.rerank, { value: hit.rerankScore.toFixed(3) })
    : t(strings.search.score, { value: hit.score.toFixed(3) });

  return (
    <div className="rounded-md border border-app-border bg-app-surface-muted p-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-medium">{title}</span>
        <span className="shrink-0 text-xs text-app-muted-foreground">{scoreText}</span>
      </div>
      {hit.snippet.trim() ? (
        <p className="mt-1 line-clamp-2 text-sm text-app-muted-foreground">{hit.snippet}</p>
      ) : null}
      <p className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.search.provenance, { group: hit.providerGroup || hit.providerId, path: provenancePath })}
      </p>
    </div>
  );
}
