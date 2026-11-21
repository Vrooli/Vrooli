/**
 * Discovery suggestion hooks. Listing target/destination suggestions is a plain
 * query; dismissing invalidates both suggestion lists so the dismissed row
 * disappears. There is no "accept" mutation here — accepting a suggestion uses
 * the existing `useRegisterTarget` / `useCreateDestination` mutations, then the
 * caller invalidates suggestions so the now-registered entry drops off the list.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  dismissSuggestion,
  listDestinationSuggestions,
  listTargetSuggestions,
} from "../api/discovery";
import { queryKeys } from "./keys";

export function useTargetSuggestions() {
  return useQuery({
    queryKey: queryKeys.targetSuggestions,
    queryFn: listTargetSuggestions,
  });
}

export function useDestinationSuggestions() {
  return useQuery({
    queryKey: queryKeys.destinationSuggestions,
    queryFn: listDestinationSuggestions,
  });
}

/** Invalidates both suggestion lists — used after dismiss AND after accept. */
export function useInvalidateSuggestions() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: queryKeys.targetSuggestions });
    void qc.invalidateQueries({ queryKey: queryKeys.destinationSuggestions });
  };
}

export function useDismissSuggestion() {
  const invalidate = useInvalidateSuggestions();
  return useMutation({
    mutationFn: (id: string) => dismissSuggestion(id),
    onSuccess: invalidate,
  });
}
