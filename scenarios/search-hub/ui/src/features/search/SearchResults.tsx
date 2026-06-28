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
              <HitRow hit={hit} />
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
        <div className="flex flex-col gap-2">
          {group.hits.every((hit) => hit.confidence?.weak) ? (
            <p className="text-xs font-medium text-app-muted-foreground">{t(strings.search.noConfidentMatch)}</p>
          ) : null}
          <ol className="flex flex-col gap-2">
            {group.hits.map((hit, i) => (
              <li key={`${hit.id}-${i}`}>
                <HitRow hit={hit} />
              </li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
}

function HitRow({ hit }: { hit: SearchHit }) {
  const { t } = useTranslation();
  const title = hit.title.trim() || hit.id;
  const locations = (hit.locations ?? []).map((location) => location.trim()).filter(Boolean);
  const provenancePath = locations[0] ?? (hit.path.trim() || hit.id);
  const confidenceText = confidenceLabel(hit);

  return (
    <div className="rounded-md border border-app-border bg-app-surface-muted p-2">
      <div className="flex items-baseline justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="font-medium">{title}</span>
          {hit.confidence?.weak ? (
            <span className="rounded-full border border-app-warning/40 px-2 py-0.5 text-xs text-app-warning">
              {t(strings.search.weak)}
            </span>
          ) : null}
        </div>
        <span className="shrink-0 text-xs text-app-muted-foreground">{confidenceText}</span>
      </div>
      {hit.snippet.trim() ? (
        <p className="mt-1 line-clamp-2 text-sm text-app-muted-foreground">{hit.snippet}</p>
      ) : null}
      <p className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.search.provenance, { group: hit.providerGroup || hit.providerId, path: provenancePath })}
      </p>
      {locations.length > 0 ? (
        <p className="mt-1 text-xs text-app-muted-foreground">
          {t(strings.search.locations, { value: locationSummary(locations) })}
        </p>
      ) : null}
    </div>
  );

  function confidenceLabel(hit: SearchHit) {
    const regime = hit.confidence?.regime?.trim();
    if (!hit.confidence) {
      return t(strings.search.confidenceUnknown);
    }
    return t(hit.confidence.weak ? strings.search.confidenceWeak : strings.search.confidenceStrong, {
      regime: regime ? `/${regime}` : "",
    });
  }
}

function locationSummary(locations: string[]) {
  if (locations.length <= 2) {
    return locations.join(", ");
  }
  return `${locations.slice(0, 2).join(", ")} (+${locations.length - 2} more)`;
}
