import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Cpu, Zap } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { blobUrl } from "../../api/client";
import { JobLane, JobState, jobsClient, type Job } from "../../api/jobs";
import { errorMessage } from "../../lib/errorMessage";
import { useJobProgress } from "../jobs/useJobProgress";
import { operationLabelKey } from "../workspace/operationLabel";
import { useReopenOutput } from "../library/useReopenOutput";

const JOBS_QUERY_KEY = ["jobs"] as const;

const LANE_LABEL: Record<JobLane, (typeof strings.jobs.lane)[keyof typeof strings.jobs.lane]> = {
  [JobLane.UNSPECIFIED]: strings.jobs.lane.unspecified,
  [JobLane.GPU]: strings.jobs.lane.gpu,
  [JobLane.CPU]: strings.jobs.lane.cpu,
};

const STATE_LABEL: Record<JobState, (typeof strings.jobs.state)[keyof typeof strings.jobs.state]> = {
  [JobState.UNSPECIFIED]: strings.jobs.state.unspecified,
  [JobState.QUEUED]: strings.jobs.state.queued,
  [JobState.RUNNING]: strings.jobs.state.running,
  [JobState.SUCCEEDED]: strings.jobs.state.succeeded,
  [JobState.FAILED]: strings.jobs.state.failed,
  [JobState.CANCELED]: strings.jobs.state.canceled,
};

const isActive = (state: JobState) => state === JobState.QUEUED || state === JobState.RUNNING;

interface JobRowProps {
  job: Job;
  index: number;
  onCancel: (id: string) => void;
  onOpen: (job: Job) => void;
  cancelPending: boolean;
}

/**
 * One activity row — a result thumbnail (succeeded jobs), a friendly op label,
 * the GPU/CPU lane chip, the live state + progress, a readable status message
 * (which carries the GPU→CPU→cloud fallback line), and Cancel / Open-output.
 * Active jobs overlay live WatchJob progress on the polled snapshot.
 */
function JobRow({ job, index, onCancel, onOpen, cancelPending }: JobRowProps) {
  const { t } = useTranslation();
  const live = useJobProgress(job.id, isActive(job.state));

  const state = live?.state ?? job.state;
  const progress = live?.progress ?? job.progress;
  const message = live?.message || job.message;
  const labelKey = operationLabelKey(job.operation);
  const label = labelKey ? t(labelKey) : job.operation;
  const succeeded = state === JobState.SUCCEEDED && job.resultRef !== "";
  const LaneIcon = job.lane === JobLane.GPU ? Zap : Cpu;

  return (
    <li className="flex gap-3 rounded-lg border border-app-border p-3">
      {succeeded ? (
        <img
          src={blobUrl(job.resultRef)}
          alt={t(strings.activity.thumbnailAlt, { operation: label })}
          loading="lazy"
          className="h-16 w-16 shrink-0 rounded-control border border-app-border object-cover"
          onError={(e) => {
            e.currentTarget.style.display = "none";
          }}
        />
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span data-testid={selectors.jobs.operation} className="truncate font-medium text-app-foreground">
            {label}
          </span>
          <span
            data-testid={selectors.jobs.lane}
            className="inline-flex items-center gap-1 rounded-control border border-app-border px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-muted-foreground"
          >
            <LaneIcon aria-hidden="true" className="h-3 w-3" />
            {t(LANE_LABEL[job.lane])}
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
        {isActive(state) ? (
          <progress
            data-testid={selectors.jobs.progress}
            className="mt-1 h-1.5 w-full overflow-hidden rounded bg-app-surface-muted [&::-webkit-progress-bar]:bg-app-surface-muted [&::-webkit-progress-value]:bg-app-primary"
            value={progress}
            max={100}
            aria-label={t(strings.jobs.progressLabel, { count: progress })}
          />
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
              variant="outline"
              data-testid={selectors.activity.openOutput({ index })}
              onClick={() => onOpen(job)}
            >
              {t(strings.activity.openOutput)}
            </Button>
          ) : null}
          {isActive(state) ? (
            <Button
              size="sm"
              data-testid={selectors.jobs.cancelButton}
              onClick={() => onCancel(job.id)}
              disabled={cancelPending}
            >
              {t(strings.jobs.cancel)}
            </Button>
          ) : null}
        </div>
      </div>
    </li>
  );
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
      {jobsQuery.data && jobs.length === 0 ? (
        <EmptyState
          testId={selectors.activity.empty}
          Icon={Zap}
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
                })
              }
              cancelPending={cancelJobMutation.isPending}
            />
          ))}
        </ul>
      ) : null}
      {cancelJobMutation.error ? (
        <p data-testid={selectors.activity.error} className="text-app-danger">
          {errorMessage(cancelJobMutation.error, t)}
        </p>
      ) : null}
    </section>
  );
}
