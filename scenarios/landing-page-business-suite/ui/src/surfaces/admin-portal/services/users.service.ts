import { apiGet, apiPost, apiDelete } from '../../../shared/api/common';

/**
 * Subscription info for a user
 */
export interface SubscriptionInfo {
  status: string;
  plan_tier: string;
}

/**
 * Credit balance info for a user
 */
export interface CreditInfo {
  balance: number;
  bonus: number;
}

/**
 * User account with enriched data
 */
export interface UserAccount {
  id: string;
  email: string;
  email_verified: boolean;
  stripe_customer_id?: string;
  created_at: string;
  last_login_at?: string;
  subscription?: SubscriptionInfo;
  credits?: CreditInfo;
  session_count: number;
}

/**
 * User session info
 */
export interface UserSession {
  id: string;
  created_at: string;
  last_used_at: string;
  expires_at: string;
  ip_address?: string;
  user_agent?: string;
  revoked: boolean;
}

/**
 * Paginated response for user listing
 */
export interface UsersListResponse {
  users: UserAccount[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

/**
 * Parameters for listing users
 */
export interface ListUsersParams {
  search?: string;
  page?: number;
  per_page?: number;
}

/**
 * Fetch paginated list of users with optional search
 */
export async function listUsers(params: ListUsersParams = {}): Promise<UsersListResponse> {
  const queryParams = new URLSearchParams();
  if (params.search) queryParams.set('search', params.search);
  if (params.page) queryParams.set('page', params.page.toString());
  if (params.per_page) queryParams.set('per_page', params.per_page.toString());

  const queryString = queryParams.toString();
  const endpoint = `/admin/users${queryString ? `?${queryString}` : ''}`;

  return apiGet<UsersListResponse>(endpoint);
}

/**
 * Fetch details for a specific user
 */
export async function getUserDetails(id: string): Promise<UserAccount> {
  return apiGet<UserAccount>(`/admin/users/${id}`);
}

/**
 * Fetch sessions for a specific user
 */
export async function getUserSessions(id: string): Promise<UserSession[]> {
  return apiGet<UserSession[]>(`/admin/users/${id}/sessions`);
}

/**
 * Revoke a specific user session
 */
export async function revokeSession(
  userId: string,
  sessionId: string
): Promise<{ success: boolean; message: string }> {
  return apiDelete(`/admin/users/${userId}/sessions/${sessionId}`);
}

/**
 * Revoke all sessions for a user
 */
export async function revokeAllSessions(
  userId: string
): Promise<{ success: boolean; message: string; sessions_revoked: number }> {
  return apiPost(`/admin/users/${userId}/sessions/revoke-all`);
}

/**
 * Format subscription status for display
 */
export function formatSubscriptionStatus(subscription?: SubscriptionInfo): string {
  if (!subscription) return 'No subscription';
  const tier = subscription.plan_tier || 'Unknown';
  const status = subscription.status === 'active' ? '' : ` (${subscription.status})`;
  return `${tier.charAt(0).toUpperCase() + tier.slice(1)}${status}`;
}

/**
 * Format credit balance for display
 */
export function formatCredits(credits?: CreditInfo): string {
  if (!credits) return 'No credits';
  const total = credits.balance + credits.bonus;
  return total.toLocaleString();
}

/**
 * Parse user agent string into a simple device description
 */
export function parseUserAgent(userAgent?: string): string {
  if (!userAgent) return 'Unknown device';

  // Simple parsing for common browsers and platforms
  if (userAgent.includes('Chrome') && userAgent.includes('Windows')) {
    return 'Chrome on Windows';
  }
  if (userAgent.includes('Chrome') && userAgent.includes('Mac')) {
    return 'Chrome on Mac';
  }
  if (userAgent.includes('Firefox') && userAgent.includes('Windows')) {
    return 'Firefox on Windows';
  }
  if (userAgent.includes('Firefox') && userAgent.includes('Mac')) {
    return 'Firefox on Mac';
  }
  if (userAgent.includes('Safari') && !userAgent.includes('Chrome')) {
    return 'Safari on Mac';
  }
  if (userAgent.includes('Edge')) {
    return 'Edge on Windows';
  }
  if (userAgent.includes('Mobile')) {
    return 'Mobile Browser';
  }

  return 'Unknown browser';
}
