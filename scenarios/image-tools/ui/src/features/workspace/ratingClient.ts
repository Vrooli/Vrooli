import { clearJobRating, jobsClient, rateJob, type JobRating } from "../../api/jobs";

/**
 * The seam the control drives. Production uses the Jobs API; tests inject fakes
 * so the control is exercised without a transport.
 */
export interface RatingClient {
  /** The job's existing rating, or undefined when it has none. */
  load: (jobId: string) => Promise<JobRating | undefined>;
  rate: (jobId: string, score: number, note: string) => Promise<JobRating | undefined>;
  clear: (jobId: string) => Promise<void>;
}

export const liveRatingClient: RatingClient = {
  load: async (jobId) => (await jobsClient.getJob({ id: jobId })).job?.rating,
  rate: rateJob,
  clear: clearJobRating,
};
