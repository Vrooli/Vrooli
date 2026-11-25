import type {
  CorrectionCandidate,
  ProviderResultGroup,
  QueryPartEvidence,
  QueryResponse,
  SearchHit,
  SearchHitReference,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import {
  CorrectionCandidate_StrengthBand,
  QueryPartEvidence_Source,
  QueryPartEvidence_Status,
  SearchHitReference_RetrievalSource,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import type { TFunction } from "i18next";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const visibleHitBudget = 4;

/** SearchResults keeps human views evidence-selected while the parent owns machine serialization. */
export function SearchResults({ data, query = "" }: { data: QueryResponse; query?: string }) {
  const { t } = useTranslation();
  const queryParts = data.queryParts ?? [];
  const correctionCandidates = data.correctionCandidates ?? [];
  const taskView = queryParts.length > 0;
  const correctionView = !taskView && correctionCandidates.length > 0;
  const directHit = !taskView && !correctionView ? exactEntityMatch(data, query) : undefined;
  const rankedView = !taskView && !correctionView && !directHit && data.reranked && data.ranked.length > 0;
  const heading = taskView
    ? t(strings.search.taskHeading)
    : correctionView
      ? t(strings.search.correctionHeading)
      : directHit
        ? t(strings.search.directHeading)
        : rankedView
          ? t(strings.search.rankedHeading)
          : t(strings.search.groupedHeading);

  return (
    <div data-testid={selectors.search.results} className="flex flex-col gap-4">
      <h3 className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground">{heading}</h3>
      <ResponseCoverage data={data} />
      {taskView || correctionView ? (
        <TaskEvidence data={data} corrections={correctionView} />
      ) : directHit ? (
        <LimitedHitList hits={directEntityHits(data, directHit)} groups={data.groups} />
      ) : rankedView ? (
        <LimitedHitList hits={data.ranked} groups={data.groups} />
      ) : (
        <GroupedResults groups={data.groups} />
      )}
    </div>
  );
}

function ResponseCoverage({ data }: { data: QueryResponse }) {
  const { t } = useTranslation();
  const degradations = data.degradations ?? [];
  if (!data.degraded && !data.partial && degradations.length === 0) return null;
  return (
    <aside role="status" className="rounded border border-app-warning/40 bg-app-surface-muted p-2 text-sm">
      {data.degraded || data.partial ? (
        <p className="font-medium text-app-warning">
          {data.partial ? t(strings.search.responsePartial, { pending: data.pendingProviders }) : t(strings.search.degraded)}
        </p>
      ) : null}
      {degradations.slice(0, 3).map((item, index) => (
        <p key={`${item.component}-${item.providerId}-${index}`} className="mt-1 text-xs text-app-muted-foreground">
          {[item.component, item.providerId, item.reason, item.detail].filter(Boolean).join(": ")}
          {item.fallbackActive ? " (fallback active)" : ""}
        </p>
      ))}
    </aside>
  );
}

function GroupedResults({ groups }: { groups: ProviderResultGroup[] }) {
  const plan = limitedGroupedHits(groups);
  return (
    <div className="flex flex-col gap-4">
      {groups.map((group) => (
        <GroupBlock key={group.providerId} group={group} hits={plan.byProvider.get(group.providerId) ?? []} />
      ))}
      {plan.omitted > 0 ? <OmittedHits count={plan.omitted} /> : null}
    </div>
  );
}

function GroupBlock({ group, hits }: { group: ProviderResultGroup; hits: SearchHit[] }) {
  const { t } = useTranslation();
  const degradations = group.degradations ?? [];
  const hasDegradation = group.degraded || degradations.length > 0;
  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-3">
      <header className="mb-2 flex items-center gap-2">
        <span className="font-medium">{group.providerId}</span>
        {hasDegradation ? (
          <span className="rounded-full border border-app-destructive/40 px-2 py-0.5 text-xs text-app-destructive">
            {t(strings.search.degraded)} ({group.count})
          </span>
        ) : (
          <span className="text-xs text-app-muted-foreground">({group.count})</span>
        )}
      </header>
      {hasDegradation && group.note ? <p className="mb-2 text-xs text-app-destructive">{group.note}</p> : null}
      {degradations.slice(0, 2).map((item, index) => (
        <p key={`${item.leg}-${index}`} className="mb-2 text-xs text-app-warning">
          {[item.leg, item.reason, item.detail].filter(Boolean).join(": ")}
        </p>
      ))}
      {hits.length === 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.search.groupEmpty)}</p>
      ) : (
        <>
          {hits.every((hit) => hit.confidence?.weak) ? (
            <p className="mb-2 text-xs font-medium text-app-muted-foreground">{t(strings.search.noConfidentMatch)}</p>
          ) : null}
          <HitList hits={hits} groups={[group]} />
        </>
      )}
    </section>
  );
}

