import { useState, useEffect, useCallback, useMemo } from 'react';
import type { WaitlistEmail } from '../../../shared/api';
import {
  calculateStats,
  fetchWaitlistEmails,
  deleteWaitlistEmail,
  fetchBranding,
  toggleComingSoonMode,
  exportToCsv,
  type WaitlistStats,
} from '../services/waitlist.service';
import { removeById } from '../../../shared/lib/collections';

export interface UseWaitlistFormReturn {
  // Data state
  emails: WaitlistEmail[];
  comingSoonEnabled: boolean;
  stats: WaitlistStats;

  // UI state
  loading: boolean;
  error: string | null;
  deleting: number | null;
  togglingComingSoon: boolean;

  // Actions
  loadData: () => Promise<void>;
  handleDelete: (id: number) => Promise<{ success: boolean; message?: string }>;
  handleToggleComingSoon: () => Promise<{ success: boolean; message?: string }>;
  handleExport: () => void;
  clearError: () => void;
}

/**
 * Reactive hook for waitlist management
 *
 * Provides state and handlers for:
 * - Loading waitlist emails and coming soon status
 * - Deleting individual emails
 * - Toggling coming soon mode
 * - Exporting to CSV
 */
export function useWaitlistForm(): UseWaitlistFormReturn {
  // Data state
  const [emails, setEmails] = useState<WaitlistEmail[]>([]);
  const [comingSoonEnabled, setComingSoonEnabled] = useState(false);

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const [togglingComingSoon, setTogglingComingSoon] = useState(false);

  /**
   * Load all data from APIs
   */
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [emailsData, brandingData] = await Promise.all([
        fetchWaitlistEmails(),
        fetchBranding(),
      ]);
      setEmails(emailsData);
      setComingSoonEnabled(brandingData.coming_soon_enabled ?? false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void loadData();
  }, [loadData]);

  /**
   * Delete a waitlist email
   */
  const handleDelete = useCallback(
    async (id: number): Promise<{ success: boolean; message?: string }> => {
      setDeleting(id);
      try {
        await deleteWaitlistEmail(id);
        // Optimistically remove from local state
        setEmails((prev) => removeById(prev, id));
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to delete email';
        setError(message);
        return { success: false, message };
      } finally {
        setDeleting(null);
      }
    },
    []
  );

  /**
   * Toggle coming soon mode
   */
  const handleToggleComingSoon = useCallback(
    async (): Promise<{ success: boolean; message?: string }> => {
      setTogglingComingSoon(true);
      try {
        const newValue = await toggleComingSoonMode(comingSoonEnabled);
        setComingSoonEnabled(newValue);
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to toggle coming soon mode';
        setError(message);
        return { success: false, message };
      } finally {
        setTogglingComingSoon(false);
      }
    },
    [comingSoonEnabled]
  );

  /**
   * Export emails to CSV
   */
  const handleExport = useCallback(() => {
    exportToCsv();
  }, []);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // Computed values
  const stats = useMemo(() => calculateStats(emails), [emails]);

  return {
    // Data state
    emails,
    comingSoonEnabled,
    stats,

    // UI state
    loading,
    error,
    deleting,
    togglingComingSoon,

    // Actions
    loadData,
    handleDelete,
    handleToggleComingSoon,
    handleExport,
    clearError,
  };
}
