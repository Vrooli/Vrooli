import { API_BASE, apiCall } from './common';
import type { WaitlistEmail } from './types';

/**
 * Submit an email to the waitlist (public, no auth required)
 */
export async function submitWaitlistEmail(
  email: string,
  source = 'coming_soon'
): Promise<{ success: boolean; message?: string }> {
  return apiCall('/waitlist', {
    method: 'POST',
    body: JSON.stringify({ email, source }),
  });
}

/**
 * Get all waitlist emails (admin only)
 */
export async function getWaitlistEmails(): Promise<WaitlistEmail[]> {
  return apiCall('/admin/waitlist');
}

/**
 * Delete a waitlist email by ID (admin only)
 */
export async function deleteWaitlistEmail(id: number): Promise<{ success: boolean }> {
  return apiCall(`/admin/waitlist/${id}`, {
    method: 'DELETE',
  });
}

/**
 * Get the export URL for downloading waitlist as CSV (admin only)
 */
export function getWaitlistExportUrl(): string {
  return `${API_BASE}/admin/waitlist/export`;
}
