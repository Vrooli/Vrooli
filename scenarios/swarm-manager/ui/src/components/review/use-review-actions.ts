/**
 * useReviewActions — Manages review trigger state for post-execution checks.
 *
 * Extracts the review trigger logic that was previously inline in JSX
 * (ScenarioReviewResults). Provides loading/error state and a clean
 * testing seam via the service parameter.
 */

import { useCallback, useState } from "react";
import { executionService } from "../../services";

export function useReviewActions(executionId: string | undefined) {
  const [isTriggering, setIsTriggering] = useState(false);
  const [triggerError, setTriggerError] = useState<string | null>(null);

  const triggerReview = useCallback(async () => {
    if (!executionId) return;
    setIsTriggering(true);
    setTriggerError(null);
    try {
      await executionService.triggerReview(executionId);
    } catch (err) {
      setTriggerError(err instanceof Error ? err.message : "Failed to trigger review");
    } finally {
      setIsTriggering(false);
    }
  }, [executionId]);

  return { triggerReview, isTriggering, triggerError };
}