function LimitedHitList({ hits, groups }: { hits: SearchHit[]; groups: ProviderResultGroup[] }) {
  const limited = uniqueLimited(hits, visibleHitBudget);
  return (
    <>
      <HitList hits={limited.shown} groups={groups} />
      {limited.omitted > 0 ? <OmittedHits count={limited.omitted} /> : null}
    </>
  );
}

function HitList({ hits, groups }: { hits: SearchHit[]; groups: ProviderResultGroup[] }) {
  return (
    <ol className="flex flex-col gap-2">
      {hits.map((hit, index) => (
        <li key={`${hit.providerId}-${hit.id}-${index}`}>
          <HitRow hit={hit} groups={groups} />
        </li>
      ))}
    </ol>
  );
}

function TaskEvidence({ data, corrections }: { data: QueryResponse; corrections: boolean }) {
  const { t } = useTranslation();
  const plan = taskEvidencePlan(data);
  return (
    <div className="flex flex-col gap-3">
      {(data.queryParts ?? []).map((part, index) => {
        const evidence = plan.parts[index];
        return (
          <section key={part.partId || index} className="rounded-panel border border-app-border p-3">
            <h4 className="font-medium">{part.label || part.partId || t(strings.search.partStatusUnknown)}</h4>
            <p className="mt-1 text-sm text-app-muted-foreground">{partStatus(part, data.groups, t)}</p>
            {evidence?.hits.map(({ hit, alsoFoundForFullQuery, unknownRetrievalSources }, hitIndex) => (
              <div key={`${hit.providerId}-${hit.id}-${hitIndex}`}>
                <HitRow hit={hit} groups={data.groups} compact />
                {alsoFoundForFullQuery ? <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.search.alsoFoundForFullQuery)}</p> : null}
                {unknownRetrievalSources.map((source) => (
                  <p key={source} className="mt-1 text-xs text-app-warning">
                    {t(strings.search.unknownRetrievalSource, { source })}
                  </p>
                ))}
              </div>
            ))}
            {evidence?.duplicate ? (
              <p className="mt-2 text-xs text-app-muted-foreground">
                {t(strings.search.duplicateEvidence)}
                {evidence.duplicateAlsoFoundForFullQuery ? ` ${t(strings.search.duplicateAlsoFoundForFullQuery)}` : ""}
              </p>
            ) : null}
            {evidence?.duplicateUnknownRetrievalSources.map((source) => (
              <p key={source} className="mt-1 text-xs text-app-warning">
                {t(strings.search.unknownRetrievalSource, { source })}
              </p>
            ))}
            {evidence?.omitted ? <p data-omitted-count={evidence.omitted} className="mt-2 text-xs text-app-muted-foreground">{t(strings.search.evidenceOmitted, { count: evidence.omitted })}</p> : null}
            {evidence?.missingReference ? <p className="mt-2 text-xs text-app-warning">{t(strings.search.supportUnavailable)}</p> : null}
          </section>
        );
      })}
      {plan.corrections.length > 0 ? (
        <section aria-label={t(strings.search.possibleCorrections)} className="rounded-panel border border-app-border p-3">
          <h4 className="mb-2 font-medium">{t(strings.search.possibleCorrections)}</h4>
          <ul className="flex flex-col gap-2">
            {plan.corrections.map(({ candidate, hit, duplicate }, index) => (
              <li key={`${candidate.submittedValue}-${candidate.candidateValue}-${index}`} className="text-sm">
                <span>{candidate.submittedValue} → {candidate.candidateValue}</span>
                <span className="ms-2 text-xs text-app-warning">
                  {t(strings.search.unverified)} · {correctionStrength(candidate)}
                </span>
                {duplicate ? (
                  <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.search.duplicateEvidence)}</p>
                ) : (
                  <p className="mt-1 text-xs text-app-muted-foreground">{hit.title || hit.id} · {hit.providerId}</p>
                )}
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {plan.omittedCorrections > 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.search.omittedCorrections, { count: plan.omittedCorrections })}</p>
      ) : null}
      {plan.omittedHits > 0 ? <OmittedHits count={plan.omittedHits} /> : null}
      {corrections && plan.corrections.length === 0 && plan.omittedCorrections === 0 ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.search.noGroundedCorrections)}</p>
      ) : null}
    </div>
  );
}

