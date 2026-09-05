/**
 * Destinations query + mutation hooks. Mutations invalidate the list (and the
 * affected destination) so usage bars and cap labels stay current.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  analyzeDestination,
  createDestination,
  deleteDestination,
  getDestination,
  getDestinationUsage,
  listDestinations,
  updateDestination,
  type AnalyzeDestinationInput,
  type CreateDestinationInput,
  type UpdateDestinationInput,
} from "../api/destinations";
import { queryKeys } from "./keys";

export function useDestinations() {
  return useQuery({
    queryKey: queryKeys.destinations,
    queryFn: listDestinations,
  });
}

export function useDestination(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.destination(id ?? ""),
    queryFn: () => getDestination(id ?? ""),
    enabled: Boolean(id),
  });
}

export function useDestinationUsage(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.destinationUsage(id ?? ""),
    queryFn: () => getDestinationUsage(id ?? ""),
    enabled: Boolean(id),
  });
}

function useDestinationInvalidation() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: queryKeys.destinations });
    void qc.invalidateQueries({ queryKey: ["destinationUsage"] });
  };
}

export function useCreateDestination() {
  const invalidate = useDestinationInvalidation();
  return useMutation({
    mutationFn: (input: CreateDestinationInput) => createDestination(input),
    onSuccess: invalidate,
  });
}

export function useAnalyzeDestination() {
  return useMutation({
    mutationFn: (input: AnalyzeDestinationInput) => analyzeDestination(input),
  });
}

export function useUpdateDestination() {
  const invalidate = useDestinationInvalidation();
  return useMutation({
    mutationFn: (input: UpdateDestinationInput) => updateDestination(input),
    onSuccess: invalidate,
  });
}

export function useDeleteDestination() {
  const invalidate = useDestinationInvalidation();
  return useMutation({
    mutationFn: ({ id, deleteRepository }: { id: string; deleteRepository: boolean }) =>
      deleteDestination(id, deleteRepository),
    onSuccess: invalidate,
  });
}
