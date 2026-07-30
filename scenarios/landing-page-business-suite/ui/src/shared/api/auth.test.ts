import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';

const adminAuthClient = vi.hoisted(() => ({ login: vi.fn(), logout: vi.fn(), session: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => adminAuthClient) }));
import {
  adminLogin,
  adminLogout,
  checkAdminSession,
  getAdminProfile,
  updateAdminProfile,
  requestMagicLink,
  verifyMagicLink,
  refreshUserTokens,
  userLogout,
  getUserMe,
  type AdminSessionResponse,
  type AdminProfile,
  type MagicLinkResponse,
  type VerifyMagicLinkResponse,
  type UserAuthTokens,
  type UserAuthMeResponse,
} from './auth';
import { ApiError } from './common';
import { createFetchMock, mockResponses, installFetchMock, getFetchCall, parseJsonBody } from '../test-utils/api-mocks';

describe('auth API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Admin Auth', () => {
    describe('adminLogin', () => {
      it('sends the generated login request', async () => {
        adminAuthClient.login.mockResolvedValue({ authenticated: true, email: 'admin@example.com', resetEnabled: false });

        await adminLogin('admin@example.com', 'password123');

        expect(adminAuthClient.login).toHaveBeenCalledWith({ email: 'admin@example.com', password: 'password123' });
      });

      it('returns session response on success', async () => {
        const sessionResponse: AdminSessionResponse = { authenticated: true, email: 'admin@example.com', reset_enabled: true };
        adminAuthClient.login.mockResolvedValue({ authenticated: true, email: 'admin@example.com', resetEnabled: true });

        const result = await adminLogin('admin@example.com', 'password123');

        expect(result).toEqual(sessionResponse);
      });

      it('propagates a generated-client authentication failure', async () => {
        adminAuthClient.login.mockRejectedValue(new Error('Invalid credentials'));

        await expect(adminLogin('admin@example.com', 'wrong')).rejects.toThrow('Invalid credentials');
      });
    });

    describe('adminLogout', () => {
      it('calls the generated logout procedure', async () => {
        adminAuthClient.logout.mockResolvedValue({ success: true });

        await adminLogout();

        expect(adminAuthClient.logout).toHaveBeenCalledWith({});
      });

      it('returns success response', async () => {
        adminAuthClient.logout.mockResolvedValue({ success: true });

        const result = await adminLogout();

        expect(result).toEqual({ success: true });
      });
    });

    describe('checkAdminSession', () => {
      it('calls the generated session procedure', async () => {
        adminAuthClient.session.mockResolvedValue({ authenticated: true, email: 'admin@example.com', resetEnabled: false });

        await checkAdminSession();

        expect(adminAuthClient.session).toHaveBeenCalledWith({});
      });

      it('returns authenticated session', async () => {
        adminAuthClient.session.mockResolvedValue({ authenticated: true, email: 'admin@example.com', resetEnabled: false });

        const result = await checkAdminSession();

        expect(result.authenticated).toBe(true);
        expect(result.email).toBe('admin@example.com');
      });

      it('returns unauthenticated session', async () => {
        adminAuthClient.session.mockResolvedValue({ authenticated: false, email: '', resetEnabled: false });

        const result = await checkAdminSession();

        expect(result.authenticated).toBe(false);
      });
    });

    describe('getAdminProfile', () => {
      it('returns admin profile', async () => {
        const profile: AdminProfile = {
          email: 'admin@example.com',
          is_default_email: false,
          is_default_password: false,
        };
        fetchMock.mockResolvedValue(mockResponses.success(profile));

        const result = await getAdminProfile();

        expect(result).toEqual(profile);
      });

      it('indicates default credentials', async () => {
        const profile: AdminProfile = {
          email: 'admin@localhost',
          is_default_email: true,
          is_default_password: true,
        };
        fetchMock.mockResolvedValue(mockResponses.success(profile));

        const result = await getAdminProfile();

        expect(result.is_default_email).toBe(true);
        expect(result.is_default_password).toBe(true);
      });
    });

    describe('updateAdminProfile', () => {
      it('sends PUT request with update payload', async () => {
        const profile: AdminProfile = {
          email: 'new@example.com',
          is_default_email: false,
          is_default_password: false,
        };
        fetchMock.mockResolvedValue(mockResponses.success(profile));

        await updateAdminProfile({
          current_password: 'oldpassword',
          new_email: 'new@example.com',
        });

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('PUT');
        expect(parseJsonBody(options.body)).toEqual({
          current_password: 'oldpassword',
          new_email: 'new@example.com',
        });
      });

      it('allows updating password', async () => {
        const profile: AdminProfile = {
          email: 'admin@example.com',
          is_default_email: false,
          is_default_password: false,
        };
        fetchMock.mockResolvedValue(mockResponses.success(profile));

        await updateAdminProfile({
          current_password: 'oldpassword',
          new_password: 'newpassword',
        });

        const [, options] = getFetchCall(fetchMock);
        expect(parseJsonBody(options.body)).toEqual({
          current_password: 'oldpassword',
          new_password: 'newpassword',
        });
      });

      it('throws on incorrect current password', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(400, 'Current password is incorrect'));

        await expect(
          updateAdminProfile({
            current_password: 'wrong',
            new_email: 'new@example.com',
          })
        ).rejects.toBeInstanceOf(ApiError);
      });
    });
  });

  describe('User Auth', () => {
    describe('requestMagicLink', () => {
      it('sends POST request with email', async () => {
        const response: MagicLinkResponse = {
          message: 'If an account exists, a magic link has been sent.',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        await requestMagicLink('user@example.com');

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(parseJsonBody(options.body)).toEqual({
          email: 'user@example.com',
        });
      });

      it('always returns success to prevent email enumeration', async () => {
        const response: MagicLinkResponse = {
          message: 'If an account exists, a magic link has been sent.',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await requestMagicLink('nonexistent@example.com');

        expect(result.message).toBeDefined();
      });

      it('returns consistent response regardless of email existence', async () => {
        const response: MagicLinkResponse = {
          message: 'If an account exists, a magic link has been sent.',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result1 = await requestMagicLink('existing@example.com');
        const result2 = await requestMagicLink('nonexistent@example.com');

        expect(result1.message).toBe(result2.message);
      });
    });

    describe('verifyMagicLink', () => {
      it('sends GET request with token', async () => {
        const response: VerifyMagicLinkResponse = {
          user: {
            id: '123',
            email: 'user@example.com',
            email_verified: true,
          },
          access_token: 'access_token_value',
          refresh_token: 'refresh_token_value',
          expires_at: '2025-01-01T00:00:00Z',
          token_type: 'Bearer',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        await verifyMagicLink('valid_token_123');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain('/auth/verify');
        expect(url).toContain('token=valid_token_123');
      });

      it('URL encodes the token', async () => {
        const response: VerifyMagicLinkResponse = {
          user: {
            id: '123',
            email: 'user@example.com',
            email_verified: true,
          },
          access_token: 'access',
          refresh_token: 'refresh',
          expires_at: '2025-01-01T00:00:00Z',
          token_type: 'Bearer',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        await verifyMagicLink('token+with=special&chars');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain(encodeURIComponent('token+with=special&chars'));
      });

      it('returns user and tokens on success', async () => {
        const response: VerifyMagicLinkResponse = {
          user: {
            id: '123',
            email: 'user@example.com',
            email_verified: true,
            stripe_customer_id: 'cus_123',
          },
          access_token: 'access_token',
          refresh_token: 'refresh_token',
          expires_at: '2025-01-01T00:00:00Z',
          token_type: 'Bearer',
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await verifyMagicLink('valid_token');

        expect(result.user.id).toBe('123');
        expect(result.user.email).toBe('user@example.com');
        expect(result.access_token).toBe('access_token');
        expect(result.refresh_token).toBe('refresh_token');
      });

      it('throws on invalid token', async () => {
        fetchMock.mockResolvedValue(mockResponses.error(400, 'Invalid or expired token'));

        await expect(verifyMagicLink('invalid_token')).rejects.toBeInstanceOf(ApiError);
      });
    });

    describe('refreshUserTokens', () => {
      it('sends POST request with refresh token', async () => {
        const tokens: UserAuthTokens = {
          access_token: 'new_access',
          refresh_token: 'new_refresh',
          expires_at: '2025-01-01T00:00:00Z',
          token_type: 'Bearer',
        };
        fetchMock.mockResolvedValue(mockResponses.success(tokens));

        await refreshUserTokens('old_refresh_token');

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(parseJsonBody(options.body)).toEqual({
          refresh_token: 'old_refresh_token',
        });
      });

      it('returns new tokens', async () => {
        const tokens: UserAuthTokens = {
          access_token: 'new_access',
          refresh_token: 'new_refresh',
          expires_at: '2025-01-01T00:00:00Z',
          token_type: 'Bearer',
        };
        fetchMock.mockResolvedValue(mockResponses.success(tokens));

        const result = await refreshUserTokens('old_refresh_token');

        expect(result.access_token).toBe('new_access');
        expect(result.refresh_token).toBe('new_refresh');
      });

      it('throws on invalid refresh token', async () => {
        fetchMock.mockResolvedValue(mockResponses.unauthorized('Invalid refresh token'));

        await expect(refreshUserTokens('invalid_token')).rejects.toBeInstanceOf(ApiError);
      });
    });

    describe('userLogout', () => {
      it('sends POST request to logout endpoint', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await userLogout();

        const [url, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(url).toContain('/auth/logout');
      });
    });

    describe('getUserMe', () => {
      it('returns current user information', async () => {
        const response: UserAuthMeResponse = {
          user: {
            id: '123',
            email: 'user@example.com',
            email_verified: true,
            stripe_customer_id: 'cus_123',
            created_at: '2024-01-01T00:00:00Z',
            last_login_at: '2024-06-01T00:00:00Z',
          },
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await getUserMe();

        expect(result.user.id).toBe('123');
        expect(result.user.email).toBe('user@example.com');
        expect(result.user.email_verified).toBe(true);
      });

      it('handles user without stripe customer', async () => {
        const response: UserAuthMeResponse = {
          user: {
            id: '123',
            email: 'user@example.com',
            email_verified: false,
          },
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await getUserMe();

        expect(result.user.stripe_customer_id).toBeUndefined();
      });

      it('throws when not authenticated', async () => {
        fetchMock.mockResolvedValue(mockResponses.unauthorized());

        await expect(getUserMe()).rejects.toBeInstanceOf(ApiError);
      });
    });
  });
});
