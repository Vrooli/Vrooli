import {
  clearSafetyFeedback,
  listSafetyFeedback,
  recordSafetyFeedback,
  type FeedbackChoice,
  type SafetyFeedbackCase,
} from "../../api/safetyFeedback";

/**
 * The seam the control drives. Production uses the Safety feedback API; tests
 * inject a fake so the control is exercised without a transport.
 */
export interface SafetyFeedbackClient {
  /** The job's existing job-level judgement, or null when it has none. */
  load: (jobId: string) => Promise<SafetyFeedbackCase | null>;
  record: (jobId: string, choice: FeedbackChoice, note: string) => Promise<SafetyFeedbackCase | undefined>;
  clear: (jobId: string) => Promise<boolean>;
}

export const liveSafetyFeedbackClient: SafetyFeedbackClient = {
  load: async (jobId) => {
    const resp = await listSafetyFeedback({ jobId, limit: 50 });
    return resp.cases.find((item) => item.outputRef === "") ?? null;
  },
  record: (jobId, choice, note) => recordSafetyFeedback(jobId, choice, { note }),
  clear: (jobId) => clearSafetyFeedback(jobId),
};