function HitRow({ hit, groups, compact = false }: { hit: SearchHit; groups: ProviderResultGroup[]; compact?: boolean }) {
  const { t } = useTranslation();
  const title = hit.title.trim() || hit.id;
  const locations = hit.locations.map((location) => location.trim()).filter(Boolean);
  const provenancePath = locations[0] ?? (hit.path.trim() || hit.id);
  const confidenceText = confidenceLabel(hit);
  const group = groups.find((item) => item.providerId === hit.providerId);
  const groupDegraded = group && (group.degraded || (group.degradations ?? []).length > 0);

  return (
    <div className={`rounded-md border border-app-border bg-app-surface-muted ${compact ? "mt-2 p-2" : "p-2"}`}>
      <div className="flex items-baseline justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="font-medium">{title}</span>
          {hit.confidence?.weak ? (
            <span className="rounded-full border border-app-warning/40 px-2 py-0.5 text-xs text-app-warning">{t(strings.search.weak)}</span>
          ) : null}
        </div>
        <span className="shrink-0 text-xs text-app-muted-foreground">{confidenceText}</span>
      </div>
      {hit.snippet.trim() ? <p className="mt-1 line-clamp-2 text-sm text-app-muted-foreground">{hit.snippet}</p> : null}
      <p className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.search.provenance, { group: hit.providerGroup || hit.providerId, path: provenancePath })}
      </p>
      {groupDegraded ? <p className="mt-1 text-xs text-app-warning">{t(strings.search.providerCoverageDegraded, { note: group.note || t(strings.search.notReported) })}</p> : null}
      {locations.length > 0 ? (
        <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.search.locations, { value: locationSummary(locations) })}</p>
      ) : null}
    </div>
  );

  function confidenceLabel(value: SearchHit) {
    const confidence = value.confidence;
    if (!confidence) return t(strings.search.confidenceUnknown);
    const regime = confidence.regime.trim();
    return t(confidence.weak ? strings.search.confidenceWeak : strings.search.confidenceStrong, { regime: regime ? `/${regime}` : "" });
  }
}

