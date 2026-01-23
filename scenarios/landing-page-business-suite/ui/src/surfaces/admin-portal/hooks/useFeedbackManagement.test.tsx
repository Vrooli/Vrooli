import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useFeedbackManagement } from './useFeedbackManagement';
import * as feedbackService from '../services/feedback.service';
import type { FeedbackRequest } from '../../../shared/api';

// Mock the service module
vi.mock('../services/feedback.service', async () => {
  const actual = await vi.importActual('../services/feedback.service');
  return {
    ...actual,
    fetchFeedbackList: vi.fn(),
    updateFeedbackStatus: vi.fn(),
    deleteFeedback: vi.fn(),
    deleteFeedbackBulk: vi.fn(),
    openEmailReply: vi.fn(),
  };
});

const mockFetchFeedbackList = vi.mocked(feedbackService.fetchFeedbackList);
const mockUpdateFeedbackStatus = vi.mocked(feedbackService.updateFeedbackStatus);
const mockDeleteFeedback = vi.mocked(feedbackService.deleteFeedback);
const mockDeleteFeedbackBulk = vi.mocked(feedbackService.deleteFeedbackBulk);
const mockOpenEmailReply = vi.mocked(feedbackService.openEmailReply);

const createMockFeedback = (overrides: Partial<FeedbackRequest> = {}): FeedbackRequest => ({
  id: 1,
  email: 'test@example.com',
  subject: 'Test Subject',
  message: 'Test message content',
  type: 'general',
  status: 'pending',
  created_at: '2024-01-15T10:30:00Z',
  updated_at: '2024-01-15T10:30:00Z',
  ...overrides,
});

