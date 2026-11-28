import type { FeedbackChoice } from "../../api/safetyFeedback";
import {
  listQuarantinedOutputs,
  quarantineImageUrl,
  type QuarantinedOutput,
} from "../../api/quarantine";
import { recordSafetyFeedback } from "../../api/safetyFeedback";

/**
 * The seam the review queue drives. Production uses the Safety API; tests
 * inject a fake so the reveal, label and download paths are exercised without a
 * transport or real refused bytes.
 */
export interface QuarantineClient {
  list: () => Promise<QuarantinedOutput[]>;
  label: (jobId: string, outputRef: string, choice: FeedbackChoice) => Promise<void>;
  imageUrl: (id: bigint | number, download?: boolean) => string;
}

export const liveQuarantineClient: QuarantineClient = {
  list: async () => (await listQuarantinedOutputs()).outputs,
  label: async (jobId, outputRef, choice) => {
    await recordSafetyFeedback(jobId, choice, { outputRef });
  },
  imageUrl: quarantineImageUrl,
};