function taskEvidencePlan(data: QueryResponse) {
  const seen = new Map<string, "shown" | "omitted">();
  const omittedHitKeys = new Set<string>();
  let shown = 0;
  const add = (hit: SearchHit): "shown" | "duplicate" | "omitted" => {
    const key = hitKey(hit);
    const previous = seen.get(key);
    if (previous === "shown") return "duplicate";
    if (previous === "omitted") return "omitted";
    if (shown >= visibleHitBudget) {
      seen.set(key, "omitted");
      omittedHitKeys.add(key);
      return "omitted";
    }
    seen.set(key, "shown");
    shown++;
    return "shown";
  };

  const parts = (data.queryParts ?? []).map((part) => {
    const hits: { hit: SearchHit; alsoFoundForFullQuery: boolean; unknownRetrievalSources: number[] }[] = [];
    const omittedKeys = new Set<string>();
    let duplicate = false;
    let duplicateAlsoFoundForFullQuery = false;
    const duplicateUnknownRetrievalSources = new Set<number>();
    let resolved = 0;
    for (const reference of part.evidence) {
      const hit = resolveReference(data, reference);
      if (!hit) continue;
      resolved++;
      const result = add(hit);
      const alsoFoundForFullQuery = part.source === QueryPartEvidence_Source.DERIVED_SEGMENT
        && reference.retrievalSources.includes(SearchHitReference_RetrievalSource.USER_QUERY);
      const unknownRetrievalSources = [...new Set(reference.retrievalSources.filter((source) => ![
        SearchHitReference_RetrievalSource.UNSPECIFIED,
        SearchHitReference_RetrievalSource.USER_QUERY,
        SearchHitReference_RetrievalSource.DERIVED_SEGMENT,
      ].includes(source)))].sort((a, b) => a - b);
      if (result === "shown") hits.push({ hit, alsoFoundForFullQuery, unknownRetrievalSources });
      else if (result === "duplicate") {
        duplicate = true;
        duplicateAlsoFoundForFullQuery ||= alsoFoundForFullQuery;
        unknownRetrievalSources.forEach((source) => duplicateUnknownRetrievalSources.add(source));
      }
      else omittedKeys.add(hitKey(hit));
    }
    return {
      hits,
      duplicate,
      duplicateAlsoFoundForFullQuery,
      duplicateUnknownRetrievalSources: [...duplicateUnknownRetrievalSources].sort((a, b) => a - b),
      omitted: omittedKeys.size,
      missingReference: resolved === 0 && part.status === QueryPartEvidence_Status.CANDIDATES_RETURNED,
    };
  });
  const corrections: { candidate: CorrectionCandidate; hit: SearchHit; duplicate: boolean }[] = [];
  let omittedCorrections = 0;
  let correctionRows = 0;
  for (const candidate of data.correctionCandidates ?? []) {
    const hit = candidate.evidence ? resolveReference(data, candidate.evidence) : undefined;
    if (!hit || !candidate.submittedValue.trim() || !candidate.candidateValue.trim()) {
      omittedCorrections++;
      continue;
    }
    if (correctionRows >= visibleHitBudget) {
      omittedCorrections++;
      continue;
    }
    const result = add(hit);
    if (result === "omitted") {
      omittedCorrections++;
      continue;
    }
    correctionRows++;
    corrections.push({ candidate, hit, duplicate: result === "duplicate" });
  }
  return { corrections, parts, omittedCorrections, omittedHits: omittedHitKeys.size };
}

function resolveReference(data: QueryResponse, ref: SearchHitReference): SearchHit | undefined {
  let hit: SearchHit | undefined;
  if (ref.rankedIndex !== undefined) hit = data.ranked[ref.rankedIndex];
  else if (ref.groupIndex !== undefined && ref.hitIndex !== undefined) hit = data.groups[ref.groupIndex]?.hits[ref.hitIndex];
  if (!hit) return undefined;
  if (ref.hitId && ref.hitId !== hit.id) return undefined;
  if (ref.providerId && ref.providerId !== hit.providerId) return undefined;
  if (ref.type && ref.type !== hit.type) return undefined;
  if (ref.path && ref.path !== hit.path) return undefined;
  return hit;
}

