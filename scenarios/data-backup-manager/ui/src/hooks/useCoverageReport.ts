/**
 * Coverage report query + bulk default-acceptance mutation. The report is a
 * live derivation (registered vs recommended vs sensitive, plus planned /
 * backed-up / verified posture); accepting defaults registers the non-sensitive
 * recommendations, so on success we invalidate the report, the target lists and
 * the discovery suggestions — every surface that reflects registration state.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  acceptDefaultTargets,
  getCoverageReport,
  type AcceptDefaultsInput,
} from "../api/coverage";
import { queryKeys } from "./keys";

export function useCoverageReport() {
  return useQuery({
    queryKey: queryKeys.coverageReport,
    queryFn: getCoverageReport,
  });
}

/** Invalidates everything registration state touches after a bulk accept. */
export function useInvalidateCoverage() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: queryKeys.coverageReport });
    void qc.invalidateQueries({ queryKey: ["targets"] });
    void qc.invalidateQueries({ queryKey: ["targetStatus"] });
    void qc.invalidateQueries({ queryKey: queryKeys.targetSuggestions });
  };
}

export function useAcceptDefaultTargets() {
  const invalidate = useInvalidateCoverage();
  return useMutation({
    mutationFn: (input: AcceptDefaultsInput = {}) => acceptDefaultTargets(input),
    onSuccess: (res) => {
      // A dry run mutates nothing, so don't churn the caches it would refresh.
      if (!res.dryRun) {
        invalidate();
      }
    },
  });
}
