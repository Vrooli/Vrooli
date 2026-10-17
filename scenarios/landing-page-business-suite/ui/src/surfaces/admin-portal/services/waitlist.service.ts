import type { WaitlistEmail, SiteBranding } from '../../../shared/api';
import {
  getWaitlistEmails,
  deleteWaitlistEmail as apiDeleteWaitlistEmail,
  exportWaitlistCsv,
  getBranding,
  updateBranding,
} from '../../../shared/api';

/**
 * Stats calculated from waitlist data
 */
export interface WaitlistStats {
  totalSignups: number;
  comingSoonCount: number;
}

/**
 * Calculate waitlist statistics from email list
 */
export function calculateStats(emails: WaitlistEmail[]): WaitlistStats {
  return {
    totalSignups: emails.length,
    comingSoonCount: emails.filter((e) => e.source === 'coming_soon').length,
  };
}

// API wrapper functions

/**
 * Fetch all waitlist emails from API
 */
export async function fetchWaitlistEmails(): Promise<WaitlistEmail[]> {
  const emails = await getWaitlistEmails();
  const rawEmails = emails as WaitlistEmail[] | null | undefined;
  return rawEmails ?? [];
}

/**
 * Delete a waitlist email via API
 */
export async function deleteWaitlistEmail(id: number): Promise<void> {
  await apiDeleteWaitlistEmail(id);
}

/**
 * Fetch branding data for coming soon status
 */
export async function fetchBranding(): Promise<SiteBranding> {
  return getBranding();
}

/**
 * Toggle coming soon mode via branding update
 */
export async function toggleComingSoonMode(
  currentValue: boolean
): Promise<boolean> {
  const newValue = !currentValue;
  await updateBranding({ coming_soon_enabled: newValue });
  return newValue;
}

/**
 * Get export URL for CSV download
 */
export async function exportToCsv(): Promise<void> {
  const { csv, filename } = await exportWaitlistCsv();
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
