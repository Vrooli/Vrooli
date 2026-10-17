import { useState, useEffect, useCallback, useMemo } from 'react';
import type { FeedbackRequest } from '../../../shared/api';
import {
  type FeedbackStatus,
  openEmailReply,
  fetchFeedbackList,
  updateFeedbackStatus,
  deleteFeedback,
  deleteFeedbackBulk,
} from '../services/feedback.service';
import {
  filterByProperties,
  countByProperty,
  toggleInSet,
  selectAll,
  removeById,
  removeByIds,
  removeFromSet,
  updateById,
} from '../../../shared/lib/collections';

export interface UseFeedbackManagementReturn {
  // Data state
  feedbackList: FeedbackRequest[];
  filteredFeedback: FeedbackRequest[];
  pendingCount: number;
  inProgressCount: number;

  // Filter state
  statusFilter: string;
  typeFilter: string;

  // Selection state
  selectedIds: Set<number>;
  expandedId: number | null;

  // UI state
  loading: boolean;
  error: string | null;
  actionLoading: number | null;
  bulkActionLoading: boolean;

  // Filter actions
  setStatusFilter: (filter: string) => void;
  setTypeFilter: (filter: string) => void;

  // Selection actions
  handleToggleSelect: (id: number) => void;
  handleToggleSelectAll: () => void;
  setExpandedId: (id: number | null) => void;

  // Data actions
  loadFeedback: () => Promise<void>;
  handleStatusChange: (id: number, newStatus: FeedbackStatus) => Promise<{ success: boolean; message?: string }>;
  handleDelete: (id: number) => Promise<{ success: boolean; message?: string }>;
  handleBulkDelete: () => Promise<{ success: boolean; message?: string }>;
  handleReply: (email: string, subject: string) => void;

  // Error handling
  clearError: () => void;
}

/**
 * Reactive hook for feedback management
 *
 * Provides state and handlers for:
 * - Loading feedback list
 * - Filtering by status and type
 * - Selecting and expanding items
 * - Updating status, deleting, and bulk operations
 * - Replying via email
 */
export function useFeedbackManagement(): UseFeedbackManagementReturn {
  // Data state
  const [feedbackList, setFeedbackList] = useState<FeedbackRequest[]>([]);

  // Filter state
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [typeFilter, setTypeFilter] = useState<string>('all');

  // Selection state
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [expandedId, setExpandedId] = useState<number | null>(null);

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [bulkActionLoading, setBulkActionLoading] = useState(false);

  /**
   * Load feedback list from API
   */
  const loadFeedback = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchFeedbackList();
      setFeedbackList(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load feedback');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void loadFeedback();
  }, [loadFeedback]);

  /**
   * Update feedback status
   */
  const handleStatusChange = useCallback(
    async (id: number, newStatus: FeedbackStatus): Promise<{ success: boolean; message?: string }> => {
      setActionLoading(id);
      try {
        const updated = await updateFeedbackStatus(id, newStatus);
        setFeedbackList((prev) => updateById(prev, id, updated));
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to update status';
        console.error('Failed to update status:', err);
        return { success: false, message };
      } finally {
        setActionLoading(null);
      }
    },
    []
  );

  /**
   * Delete a single feedback item
   */
  const handleDelete = useCallback(
    async (id: number): Promise<{ success: boolean; message?: string }> => {
      setActionLoading(id);
      try {
        await deleteFeedback(id);
        setFeedbackList((prev) => removeById(prev, id));
        setSelectedIds((prev) => removeFromSet(prev, id));
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to delete feedback';
        console.error('Failed to delete feedback:', err);
        return { success: false, message };
      } finally {
        setActionLoading(null);
      }
    },
    []
  );

  /**
   * Delete multiple selected feedback items
   */
  const handleBulkDelete = useCallback(async (): Promise<{ success: boolean; message?: string }> => {
    if (selectedIds.size === 0) {
      return { success: false, message: 'No items selected' };
    }

    setBulkActionLoading(true);
    try {
      await deleteFeedbackBulk(Array.from(selectedIds));
      setFeedbackList((prev) => removeByIds(prev, selectedIds));
      setSelectedIds(new Set());
      return { success: true };
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to bulk delete';
      console.error('Failed to bulk delete:', err);
      return { success: false, message };
    } finally {
      setBulkActionLoading(false);
    }
  }, [selectedIds]);

  /**
   * Toggle selection of a single item
   */
  const handleToggleSelect = useCallback((id: number) => {
    setSelectedIds((prev) => toggleInSet(prev, id));
  }, []);

  /**
   * Computed filtered feedback list
   */
  const filteredFeedback = useMemo(
    () => filterByProperties(feedbackList, {
      status: statusFilter,
      type: typeFilter,
    }),
    [feedbackList, statusFilter, typeFilter]
  );

  /**
   * Toggle select all (based on filtered list)
   */
  const handleToggleSelectAll = useCallback(() => {
    setSelectedIds((prev) => selectAll(filteredFeedback, prev));
  }, [filteredFeedback]);

  /**
   * Reply via email
   */
  const handleReply = useCallback((email: string, subject: string) => {
    openEmailReply(email, subject);
  }, []);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // Computed counts
  const pendingCount = useMemo(() => countByProperty(feedbackList, 'status', 'pending'), [feedbackList]);
  const inProgressCount = useMemo(() => countByProperty(feedbackList, 'status', 'in_progress'), [feedbackList]);

  return {
    // Data state
    feedbackList,
    filteredFeedback,
    pendingCount,
    inProgressCount,

    // Filter state
    statusFilter,
    typeFilter,

    // Selection state
    selectedIds,
    expandedId,

    // UI state
    loading,
    error,
    actionLoading,
    bulkActionLoading,

    // Filter actions
    setStatusFilter,
    setTypeFilter,

    // Selection actions
    handleToggleSelect,
    handleToggleSelectAll,
    setExpandedId,

    // Data actions
    loadFeedback,
    handleStatusChange,
    handleDelete,
    handleBulkDelete,
    handleReply,

    // Error handling
    clearError,
  };
}