describe('useFeedbackManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchFeedbackList.mockResolvedValue([]);
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      expect(result.current.loading).toBe(true);
      await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it('has empty feedback list initially', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.feedbackList).toEqual([]);
    });

    it('has default filter values', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.statusFilter).toBe('all');
      expect(result.current.typeFilter).toBe('all');
    });

    it('has empty selection initially', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.selectedIds.size).toBe(0);
      expect(result.current.expandedId).toBeNull();
    });

    it('has zero counts initially', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.pendingCount).toBe(0);
      expect(result.current.inProgressCount).toBe(0);
    });
  });

  describe('loading data', () => {
    it('fetches feedback on mount', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1 }),
        createMockFeedback({ id: 2 }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.feedbackList).toHaveLength(2);
      expect(mockFetchFeedbackList).toHaveBeenCalledTimes(1);
    });

    it('handles fetch error', async () => {
      mockFetchFeedbackList.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toBe('Network error');
    });

    it('can reload data', async () => {
      mockFetchFeedbackList.mockResolvedValue([]);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(mockFetchFeedbackList).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.loadFeedback();
      });

      expect(mockFetchFeedbackList).toHaveBeenCalledTimes(2);
    });
  });

  describe('filtering', () => {
    it('filters by status', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, status: 'pending' }),
        createMockFeedback({ id: 2, status: 'resolved' }),
        createMockFeedback({ id: 3, status: 'pending' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.filteredFeedback).toHaveLength(3);

      act(() => {
        result.current.setStatusFilter('pending');
      });

      expect(result.current.filteredFeedback).toHaveLength(2);
      expect(result.current.filteredFeedback.every((f) => f.status === 'pending')).toBe(true);
    });

    it('filters by type', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, type: 'bug' }),
        createMockFeedback({ id: 2, type: 'feature' }),
        createMockFeedback({ id: 3, type: 'bug' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setTypeFilter('bug');
      });

      expect(result.current.filteredFeedback).toHaveLength(2);
      expect(result.current.filteredFeedback.every((f) => f.type === 'bug')).toBe(true);
    });

    it('applies both filters together', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, type: 'bug', status: 'pending' }),
        createMockFeedback({ id: 2, type: 'bug', status: 'resolved' }),
        createMockFeedback({ id: 3, type: 'feature', status: 'pending' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setStatusFilter('pending');
        result.current.setTypeFilter('bug');
      });

      expect(result.current.filteredFeedback).toHaveLength(1);
      expect(result.current.filteredFeedback[0].id).toBe(1);
    });
  });

  describe('counts computation', () => {
    it('computes pending count correctly', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, status: 'pending' }),
        createMockFeedback({ id: 2, status: 'pending' }),
        createMockFeedback({ id: 3, status: 'resolved' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.pendingCount).toBe(2);
    });

    it('computes in_progress count correctly', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, status: 'in_progress' }),
        createMockFeedback({ id: 2, status: 'pending' }),
        createMockFeedback({ id: 3, status: 'in_progress' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.inProgressCount).toBe(2);
    });
  });

  describe('selection', () => {
    it('toggles individual selection', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1 }),
        createMockFeedback({ id: 2 }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelect(1);
      });

      expect(result.current.selectedIds.has(1)).toBe(true);
      expect(result.current.selectedIds.size).toBe(1);

      act(() => {
        result.current.handleToggleSelect(1);
      });

      expect(result.current.selectedIds.has(1)).toBe(false);
      expect(result.current.selectedIds.size).toBe(0);
    });

    it('toggles select all', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1 }),
        createMockFeedback({ id: 2 }),
        createMockFeedback({ id: 3 }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelectAll();
      });

      expect(result.current.selectedIds.size).toBe(3);

      act(() => {
        result.current.handleToggleSelectAll();
      });

      expect(result.current.selectedIds.size).toBe(0);
    });

    it('select all respects current filter', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1, status: 'pending' }),
        createMockFeedback({ id: 2, status: 'resolved' }),
        createMockFeedback({ id: 3, status: 'pending' }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setStatusFilter('pending');
      });

      act(() => {
        result.current.handleToggleSelectAll();
      });

      expect(result.current.selectedIds.size).toBe(2);
      expect(result.current.selectedIds.has(1)).toBe(true);
      expect(result.current.selectedIds.has(3)).toBe(true);
      expect(result.current.selectedIds.has(2)).toBe(false);
    });

    it('sets expanded id', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setExpandedId(5);
      });

      expect(result.current.expandedId).toBe(5);

      act(() => {
        result.current.setExpandedId(null);
      });

      expect(result.current.expandedId).toBeNull();
    });
  });

  describe('handleStatusChange', () => {
    it('updates status successfully', async () => {
      const mockFeedback = [createMockFeedback({ id: 1, status: 'pending' })];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);
      mockUpdateFeedbackStatus.mockResolvedValue(
        createMockFeedback({ id: 1, status: 'resolved' })
      );

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let updateResult: { success: boolean; message?: string };
      await act(async () => {
        updateResult = await result.current.handleStatusChange(1, 'resolved');
      });

      expect(updateResult!.success).toBe(true);
      expect(result.current.feedbackList[0].status).toBe('resolved');
      expect(mockUpdateFeedbackStatus).toHaveBeenCalledWith(1, 'resolved');
    });

    it('handles update error', async () => {
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockUpdateFeedbackStatus.mockRejectedValue(new Error('Update failed'));

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let updateResult: { success: boolean; message?: string };
      await act(async () => {
        updateResult = await result.current.handleStatusChange(1, 'resolved');
      });

      expect(updateResult!.success).toBe(false);
      expect(updateResult!.message).toBe('Update failed');
    });

    it('sets action loading state during update', async () => {
      let resolvePromise: (value: FeedbackRequest) => void;
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockUpdateFeedbackStatus.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve;
        })
      );

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let updatePromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        updatePromise = result.current.handleStatusChange(1, 'resolved');
      });

      expect(result.current.actionLoading).toBe(1);

      await act(async () => {
        resolvePromise!(createMockFeedback({ id: 1, status: 'resolved' }));
        await updatePromise;
      });

      expect(result.current.actionLoading).toBeNull();
    });
  });

  describe('handleDelete', () => {
    it('deletes feedback successfully', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1 }),
        createMockFeedback({ id: 2 }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);
      mockDeleteFeedback.mockResolvedValue({ success: true, id: 1 });

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.feedbackList).toHaveLength(2);

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDelete(1);
      });

      expect(deleteResult!.success).toBe(true);
      expect(result.current.feedbackList).toHaveLength(1);
      expect(result.current.feedbackList.find((f) => f.id === 1)).toBeUndefined();
      expect(mockDeleteFeedback).toHaveBeenCalledWith(1);
    });

    it('removes from selection when deleted', async () => {
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockDeleteFeedback.mockResolvedValue({ success: true, id: 1 });

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelect(1);
      });

      expect(result.current.selectedIds.has(1)).toBe(true);

      await act(async () => {
        await result.current.handleDelete(1);
      });

      expect(result.current.selectedIds.has(1)).toBe(false);
    });

    it('handles delete error', async () => {
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockDeleteFeedback.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDelete(1);
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('Delete failed');
    });

    it('sets action loading state during delete', async () => {
      let resolvePromise: () => void;
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockDeleteFeedback.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => resolve({ success: true, id: 1 });
        })
      );

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let deletePromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        deletePromise = result.current.handleDelete(1);
      });

      expect(result.current.actionLoading).toBe(1);

      await act(async () => {
        resolvePromise!();
        await deletePromise;
      });

      expect(result.current.actionLoading).toBeNull();
    });
  });

  describe('handleBulkDelete', () => {
    it('deletes multiple feedback items', async () => {
      const mockFeedback = [
        createMockFeedback({ id: 1 }),
        createMockFeedback({ id: 2 }),
        createMockFeedback({ id: 3 }),
      ];
      mockFetchFeedbackList.mockResolvedValue(mockFeedback);
      mockDeleteFeedbackBulk.mockResolvedValue({ success: true, deleted: 2 });

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelect(1);
        result.current.handleToggleSelect(2);
      });

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleBulkDelete();
      });

      expect(deleteResult!.success).toBe(true);
      expect(result.current.feedbackList).toHaveLength(1);
      expect(result.current.feedbackList[0].id).toBe(3);
      expect(result.current.selectedIds.size).toBe(0);
      expect(mockDeleteFeedbackBulk).toHaveBeenCalledWith([1, 2]);
    });

    it('returns failure when no items selected', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleBulkDelete();
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('No items selected');
    });

    it('handles bulk delete error', async () => {
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockDeleteFeedbackBulk.mockRejectedValue(new Error('Bulk delete failed'));

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelect(1);
      });

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleBulkDelete();
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('Bulk delete failed');
    });

    it('sets bulk action loading state during delete', async () => {
      let resolvePromise: () => void;
      mockFetchFeedbackList.mockResolvedValue([createMockFeedback({ id: 1 })]);
      mockDeleteFeedbackBulk.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => resolve({ success: true, deleted: 1 });
        })
      );

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleToggleSelect(1);
      });

      let deletePromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        deletePromise = result.current.handleBulkDelete();
      });

      expect(result.current.bulkActionLoading).toBe(true);

      await act(async () => {
        resolvePromise!();
        await deletePromise;
      });

      expect(result.current.bulkActionLoading).toBe(false);
    });
  });

  describe('handleReply', () => {
    it('calls openEmailReply with correct arguments', async () => {
      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.handleReply('test@example.com', 'Test Subject');
      });

      expect(mockOpenEmailReply).toHaveBeenCalledWith('test@example.com', 'Test Subject');
    });
  });

  describe('clearError', () => {
    it('clears error state', async () => {
      mockFetchFeedbackList.mockRejectedValue(new Error('Test error'));

      const { result } = renderHook(() => useFeedbackManagement());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });
});
