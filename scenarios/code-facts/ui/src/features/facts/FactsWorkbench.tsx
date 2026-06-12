import { useQuery } from "@tanstack/react-query";

import { describeScenario } from "../../api/facts";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const FACTS_QUERY_KEY = ["code-facts", "describe", "scenario", "code-facts"] as const;

export function FactsWorkbench() {
  const { t } = useTranslation();
  const factsQuery = useQuery({
    queryKey: FACTS_QUERY_KEY,
    queryFn: () => describeScenario("code-facts"),
  });

  return (
    <section
      data-testid={selectors.facts.workbench}
      aria-label={t(strings.facts.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-app-muted-foreground">{t(strings.facts.title)}</h2>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.facts.description)}</p>
        </div>
        <span data-testid={selectors.facts.cache} className="text-xs text-app-muted-foreground">
          {factsQuery.data?.cache?.hit ? t(strings.facts.cacheHit) : t(strings.facts.cacheMiss)}
        </span>
      </div>

      {factsQuery.isLoading && (
        <p data-testid={selectors.facts.loading} className="mt-4 text-sm text-app-muted-foreground">
          {t(strings.facts.loading)}
        </p>
      )}
      {factsQuery.error && (
        <p data-testid={selectors.facts.error} className="mt-4 text-sm text-red-400">
          {errorMessage(factsQuery.error, t)}
        </p>
      )}
      {factsQuery.data && (
        <>
          <div className="mt-4 grid gap-3 md:grid-cols-3">
            <FactStat label={t(strings.facts.surfaces)} value={factsQuery.data.surfaces.length} />
            <FactStat label={t(strings.facts.parseUnits)} value={factsQuery.data.parseUnits.length} />
            <FactStat label={t(strings.facts.warnings)} value={factsQuery.data.warnings.length} />
          </div>
          <dl className="mt-4 grid gap-2 rounded-control border border-app-border bg-app-surface-muted p-3 text-xs md:grid-cols-2">
            <CacheField label={t(strings.facts.cacheState)} value={factsQuery.data.cache?.state || "miss"} />
            <CacheField label={t(strings.facts.cacheKey)} value={factsQuery.data.cache?.cacheKey || ""} />
            <CacheField label={t(strings.facts.sourceHash)} value={factsQuery.data.cache?.sourceHash || ""} />
            <CacheField label={t(strings.facts.configHash)} value={factsQuery.data.cache?.configHash || ""} />
          </dl>
        </>
      )}
      {factsQuery.data && factsQuery.data.evidence.length > 0 && (
        <ul data-testid={selectors.facts.evidenceList} className="mt-4 space-y-2 text-sm text-app-muted-foreground">
          {factsQuery.data.evidence.map((evidence, index) => (
            <li key={`${evidence.analyzer}-${index}`} className="rounded-control border border-app-border p-2">
              <span className="font-medium text-app-foreground">{evidence.status}</span>
              <span> - {evidence.message}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function CacheField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="truncate font-mono text-app-foreground" title={value}>
        {value || "-"}
      </dd>
    </div>
  );
}

function FactStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <p className="text-xs uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}
