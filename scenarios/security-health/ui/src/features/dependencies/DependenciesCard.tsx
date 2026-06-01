import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { dependencyClient, Ecosystem, Mode } from "../../api/dependencies";

type EcosystemLabelKey =
  | typeof strings.dependencies.ecosystemAll
  | typeof strings.dependencies.ecosystemGo
  | typeof strings.dependencies.ecosystemNpm;

const ECOSYSTEM_LABEL: Record<Ecosystem, EcosystemLabelKey> = {
  [Ecosystem.UNSPECIFIED]: strings.dependencies.ecosystemAll,
  [Ecosystem.GO]: strings.dependencies.ecosystemGo,
  [Ecosystem.NPM]: strings.dependencies.ecosystemNpm,
};

/** coveragePercent renders indexed/expected as an integer percent (0 when the
 * corpus is empty), matching the CLI's `index: N/M (P%)` line. */
function coveragePercent(indexed: number, expected: number): number {
  if (expected <= 0) return 0;
  return Math.round((indexed / expected) * 100);
}

interface SearchParams {
  query: string;
  ecosystem: Ecosystem;
  vulnerableOnly: boolean;
}

/**
 * DependenciesCard is the fleet dependency-intelligence surface: a free-text
 * query composed with the `ecosystem` / `vulnerableOnly` structured filters,
 * answering "which scenarios are exposed to CVE-X?" in one query. The status
 * strip reports indexed/vulnerable counts and the last reconcile; `mode_used`
 * drives the "(text mode)" hint when the semantic backend is degraded.
 */
export function DependenciesCard() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [ecosystem, setEcosystem] = useState<Ecosystem>(Ecosystem.UNSPECIFIED);
  const [vulnerableOnly, setVulnerableOnly] = useState(false);
  const [params, setParams] = useState<SearchParams | null>(null);

  const statusQuery = useQuery({
    queryKey: ["deps-status"],
    queryFn: () => dependencyClient.status({}),
  });

  const searchQuery = useQuery({
    queryKey: ["deps-search", params],
    queryFn: () =>
      dependencyClient.search({
        query: params?.query ?? "",
        ecosystem: params?.ecosystem ?? Ecosystem.UNSPECIFIED,
        vulnerableOnly: params?.vulnerableOnly ?? false,
        limit: 50,
        mode: Mode.UNSPECIFIED,
      }),
    enabled: params !== null,
  });

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setParams({ query: query.trim(), ecosystem, vulnerableOnly });
  };

  const results = searchQuery.data?.results ?? [];
  const isTextMode = searchQuery.data?.modeUsed === Mode.TEXT;

  return (
    <section
      data-testid={selectors.dependencies.card}
      aria-label={t(strings.dependencies.title)}
      className="rounded-xl border border-app-border bg-app-surface p-4"
    >
      {statusQuery.data && (
        <div data-testid={selectors.dependencies.status} className="mb-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-app-muted-foreground">
          <span>{t(strings.dependencies.indexedCount, { count: statusQuery.data.indexedCount })}</span>
          <span>{t(strings.dependencies.vulnerableCount, { count: statusQuery.data.vulnerableCount })}</span>
          <span
            data-testid={selectors.dependencies.coverage}
            title={t(strings.dependencies.coverageTooltip)}
            className={statusQuery.data.indexReady ? undefined : "text-amber-300/80"}
          >
            {t(
              statusQuery.data.indexReady
                ? strings.dependencies.indexReady
                : strings.dependencies.indexBuilding,
              { percent: coveragePercent(statusQuery.data.indexedVectors, statusQuery.data.expectedVectors) },
            )}
          </span>
          {statusQuery.data.lastReconcileAt && (
            <span>
              {t(strings.dependencies.lastReconcile)}{" "}
              {formatDate(new Date(statusQuery.data.lastReconcileAt), { dateStyle: "medium", timeStyle: "short" })}
            </span>
          )}
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1 text-sm">
          <span className="text-app-muted-foreground">{t(strings.dependencies.queryLabel)}</span>
          <Input
            data-testid={selectors.dependencies.queryInput}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t(strings.dependencies.queryPlaceholder)}
            aria-label={t(strings.dependencies.queryLabel)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-app-muted-foreground">{t(strings.dependencies.ecosystem)}</span>
          <select
            data-testid={selectors.dependencies.ecosystemSelect}
            value={ecosystem}
            onChange={(e) => setEcosystem(Number(e.target.value))}
            aria-label={t(strings.dependencies.ecosystem)}
            className="h-10 rounded-md border border-white/20 bg-white/5 px-2 text-sm text-white"
          >
            {([Ecosystem.UNSPECIFIED, Ecosystem.GO, Ecosystem.NPM] as const).map((eco) => (
              <option key={eco} value={eco} className="text-black">
                {t(ECOSYSTEM_LABEL[eco])}
              </option>
            ))}
          </select>
        </label>
        <label className="flex h-10 items-center gap-2 text-sm">
          <input
            data-testid={selectors.dependencies.vulnerableOnly}
            type="checkbox"
            checked={vulnerableOnly}
            onChange={(e) => setVulnerableOnly(e.target.checked)}
          />
          {t(strings.dependencies.vulnerableOnly)}
        </label>
        <Button data-testid={selectors.dependencies.searchButton} type="submit" disabled={searchQuery.isFetching}>
          {searchQuery.isFetching ? t(strings.dependencies.searching) : t(strings.dependencies.search)}
        </Button>
      </form>

      {isTextMode && (
        <p data-testid={selectors.dependencies.modeHint} className="mt-2 text-xs text-amber-300/80">
          {t(strings.dependencies.textModeHint)}
        </p>
      )}

      {searchQuery.error && (
        <p data-testid={selectors.dependencies.error} className="mt-4 text-red-400">
          {errorMessage(searchQuery.error, t)}
        </p>
      )}

      {searchQuery.data && results.length === 0 && (
        <p data-testid={selectors.dependencies.empty} className="mt-4 text-app-muted-foreground">
          {t(strings.dependencies.empty)}
        </p>
      )}

      {results.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table data-testid={selectors.dependencies.results} className="w-full text-left text-sm">
            <thead className="text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="py-1 pe-3">{t(strings.dependencies.columns.scenario)}</th>
                <th className="py-1 pe-3">{t(strings.dependencies.columns.name)}</th>
                <th className="py-1 pe-3">{t(strings.dependencies.columns.version)}</th>
                <th className="py-1 pe-3">{t(strings.dependencies.columns.ecosystem)}</th>
                <th className="py-1 pe-3">{t(strings.dependencies.columns.vulns)}</th>
              </tr>
            </thead>
            <tbody>
              {results.map((r, i) => {
                const rec = r.record;
                if (!rec) return null;
                const vulnerable = rec.vulnIds.length > 0;
                return (
                  <tr
                    key={`${rec.scenario}:${rec.name}:${rec.version}:${i}`}
                    className={vulnerable ? "border-t border-red-500/20" : "border-t border-app-border"}
                  >
                    <td className="py-1.5 pe-3 font-mono text-xs">{rec.scenario}</td>
                    <td className="py-1.5 pe-3 font-mono text-xs">{rec.name}</td>
                    <td className="py-1.5 pe-3 font-mono text-xs">{rec.version}</td>
                    <td className="py-1.5 pe-3 text-xs">{t(ECOSYSTEM_LABEL[rec.ecosystem])}</td>
                    <td className="py-1.5 pe-3 text-xs">
                      {vulnerable ? (
                        <span className="text-red-300">{rec.vulnIds.join(", ")}</span>
                      ) : (
                        <span className="text-app-muted-foreground">—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
