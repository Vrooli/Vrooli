import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowUpRight, Inbox, Loader2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card, CardBody, CardDescription, CardHeader, CardTitle } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { Modal } from "../../components/ui/Modal";
import { ProgressBar } from "../../components/ui/ProgressBar";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";
import { isTerminal, type ReindexState, type ReindexStatus } from "../../api/reindex";

import {
  reindexStatusQueryKey,
  trackedFromTrigger,
  useJobStatus,
  useTrackedJobs,
  useTriggerReindex,
  type TrackedJob,
} from "./useReindexJobs";

const SCENARIO_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

const STATE_LABEL_KEY = {
  queued: strings.pages.reindex.state.queued,
  running: strings.pages.reindex.state.running,
  succeeded: strings.pages.reindex.state.succeeded,
  failed: strings.pages.reindex.state.failed,
  cancelled: strings.pages.reindex.state.cancelled,
  unknown: strings.pages.reindex.state.unknown,
} as const satisfies Record<ReindexState, string>;

const STATE_TONE: Record<ReindexState, "neutral" | "info" | "success" | "warn" | "error"> = {
  queued: "neutral",
  running: "info",
  succeeded: "success",
  failed: "error",
  cancelled: "warn",
  unknown: "neutral",
};

function formatTimestamp(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

export function ReindexPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { jobs, add, clearTerminal } = useTrackedJobs();
  const trigger = useTriggerReindex();

  const [scenario, setScenario] = useState("");
  const [dryRun, setDryRun] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [pendingSubmit, setPendingSubmit] = useState<
    { scenario: string; dryRun: boolean } | null
  >(null);

  const trimmed = scenario.trim();
  const inputInvalid = trimmed.length > 0 && !SCENARIO_NAME_PATTERN.test(trimmed);
  const wantsAllScenarios = trimmed.length === 0;

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setFormError(null);
    if (trimmed.length > 0 && !SCENARIO_NAME_PATTERN.test(trimmed)) {
      setFormError(t(strings.pages.reindex.form.scenarioHelp));
      return;
    }
    const payload = { scenario: trimmed, dryRun };
    if (wantsAllScenarios) {
      setPendingSubmit(payload);
      return;
    }
    runTrigger(payload);
  }

  function runTrigger({ scenario: s, dryRun: d }: { scenario: string; dryRun: boolean }) {
    trigger.mutate(
      { scenario: s, dryRun: d },
      {
        onSuccess: (result) => {
          add(trackedFromTrigger(s, result));
        },
        onError: (err) => {
          setFormError(err instanceof Error ? err.message : String(err));
        },
      },
    );
  }

  // Pre-compute status for every tracked job by mounting a JobStatusRow.
  // Each row owns its own polling lifecycle.

  return (
    <section
      data-testid={selectors.pages.reindex}
      aria-labelledby="reindex-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="reindex-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.reindex.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.reindex.description)}
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.reindex.form.heading)}</CardTitle>
          <CardDescription>{t(strings.pages.reindex.form.scenarioHelp)}</CardDescription>
        </CardHeader>
        <CardBody>
          <form
            onSubmit={onSubmit}
            className="flex flex-col gap-3"
            data-testid={selectors.reindex.form}
          >
            <div className="flex flex-col gap-1">
              <label
                htmlFor="reindex-scenario"
                className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground"
              >
                {t(strings.pages.reindex.form.scenarioLabel)}
              </label>
              <input
                id="reindex-scenario"
                type="text"
                value={scenario}
                onChange={(e) => setScenario(e.target.value)}
                aria-invalid={inputInvalid || undefined}
                placeholder={t(strings.pages.reindex.form.scenarioPlaceholder)}
                data-testid={selectors.reindex.scenarioInput}
                className="h-11 min-h-touch w-full rounded-control border border-app-border bg-app-background px-3 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus"
              />
            </div>
            <div className="flex items-start gap-2 text-sm text-app-foreground">
              <input
                id="reindex-dry-run"
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                data-testid={selectors.reindex.dryRunInput}
                className="mt-0.5 h-4 w-4 rounded border-app-border bg-app-background text-app-primary focus-visible:ring-2 focus-visible:ring-app-focus"
              />
              <label htmlFor="reindex-dry-run" className="flex flex-col">
                <span className="font-medium">{t(strings.pages.reindex.form.dryRunLabel)}</span>
                <span className="text-xs text-app-muted-foreground">
                  {t(strings.pages.reindex.form.dryRunHelp)}
                </span>
              </label>
            </div>
            <Button
              type="submit"
              loading={trigger.isPending}
              disabled={trigger.isPending}
              data-testid={selectors.reindex.submit}
              className="self-start"
            >
              {trigger.isPending
                ? t(strings.pages.reindex.form.submitting)
                : t(strings.pages.reindex.form.submit)}
            </Button>
            {formError ? (
              <p
                role="alert"
                data-testid={selectors.reindex.error}
                className="text-sm text-app-danger"
              >
                {t(strings.pages.reindex.error, { message: formError })}
              </p>
            ) : null}
          </form>
        </CardBody>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>{t(strings.pages.reindex.jobs.heading)}</CardTitle>
          {jobs.length > 0 ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                const snapshot = jobs.reduce<Record<string, ReindexState | undefined>>(
                  (acc, j) => {
                    const cached = queryClient.getQueryData<ReindexStatus>(
                      reindexStatusQueryKey(j.jobId),
                    );
                    acc[j.jobId] = cached?.state;
                    return acc;
                  },
                  {},
                );
                clearTerminal(snapshot);
              }}
              data-testid={selectors.reindex.clearTerminal}
            >
              {t(strings.pages.reindex.jobs.clearTerminal)}
            </Button>
          ) : null}
        </CardHeader>
        <CardBody>
          {jobs.length === 0 ? (
            <EmptyState
              icon={Inbox}
              title={t(strings.pages.reindex.jobs.emptyTitle)}
              description={t(strings.pages.reindex.jobs.emptyDescription)}
              data-testid={selectors.reindex.emptyJobs}
            />
          ) : (
            <ul
              className="flex flex-col gap-2"
              data-testid={selectors.reindex.jobsList}
              aria-label={t(strings.pages.reindex.jobs.heading)}
            >
              {jobs.map((job, idx) => (
                <li key={job.jobId}>
                  <JobRow job={job} index={idx} />
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      <Modal
        open={pendingSubmit !== null}
        onClose={() => setPendingSubmit(null)}
        title={t(strings.pages.reindex.confirm.title)}
        description={t(strings.pages.reindex.confirm.description)}
        closeLabel={t(strings.pages.reindex.confirm.close)}
        backdropCloseLabel={t(strings.pages.reindex.confirm.backdrop)}
        data-testid={selectors.reindex.confirmModal}
        footer={
          <>
            <Button
              variant="ghost"
              onClick={() => setPendingSubmit(null)}
              data-testid={selectors.reindex.confirmCancel}
            >
              {t(strings.pages.reindex.confirm.cancel)}
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                if (pendingSubmit) runTrigger(pendingSubmit);
                setPendingSubmit(null);
              }}
              data-testid={selectors.reindex.confirmAccept}
            >
              {t(strings.pages.reindex.confirm.accept)}
            </Button>
          </>
        }
      />
    </section>
  );
}

function JobRow({ job, index }: { job: TrackedJob; index: number }) {
  const { t } = useTranslation();
  const status = useJobStatus(job.jobId);
  const state: ReindexState = status.data?.state ?? "queued";
  const processed = status.data?.processed ?? 0;
  const total = status.data?.total ?? 0;
  const pct = total > 0 ? (processed / total) * 100 : state === "succeeded" ? 100 : 0;
  const tone = state === "failed" ? "danger" : state === "succeeded" ? "success" : "default";
  const terminal = isTerminal(state);

  const scenarioLabel = job.scenario.length === 0
    ? t(strings.pages.reindex.jobs.allScenarios)
    : job.scenario;

  return (
    <article
      data-testid={selectors.reindex.jobRow({ index })}
      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3"
      aria-labelledby={`reindex-job-${index}-title`}
    >
      <header className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3
            id={`reindex-job-${index}-title`}
            className="flex items-center gap-2 text-sm font-semibold tracking-tight"
          >
            <span className="break-all font-mono text-xs text-app-muted-foreground">
              {job.jobId}
            </span>
            {job.dryRun ? <Badge tone="info">{t(strings.pages.reindex.jobs.columns.dryRun)}</Badge> : null}
          </h3>
          <p className="text-xs text-app-muted-foreground">
            <span className="font-mono">{scenarioLabel}</span>
            <span aria-hidden> · </span>
            <time>{formatTimestamp(job.triggeredAt)}</time>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Badge tone={STATE_TONE[state]} data-testid={selectors.reindex.jobState({ index })}>
            {status.isFetching && !terminal ? (
              <Loader2 aria-hidden className="h-3 w-3 animate-spin" />
            ) : null}
            <span>{t(STATE_LABEL_KEY[state])}</span>
          </Badge>
          <Link
            to={ROUTES.reindexJob(job.jobId)}
            className="inline-flex items-center gap-1 text-xs font-medium text-app-primary hover:underline"
            data-testid={selectors.reindex.jobOpen({ index })}
            aria-label={t(strings.pages.reindex.jobs.open, { jobId: job.jobId })}
          >
            {t(strings.pages.reindex.jobs.columns.progress)}
            <ArrowUpRight aria-hidden className="h-3.5 w-3.5" />
          </Link>
        </div>
      </header>

      <ProgressBar
        value={pct}
        label={t(strings.pages.reindex.jobs.columns.progress)}
        tone={tone}
      />
      <p className="text-xs text-app-muted-foreground tabular-nums">
        {processed} / {total}
      </p>

      {status.data?.error ? (
        <p role="alert" className="text-xs text-app-danger">
          {status.data.error}
        </p>
      ) : null}
    </article>
  );
}
