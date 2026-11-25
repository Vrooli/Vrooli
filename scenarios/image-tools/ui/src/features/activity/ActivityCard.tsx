import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { TFunction } from "i18next";
import { Cloud, Cpu, Layers, ListOrdered } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { EmptyState } from "../../components/ui/empty-state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { blobUrl } from "../../api/client";
import { JobLane, JobQueue, JobState, jobsClient, type Job } from "../../api/jobs";
import { errorMessage } from "../../lib/errorMessage";
import { useJobProgress } from "../jobs/useJobProgress";
import { operationLabelKey } from "../workspace/operationLabel";
import { useReopenOutput } from "../library/useReopenOutput";
import { cn } from "../../lib/utils";
import { SensitiveImage } from "../sensitive/SensitiveImage";
import { isSensitiveRef } from "../../lib/sensitiveMedia";
import { SafetyFeedbackControl } from "./SafetyFeedbackControl";

const JOBS_QUERY_KEY = ["jobs"] as const;

const QUEUE_LABEL: Record<JobQueue, (typeof strings.jobs.queue)[keyof typeof strings.jobs.queue]> = {
  [JobQueue.UNSPECIFIED]: strings.jobs.queue.unspecified,
  [JobQueue.LOCAL_SERIAL]: strings.jobs.queue.localSerial,
  [JobQueue.CPU_POOL]: strings.jobs.queue.cpuPool,
  [JobQueue.NETWORK_POOL]: strings.jobs.queue.networkPool,
  [JobQueue.CAPACITY]: strings.jobs.queue.capacity,
};

const QUEUE_ICON = {
  [JobQueue.UNSPECIFIED]: ListOrdered,
  [JobQueue.LOCAL_SERIAL]: Layers,
  [JobQueue.CPU_POOL]: Cpu,
  [JobQueue.NETWORK_POOL]: Cloud,
  [JobQueue.CAPACITY]: Cpu,
};

const STATE_LABEL: Record<JobState, (typeof strings.jobs.state)[keyof typeof strings.jobs.state]> = {
  [JobState.UNSPECIFIED]: strings.jobs.state.unspecified,
  [JobState.QUEUED]: strings.jobs.state.queued,
  [JobState.RUNNING]: strings.jobs.state.running,
  [JobState.SUCCEEDED]: strings.jobs.state.succeeded,
  [JobState.FAILED]: strings.jobs.state.failed,
  [JobState.CANCELED]: strings.jobs.state.canceled,
  [JobState.ADMITTING]: strings.jobs.state.admitting,
  [JobState.CANCELING]: strings.jobs.state.canceling,
};

const isActive = (state: JobState) => [JobState.QUEUED, JobState.ADMITTING, JobState.RUNNING, JobState.CANCELING].includes(state);

interface JobRowProps {
  job: Job;
  index: number;
  onCancel: (id: string) => void;
  onOpen: (job: Job) => void;
  cancelPending: boolean;
}

/**
 * One activity row — a result thumbnail (succeeded jobs), a friendly op label,
 * the scheduler queue chip, the live state + progress, a readable status message
 * (which carries the GPU→CPU→cloud fallback line), and Cancel / Open-output.
 * Active jobs overlay live WatchJob progress on the polled snapshot.
 */
