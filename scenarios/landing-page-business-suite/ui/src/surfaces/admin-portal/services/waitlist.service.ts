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
 * Format a date string for display
 */
export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString();
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

/**
 * Filter emails by source
 */
export function filterEmailsBySource(
  emails: WaitlistEmail[],
  source: string
): WaitlistEmail[] {
  return emails.filter((e) => e.source === source);
}

/**
 * Search emails by email address
 */
export function searchEmails(
  emails: WaitlistEmail[],
  query: string
): WaitlistEmail[] {
  if (!query.trim()) {
    return emails;
  }
  const lowerQuery = query.toLowerCase();
  return emails.filter((e) =>
    e.email.toLowerCase().includes(lowerQuery)
  );
}

/**
 * Sort emails by date (newest first by default)
 */
export function sortEmailsByDate(
  emails: WaitlistEmail[],
  ascending = false
): WaitlistEmail[] {
  return [...emails].sort((a, b) => {
    const dateA = new Date(a.created_at).getTime();
    const dateB = new Date(b.created_at).getTime();
    return ascending ? dateA - dateB : dateB - dateA;
  });
}

/**
 * Remove an email from the list locally (for optimistic updates)
 */
export function removeEmailFromList(
  emails: WaitlistEmail[],
  idToRemove: number
): WaitlistEmail[] {
  return emails.filter((e) => e.id !== idToRemove);
}

/**
 * Get unique sources from email list
 */
export function getUniqueSources(emails: WaitlistEmail[]): string[] {
  const sources = new Set(emails.map((e) => e.source));
  return Array.from(sources).sort();
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
