import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { JobLane, JobState, jobsClient, type Job } from "../../api/jobs";
import { errorMessage } from "../../lib/errorMessage";
import { useJobProgress } from "./useJobProgress";

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

/** Non-terminal jobs can still be canceled and stream live progress. */
const isActive = (state: JobState) => state === JobState.QUEUED || state === JobState.RUNNING;

interface JobRowProps {
  job: Job;
  onCancel: (id: string) => void;
  cancelPending: boolean;
}

/**
 * One job row. While the job is active it subscribes to WatchJob and overlays
 * the live progress/state/message on top of the polled ListJobs snapshot, so
 * the bar advances without waiting for the next poll.
 */
function JobRow({ job, onCancel, cancelPending }: JobRowProps) {
  const { t } = useTranslation();
  const live = useJobProgress(job.id, isActive(job.state));

  const state = live?.state ?? job.state;
  const progress = live?.progress ?? job.progress;
  const message = live?.message || job.message;

  return (
    <li className="rounded-lg border border-app-border p-3">
      <div className="flex items-center gap-2">
        <span className="font-medium">{job.id}</span>
        {live && (
          <span
            data-testid={selectors.jobs.liveBadge}
            className="rounded border border-app-success/60 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-success"
          >
            {t(strings.jobs.liveBadge)}
          </span>
        )}
      </div>
      <div data-testid={selectors.jobs.operation} className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.jobs.operationLabel)} {job.operation}
      </div>
      <div data-testid={selectors.jobs.lane} className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.jobs.laneLabel)} {t(LANE_LABEL[job.lane])}
      </div>
      <div data-testid={selectors.jobs.state} className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.jobs.stateLabel)} {t(STATE_LABEL[state])}
      </div>
      <div data-testid={selectors.jobs.progress} className="mt-1 text-xs text-app-muted-foreground">
        {t(strings.jobs.progressLabel, { count: progress })}
      </div>
      <progress
        className="mt-1 h-1.5 w-full overflow-hidden rounded bg-app-surface-muted [&::-webkit-progress-bar]:bg-app-surface-muted [&::-webkit-progress-value]:bg-app-primary"
        value={progress}
        max={100}
        aria-label={t(strings.jobs.progressLabel, { count: progress })}
      />
      {message && (
        <div data-testid={selectors.jobs.message} className="mt-1 text-xs text-app-muted-foreground">
          {t(strings.jobs.messageLabel)} {message}
        </div>
      )}
      {state === JobState.SUCCEEDED && job.resultRef && (
        <div data-testid={selectors.jobs.result} className="mt-1 break-all text-xs text-app-muted-foreground">
          {t(strings.jobs.resultLabel)} {job.resultRef}
        </div>
      )}
      {state === JobState.FAILED && job.error && (
        <div data-testid={selectors.jobs.errorDetail} className="mt-1 text-xs text-app-danger">
          {t(strings.jobs.errorLabel)} {job.error}
        </div>
      )}
      {isActive(state) && (
        <Button
          data-testid={selectors.jobs.cancelButton}
          className="mt-2"
          onClick={() => onCancel(job.id)}
          disabled={cancelPending}
        >
          {t(strings.jobs.cancel)}
        </Button>
      )}
    </li>
  );
}

/**
 * JobsCard is the durable-async-work monitor: it lists recent jobs (polled via
 * ListJobs) and, for each active job, overlays live WatchJob progress with a
 * Cancel action (CancelJob). Loading / empty / error states mirror the other
 * read-oriented cards.
 */
export function JobsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const jobsQuery = useQuery({
    queryKey: JOBS_QUERY_KEY,
    queryFn: () => jobsClient.listJobs({ limit: 20 }),
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
      data-testid={selectors.jobs.card}
      aria-label={t(strings.jobs.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4"
    >
      <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.jobs.title)}</h2>
      {jobsQuery.isLoading && (
        <p data-testid={selectors.jobs.loading} className="mt-2 text-app-foreground">
          {t(strings.jobs.loading)}
        </p>
      )}
      {jobsQuery.error && (
        <p data-testid={selectors.jobs.error} className="mt-2 text-app-danger">
          {errorMessage(jobsQuery.error, t)}
        </p>
      )}
      {jobsQuery.data && jobs.length === 0 && (
        <p data-testid={selectors.jobs.empty} className="mt-2 text-app-foreground">
          {t(strings.jobs.empty)}
        </p>
      )}
      {jobs.length > 0 && (
        <ul data-testid={selectors.jobs.list} className="mt-2 space-y-1 text-sm text-app-foreground">
          {jobs.map((job) => (
            <JobRow
              key={job.id}
              job={job}
              onCancel={(id) => cancelJobMutation.mutate(id)}
              cancelPending={cancelJobMutation.isPending}
            />
          ))}
        </ul>
      )}
      {cancelJobMutation.error && (
        <p data-testid={selectors.jobs.error} className="mt-2 text-app-danger">
          {errorMessage(cancelJobMutation.error, t)}
        </p>
      )}
    </section>
  );
}
