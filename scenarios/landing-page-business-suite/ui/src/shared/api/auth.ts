import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { AdminAuthService, AdminProfileService } from '@vrooli/proto-types/landing-page-business-suite/v1/admin_pb';
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

export async function adminLogin(email: string, password: string, totpCode = '') {
  return adminAuthClient.login({ email, password, totpCode }).then((resp) => {
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

// ===== Admin Two-Factor =====

export interface AdminMFAStatus {
  enabled: boolean;
  enabled_at?: string;
  recovery_codes_left: number;
  enrollment_in_progress: boolean;
}

export interface AdminMFAEnrollment {
  secret: string;
  otpauth_uri: string;
}

export async function getAdminMFAStatus(): Promise<AdminMFAStatus> {
  return apiGet<AdminMFAStatus>('/admin/mfa');
}

export async function beginAdminMFAEnrollment(): Promise<AdminMFAEnrollment> {
  return apiPost<AdminMFAEnrollment>('/admin/mfa/enroll', {});
}

export async function confirmAdminMFAEnrollment(code: string): Promise<{ recovery_codes: string[] }> {
  return apiPost<{ recovery_codes: string[] }>('/admin/mfa/confirm', { code });
}

export async function disableAdminMFA(code: string): Promise<void> {
  await apiPost<unknown>('/admin/mfa/disable', { code });
}

export async function regenerateAdminRecoveryCodes(code: string): Promise<{ recovery_codes: string[] }> {
  return apiPost<{ recovery_codes: string[] }>('/admin/mfa/recovery-codes', { code });
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

export interface DesktopLinkIssueRequest {
	business_account_id?: string;
	installation_id: string;
  resource: string;
  audience: string;
  scopes: string[];
  code_challenge: string;
  code_challenge_method: 'S256';
  redirect_uri: string;
}

export interface DesktopLinkIssueResponse {
  code: string;
  expires_at: string;
  installation_id: string;
  resource: string;
  audience: string;
  scopes: string[];
}

export interface BusinessAccount {
	id: string;
	display_name: string;
	role: string;
	created_at?: string;
}

// ===== User Auth Functions =====

/** How a pending sign-in should finish once the address is proven. */
export type SignInFlow = 'browser' | 'native_app' | 'desktop_link';

/**
 * App parameters captured on the sign-in page. They are stored server-side
 * with the request so a link opened in another tab can resume the same flow.
 */
export interface SignInContext {
  redirect_uri?: string;
  app?: string;
  state?: string;
  code_challenge?: string;
  code_challenge_method?: string;
  desktop_link?: boolean;
  installation_id?: string;
  resource?: string;
  audience?: string;
  scopes?: string[];
  business_account_id?: string;
}

export interface SignInStarted {
  message: string;
  expires_at?: string;
}

export interface SignInPreview {
  email_hint: string;
  expires_at: string;
  flow: SignInFlow;
  same_browser: boolean;
  context?: SignInContext;
}

export interface SignInResult extends VerifyMagicLinkResponse {
  same_browser?: boolean;
  context?: SignInContext;
}

/** Start sign-in: emails a 6-digit code and a one-use link. */
export async function requestMagicLink(email: string, options: { browserBinding?: string; context?: SignInContext } = {}): Promise<SignInStarted> {
  const resp = await apiPost<SignInStarted>('/auth/magic-link', {
    email,
    ...(options.browserBinding ? { browser_binding: options.browserBinding } : {}),
    ...(options.context ? { context: options.context } : {}),
  });
  const validated = parseOrNull(MagicLinkResponseSchema, resp, 'MagicLinkResponse');
  return validated ? { ...resp, message: validated.message } : { message: 'Request sent' };
}

/** Describe a sign-in link without using it up. */
export async function previewSignIn(token: string, browserBinding?: string): Promise<SignInPreview> {
  return apiPost<SignInPreview>('/auth/magic-link/preview', { token, ...(browserBinding ? { browser_binding: browserBinding } : {}) });
}

/** Complete sign-in with the emailed link after the person confirms. */
export async function verifyMagicLink(token: string, browserBinding?: string): Promise<SignInResult> {
  const resp = await apiPost<SignInResult>('/auth/verify', { token, ...(browserBinding ? { browser_binding: browserBinding } : {}) });
  const validated = parseOrNull(VerifyMagicLinkResponseSchema, resp, 'VerifyMagicLinkResponse');
  if (!validated) {
    throw new Error('Invalid verify magic link response from API');
  }
  return { ...resp, ...validated };
}

/** Complete sign-in with the emailed code from the requesting browser. */
export async function verifySignInCode(email: string, code: string, browserBinding: string): Promise<SignInResult> {
  const resp = await apiPost<SignInResult>('/auth/verify-code', { email, code, browser_binding: browserBinding });
  const validated = parseOrNull(VerifyMagicLinkResponseSchema, resp, 'VerifyMagicLinkResponse');
  if (!validated) {
    throw new Error('Invalid verify code response from API');
  }
  return { ...resp, ...validated };
}

/** Proof accepted by the native-app authorization endpoint. */
export type NativeSignInProof =
  | { token: string; browserBinding?: string }
  | { email: string; code: string; browserBinding: string };

/**
 * Exchange a link or code for a one-use PKCE authorization code and return the
 * loopback URL to hand back to the native app. Tokens never enter the URL.
 */
export async function authorizeNativeApp(proof: NativeSignInProof, context: SignInContext): Promise<string> {
  const body: Record<string, string> = {
    code_challenge: context.code_challenge ?? '',
    code_challenge_method: context.code_challenge_method ?? '',
    redirect_uri: context.redirect_uri ?? '',
    state: context.state ?? '',
  };
  if ('token' in proof) {
    body.token = proof.token;
    if (proof.browserBinding) body.browser_binding = proof.browserBinding;
  } else {
    body.email = proof.email;
    body.code = proof.code;
    body.browser_binding = proof.browserBinding;
  }
  const resp = await apiPost<{ redirect_url?: string }>('/auth/authorize', body);
  if (typeof resp.redirect_url !== 'string' || !resp.redirect_url) {
    throw new Error('Invalid native authorization response from API');
  }
  return resp.redirect_url;
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

/** Issue a one-use, scoped desktop-link code for the authenticated browser. */
export async function issueDesktopLink(request: DesktopLinkIssueRequest): Promise<DesktopLinkIssueResponse> {
	return apiPost<DesktopLinkIssueResponse>('/desktop/links', request);
}

/** List LPBS accounts the authenticated browser may select for a link. */
export async function listBusinessAccounts(): Promise<BusinessAccount[]> {
	const response = await apiGet<{ accounts?: BusinessAccount[] }>('/business-accounts');
	return Array.isArray(response.accounts) ? response.accounts : [];
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
