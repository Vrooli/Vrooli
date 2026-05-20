import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, CheckCircle2, Info, Loader2, RefreshCw, ShieldCheck, XCircle } from "lucide-react";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { StatusPill } from "../../components/ui/StatusPill";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";

import { useQuery } from "@tanstack/react-query";

import { useRecentRuns, useValidateScenario } from "./useValidation";
import {
  validateScenario,
  type FindingSeverity,
  type ValidationFinding,
} from "../../api/validation";

type SeverityFilter = "all" | "error" | "warning" | "info";

const SEVERITY_TONE: Record<FindingSeverity, "error" | "warn" | "info" | "neutral"> = {
  error: "error",
  warning: "warn",
  info: "info",
  unspecified: "neutral",
};

function severityToFilter(s: FindingSeverity): SeverityFilter | null {
  if (s === "error" || s === "warning" || s === "info") return s;
  return null;
}

function formatRanAt(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

export function ValidationDetailPage() {
  const { t } = useTranslation();
  const { scenarioId = "" } = useParams<{ scenarioId: string }>();
  const { record } = useRecentRuns();
  const [filter, setFilter] = useState<SeverityFilter>("all");

  const query = useQuery({
    queryKey: ["validation", "result", scenarioId],
    queryFn: () => validateScenario(scenarioId),
    enabled: scenarioId.length > 0,
    refetchOnWindowFocus: false,
    staleTime: Infinity,
  });

  const mutation = useValidateScenario();
  const result = mutation.data ?? query.data ?? null;

  useEffect(() => {
    if (result) record(result);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- record is stable per useRecentRuns; recording on every result update is desired
  }, [result?.scenario, result?.ranAt]);

  const filtered = useMemo<ValidationFinding[]>(() => {
    if (!result) return [];
    if (filter === "all") return result.findings;
    return result.findings.filter((f) => severityToFilter(f.severity) === filter);
  }, [result, filter]);

  const isLoading = query.isLoading || mutation.isPending;
  const error = mutation.error ?? query.error;

  return (
    <section
      data-testid={selectors.pages.validationDetail}
      aria-labelledby="validation-detail-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <Link
          to={ROUTES.validation}
          className="inline-flex items-center gap-1 self-start text-sm text-app-muted-foreground hover:text-app-foreground"
        >
          <ArrowLeft aria-hidden className="h-4 w-4" />
          {t(strings.pages.validation.back)}
        </Link>
        <div className="flex flex-wrap items-center gap-3">
          <h2 id="validation-detail-heading" className="text-2xl font-semibold tracking-tight break-all">
            {scenarioId}
          </h2>
          {result ? (
            <StatusPill
              data-testid={selectors.validation.detail.statusBadge}
              status={result.passed ? "ok" : "error"}
              label={
                result.passed
                  ? t(strings.pages.validation.status.passed)
                  : t(strings.pages.validation.status.failed)
              }
            />
          ) : null}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => mutation.mutate(scenarioId)}
            disabled={isLoading || scenarioId.length === 0}
            loading={mutation.isPending}
            data-testid={selectors.validation.detail.revalidate}
          >
            <RefreshCw aria-hidden className="h-4 w-4" />
            {t(strings.pages.validation.revalidate)}
          </Button>
        </div>
      </header>

      {isLoading && !result ? (
        <p
          className="flex items-center gap-2 text-sm text-app-muted-foreground"
          role="status"
          aria-live="polite"
          data-testid={selectors.validation.detail.loading}
        >
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.validation.loading)}
        </p>
      ) : null}

      {error ? (
        <div
          role="alert"
          data-testid={selectors.validation.detail.error}
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          {t(strings.pages.validation.error, {
            message: error instanceof Error ? error.message : String(error),
          })}
        </div>
      ) : null}

      {result ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t(strings.pages.validation.summary.heading)}</CardTitle>
            </CardHeader>
            <CardBody>
              <dl
                data-testid={selectors.validation.detail.summary}
                className="grid grid-cols-2 gap-4 sm:grid-cols-4"
              >
                <SummaryStat
                  icon={XCircle}
                  tone="error"
                  label={t(strings.pages.validation.summary.errors)}
                  value={result.summary.errors}
                />
                <SummaryStat
                  icon={Info}
                  tone="warn"
                  label={t(strings.pages.validation.summary.warnings)}
                  value={result.summary.warnings}
                />
                <SummaryStat
                  icon={Info}
                  tone="info"
                  label={t(strings.pages.validation.summary.infos)}
                  value={result.summary.infos}
                />
                <div className="flex flex-col gap-1">
                  <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">
                    {t(strings.pages.validation.summary.ranAt)}
                  </dt>
                  <dd className="text-sm">
                    <time dateTime={result.ranAt}>{formatRanAt(result.ranAt)}</time>
                  </dd>
                </div>
              </dl>
            </CardBody>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t(strings.pages.validation.findings.heading)}</CardTitle>
            </CardHeader>
            <CardBody>
              <fieldset className="flex flex-wrap items-center gap-2 pb-4">
                <legend className="sr-only">
                  {t(strings.pages.validation.filters.heading)}
                </legend>
                {(["all", "error", "warning", "info"] as const).map((s) => (
                  <FilterChip
                    key={s}
                    label={t(
                      s === "all"
                        ? strings.pages.validation.filters.all
                        : s === "error"
                        ? strings.pages.validation.filters.error
                        : s === "warning"
                        ? strings.pages.validation.filters.warning
                        : strings.pages.validation.filters.info,
                    )}
                    active={filter === s}
                    onClick={() => setFilter(s)}
                    testId={selectors.validation.severityFilter({ severity: s })}
                  />
                ))}
              </fieldset>
              {result.findings.length === 0 ? (
                <EmptyState
                  icon={CheckCircle2}
                  title={t(strings.pages.validation.findings.empty)}
                  data-testid={selectors.validation.detail.empty}
                />
              ) : filtered.length === 0 ? (
                <EmptyState
                  icon={ShieldCheck}
                  title={t(strings.pages.validation.findings.noneForFilter)}
                />
              ) : (
                <ul
                  data-testid={selectors.validation.detail.findings}
                  className="flex flex-col gap-3"
                >
                  {filtered.map((finding, idx) => (
                    <FindingRow
                      key={`${finding.code}-${finding.location}-${idx}`}
                      finding={finding}
                      index={idx}
                      t={t}
                    />
                  ))}
                </ul>
              )}
            </CardBody>
          </Card>
        </>
      ) : null}
    </section>
  );
}

