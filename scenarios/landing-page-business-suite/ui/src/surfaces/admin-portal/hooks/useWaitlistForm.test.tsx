import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useWaitlistForm } from './useWaitlistForm';
import * as waitlistService from '../services/waitlist.service';
import type { WaitlistEmail, SiteBranding } from '../../../shared/api';

// Mock the service module
vi.mock('../services/waitlist.service', async () => {
  const actual = await vi.importActual('../services/waitlist.service');
  return {
    ...actual,
    fetchWaitlistEmails: vi.fn(),
    deleteWaitlistEmail: vi.fn(),
    fetchBranding: vi.fn(),
    toggleComingSoonMode: vi.fn(),
    exportToCsv: vi.fn(),
  };
});

const mockFetchWaitlistEmails = vi.mocked(waitlistService.fetchWaitlistEmails);
const mockDeleteWaitlistEmail = vi.mocked(waitlistService.deleteWaitlistEmail);
const mockFetchBranding = vi.mocked(waitlistService.fetchBranding);
const mockToggleComingSoonMode = vi.mocked(waitlistService.toggleComingSoonMode);
const mockExportToCsv = vi.mocked(waitlistService.exportToCsv);

const createMockEmail = (overrides: Partial<WaitlistEmail> = {}): WaitlistEmail => ({
  id: 1,
  email: 'test@example.com',
  source: 'coming_soon',
  created_at: '2024-01-15T10:30:00Z',
  ...overrides,
});

const createMockBranding = (overrides: Partial<SiteBranding> = {}): SiteBranding => ({
  id: 1,
  site_name: 'Test Site',
  coming_soon_enabled: false,
  ...overrides,
});

