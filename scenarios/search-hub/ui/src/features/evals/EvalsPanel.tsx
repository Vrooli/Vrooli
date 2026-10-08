import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import type { EvalRun, EvalSuite, CompareRunsResponse } from "../../api/evals";
import { listSuites, listRuns, compareRuns } from "../../api/evals";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Trend } from "./Trend";

/**
 * EvalsPanel is the search-quality baseline surface: pick a registered suite on
 * the left, see its tagged run history (with a trend) on the right, expand any
 * run to its per-case table, and tick two runs to A/B compare them.
 *
 * It is read-only observability — running a suite is a deliberate (often
 * attended) act owned by the CLI (`search-hub evals run …`). All loading /
 * error / empty states fall out of the react-query lifecycle, mirroring the
 * search panel.
 */
export function EvalsPanel() {
  const { t } = useTranslation();
  const [selectedSuite, setSelectedSuite] = useState<string | null>(null);

  const suitesQuery = useQuery({ queryKey: ["eval-suites"], queryFn: () => listSuites() });

  // Default to the first suite once they load.
  useEffect(() => {
    if (selectedSuite === null && suitesQuery.data && suitesQuery.data.length > 0) {
      setSelectedSuite(suitesQuery.data[0]?.suiteId ?? null);
    }
  }, [selectedSuite, suitesQuery.data]);

  if (suitesQuery.isLoading) {
    return (
      <p data-testid={selectors.evals.loading} className="text-sm text-app-muted-foreground">
        {t(strings.evals.loading)}
      </p>
    );
  }
  if (suitesQuery.isError) {
    return (
      <p data-testid={selectors.evals.error} className="text-sm text-app-destructive">
        {t(strings.evals.error)}
      </p>
    );
  }

  const suites = suitesQuery.data ?? [];

  return (
    <div data-testid={selectors.evals.panel} className="grid grid-cols-1 gap-4 md:grid-cols-[16rem_1fr]">
      <SuiteList suites={suites} selected={selectedSuite} onSelect={setSelectedSuite} />
      {selectedSuite ? (
        <RunHistory key={selectedSuite} suiteId={selectedSuite} />
      ) : (
        <p data-testid={selectors.evals.selectSuite} className="text-sm text-app-muted-foreground">
          {t(strings.evals.selectSuite)}
        </p>
      )}
    </div>
  );
}

function SuiteList({
  suites,
  selected,
  onSelect,
}: {
  suites: readonly EvalSuite[];
  selected: string | null;
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation();
  return (
    <nav
      data-testid={selectors.evals.suiteList}
      aria-label={t(strings.evals.suitesHeading)}
      className="flex flex-col gap-1"
    >
      <p className="px-2 pb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
        {t(strings.evals.suitesHeading)}
      </p>
      {suites.length === 0 ? (
        <p data-testid={selectors.evals.noSuites} className="px-2 text-sm text-app-muted-foreground">
          {t(strings.evals.noSuites)}
        </p>
      ) : (
        suites.map((s) => {
          const active = s.suiteId === selected;
          return (
            <button
              key={s.suiteId}
              type="button"
              data-testid={selectors.evals.suiteItem({ suiteId: s.suiteId })}
              aria-pressed={active}
              onClick={() => onSelect(s.suiteId)}
              className={
                "rounded-control px-3 py-2 text-start text-sm transition-colors " +
                (active
                  ? "bg-app-primary text-app-primary-foreground"
                  : "text-app-foreground hover:bg-app-surface-muted")
              }
            >
              <span className="block font-medium">{s.name || s.suiteId}</span>
              <span className="block text-xs opacity-80">
                {t(strings.evals.suiteMeta, { provider: s.providerId, cases: s.cases.length })}
              </span>
            </button>
          );
        })
      )}
    </nav>
  );
}

function RunHistory({ suiteId }: { suiteId: string }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState<string | null>(null);
  const [selected, setSelected] = useState<readonly string[]>([]);
  const [comparePair, setComparePair] = useState<[string, string] | null>(null);

  const runsQuery = useQuery({ queryKey: ["eval-runs", suiteId], queryFn: () => listRuns(suiteId) });

  const compareQuery = useQuery({
    queryKey: ["eval-compare", comparePair],
    queryFn: () => compareRuns((comparePair as [string, string])[0], (comparePair as [string, string])[1]),
    enabled: comparePair !== null,
  });

  const runs = useMemo(() => runsQuery.data ?? [], [runsQuery.data]);

  const toggleSelect = (runId: string) => {
    setSelected((prev) => {
      if (prev.includes(runId)) return prev.filter((x) => x !== runId);
      // Keep at most two: drop the oldest selection when a third is ticked.
      return prev.length >= 2 ? [prev[1] as string, runId] : [...prev, runId];
    });
  };

  if (runsQuery.isLoading) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.evals.loading)}</p>;
  }
  if (runs.length === 0) {
    return (
      <p data-testid={selectors.evals.noRuns} className="text-sm text-app-muted-foreground">
        {t(strings.evals.noRuns)}
      </p>
    );
  }

  return (
    <section data-testid={selectors.evals.runHistory} className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <h3 className="text-lg font-semibold">{t(strings.evals.trendHeading)}</h3>
        <div data-testid={selectors.evals.trend}>
          <Trend runs={runs} />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">{t(strings.evals.runsHeading)}</h3>
          <Button
            type="button"
            data-testid={selectors.evals.compareButton}
            disabled={selected.length !== 2}
            onClick={() => setComparePair([selected[0] as string, selected[1] as string])}
          >
            {t(strings.evals.compareButton)}
          </Button>
        </div>
        <p className="text-xs text-app-muted-foreground">{t(strings.evals.compareHint)}</p>

        <ul className="flex flex-col gap-2">
          {runs.map((run) => (
            <RunRow
              key={run.runId}
              run={run}
              checked={selected.includes(run.runId)}
              onToggle={() => toggleSelect(run.runId)}
              expanded={expanded === run.runId}
              onExpand={() => setExpanded((cur) => (cur === run.runId ? null : run.runId))}
            />
          ))}
        </ul>
      </div>

      {comparePair && compareQuery.data ? <CompareResult data={compareQuery.data} /> : null}
    </section>
  );
}

