import { FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { History, Loader2 } from "lucide-react";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card, CardBody, CardDescription, CardHeader, CardTitle } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { Table, type ColumnDef } from "../../components/ui/Table";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";

import { useRecentRuns, useValidateScenario, type RecentRun } from "./useValidation";

const SCENARIO_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

function formatRanAt(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

function RunStatusBadge({ run, t }: { run: RecentRun; t: ReturnType<typeof useTranslation>["t"] }) {
  return run.passed ? (
    <Badge tone="success">{t(strings.pages.validation.status.passed)}</Badge>
  ) : (
    <Badge tone="error">{t(strings.pages.validation.status.failed)}</Badge>
  );
}

export function ValidationListPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { runs, record, clear } = useRecentRuns();
  const mutation = useValidateScenario();

  const [scenario, setScenario] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const trimmed = scenario.trim();
  const inputInvalid = trimmed.length > 0 && !SCENARIO_NAME_PATTERN.test(trimmed);

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setValidationError(null);
    if (trimmed.length === 0 || !SCENARIO_NAME_PATTERN.test(trimmed)) {
      setValidationError(t(strings.pages.validation.form.scenarioHelp));
      return;
    }
    mutation.mutate(trimmed, {
      onSuccess: (result) => {
        record(result);
        navigate(ROUTES.validationDetail(result.scenario));
      },
      onError: (err) => {
        setValidationError(err instanceof Error ? err.message : String(err));
      },
    });
  }

  const columns: ColumnDef<RecentRun>[] = [
    {
      key: "scenario",
      header: t(strings.pages.validation.recent.columns.scenario),
      cell: (row) => (
        <Link
          to={ROUTES.validationDetail(row.scenario)}
          className="font-medium text-app-primary hover:underline break-all"
          aria-label={t(strings.pages.validation.recent.open, { scenario: row.scenario })}
          data-testid={selectors.validation.recentRow({ scenario: row.scenario })}
        >
          {row.scenario}
        </Link>
      ),
    },
    {
      key: "status",
      header: t(strings.pages.validation.recent.columns.status),
      cell: (row) => <RunStatusBadge run={row} t={t} />,
    },
    {
      key: "errors",
      header: t(strings.pages.validation.recent.columns.errors),
      cell: (row) =>
        row.errors > 0 ? <Badge tone="error">{row.errors}</Badge> : <span aria-hidden>0</span>,
      align: "right",
    },
    {
      key: "warnings",
      header: t(strings.pages.validation.recent.columns.warnings),
      cell: (row) =>
        row.warnings > 0 ? <Badge tone="warn">{row.warnings}</Badge> : <span aria-hidden>0</span>,
      align: "right",
    },
    {
      key: "ranAt",
      header: t(strings.pages.validation.recent.columns.ranAt),
      cell: (row) => (
        <time dateTime={row.ranAt} className="text-app-muted-foreground">
          {formatRanAt(row.ranAt)}
        </time>
      ),
    },
  ];

  return (
    <section
      data-testid={selectors.pages.validation}
      aria-labelledby="validation-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="validation-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.validation.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.validation.description)}
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.validation.form.heading)}</CardTitle>
          <CardDescription>{t(strings.pages.validation.form.scenarioHelp)}</CardDescription>
        </CardHeader>
        <CardBody>
          <form
            data-testid={selectors.validation.form}
            onSubmit={onSubmit}
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
            noValidate
          >
            <div className="flex flex-1 flex-col gap-1">
              <label htmlFor="validation-scenario" className="text-sm font-medium">
                {t(strings.pages.validation.form.scenarioLabel)}
              </label>
              <input
                id="validation-scenario"
                data-testid={selectors.validation.scenarioInput}
                type="text"
                inputMode="text"
                autoComplete="off"
                spellCheck={false}
                value={scenario}
                onChange={(e) => {
                  setScenario(e.target.value);
                  if (validationError) setValidationError(null);
                }}
                placeholder={t(strings.pages.validation.form.scenarioPlaceholder)}
                aria-invalid={inputInvalid || Boolean(validationError) || undefined}
                aria-describedby="validation-scenario-help"
                className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus"
              />
              <p
                id="validation-scenario-help"
                className={
                  validationError
                    ? "text-xs text-app-danger"
                    : "text-xs text-app-muted-foreground"
                }
                role={validationError ? "alert" : undefined}
              >
                {validationError ?? t(strings.pages.validation.form.scenarioHelp)}
              </p>
            </div>
            <Button
              type="submit"
              data-testid={selectors.validation.submit}
              disabled={mutation.isPending || trimmed.length === 0}
              loading={mutation.isPending}
            >
              {mutation.isPending
                ? t(strings.pages.validation.form.submitting)
                : t(strings.pages.validation.form.submit)}
            </Button>
          </form>
        </CardBody>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t(strings.pages.validation.recent.heading)}</CardTitle>
          {runs.length > 0 ? (
            <Button variant="ghost" size="sm" onClick={() => clear()}>
              {t(strings.pages.validation.recent.clear)}
            </Button>
          ) : null}
        </CardHeader>
        <CardBody>
          {runs.length === 0 ? (
            <EmptyState
              icon={History}
              title={t(strings.pages.validation.recent.empty)}
              data-testid={selectors.validation.emptyRecent}
            />
          ) : (
            <Table<RecentRun>
              data-testid={selectors.validation.recentList}
              columns={columns}
              rows={runs}
              rowKey={(row) => row.scenario}
              emptyTitle={t(strings.pages.validation.recent.empty)}
            />
          )}
        </CardBody>
      </Card>

      {mutation.isPending ? (
        <p
          className="flex items-center gap-2 text-sm text-app-muted-foreground"
          role="status"
          aria-live="polite"
        >
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.validation.loading)}
        </p>
      ) : null}
    </section>
  );
}
