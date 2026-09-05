import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useSearchParams } from "react-router-dom";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { fetchScore, fetchScores, fetchScoreTrend } from "../../api/scoring";
import { errorMessage } from "../../lib/errorMessage";

import { CompositeCard } from "./CompositeCard";
import { FleetTable } from "./FleetTable";
import { FreshnessCard } from "./FreshnessCard";
import { ImportanceCard } from "./ImportanceCard";
import { MaturityCard } from "./MaturityCard";
import { RecommendationsCard } from "./RecommendationsCard";
import { TrendCard } from "./TrendCard";

/**
 * ScoreDashboard is this scenario's primary surface: pick a scenario,
 * render its full cached status payload (ScoreService.GetScore) —
 * maturity rung "as of digest", composite score, per-phase freshness with
 * the copy-pastable refresh command, recommendations, and the action plan.
 *
 * The selected scenario lives in the `?scenario=` search param so a
 * dashboard view is shareable / refresh-stable; the input is a draft until
 * submitted.
 */
export function ScoreDashboard() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const scenario = searchParams.get("scenario") ?? "";
  const [draft, setDraft] = useState(scenario);
  const [fleetPageToken, setFleetPageToken] = useState("");

  const scoreQuery = useQuery({
    queryKey: ["score", scenario],
    queryFn: () => fetchScore(scenario),
    enabled: scenario !== "",
  });

  const trendQuery = useQuery({
    queryKey: ["scoreTrend", scenario],
    queryFn: () => fetchScoreTrend(scenario),
    enabled: scenario !== "",
  });

  const fleetQuery = useQuery({
    queryKey: ["scoreFleet", fleetPageToken],
    queryFn: () => fetchScores({ pageToken: fleetPageToken, pageSize: 10 }),
  });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const next = draft.trim();
    setSearchParams(next === "" ? {} : { scenario: next });
  };

  const data = scoreQuery.data;
  const fleet = fleetQuery.data;

  return (
    <div data-testid={selectors.scoring.dashboard} className="flex flex-col gap-4">
      <form
        data-testid={selectors.scoring.form}
        onSubmit={handleSubmit}
        className="flex max-w-xl items-end gap-2"
      >
        <label className="flex-1 text-sm">
          <span className="mb-1 block font-medium">{t(strings.scoring.form.label)}</span>
          <Input
            data-testid={selectors.scoring.input}
            name="scenario"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder={t(strings.scoring.form.placeholder)}
            autoComplete="off"
          />
        </label>
        <Button data-testid={selectors.scoring.submit} type="submit">
          {t(strings.scoring.form.submit)}
        </Button>
      </form>

      {scenario === "" && (
        <p data-testid={selectors.scoring.empty} className="text-app-muted-foreground">
          {t(strings.scoring.empty)}
        </p>
      )}
      {scoreQuery.isLoading && scenario !== "" && (
        <p data-testid={selectors.scoring.loading} className="text-app-muted-foreground">
          {t(strings.scoring.loading)}
        </p>
      )}
      {scoreQuery.error && (
        <p data-testid={selectors.scoring.error} role="alert" className="text-red-600 dark:text-red-400">
          {errorMessage(scoreQuery.error, t)}
        </p>
      )}

      {data && (
        <>
          <div className="flex flex-wrap items-baseline gap-x-3">
            <h3 className="text-xl font-semibold">{data.scenario}</h3>
            <span className="text-sm text-app-muted-foreground">{data.category}</span>
            {data.calculatedAt && (
              <span className="text-xs text-app-muted-foreground">
                {t(strings.scoring.calculatedAtLabel)}{" "}
                {formatDate(timestampDate(data.calculatedAt), {
                  dateStyle: "medium",
                  timeStyle: "short",
                })}
              </span>
            )}
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            {data.composite && <CompositeCard composite={data.composite} />}
            {data.maturity && (
              <MaturityCard
                maturity={data.maturity}
                digest={data.freshness?.currentDigest ?? ""}
                digestError={data.freshness?.digestError ?? ""}
              />
            )}
            {data.freshness && <FreshnessCard freshness={data.freshness} />}
            {data.importance && <ImportanceCard importance={data.importance} />}
            <TrendCard snapshots={trendQuery.data?.snapshots ?? []} trend={data.trend} />
            {data.recommendations.length > 0 && data.composite && (
              <RecommendationsCard
                recommendations={data.recommendations}
                actionPlan={data.actionPlan}
                compositeScore={data.composite.score}
              />
            )}
          </div>
          {data.degradations.length > 0 && (
            <section
              data-testid={selectors.scoring.degradations.card}
              aria-label={t(strings.scoring.degradations.title)}
              className="rounded-panel border border-amber-500/40 bg-amber-500/10 p-4 text-sm"
            >
              <h3 className="font-semibold uppercase text-app-muted-foreground">
                {t(strings.scoring.degradations.title)}
              </h3>
              <ul className="mt-1 space-y-0.5">
                {data.degradations.map((degradation) => (
                  <li key={degradation.collector}>
                    {t(strings.scoring.degradations.line, {
                      collector: degradation.collector,
                      state: degradation.state,
                      reason: degradation.reason,
                    })}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
      <FleetTable
        rows={fleet?.scores ?? []}
        hasNextPage={(fleet?.nextPageToken ?? "") !== ""}
        loading={fleetQuery.isFetching}
        onNextPage={() => {
          const next = fleet?.nextPageToken ?? "";
          if (next !== "") {
            setFleetPageToken(next);
          }
        }}
      />
    </div>
  );
}