function RunRow({
  run,
  checked,
  onToggle,
  expanded,
  onExpand,
}: {
  run: EvalRun;
  checked: boolean;
  onToggle: () => void;
  expanded: boolean;
  onExpand: () => void;
}) {
  const { t } = useTranslation();
  const agg = run.aggregate;
  const tag = run.tag.trim() === "" ? t(strings.evals.untagged) : run.tag;
  return (
    <li
      data-testid={selectors.evals.runRow({ runId: run.runId })}
      className="rounded-control border border-app-border bg-app-surface p-3"
    >
      <div className="flex flex-wrap items-center gap-3 text-sm">
        <input
          type="checkbox"
          aria-label={tag}
          data-testid={selectors.evals.runSelect({ runId: run.runId })}
          checked={checked}
          onChange={onToggle}
        />
        <span className="rounded-full border border-app-border px-2 py-0.5 text-xs">
          {t(strings.evals.tagLabel, { tag })}
        </span>
        <span className="text-app-muted-foreground">{run.createdAt}</span>
        <span className="font-medium">
          {t(strings.evals.runMet, { met: agg?.met ?? 0, cases: agg?.cases ?? 0 })}
        </span>
        <span className="font-mono text-xs text-app-muted-foreground">
          {t(strings.evals.trendStrong)} {(agg?.meanStrongTop1 ?? 0).toFixed(3)} · {t(strings.evals.trendGibberish)}{" "}
          {(agg?.maxGibberishScore ?? 0).toFixed(3)}
        </span>
        {run.config ? (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.evals.runConfig, { reranker: run.config.rerankerLeg || "none" })}
          </span>
        ) : null}
        <button
          type="button"
          onClick={onExpand}
          className="ms-auto text-xs text-app-primary underline"
        >
          {expanded ? t(strings.evals.hideRun) : t(strings.evals.viewRun)}
        </button>
      </div>
      {expanded ? <CaseTable run={run} /> : null}
    </li>
  );
}

function CaseTable({ run }: { run: EvalRun }) {
  const { t } = useTranslation();
  return (
    <table className="mt-3 w-full text-start text-sm">
      <caption className="sr-only">{t(strings.evals.casesHeading)}</caption>
      <thead className="text-xs uppercase text-app-muted-foreground">
        <tr>
          <th className="py-1 text-start">{t(strings.evals.colCase)}</th>
          <th className="py-1 text-start">{t(strings.evals.colOutcome)}</th>
          <th className="py-1 text-end">{t(strings.evals.colTop)}</th>
          <th className="py-1 text-end">{t(strings.evals.colRank)}</th>
        </tr>
      </thead>
      <tbody>
        {run.results.map((cr) => (
          <tr key={cr.caseId} className="border-t border-app-border/60">
            <td className="py-1 font-mono">{cr.caseId}</td>
            <td className="py-1">
              <OutcomeBadge outcome={cr.outcome} />
            </td>
            <td className="py-1 text-end font-mono">{cr.observedTopScore.toFixed(3)}</td>
            <td className="py-1 text-end font-mono">{cr.expectedRank > 0 ? cr.expectedRank : "—"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  const tone =
    outcome === "met"
      ? "border-app-primary/40 text-app-primary"
      : outcome === "below_expectation" || outcome === "unexpected_hit"
        ? "border-app-destructive/40 text-app-destructive"
        : "border-app-border text-app-muted-foreground";
  return <span className={`rounded-full border px-2 py-0.5 text-xs ${tone}`}>{outcome}</span>;
}

function CompareResult({ data }: { data: CompareRunsResponse }) {
  const { t } = useTranslation();
  const tagA = data.runA?.tag || t(strings.evals.untagged);
  const tagB = data.runB?.tag || t(strings.evals.untagged);
  return (
    <section
      data-testid={selectors.evals.compareResult}
      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3"
    >
      <h3 className="text-lg font-semibold">{t(strings.evals.compareHeading)}</h3>
      <p className="text-xs text-app-muted-foreground">
        {t(strings.evals.compareLegend, { a: tagA, b: tagB })}
      </p>
      <table className="w-full text-start text-sm">
        <thead className="text-xs uppercase text-app-muted-foreground">
          <tr>
            <th className="py-1 text-start">{t(strings.evals.colCase)}</th>
            <th className="py-1 text-start">{t(strings.evals.compareColA)}</th>
            <th className="py-1 text-start">{t(strings.evals.compareColB)}</th>
            <th className="py-1 text-end">{t(strings.evals.colDelta)}</th>
          </tr>
        </thead>
        <tbody>
          {data.deltas.map((d) => {
            const delta = d.topScoreB - d.topScoreA;
            const sign = delta > 0 ? "+" : "";
            return (
              <tr key={d.caseId} className="border-t border-app-border/60">
                <td className="py-1 font-mono">{d.caseId}</td>
                <td className="py-1">
                  <OutcomeBadge outcome={d.outcomeA || "—"} />
                </td>
                <td className="py-1">
                  <OutcomeBadge outcome={d.outcomeB || "—"} />
                </td>
                <td
                  className={
                    "py-1 text-end font-mono " +
                    (delta > 0 ? "text-app-primary" : delta < 0 ? "text-app-destructive" : "")
                  }
                >
                  {sign}
                  {delta.toFixed(3)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}
