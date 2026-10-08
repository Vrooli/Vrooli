import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, RefreshCw, XCircle } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { fleetClient } from "../../api/fleet";
import { severityChipClass } from "./severity";

const FLEET_QUERY_KEY = ["fleet-scan"] as const;

/**
 * FleetView is the structure-health fleet dashboard. It calls
 * `FleetService.ScanFleet` and renders the typed all-kind rollup:
 *
 *   - headline counters (scenarios / passing / missing-freshness / auto-fixable)
 *   - profile distribution mini-list
 *   - per-rule conformance table (server order: most-offending first)
 *   - target offenders table (kind/id, verdict, profile, error/warning/auto-fixable
 *     counts, missing-freshness badge)
 *   - any ungraded targets with their reason
 *
 * Verdicts read strictly off `entry.passed` so the visual semantics match the
 * gating contract (`passed=false` iff a target has any error-severity
 * structure finding).
 */
export function FleetView() {
  const { t } = useTranslation();

  const query = useQuery({
    queryKey: FLEET_QUERY_KEY,
    queryFn: () => fleetClient.scanFleet({}),
  });

  const data = query.data;
  const entries = data?.entries ?? [];
  const rules = data?.ruleConformance ?? [];
  const profiles = data?.profileDistribution ?? [];
  const scanErrors = data?.errors ?? [];

  return (
    <section
      data-testid={selectors.fleet.view}
      aria-label={t(strings.fleet.title)}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.fleet.title)}
        </h3>
        <Button
          data-testid={selectors.fleet.refreshButton}
          variant="outline"
          size="sm"
          className="ms-auto"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
          aria-label={t(strings.fleet.refresh)}
        >
          <RefreshCw
            aria-hidden="true"
            className={["h-4 w-4", query.isFetching ? "animate-spin" : ""].join(" ")}
          />
        </Button>
      </div>

      {query.isLoading && (
        <p data-testid={selectors.fleet.loading} className="text-app-muted-foreground">
          {t(strings.fleet.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.fleet.error} className="text-app-danger">
          {errorMessage(query.error, t)}
        </p>
      )}

      {data && entries.length === 0 && scanErrors.length === 0 && (
        <p data-testid={selectors.fleet.empty} className="text-app-muted-foreground">
          {t(strings.fleet.empty)}
        </p>
      )}

      {data && (entries.length > 0 || scanErrors.length > 0) && (
        <>
          <SummaryStats
            targetCount={data.targetCount || data.scenarioCount}
            passingTargetCount={data.passingTargetCount || data.passingCount}
            autofixableTotal={data.autofixableTotal}
          />
          <ProfileDistributionCard profiles={profiles} />
          <RuleConformanceCard rules={rules} />
          <ScenarioOffendersTable entries={entries} />
          <ScanErrorsCard errors={scanErrors} />
        </>
      )}
    </section>
  );
}

function SummaryStats({
  targetCount,
  passingTargetCount,
  autofixableTotal,
}: {
  targetCount: number;
  passingTargetCount: number;
  autofixableTotal: number;
}) {
  const { t } = useTranslation();
  return (
    <dl
      data-testid={selectors.fleet.summary}
      className="grid grid-cols-2 gap-3 sm:grid-cols-3"
    >
      <Stat
        testId={selectors.fleet.summaryScenarios}
        label={t(strings.fleet.summary.scenarios)}
        value={targetCount}
      />
      <Stat
        testId={selectors.fleet.summaryPassing}
        label={t(strings.fleet.summary.passing)}
        value={passingTargetCount}
      />
      <Stat
        testId={selectors.fleet.summaryAutofixable}
        label={t(strings.fleet.summary.autofixable)}
        value={autofixableTotal}
      />
    </dl>
  );
}

function Stat({
  testId,
  label,
  value,
}: {
  testId: string;
  label: string;
  value: number;
}) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd data-testid={testId} className="mt-2 text-2xl font-semibold tabular-nums">
        {value}
      </dd>
    </div>
  );
}

