import type { WaitlistEmail, SiteBranding } from '../../../shared/api';
import {
  getWaitlistEmails,
  deleteWaitlistEmail as apiDeleteWaitlistEmail,
  getWaitlistExportUrl,
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
  return emails || [];
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
export function getExportUrl(): string {
  return getWaitlistExportUrl();
}

/**
 * Open export URL in new window
 */
export function exportToCsv(): void {
  window.open(getExportUrl(), '_blank');
}