function exactEntityMatch(data: QueryResponse, query: string): SearchHit | undefined {
  const exact = query.trim().replace(/^["'`]|["'`]$/g, "").toLocaleLowerCase();
  if (!exact) return undefined;
  const hits = data.reranked && data.ranked.length > 0 ? data.ranked : data.groups.flatMap((group) => group.hits);
  return hits.find((hit) => [hit.id, hit.title, hit.path].some((value) => value.trim().toLocaleLowerCase() === exact));
}

function directEntityHits(data: QueryResponse, exact: SearchHit) {
  const hits = data.reranked && data.ranked.length > 0 ? data.ranked : data.groups.flatMap((group) => group.hits);
  const seen = new Set<string>();
  const ordered: SearchHit[] = [];
  for (const hit of [exact, ...hits]) {
    const key = hitKey(hit);
    if (seen.has(key)) continue;
    seen.add(key);
    ordered.push(hit);
  }
  return ordered;
}

function limitedGroupedHits(groups: ProviderResultGroup[]) {
  const seen = new Set<string>();
  const byProvider = new Map<string, SearchHit[]>();
  let shown = 0;
  let total = 0;
  for (const group of groups) {
    const selected: SearchHit[] = [];
    for (const hit of group.hits) {
      const key = hitKey(hit);
      if (seen.has(key)) continue;
      seen.add(key);
      total++;
      if (shown < visibleHitBudget) {
        selected.push(hit);
        shown++;
      }
    }
    byProvider.set(group.providerId, selected);
  }
  return { byProvider, omitted: Math.max(total - shown, 0) };
}

function uniqueLimited(hits: SearchHit[], limit: number) {
  const unique = new Map<string, SearchHit>();
  for (const hit of hits) unique.set(hitKey(hit), hit);
  const all = [...unique.values()];
  return { shown: all.slice(0, limit), omitted: Math.max(all.length - limit, 0) };
}

function hitKey(hit: SearchHit) {
  const source = hit.path.trim().split("#", 1)[0];
  if (source) return `path:${source}`;
  if (hit.id) return `id:${hit.providerId}:${hit.type}:${hit.id}`;
  return `fallback:${hit.providerId}:${hit.title}:${hit.snippet}`;
}

function partStatus(part: QueryPartEvidence, groups: ProviderResultGroup[], t: TFunction) {
  const labels: Record<number, string> = {
    [QueryPartEvidence_Status.CANDIDATES_RETURNED]: t(strings.search.partCandidates),
    [QueryPartEvidence_Status.SEARCHED_EMPTY]: t(strings.search.partEmpty),
    [QueryPartEvidence_Status.PROVIDER_DEGRADED]: t(strings.search.partDegraded),
    [QueryPartEvidence_Status.NOT_ROUTED]: t(strings.search.partNotRouted),
  };
  const knownStatus = labels[part.status];
  const statusLabel = knownStatus
    || (part.status === QueryPartEvidence_Status.UNSPECIFIED
      ? t(strings.search.partStatusUnknown)
      : t(strings.search.partStatusUnknownValue, { status: part.status }));
  const details = [statusLabel];
  if (part.source === QueryPartEvidence_Source.DERIVED_SEGMENT) {
    details.push(t(strings.search.derivedSegment));
  } else if (part.source !== QueryPartEvidence_Source.USER_QUERY && part.source !== QueryPartEvidence_Source.UNSPECIFIED) {
    details.push(t(strings.search.partSourceUnknownValue, { source: part.source }));
  }
  if (part.degraded && part.status !== QueryPartEvidence_Status.PROVIDER_DEGRADED) details.push(t(strings.search.partCoverageDegraded));
  if (part.partial) details.push(t(strings.search.partCoveragePartial));
  const searchedProviderIds = part.searchedProviderIds ?? [];
  if (searchedProviderIds.length > 0) details.push(t(strings.search.searchedProviders, { providers: searchedProviderIds.slice(0, 3).join(", ") }));
  const searched = new Set(searchedProviderIds);
  const degradedProviders = groups
    .filter((group) => searched.has(group.providerId) && (group.degraded || (group.degradations ?? []).length > 0))
    .slice(0, 2)
    .map((group) => `${group.providerId}: ${group.note || t(strings.search.notReported)}`);
  if (degradedProviders.length > 0) details.push(`${t(strings.search.partCoverageDegraded)} (${degradedProviders.join("; ")})`);
  const reason = partReason(part.reason, t);
  if (reason) details.push(reason);
  return details.join(" · ");
}

function partReason(reason: string, t: TFunction) {
  switch (reason) {
    case "ambiguous_provider_evidence": return t(strings.search.reasonAmbiguousOwners);
    case "no_positive_provider_evidence": return t(strings.search.reasonNoPositive);
    case "ambiguous_lexical_boundary": return t(strings.search.reasonAmbiguousBoundary);
    default: return reason;
  }
}

function correctionStrength(candidate: CorrectionCandidate) {
  switch (candidate.strength) {
    case CorrectionCandidate_StrengthBand.STRONG: return "strong candidate";
    case CorrectionCandidate_StrengthBand.POSSIBLE: return "possible candidate";
    default: return "strength unreported";
  }
}

function OmittedHits({ count }: { count: number }) {
  const { t } = useTranslation();
  return <p data-omitted-count={count} className="text-xs text-app-muted-foreground">{t(strings.search.omittedHits, { count })}</p>;
}

function locationSummary(locations: string[]) {
  if (locations.length <= 2) return locations.join(", ");
  return `${locations.slice(0, 2).join(", ")} (+${locations.length - 2} more)`;
}
