import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { VerifyMagicLink } from './VerifyMagicLink';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return { ...actual, verifyMagicLink: vi.fn() };
});

const verifyMagicLink = vi.mocked(api.verifyMagicLink);
const validResponse: api.VerifyMagicLinkResponse = {
  access_token: 'access-token',
  refresh_token: 'refresh-token',
  expires_at: '2026-12-31T00:00:00Z',
  token_type: 'Bearer',
  user: { id: 'user-1', email: 'buyer@example.com', email_verified: true },
};

function renderVerify(token = 'valid-token', redirectTo?: (url: string) => void) {
  return renderWithProviders(<MemoryRouter initialEntries={[`/auth/verify?token=${token}`]}><VerifyMagicLink redirectTo={redirectTo} /></MemoryRouter>);
}

describe('VerifyMagicLink', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('rejects an absent verification token before network access', async () => {
    renderVerify('');
    expect(await screen.findByText('No verification token provided')).toBeInTheDocument();
    expect(verifyMagicLink).not.toHaveBeenCalled();
    expect(screen.getByRole('link', { name: /request new link/i })).toHaveAttribute('href', '/auth/login');
  });

  it('does not redirect authentication tokens to an untrusted callback URL', async () => {
    verifyMagicLink.mockResolvedValue(validResponse);
    sessionStorage.setItem('auth_callback_params', JSON.stringify({ redirect_uri: 'https://attacker.example/callback', app: 'Untrusted', state: 'nonce' }));
    renderVerify();

    expect(await screen.findByText('Verification successful')).toBeInTheDocument();
    expect(sessionStorage.getItem('auth_callback_params')).toBeNull();
    expect(window.location.href).not.toContain('access-token');
  });

  it('recovers safely from malformed persisted callback state', async () => {
    verifyMagicLink.mockResolvedValue(validResponse);
    sessionStorage.setItem('auth_callback_params', '{not-json');
    renderVerify();

    expect(await screen.findByText('Verification successful')).toBeInTheDocument();
    expect(window.location.href).not.toContain('access-token');
  });

  it('recovers safely when persisted callback state lacks required fields', async () => {
    verifyMagicLink.mockResolvedValue(validResponse);
    sessionStorage.setItem('auth_callback_params', JSON.stringify({ app: 'Desktop' }));
    renderVerify();

    expect(await screen.findByText('Verification successful')).toBeInTheDocument();
    expect(window.location.href).not.toContain('access-token');
  });

  it('sends a one-use PKCE authorization request instead of tokens through the callback', async () => {
    const redirectTo = vi.fn();
    sessionStorage.setItem('auth_callback_params', JSON.stringify({
      redirect_uri: 'http://127.0.0.1:43123/callback',
      app: 'Desktop',
      state: 'nonce',
      code_challenge: 'challenge',
      code_challenge_method: 'S256',
    }));
    renderVerify('valid-token', redirectTo);

    expect(await screen.findByText('Signed in!')).toBeInTheDocument();
    expect(screen.getByText('Redirecting you back to the app...')).toBeInTheDocument();
    expect(verifyMagicLink).not.toHaveBeenCalled();
    expect(redirectTo).toHaveBeenCalledWith(expect.stringContaining('/api/v1/auth/authorize'));
    expect(redirectTo).toHaveBeenCalledWith(expect.stringContaining('code_challenge=challenge'));
    expect(redirectTo).toHaveBeenCalledWith(expect.not.stringContaining('access-token'));
    expect(redirectTo).toHaveBeenCalledWith(expect.not.stringContaining('refresh-token'));
    expect(sessionStorage.getItem('auth_callback_params')).toBeNull();
  });

  it('offers a retry path for network failures and completes after recovery', async () => {
    verifyMagicLink.mockRejectedValueOnce(new api.ApiError('offline', 'network'));
    renderVerify();
    expect(await screen.findByText('Unable to reach the server. Please check your connection.')).toBeInTheDocument();

    verifyMagicLink.mockResolvedValueOnce(validResponse);
    fireEvent.click(screen.getByRole('button', { name: /try again/i }));
    await waitFor(() => {
      expect(verifyMagicLink).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('Verification successful')).toBeInTheDocument();
  });

  it.each([
    ['expired', 'This link has expired', 'Request new link'],
    ['already used', 'This link was already used', 'Request new link'],
    ['invalid', 'This link is invalid', 'Request new link'],
    ['unexpected failure', 'An unexpected failure occurred', null],
  ] as const)('classifies %s verification failures without exposing credentials', async (message, userMessage, expectedAction) => {
    const error = new api.ApiError(message, 'unknown', undefined, userMessage);
    verifyMagicLink.mockRejectedValue(error);
    renderVerify();

    expect(await screen.findByText(userMessage)).toBeInTheDocument();
    if (expectedAction) {
      expect(screen.getByRole('link', { name: expectedAction })).toHaveAttribute('href', '/auth/login');
    } else {
      expect(screen.queryByRole('link', { name: /request new link/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /try again/i })).not.toBeInTheDocument();
    }
  });
});
