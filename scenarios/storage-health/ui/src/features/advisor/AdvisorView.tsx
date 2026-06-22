import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  storageClient,
  type AdviseEnginesResponse,
  type AnalyzeMigrationsResponse,
  type EngineCandidate,
  type MigrationHygiene,
} from "../../api/storage";
import { FitnessMeter } from "../storage/format";

type Tab = "engines" | "migrations";

/**
 * Migration advisor workflow. Two tabs over AdvisorService:
 *   - Engine fitness   (AdviseEngines)    — ranked Postgres→SQLite candidates
 *   - Migration hygiene (AnalyzeMigrations) — scenarios carrying migration debt
 */
export function AdvisorView() {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>("engines");

  return (
    <section
      data-testid={selectors.pages.advisor}
      aria-labelledby="advisor-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-1">
        <h2 id="advisor-heading" className="text-2xl font-semibold">
          {t(strings.advisor.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.advisor.description)}</p>
      </header>

      <div
        data-testid={selectors.advisor.tabs}
        role="tablist"
        aria-label={t(strings.advisor.title)}
        className="flex gap-1"
      >
        {(["engines", "migrations"] as const).map((tk) => (
          <button
            key={tk}
            type="button"
            role="tab"
            aria-selected={tab === tk}
            data-testid={selectors.advisor.tab({ tab: tk })}
            onClick={() => setTab(tk)}
            className={
              tab === tk
                ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                : "rounded-control border border-app-border px-3 py-1 text-sm hover:bg-app-surface-muted"
            }
          >
            {tk === "engines" ? t(strings.advisor.tab.engines) : t(strings.advisor.tab.migrations)}
          </button>
        ))}
      </div>

      {tab === "engines" ? <EngineFitnessPanel /> : <MigrationHygienePanel />}
    </section>
  );
}

function EngineFitnessPanel() {
  const { t } = useTranslation();
  const query = useQuery<AdviseEnginesResponse>({
    queryKey: ["advise-engines"],
    queryFn: () => storageClient.adviseEngines({}),
  });

  const candidates = [...(query.data?.candidates ?? [])].sort(
    (a, b) => b.fitnessScore - a.fitnessScore,
  );

  return (
    <div
      data-testid={selectors.advisor.enginesPanel}
      role="tabpanel"
      className="flex flex-col gap-3"
    >
      {query.isLoading && (
        <LoadingState
          testId={selectors.advisor.loading}
          title={t(strings.advisor.loadingTitle)}
          skeleton={<Skeleton className="h-28 w-full" />}
        />
      )}
      {query.error && (
        <ErrorState
          testId={selectors.advisor.error}
          title={t(strings.advisor.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}
      {query.data && !query.error && candidates.length === 0 && (
        <EmptyState
          testId={selectors.advisor.enginesEmpty}
          title={t(strings.advisor.engines.empty.title)}
          message={t(strings.advisor.engines.empty.message)}
        />
      )}
      {query.data &&
        candidates.map((c) => <CandidateCard key={c.scenario} candidate={c} />)}
    </div>
  );
}

function CandidateCard({ candidate }: { candidate: EngineCandidate }) {
  const { t } = useTranslation();
  return (
    <article
      data-testid={selectors.advisor.candidate({ scenario: candidate.scenario })}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-3">
        <Link
          to={`/validate?scenario=${encodeURIComponent(candidate.scenario)}`}
          className="font-medium text-app-primary underline"
        >
          {candidate.scenario}
        </Link>
        <span className="text-sm text-app-muted-foreground">
          <span className="text-xs uppercase">{t(strings.advisor.engines.currentLabel)} </span>
          {candidate.currentEngine || "—"}
          {" → "}
          <span className="text-xs uppercase">{t(strings.advisor.engines.recommendedLabel)} </span>
          {candidate.recommendedEngine || "—"}
        </span>
        <span className="ms-auto">
          <FitnessMeter score={candidate.fitnessScore} label={t(strings.advisor.engines.fitnessLabel)} />
        </span>
      </div>
      {candidate.rationale && (
        <p className="mt-2 text-sm text-app-foreground">
          <span className="font-medium">{t(strings.advisor.engines.rationaleLabel)}: </span>
          {candidate.rationale}
        </p>
      )}
      {candidate.blockers.length > 0 && (
        <div className="mt-2 text-sm">
          <span className="font-medium text-app-danger">
            {t(strings.advisor.engines.blockersLabel)}:
          </span>
          <ul className="ms-4 list-disc text-app-foreground">
            {candidate.blockers.map((b, i) => (
              <li key={i}>{b}</li>
            ))}
          </ul>
        </div>
      )}
    </article>
  );
}

function MigrationHygienePanel() {
  const { t } = useTranslation();
  const query = useQuery<AnalyzeMigrationsResponse>({
    queryKey: ["analyze-migrations"],
    queryFn: () => storageClient.analyzeMigrations({}),
  });

  const data = query.data;
  const withDebt = (data?.entries ?? []).filter((e) => e.migrationDebt > 0 || e.notes.length > 0);

  return (
    <div
      data-testid={selectors.advisor.migrationsPanel}
      role="tabpanel"
      className="flex flex-col gap-3"
    >
      {query.isLoading && (
        <LoadingState
          testId={selectors.advisor.loading}
          title={t(strings.advisor.loadingTitle)}
          skeleton={<Skeleton className="h-28 w-full" />}
        />
      )}
      {query.error && (
        <ErrorState
          testId={selectors.advisor.error}
          title={t(strings.advisor.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}
      {data && !query.error && (
        <>
          <p
            data-testid={selectors.advisor.migrationsSummary}
            className="text-sm text-app-muted-foreground"
          >
            {t(strings.advisor.migrations.summary, {
              withMigrations: data.withMigrationsCount,
              scenarios: data.scenarioCount,
              debt: data.debtCount,
            })}
          </p>
          {withDebt.length === 0 ? (
            <EmptyState
              testId={selectors.advisor.migrationsEmpty}
              title={t(strings.advisor.migrations.empty.title)}
              message={t(strings.advisor.migrations.empty.message)}
            />
          ) : (
            withDebt.map((e) => <MigrationRow key={e.scenario} entry={e} />)
          )}
        </>
      )}
    </div>
  );
}

function MigrationRow({ entry }: { entry: MigrationHygiene }) {
  const { t } = useTranslation();
  return (
    <article
      data-testid={selectors.advisor.migration({ scenario: entry.scenario })}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-3">
        <Link
          to={`/validate?scenario=${encodeURIComponent(entry.scenario)}`}
          className="font-medium text-app-primary underline"
        >
          {entry.scenario}
        </Link>
        <span className="text-sm text-app-muted-foreground">{entry.storageStage}</span>
        <span className="ms-auto text-sm">
          <span className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.advisor.migrations.debtLabel)}{" "}
          </span>
          <span className="font-semibold tabular-nums text-app-foreground">{entry.migrationDebt}</span>
        </span>
      </div>
      {entry.notes.length > 0 && (
        <>
        <span className="mt-2 block text-xs uppercase tracking-wide text-app-muted-foreground">
          {t(strings.advisor.migrations.notesLabel)}
        </span>
        <ul className="ms-4 list-disc text-sm text-app-foreground">
          {entry.notes.map((n, i) => (
            <li key={i}>{n}</li>
          ))}
        </ul>
        </>
      )}
    </article>
  );
}
