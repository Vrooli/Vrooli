/**
 * useLabels - Manages label CRUD operations and chat-label associations.
 *
 * This hook provides a focused API for label management, keeping
 * the main chat hook simpler.
 *
 * SEAM: Label operations are isolated here. For testing, mock the
 * label API functions or provide test data via React Query's
 * test utilities.
 */
import { useMemo, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchLabels,
  createLabel,
  deleteLabel,
  assignLabel,
  removeLabel,
  Label,
} from "../lib/api";

// Stable empty array for default value
// CRITICAL: Using `= []` in destructuring creates a NEW array on every render,
// which changes references and triggers infinite re-render loops via useMemo dependencies
const EMPTY_LABELS: Label[] = [];

// DEBUG: Track renders
let useLabelsRenderCount = 0;

export function useLabels() {
  const queryClient = useQueryClient();

  // DEBUG: Track renders
  useLabelsRenderCount++;
  console.log(`[useLabels] Render #${useLabelsRenderCount}`);

  // Fetch labels
  // NOTE: Use stable EMPTY_LABELS constant instead of `= []`
  // CRITICAL: Use aggressive caching to prevent cascading re-renders during
  // rapid state transitions (e.g., fresh chat message send).
  const {
    data: labelsData,
    isLoading: loadingLabels,
    error: labelsError,
  } = useQuery({
    queryKey: ["labels"],
    queryFn: fetchLabels,
    staleTime: 60_000, // 1 minute - data considered fresh
    gcTime: 300_000, // 5 minutes - keep in cache
    refetchOnMount: false, // Don't refetch when component mounts
    refetchOnWindowFocus: false, // Don't refetch on window focus
  });
  const labels = labelsData ?? EMPTY_LABELS;

  // Create label
  const createLabelMutation = useMutation({
    mutationFn: createLabel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["labels"] });
    },
  });

  // Delete label
  const deleteLabelMutation = useMutation({
    mutationFn: deleteLabel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["labels"] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  // Assign label to chat
  const assignLabelMutation = useMutation({
    mutationFn: ({ chatId, labelId }: { chatId: string; labelId: string }) =>
      assignLabel(chatId, labelId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["chat", variables.chatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  // Remove label from chat
  const removeLabelMutation = useMutation({
    mutationFn: ({ chatId, labelId }: { chatId: string; labelId: string }) =>
      removeLabel(chatId, labelId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["chat", variables.chatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  // CRITICAL: Memoize action functions to prevent creating new references on every render.
  // Without this, every useLabels render creates new function references that cascade
  // through useChats to all consuming components.
  const createLabelAction = useCallback(
    (data: Parameters<typeof createLabel>[0]) => createLabelMutation.mutate(data),
    [createLabelMutation.mutate]
  );
  const deleteLabelAction = useCallback(
    (labelId: string) => deleteLabelMutation.mutate(labelId),
    [deleteLabelMutation.mutate]
  );
  const assignLabelAction = useCallback(
    (params: { chatId: string; labelId: string }) => assignLabelMutation.mutate(params),
    [assignLabelMutation.mutate]
  );
  const removeLabelAction = useCallback(
    (params: { chatId: string; labelId: string }) => removeLabelMutation.mutate(params),
    [removeLabelMutation.mutate]
  );

  // Extract pending states outside useMemo to avoid including mutation objects in deps
  const isCreatingLabel = createLabelMutation.isPending;
  const isDeletingLabel = deleteLabelMutation.isPending;

  // CRITICAL: Memoize the return object to prevent creating new object references
  // on every render. Without this, every render creates a new object that cascades
  // through useChats to all consuming components, contributing to render storms.
  return useMemo(
    () => ({
      // Data
      labels,

      // Loading states
      loadingLabels,

      // Errors
      labelsError,

      // Actions - use memoized callbacks
      createLabel: createLabelAction,
      deleteLabel: deleteLabelAction,
      assignLabel: assignLabelAction,
      removeLabel: removeLabelAction,

      // Mutation states
      isCreatingLabel,
      isDeletingLabel,
    }),
    [
      labels,
      loadingLabels,
      labelsError,
      createLabelAction,
      deleteLabelAction,
      assignLabelAction,
      removeLabelAction,
      isCreatingLabel,
      isDeletingLabel,
    ]
  );
}

// Re-export Label type for convenience
export type { Label };