function SummaryStat({
  icon: Icon,
  tone,
  label,
  value,
}: {
  icon: typeof XCircle;
  tone: "error" | "warn" | "info";
  label: string;
  value: number;
}) {
  const toneClass =
    tone === "error"
      ? "text-app-danger"
      : tone === "warn"
      ? "text-app-warning"
      : "text-app-info";
  return (
    <div className="flex flex-col gap-1">
      <dt className="flex items-center gap-1 text-xs uppercase tracking-wide text-app-muted-foreground">
        <Icon aria-hidden className={`h-3.5 w-3.5 ${toneClass}`} />
        {label}
      </dt>
      <dd className="text-2xl font-semibold tabular-nums">{value}</dd>
    </div>
  );
}

function FilterChip({
  label,
  active,
  onClick,
  testId,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  testId: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      data-testid={testId}
      className={
        active
          ? "rounded-pill bg-app-primary px-3 py-1 text-xs font-medium text-app-primary-foreground min-h-touch md:min-h-0"
          : "rounded-pill border border-app-border bg-app-surface px-3 py-1 text-xs font-medium text-app-foreground hover:bg-app-surface-muted min-h-touch md:min-h-0"
      }
    >
      {label}
    </button>
  );
}

function FindingRow({
  finding,
  index,
  t,
}: {
  finding: ValidationFinding;
  index: number;
  t: ReturnType<typeof useTranslation>["t"];
}) {
  return (
    <li
      data-testid={selectors.validation.findingRow({ index })}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-2 pb-2">
        <Badge tone={SEVERITY_TONE[finding.severity]}>{finding.severity}</Badge>
        {finding.code ? (
          <span className="font-mono text-xs text-app-muted-foreground">
            <span className="sr-only">{t(strings.pages.validation.findings.code)}: </span>
            {finding.code}
          </span>
        ) : null}
      </div>
      <p className="text-sm">{finding.message}</p>
      {finding.location ? (
        <p className="pt-2 font-mono text-xs text-app-muted-foreground break-all">
          <span className="text-app-muted-foreground/80">
            {t(strings.pages.validation.findings.location)}:{" "}
          </span>
          {finding.location}
        </p>
      ) : null}
      {finding.suggestion ? (
        <p className="pt-2 text-xs text-app-muted-foreground">
          <span className="font-medium text-app-foreground">
            {t(strings.pages.validation.findings.suggestion)}:{" "}
          </span>
          {finding.suggestion}
        </p>
      ) : null}
    </li>
  );
}
