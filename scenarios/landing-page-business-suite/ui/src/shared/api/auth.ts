import { apiCall, apiPost, apiGet } from './common';
import { parseOrNull } from './safeParse';
import {
  AdminSessionResponseSchema,
  AdminProfileSchema,
  MagicLinkResponseSchema,
  VerifyMagicLinkResponseSchema,
  UserAuthTokensSchema,
  UserAuthMeResponseSchema,
} from './schemas/auth.schema';
import { SuccessResponseSchema } from './schemas/common.schema';

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
  }).then((resp) => {
    const validated = parseOrNull(AdminSessionResponseSchema, resp, 'AdminSessionResponse');
    if (!validated) {
      throw new Error('Invalid admin login response from API');
    }
    return validated;
  });
}

export async function adminLogout() {
  return apiCall<{ success: boolean }>('/admin/logout', {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'AdminLogoutResponse');
    if (!validated) {
      throw new Error('Invalid admin logout response from API');
    }
    return validated;
  });
}

export async function checkAdminSession() {
  return apiCall<AdminSessionResponse>('/admin/session').then((resp) => {
    const validated = parseOrNull(AdminSessionResponseSchema, resp, 'AdminSessionResponse');
    if (!validated) {
      return { authenticated: false };
    }
    return validated;
  });
}

export async function getAdminProfile() {
  return apiCall<AdminProfile>('/admin/profile').then((resp) => {
    const validated = parseOrNull(AdminProfileSchema, resp, 'AdminProfile');
    if (!validated) {
      throw new Error('Invalid admin profile response from API');
    }
    return validated;
  });
}

export async function updateAdminProfile(payload: AdminProfileUpdatePayload) {
  return apiCall<AdminProfile>('/admin/profile', {
    method: 'PUT',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const validated = parseOrNull(AdminProfileSchema, resp, 'AdminProfile');
    if (!validated) {
      throw new Error('Invalid update admin profile response from API');
    }
    return validated;
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
  return apiPost<MagicLinkResponse>('/auth/magic-link', { email }).then((resp) => {
    const validated = parseOrNull(MagicLinkResponseSchema, resp, 'MagicLinkResponse');
    if (!validated) {
      return { message: 'Request sent' };
    }
    return validated;
  });
}

/**
 * Verify a magic link token and get authentication tokens.
 * Returns tokens in response body (for JSON clients) or redirects (handled server-side).
 */
export async function verifyMagicLink(token: string): Promise<VerifyMagicLinkResponse> {
  return apiGet<VerifyMagicLinkResponse>(`/auth/verify?token=${encodeURIComponent(token)}`).then((resp) => {
    const validated = parseOrNull(VerifyMagicLinkResponseSchema, resp, 'VerifyMagicLinkResponse');
    if (!validated) {
      throw new Error('Invalid verify magic link response from API');
    }
    return validated;
  });
}

/**
 * Refresh the access token using a refresh token.
 */
export async function refreshUserTokens(refreshToken: string): Promise<UserAuthTokens> {
  return apiPost<UserAuthTokens>('/auth/refresh', { refresh_token: refreshToken }).then((resp) => {
    const validated = parseOrNull(UserAuthTokensSchema, resp, 'UserAuthTokens');
    if (!validated) {
      throw new Error('Invalid refresh tokens response from API');
    }
    return validated;
  });
}

/**
 * Log out the current user session.
 */
export async function userLogout(): Promise<void> {
  await apiPost<undefined>('/auth/logout', undefined);
}

/**
 * Get the current authenticated user's information.
 */
export async function getUserMe(): Promise<UserAuthMeResponse> {
  return apiGet<UserAuthMeResponse>('/auth/me').then((resp) => {
    const validated = parseOrNull(UserAuthMeResponseSchema, resp, 'UserAuthMeResponse');
    if (!validated) {
      throw new Error('Invalid user me response from API');
    }
    return validated;
  });
}