function JobRow({ job, index, onCancel, onOpen, cancelPending }: JobRowProps) {
  const { t } = useTranslation();
  const live = useJobProgress(job.id, isActive(job.state));

  const state = live?.state ?? job.state;
  const progress = live?.progress ?? job.progress;
  const message = live?.message || job.message;
  const estimate = jobEstimate(job, state, Date.now());
  const labelKey = operationLabelKey(job.operation);
  const label = labelKey ? t(labelKey) : job.operation;
  const succeeded = state === JobState.SUCCEEDED && job.resultRef !== "";
  const queue = schedulingQueueForJob(job);
  const QueueIcon = QUEUE_ICON[queue];

  return (
    <li className="flex gap-3 rounded-lg border border-app-border p-3">
      {succeeded ? (
        <SensitiveImage
          src={blobUrl(job.resultRef)}
          sensitive={isSensitiveRef(job, job.resultRef)}
          compact
          alt={t(strings.activity.thumbnailAlt, { operation: label })}
          loading="lazy"
          wrapperClassName="h-16 w-16 shrink-0 rounded-control"
          className="h-16 w-16 shrink-0 rounded-control border border-app-border object-cover"
          onError={(e) => {
            e.currentTarget.style.display = "none";
          }}
        />
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span data-testid={selectors.jobs.operation} className="min-w-0 max-w-full flex-1 truncate font-medium text-app-foreground">
            {label}
          </span>
          <span
            data-testid={selectors.jobs.lane}
            className="inline-flex items-center gap-1 rounded-control border border-app-border px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-muted-foreground"
          >
            <QueueIcon aria-hidden="true" className="h-3 w-3" />
            {t(QUEUE_LABEL[queue])}
          </span>
          {live ? (
            <span
              data-testid={selectors.jobs.liveBadge}
              className="rounded border border-app-success/60 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-success"
            >
              {t(strings.jobs.liveBadge)}
            </span>
          ) : null}
        </div>
        <div data-testid={selectors.jobs.state} className="mt-1 text-xs text-app-muted-foreground">
          {t(strings.jobs.stateLabel)} {t(STATE_LABEL[state])}
        </div>
        {state === JobState.RUNNING ? (
          <progress
            data-testid={selectors.jobs.progress}
            className="mt-1 h-1.5 w-full overflow-hidden rounded bg-app-surface-muted [&::-webkit-progress-bar]:bg-app-surface-muted [&::-webkit-progress-value]:bg-app-primary"
            value={progress}
            max={100}
            aria-label={t(strings.jobs.progressLabel, { count: progress, operation: label })}
          />
        ) : null}
        {estimate ? (
          <p data-testid={selectors.jobs.estimate} className="mt-1 text-xs text-app-muted-foreground">
            {estimate.kind === "unavailable" ? t(strings.jobs.estimate.unavailable) : null}
            {estimate.kind === "queued" ? t(strings.jobs.estimate.queued, { duration: formatDuration(estimate.seconds, t) }) : null}
            {estimate.kind === "running" ? t(strings.jobs.estimate.running, { duration: formatDuration(estimate.seconds, t) }) : null}
            {estimate.kind === "overdue" ? t(strings.jobs.estimate.overdue, { duration: formatDuration(estimate.seconds, t) }) : null}
          </p>
        ) : null}
        {message ? (
          <div data-testid={selectors.jobs.message} className="mt-1 text-xs text-app-muted-foreground">
            {message}
          </div>
        ) : null}
        {state === JobState.FAILED && job.error ? (
          <div data-testid={selectors.jobs.errorDetail} className="mt-1 text-xs text-app-danger">
            {t(strings.jobs.errorLabel)} {job.error}
          </div>
        ) : null}
        <div className="mt-2 flex flex-wrap gap-2">
          {succeeded ? (
            <Button
              size="sm"
              variant="ghost"
              data-testid={selectors.activity.openOutput({ index })}
              onClick={() => onOpen(job)}
              shape="square"
              className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
            >
              {t(strings.activity.openOutput)}
            </Button>
          ) : null}
          {isActive(state) ? (
            <Button
              size="sm"
              data-testid={selectors.jobs.cancelButton}
              onClick={() => onCancel(job.id)}
              disabled={cancelPending || state === JobState.CANCELING}
              shape="square"
            >
              {t(strings.jobs.cancel)}
            </Button>
          ) : null}
        </div>
        {state === JobState.FAILED || state === JobState.SUCCEEDED ? (
          <SafetyFeedbackControl job={job} />
        ) : null}
      </div>
    </li>
  );
}

// Rows written before scheduling_queue carry only the deprecated lane. Read it
// through its structural shape so the compatibility path is explicit rather than
// a lint suppression.
interface LegacyLaneJob extends Omit<Job, "lane"> {
  lane?: JobLane;
}

function legacyLane(job: Job): JobLane {
  return (job as LegacyLaneJob).lane ?? JobLane.UNSPECIFIED;
}

function schedulingQueueForJob(job: Job): JobQueue {
  if (job.schedulingQueue !== JobQueue.UNSPECIFIED) return job.schedulingQueue;
  switch (legacyLane(job)) {
    // Legacy JOB_LANE_GPU described the serialized local queue. It never
    // established which device the selected backend used.
    case JobLane.GPU:
      return JobQueue.LOCAL_SERIAL;
    case JobLane.CPU:
      return JobQueue.CPU_POOL;
    case JobLane.NETWORK:
      return JobQueue.NETWORK_POOL;
    default:
      return JobQueue.UNSPECIFIED;
  }
}

/**
 * Activity — the durable-async-work monitor (the Console-zone upgrade of the
 * old jobs card): live + recent operations with result thumbnails, friendly
 * labels, lane chips, readable fallback status, Cancel, and Open-output that
 * reopens a result in the Workspace.
 */
export function ActivityCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const reopen = useReopenOutput();

  const jobsQuery = useQuery({
    queryKey: JOBS_QUERY_KEY,
    queryFn: () => jobsClient.listJobs({ limit: 30 }),
  });

  const cancelJobMutation = useMutation({
    mutationFn: (id: string) => jobsClient.cancelJob({ id }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: JOBS_QUERY_KEY });
    },
  });

  const jobs: Job[] = jobsQuery.data?.jobs ?? [];

  return (
    <section
      data-testid={selectors.activity.card}
      aria-label={t(strings.jobs.title)}
      className="flex flex-col gap-2"
    >
      <p className="text-sm text-app-muted-foreground">{t(strings.activity.description)}</p>
      {jobsQuery.isLoading ? (
        <p data-testid={selectors.activity.loading} className="text-app-foreground">
          {t(strings.jobs.loading)}
        </p>
      ) : null}
      {jobsQuery.error ? (
        <p data-testid={selectors.activity.error} className="text-app-danger">
          {errorMessage(jobsQuery.error, t)}
        </p>
      ) : null}
      {cancelJobMutation.error ? (
        <p role="alert" data-testid={selectors.jobs.cancelError} className="text-sm text-app-danger">
          {t(strings.jobs.cancelFailed)}
        </p>
      ) : null}
      {jobsQuery.data && jobs.length === 0 ? (
        <EmptyState
          testId={selectors.activity.empty}
          Icon={Layers}
          title={t(strings.jobs.empty)}
        />
      ) : null}
      {jobs.length > 0 ? (
        <ul data-testid={selectors.activity.list} className="flex flex-col gap-2">
          {jobs.map((job, index) => (
            <JobRow
              key={job.id}
              job={job}
              index={index + 1}
              onCancel={(id) => cancelJobMutation.mutate(id)}
              onOpen={(j) =>
                void reopen({
                  jobId: j.id,
                  operation: j.operation,
                  resultRef: j.resultRef,
                  createdAtMs: 0,
                  sensitive: isSensitiveRef(j, j.resultRef),
                })
              }
              cancelPending={cancelJobMutation.isPending}
            />
          ))}
        </ul>
      ) : null}
    </section>
  );
}