describe('useWaitlistForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchWaitlistEmails.mockResolvedValue([]);
    mockFetchBranding.mockResolvedValue(createMockBranding());
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useWaitlistForm());
      expect(result.current.loading).toBe(true);
      await waitFor(() => { expect(result.current.loading).toBe(false); });
    });

    it('has empty emails initially', async () => {
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.emails).toEqual([]);
    });

    it('has coming soon disabled by default', async () => {
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.comingSoonEnabled).toBe(false);
    });

    it('has stats initialized to zero', async () => {
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.stats).toEqual({ totalSignups: 0, comingSoonCount: 0 });
    });
  });

  describe('loading data', () => {
    it('fetches emails and branding on mount', async () => {
      const mockEmails = [createMockEmail({ id: 1 }), createMockEmail({ id: 2 })];
      mockFetchWaitlistEmails.mockResolvedValue(mockEmails);
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: true }));

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.emails).toHaveLength(2);
      expect(result.current.comingSoonEnabled).toBe(true);
      expect(mockFetchWaitlistEmails).toHaveBeenCalledTimes(1);
      expect(mockFetchBranding).toHaveBeenCalledTimes(1);
    });

    it('handles fetch error', async () => {
      mockFetchWaitlistEmails.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.error).toBe('Network error');
    });

    it('uses safe defaults for absent branding state and non-Error load failures', async () => {
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: undefined }));
      const initial = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(initial.result.current.loading).toBe(false); });
      expect(initial.result.current.comingSoonEnabled).toBe(false);
      initial.unmount();

      mockFetchWaitlistEmails.mockRejectedValue('offline');
      const failed = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(failed.result.current.loading).toBe(false); });
      expect(failed.result.current.error).toBe('Failed to load data');
    });

    it('can reload data', async () => {
      mockFetchWaitlistEmails.mockResolvedValue([]);
      mockFetchBranding.mockResolvedValue(createMockBranding());

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(mockFetchWaitlistEmails).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.loadData();
      });

      expect(mockFetchWaitlistEmails).toHaveBeenCalledTimes(2);
    });
  });

  describe('stats computation', () => {
    it('computes stats correctly', async () => {
      const mockEmails = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
        createMockEmail({ id: 3, source: 'newsletter' }),
      ];
      mockFetchWaitlistEmails.mockResolvedValue(mockEmails);

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.stats.totalSignups).toBe(3);
      expect(result.current.stats.comingSoonCount).toBe(2);
    });
  });

  describe('handleDelete', () => {
    it('deletes email successfully', async () => {
      const mockEmails = [
        createMockEmail({ id: 1 }),
        createMockEmail({ id: 2 }),
      ];
      mockFetchWaitlistEmails.mockResolvedValue(mockEmails);
      mockDeleteWaitlistEmail.mockResolvedValue(undefined);

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.emails).toHaveLength(2);

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDelete(1);
      });

      expect(deleteResult!.success).toBe(true);
      expect(result.current.emails).toHaveLength(1);
      expect(result.current.emails.find((e) => e.id === 1)).toBeUndefined();
      expect(mockDeleteWaitlistEmail).toHaveBeenCalledWith(1);
    });

    it('handles delete error', async () => {
      mockFetchWaitlistEmails.mockResolvedValue([createMockEmail({ id: 1 })]);
      mockDeleteWaitlistEmail.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDelete(1);
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('Delete failed');
      expect(result.current.error).toBe('Delete failed');
    });

    it('uses a safe fallback for a non-Error delete failure', async () => {
      mockDeleteWaitlistEmail.mockRejectedValue('offline');
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const response = await act(async () => result.current.handleDelete(1));

      expect(response).toEqual({ success: false, message: 'Failed to delete email' });
      expect(result.current.error).toBe('Failed to delete email');
    });

    it('sets deleting state during delete', async () => {
      let resolvePromise: () => void;
      mockFetchWaitlistEmails.mockResolvedValue([createMockEmail({ id: 1 })]);
      mockDeleteWaitlistEmail.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => { resolve(undefined); };
        })
      );

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let deletePromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        deletePromise = result.current.handleDelete(1);
      });

      expect(result.current.deleting).toBe(1);

      await act(async () => {
        resolvePromise!();
        await deletePromise;
      });

      expect(result.current.deleting).toBeNull();
    });
  });

  describe('handleToggleComingSoon', () => {
    it('toggles coming soon mode on', async () => {
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: false }));
      mockToggleComingSoonMode.mockResolvedValue(true);

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.comingSoonEnabled).toBe(false);

      let toggleResult: { success: boolean; message?: string };
      await act(async () => {
        toggleResult = await result.current.handleToggleComingSoon();
      });

      expect(toggleResult!.success).toBe(true);
      expect(result.current.comingSoonEnabled).toBe(true);
      expect(mockToggleComingSoonMode).toHaveBeenCalledWith(false);
    });

    it('toggles coming soon mode off', async () => {
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: true }));
      mockToggleComingSoonMode.mockResolvedValue(false);

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.comingSoonEnabled).toBe(true);

      let toggleResult: { success: boolean; message?: string };
      await act(async () => {
        toggleResult = await result.current.handleToggleComingSoon();
      });

      expect(toggleResult!.success).toBe(true);
      expect(result.current.comingSoonEnabled).toBe(false);
    });

    it('handles toggle error', async () => {
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: false }));
      mockToggleComingSoonMode.mockRejectedValue(new Error('Toggle failed'));

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let toggleResult: { success: boolean; message?: string };
      await act(async () => {
        toggleResult = await result.current.handleToggleComingSoon();
      });

      expect(toggleResult!.success).toBe(false);
      expect(toggleResult!.message).toBe('Toggle failed');
      expect(result.current.error).toBe('Toggle failed');
    });

    it('uses a safe fallback for a non-Error toggle failure', async () => {
      mockToggleComingSoonMode.mockRejectedValue('offline');
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const response = await act(async () => result.current.handleToggleComingSoon());

      expect(response).toEqual({ success: false, message: 'Failed to toggle coming soon mode' });
      expect(result.current.error).toBe('Failed to toggle coming soon mode');
    });

    it('sets togglingComingSoon state during toggle', async () => {
      let resolvePromise: () => void;
      mockFetchBranding.mockResolvedValue(createMockBranding({ coming_soon_enabled: false }));
      mockToggleComingSoonMode.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => { resolve(true); };
        })
      );

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let togglePromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        togglePromise = result.current.handleToggleComingSoon();
      });

      expect(result.current.togglingComingSoon).toBe(true);

      await act(async () => {
        resolvePromise!();
        await togglePromise;
      });

      expect(result.current.togglingComingSoon).toBe(false);
    });
  });

  describe('handleExport', () => {
    it('calls exportToCsv', async () => {
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await result.current.handleExport();
      });

      expect(mockExportToCsv).toHaveBeenCalled();
    });

    it('keeps the operator-visible export error safe for non-Error failures', async () => {
      mockExportToCsv.mockRejectedValue('offline');
      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await result.current.handleExport();
      });

      expect(result.current.error).toBe('Failed to export waitlist');
    });
  });

  describe('clearError', () => {
    it('clears error state', async () => {
      mockFetchWaitlistEmails.mockRejectedValue(new Error('Test error'));

      const { result } = renderHook(() => useWaitlistForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });
});
