import { useQuery } from "@tanstack/react-query";

import { getSafetyFeedbackStats } from "../../api/safetyFeedback";

export const SAFETY_FEEDBACK_STATS_KEY = ["safety-feedback-stats"] as const;

/** The labelled corpus's aggregate false-positive record. */
export function useSafetyFeedbackStats() {
  return useQuery({
    queryKey: SAFETY_FEEDBACK_STATS_KEY,
    queryFn: () => getSafetyFeedbackStats(),
  });
}