type JobEstimate =
  | { kind: "queued" | "running" | "overdue"; seconds: number }
  | { kind: "unavailable" };

function jobEstimate(job: Job, state: JobState, now: number): JobEstimate | null {
  if (![JobState.QUEUED, JobState.ADMITTING, JobState.RUNNING].includes(state)) {
    return null;
  }
  const estimateSeconds = job.estimatedSeconds;
  if (!Number.isFinite(estimateSeconds) || estimateSeconds <= 0) {
    return { kind: "unavailable" };
  }

  if (state === JobState.QUEUED || state === JobState.ADMITTING) {
    // The caller's estimate describes execution time; queue and capacity wait
    // are separate and must not be hidden inside the runtime estimate.
    return { kind: "queued", seconds: estimateSeconds };
  }

  const startedAt = job.startedAt;
  if (!startedAt) return { kind: "unavailable" };
  const startedMs = Number(startedAt.seconds) * 1000 + Math.floor(startedAt.nanos / 1_000_000);
  if (!Number.isFinite(startedMs) || startedMs <= 0) return { kind: "unavailable" };

  const elapsedSeconds = Math.max(0, Math.floor((now - startedMs) / 1000));
  const remainingSeconds = Math.ceil(estimateSeconds - elapsedSeconds);
  return remainingSeconds > 0
    ? { kind: "running", seconds: remainingSeconds }
    : { kind: "overdue", seconds: estimateSeconds };
}

function formatDuration(seconds: number, t: TFunction): string {
  if (seconds < 60) return t(strings.jobs.estimate.seconds, { count: Math.max(1, Math.ceil(seconds)) });
  const minutes = Math.ceil(seconds / 60);
  if (minutes < 60) return t(strings.jobs.estimate.minutes, { count: minutes });
  const hours = Math.floor(minutes / 60);
  const extraMinutes = minutes % 60;
  return extraMinutes === 0
    ? t(strings.jobs.estimate.hours, { count: hours })
    : t(strings.jobs.estimate.hoursMinutes, { hours, minutes: extraMinutes });
}
