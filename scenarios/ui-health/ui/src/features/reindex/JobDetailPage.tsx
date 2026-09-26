import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Loader2 } from "lucide-react";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import { ProgressBar } from "../../components/ui/ProgressBar";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";
import { isTerminal, type ReindexState } from "../../api/reindex";

import {
  useCancelJob,
  useJobStatus,
  useTrackedJobs,
} from "./useReindexJobs";

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

export function JobDetailPage() {
  const { t } = useTranslation();
  const { jobId = "" } = useParams<{ jobId: string }>();
  const { jobs } = useTrackedJobs();
  const tracked = jobs.find((j) => j.jobId === jobId) ?? null;
  const status = useJobStatus(jobId);
  const cancel = useCancelJob();

  const state: ReindexState = status.data?.state ?? "unknown";
  const terminal = isTerminal(state);
  const processed = status.data?.processed ?? 0;
  const total = status.data?.total ?? 0;
  const pct = total > 0 ? (processed / total) * 100 : state === "succeeded" ? 100 : 0;
  const tone = state === "failed" ? "danger" : state === "succeeded" ? "success" : "default";

  return (
    <section
      data-testid={selectors.pages.reindexJob}
      aria-labelledby="reindex-job-heading"
      className="flex flex-col gap-4"
    >
      <Link
        to={ROUTES.reindex}
        className="inline-flex items-center gap-1 self-start text-sm font-medium text-app-primary hover:underline"
        data-testid={selectors.reindex.detail.back}
      >
        <ArrowLeft aria-hidden className="h-3.5 w-3.5" />
        {t(strings.pages.reindex.job.back)}
      </Link>
      <h2 id="reindex-job-heading" className="text-2xl font-semibold tracking-tight">
        {t(strings.pages.reindex.job.title)}
      </h2>

      {status.isLoading ? (
        <p className="flex items-center gap-2 text-sm text-app-muted-foreground" role="status">
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.reindex.job.loading)}
        </p>
      ) : null}

      {status.error ? (
        <div
          role="alert"
          data-testid={selectors.reindex.detail.error}
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          {t(strings.pages.reindex.job.loadError, {
            message:
              status.error instanceof Error ? status.error.message : String(status.error),
          })}
        </div>
      ) : null}

      {!status.isLoading && !status.data ? (
        <EmptyState
          title={t(strings.pages.reindex.job.notFound.title)}
          description={t(strings.pages.reindex.job.notFound.description, { jobId })}
          data-testid={selectors.reindex.detail.notFound}
        />
      ) : null}

      {status.data ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>
                <span className="break-all font-mono text-sm">{jobId}</span>
              </CardTitle>
            </CardHeader>
            <CardBody>
              <dl
                className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-[max-content_1fr]"
                data-testid={selectors.reindex.detail.meta}
              >
                <DefRow label={t(strings.pages.reindex.job.meta.state)}>
                  <Badge tone={STATE_TONE[state]}>{t(STATE_LABEL_KEY[state])}</Badge>
                </DefRow>
                <DefRow label={t(strings.pages.reindex.job.meta.scenario)}>
                  <span className="font-mono break-all">
                    {tracked?.scenario && tracked.scenario.length > 0
                      ? tracked.scenario
                      : t(strings.pages.reindex.jobs.allScenarios)}
                  </span>
                </DefRow>
                {tracked ? (
                  <DefRow label={t(strings.pages.reindex.job.meta.triggeredAt)}>
                    <time>{formatTimestamp(tracked.triggeredAt)}</time>
                  </DefRow>
                ) : null}
                {tracked ? (
                  <DefRow label={t(strings.pages.reindex.job.meta.dryRun)}>
                    {tracked.dryRun ? (
                      <Badge tone="info">{t(strings.pages.reindex.jobs.columns.dryRun)}</Badge>
                    ) : (
                      <span aria-hidden>—</span>
                    )}
                  </DefRow>
                ) : null}
                <DefRow label={t(strings.pages.reindex.job.meta.processed)}>
                  <span className="tabular-nums font-mono">{processed}</span>
                </DefRow>
                <DefRow label={t(strings.pages.reindex.job.meta.total)}>
                  <span className="tabular-nums font-mono">{total}</span>
                </DefRow>
              </dl>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <ProgressBar
                value={pct}
                label={t(strings.pages.reindex.job.progress)}
                tone={tone}
                data-testid={selectors.reindex.detail.progress}
              />
            </CardBody>
          </Card>

          {status.data.error ? (
            <div
              role="alert"
              data-testid={selectors.reindex.detail.error}
              className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
            >
              {t(strings.pages.reindex.job.error, { message: status.data.error })}
            </div>
          ) : null}

          {!terminal ? (
            <Button
              type="button"
              variant="danger"
              onClick={() => cancel.mutate(jobId)}
              loading={cancel.isPending}
              data-testid={selectors.reindex.detail.cancel}
              className="self-start"
            >
              {cancel.isPending
                ? t(strings.pages.reindex.job.cancelling)
                : t(strings.pages.reindex.job.cancel)}
            </Button>
          ) : null}
        </>
      ) : null}
    </section>
  );
}

function DefRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="text-sm text-app-foreground">{children}</dd>
    </>
  );
}
