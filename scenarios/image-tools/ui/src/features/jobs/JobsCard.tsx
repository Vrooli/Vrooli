import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { JobLane, JobState, jobsClient, type Job } from "../../api/jobs";
import { errorMessage } from "../../lib/errorMessage";

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

/** Terminal jobs can no longer be canceled. */
const isCancelable = (state: JobState) =>
  state === JobState.QUEUED || state === JobState.RUNNING;

/**
 * JobsCard is the durable-async-work surface: it lists the most recent
 * jobs (id, operation, lane, state, progress) and lets the operator
 * cancel an in-flight job. Loading / empty / error states mirror the
 * other read-oriented cards.
 *
 * Live progress (the JobsService.WatchJob server stream) is intentionally
 * out of scope here — see the TODO in `api/jobs.ts`.
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
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.jobs.title)}</h2>
      {jobsQuery.isLoading && (
        <p data-testid={selectors.jobs.loading} className="mt-2 text-slate-200">
          {t(strings.jobs.loading)}
        </p>
      )}
      {jobsQuery.error && (
        <p data-testid={selectors.jobs.error} className="mt-2 text-red-400">
          {errorMessage(jobsQuery.error, t)}
        </p>
      )}
      {jobsQuery.data && jobs.length === 0 && (
        <p data-testid={selectors.jobs.empty} className="mt-2 text-slate-200">
          {t(strings.jobs.empty)}
        </p>
      )}
      {jobs.length > 0 && (
        <ul data-testid={selectors.jobs.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {jobs.map((job) => (
            <li key={job.id} className="rounded-lg border border-white/10 p-3">
              <div className="font-medium">{job.id}</div>
              <div data-testid={selectors.jobs.operation} className="mt-1 text-xs text-slate-400">
                {t(strings.jobs.operationLabel)} {job.operation}
              </div>
              <div data-testid={selectors.jobs.lane} className="mt-1 text-xs text-slate-400">
                {t(strings.jobs.laneLabel)} {t(LANE_LABEL[job.lane])}
              </div>
              <div data-testid={selectors.jobs.state} className="mt-1 text-xs text-slate-400">
                {t(strings.jobs.stateLabel)} {t(STATE_LABEL[job.state])}
              </div>
              <div data-testid={selectors.jobs.progress} className="mt-1 text-xs text-slate-400">
                {t(strings.jobs.progressLabel, { count: job.progress })}
              </div>
              {isCancelable(job.state) && (
                <Button
                  data-testid={selectors.jobs.cancelButton}
                  className="mt-2"
                  onClick={() => cancelJobMutation.mutate(job.id)}
                  disabled={cancelJobMutation.isPending}
                >
                  {t(strings.jobs.cancel)}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      {cancelJobMutation.error && (
        <p data-testid={selectors.jobs.error} className="mt-2 text-red-400">
          {errorMessage(cancelJobMutation.error, t)}
        </p>
      )}
    </section>
  );
}
