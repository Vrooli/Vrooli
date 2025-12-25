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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchLabels,
  createLabel,
  deleteLabel,
  assignLabel,
  removeLabel,
  Label,
} from "../lib/api";

export function useLabels() {
  const queryClient = useQueryClient();

  // Fetch labels
  const {
    data: labels = [],
    isLoading: loadingLabels,
    error: labelsError,
  } = useQuery({
    queryKey: ["labels"],
    queryFn: fetchLabels,
  });

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

  return {
    // Data
    labels,

    // Loading states
    loadingLabels,

    // Errors
    labelsError,

    // Actions
    createLabel: createLabelMutation.mutate,
    deleteLabel: deleteLabelMutation.mutate,
    assignLabel: assignLabelMutation.mutate,
    removeLabel: removeLabelMutation.mutate,

    // Mutation states
    isCreatingLabel: createLabelMutation.isPending,
    isDeletingLabel: deleteLabelMutation.isPending,
  };
}

// Re-export Label type for convenience
export type { Label };
