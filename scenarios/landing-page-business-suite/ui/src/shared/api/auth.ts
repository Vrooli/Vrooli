import { apiCall, apiPost, apiGet } from './common';

// ===== Admin Auth Types =====

export interface AdminSessionResponse {
  authenticated: boolean;
  email?: string;
  reset_enabled?: boolean;
}

export interface AdminProfile {
  email: string;
  is_default_email: boolean;
  is_default_password: boolean;
}

export interface AdminProfileUpdatePayload {
  current_password: string;
  new_email?: string;
  new_password?: string;
}

export async function adminLogin(email: string, password: string) {
  return apiCall<AdminSessionResponse>('/admin/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function adminLogout() {
  return apiCall<{ success: boolean }>('/admin/logout', {
    method: 'POST',
  });
}

export async function checkAdminSession() {
  return apiCall<AdminSessionResponse>('/admin/session');
}

export async function getAdminProfile() {
  return apiCall<AdminProfile>('/admin/profile');
}

export async function updateAdminProfile(payload: AdminProfileUpdatePayload) {
  return apiCall<AdminProfile>('/admin/profile', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

// ===== User Auth Types =====

export interface UserAuthUser {
  id: string;
  email: string;
  email_verified: boolean;
  stripe_customer_id?: string;
  created_at?: string;
  last_login_at?: string | null;
}

export interface UserAuthTokens {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
}

export interface MagicLinkResponse {
  message: string;
}

export interface VerifyMagicLinkResponse extends UserAuthTokens {
  user: UserAuthUser;
}

export interface UserAuthMeResponse {
  user: UserAuthUser;
}

// ===== User Auth Functions =====

/**
 * Request a magic link to be sent to the user's email.
 * Always returns success to prevent email enumeration.
 */
export async function requestMagicLink(email: string): Promise<MagicLinkResponse> {
  return apiPost<MagicLinkResponse>('/auth/magic-link', { email });
}

/**
 * Verify a magic link token and get authentication tokens.
 * Returns tokens in response body (for JSON clients) or redirects (handled server-side).
 */
export async function verifyMagicLink(token: string): Promise<VerifyMagicLinkResponse> {
  return apiGet<VerifyMagicLinkResponse>(`/auth/verify?token=${encodeURIComponent(token)}`);
}

/**
 * Refresh the access token using a refresh token.
 */
export async function refreshUserTokens(refreshToken: string): Promise<UserAuthTokens> {
  return apiPost<UserAuthTokens>('/auth/refresh', { refresh_token: refreshToken });
}

/**
 * Log out the current user session.
 */
export async function userLogout(): Promise<void> {
  return apiPost<void>('/auth/logout', undefined);
}

/**
 * Get the current authenticated user's information.
 */
export async function getUserMe(): Promise<UserAuthMeResponse> {
  return apiGet<UserAuthMeResponse>('/auth/me');
}