function ProfileDistributionCard({
  profiles,
}: {
  profiles: { profileId: string; scenarioCount: number; recognized: boolean }[];
}) {
  const { t } = useTranslation();
  if (profiles.length === 0) return null;
  return (
    <section
      data-testid={selectors.fleet.profiles}
      aria-label={t(strings.fleet.profiles.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h4 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.fleet.profiles.title)}
      </h4>
      <ul className="mt-3 flex flex-col gap-2">
        {profiles.map((profile) => (
          <li
            key={profile.profileId}
            data-testid={selectors.fleet.profileRow({ profileId: profile.profileId })}
            className="flex items-center justify-between gap-2 text-sm"
          >
            <span className="flex items-center gap-2">
              <span className="font-medium text-app-foreground">{profile.profileId}</span>
              {!profile.recognized && (
                <span className="rounded-control border border-app-border bg-app-surface-muted px-1.5 py-0.5 text-xs text-app-muted-foreground">
                  {t(strings.fleet.profiles.unrecognized)}
                </span>
              )}
            </span>
            <span className="tabular-nums text-app-muted-foreground">
              {t(strings.fleet.profiles.countLabel, { count: profile.scenarioCount })}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function RuleConformanceCard({
  rules,
}: {
  rules: {
    code: string;
    offendingScenarios: number;
    totalFindings: number;
    autofixable: number;
    worstSeverity: string;
  }[];
}) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.fleet.rules}
      aria-label={t(strings.fleet.rules.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h4 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.fleet.rules.title)}
      </h4>
      {rules.length === 0 ? (
        <p data-testid={selectors.fleet.rulesEmpty} className="mt-3 text-app-muted-foreground">
          {t(strings.fleet.rules.empty)}
        </p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
                <th scope="col" className="px-2 py-1 text-start font-medium">
                  {t(strings.fleet.rules.col.code)}
                </th>
                <th scope="col" className="px-2 py-1 text-start font-medium">
                  {t(strings.fleet.rules.col.severity)}
                </th>
                <th scope="col" className="px-2 py-1 text-end font-medium">
                  {t(strings.fleet.rules.col.offenders)}
                </th>
                <th scope="col" className="px-2 py-1 text-end font-medium">
                  {t(strings.fleet.rules.col.findings)}
                </th>
                <th scope="col" className="px-2 py-1 text-end font-medium">
                  {t(strings.fleet.rules.col.autofixable)}
                </th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr
                  key={rule.code}
                  data-testid={selectors.fleet.ruleRow({ code: rule.code })}
                  className="border-t border-app-border"
                >
                  <td className="px-2 py-1.5 font-medium text-app-foreground">{rule.code}</td>
                  <td className="px-2 py-1.5">
                    <span
                      className={[
                        "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
                        severityChipClass(rule.worstSeverity),
                      ].join(" ")}
                    >
                      {rule.worstSeverity}
                    </span>
                  </td>
                  <td className="px-2 py-1.5 text-end tabular-nums">{rule.offendingScenarios}</td>
                  <td className="px-2 py-1.5 text-end tabular-nums">{rule.totalFindings}</td>
                  <td className="px-2 py-1.5 text-end tabular-nums text-app-info">
                    {rule.autofixable}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function ScenarioOffendersTable({
  entries,
}: {
  entries: {
    scenario: string;
    targetKind: string;
    targetId: string;
    targetPath: string;
    passed: boolean;
    profileId: string;
    profileRecognized: boolean;
    errorCount: number;
    warningCount: number;
    autofixableCount: number;
    degradedReason: string;
  }[];
}) {
  const { t } = useTranslation();
  if (entries.length === 0) {
    return (
      <section
        data-testid={selectors.fleet.scenarios}
        aria-label={t(strings.fleet.scenarios.title)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h4 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.fleet.scenarios.title)}
        </h4>
        <p data-testid={selectors.fleet.scenariosEmpty} className="mt-3 text-app-muted-foreground">
          {t(strings.fleet.empty)}
        </p>
      </section>
    );
  }
  return (
    <section
      data-testid={selectors.fleet.scenarios}
      data-target-kind-axis="all"
      aria-label={t(strings.fleet.scenarios.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h4 className="text-sm font-medium text-app-muted-foreground">
        {t(strings.fleet.scenarios.title)}
      </h4>
      <div className="mt-3 overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
              <th scope="col" className="px-2 py-1 text-start font-medium">
                {t(strings.fleet.scenarios.col.target)}
              </th>
              <th scope="col" className="px-2 py-1 text-start font-medium">
                {t(strings.fleet.scenarios.col.verdict)}
              </th>
              <th scope="col" className="px-2 py-1 text-start font-medium">
                {t(strings.fleet.scenarios.col.profile)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.fleet.scenarios.col.errors)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.fleet.scenarios.col.warnings)}
              </th>
              <th scope="col" className="px-2 py-1 text-end font-medium">
                {t(strings.fleet.scenarios.col.autofixable)}
              </th>
            </tr>
          </thead>
          <tbody>
            {entries.map((entry) => (
              <tr
                key={`${entry.targetKind || "scenario"}:${entry.targetId || entry.scenario}`}
                data-testid={
                  entry.targetKind === "" || entry.targetKind === "scenario"
                    ? selectors.fleet.scenarioRow({ scenario: entry.scenario })
                    : undefined
                }
                data-passed={entry.passed}
                className="border-t border-app-border"
              >
                <td className="px-2 py-1.5">
                  <span
                    data-testid={selectors.fleet.targetRow({
                      kind: entry.targetKind || "scenario",
                      id: entry.targetId || entry.scenario,
                    })}
                    className="sr-only"
                  />
                  <span className="font-medium text-app-foreground">
                    <span className="me-2 rounded-control border border-app-border px-1.5 py-0.5 text-xs uppercase text-app-muted-foreground">
                      {entry.targetKind || "scenario"}
                    </span>
                    {entry.targetId || entry.scenario}
                  </span>
                  {entry.degradedReason && (
                    <span className="ms-2 text-xs text-app-warning" title={entry.degradedReason}>
                      ⚠ {entry.degradedReason}
                    </span>
                  )}
                </td>
                <td className="px-2 py-1.5">
                  <span
                    className={[
                      "inline-flex items-center gap-1 rounded-control px-1.5 py-0.5 text-xs font-medium",
                      entry.passed
                        ? "border border-app-success/40 bg-app-success/10 text-app-success"
                        : "border border-app-danger/40 bg-app-danger/10 text-app-danger",
                    ].join(" ")}
                  >
                    {entry.passed ? (
                      <CheckCircle2 aria-hidden="true" className="h-3.5 w-3.5" />
                    ) : (
                      <XCircle aria-hidden="true" className="h-3.5 w-3.5" />
                    )}
                    {entry.passed
                      ? t(strings.fleet.scenarios.passed)
                      : t(strings.fleet.scenarios.failed)}
                  </span>
                </td>
                <td className="px-2 py-1.5">
                  <span className="text-app-foreground">{entry.profileId}</span>
                  <span className="ms-1 text-xs text-app-muted-foreground">
                    (
                    {entry.profileRecognized
                      ? t(strings.fleet.scenarios.recognized)
                      : t(strings.fleet.scenarios.unrecognized)}
                    )
                  </span>
                </td>
                <td className="px-2 py-1.5 text-end tabular-nums text-app-danger">
                  {entry.errorCount}
                </td>
                <td className="px-2 py-1.5 text-end tabular-nums text-app-warning">
                  {entry.warningCount}
                </td>
                <td className="px-2 py-1.5 text-end tabular-nums text-app-info">
                  {entry.autofixableCount}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function ScanErrorsCard({
  errors,
}: {
  errors: { scenario: string; reason: string }[];
}) {
  const { t } = useTranslation();
  if (errors.length === 0) return null;
  return (
    <section
      data-testid={selectors.fleet.errors}
      aria-label={t(strings.fleet.errors.title)}
      className="rounded-panel border border-app-danger/40 bg-app-danger/5 p-4"
    >
      <h4 className="text-sm font-medium text-app-danger">{t(strings.fleet.errors.title)}</h4>
      <ul className="mt-3 flex flex-col gap-1 text-sm">
        {errors.map((scanError) => (
          <li key={scanError.scenario} className="flex flex-wrap gap-2">
            <span className="font-medium text-app-foreground">{scanError.scenario}</span>
            <span className="text-app-muted-foreground">{scanError.reason}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
