import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils/renderWithProviders';
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

function renderVerify(token = 'valid-token') {
  return renderWithProviders(<VerifyMagicLink />, { route: `/auth/verify?token=${token}` });
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
});
