import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { AdminAuthService, AdminProfileService } from '@vrooli/proto-types/landing-page-business-suite/admin_pb';
import { apiPost, apiGet, CONNECT_API_BASE } from './common';
import { parseOrNull } from './safeParse';
import {
  AdminSessionResponseSchema,
  AdminProfileSchema,
  MagicLinkResponseSchema,
  VerifyMagicLinkResponseSchema,
  UserAuthTokensSchema,
  UserAuthMeResponseSchema,
} from './schemas/auth.schema';

const adminAuthClient = createClient(AdminAuthService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));
const adminProfileClient = createClient(AdminProfileService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

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
  return adminAuthClient.login({ email, password }).then((resp) => {
    const validated = parseOrNull(AdminSessionResponseSchema, {
      authenticated: resp.authenticated,
      ...(resp.email ? { email: resp.email } : {}),
      reset_enabled: resp.resetEnabled,
    }, 'AdminSessionResponse');
    if (!validated) {
      throw new Error('Invalid admin login response from API');
    }
    return validated;
  });
}

export async function adminLogout() {
  return adminAuthClient.logout({}).then((resp) => {
    if (!resp.success) {
      throw new Error('Invalid admin logout response from API');
    }
    return { success: true };
  });
}

export async function checkAdminSession() {
  return adminAuthClient.session({}).then((resp) => {
    const validated = parseOrNull(AdminSessionResponseSchema, {
      authenticated: resp.authenticated,
      ...(resp.email ? { email: resp.email } : {}),
      reset_enabled: resp.resetEnabled,
    }, 'AdminSessionResponse');
    if (!validated) {
      return { authenticated: false };
    }
    return validated;
  });
}

export async function getAdminProfile() {
  return adminProfileClient.getAdminProfile({}).then((resp) => {
    const profile = resp.profile;
    const validated = parseOrNull(AdminProfileSchema, profile && {
      email: profile.email,
      is_default_email: profile.isDefaultEmail,
      is_default_password: profile.isDefaultPassword,
    }, 'AdminProfile');
    if (!validated) {
      throw new Error('Invalid admin profile response from API');
    }
    return validated;
  });
}

export async function updateAdminProfile(payload: AdminProfileUpdatePayload) {
  return adminProfileClient.updateAdminProfile({
    currentPassword: payload.current_password,
    newEmail: payload.new_email ?? '',
    newPassword: payload.new_password ?? '',
  }).then((resp) => {
    const profile = resp.profile;
    const validated = parseOrNull(AdminProfileSchema, profile && {
      email: profile.email,
      is_default_email: profile.isDefaultEmail,
      is_default_password: profile.isDefaultPassword,
    }, 'AdminProfile');
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
