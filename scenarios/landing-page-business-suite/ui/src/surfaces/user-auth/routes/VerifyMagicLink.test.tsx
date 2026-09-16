import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { VerifyMagicLink } from './VerifyMagicLink';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    previewSignIn: vi.fn(),
    verifyMagicLink: vi.fn(),
    authorizeNativeApp: vi.fn(),
    issueDesktopLink: vi.fn(),
    listBusinessAccounts: vi.fn(),
  };
});

const previewSignIn = vi.mocked(api.previewSignIn);
const verifyMagicLink = vi.mocked(api.verifyMagicLink);
const authorizeNativeApp = vi.mocked(api.authorizeNativeApp);
const issueDesktopLink = vi.mocked(api.issueDesktopLink);
const listBusinessAccounts = vi.mocked(api.listBusinessAccounts);

const signedIn: api.SignInResult = {
  access_token: 'access-token', refresh_token: 'refresh-token', expires_at: '2030-01-01T00:00:00Z', token_type: 'Bearer',
  user: { id: 'user-1', email: 'buyer@example.com', email_verified: true },
};

const browserPreview: api.SignInPreview = { email_hint: 'bu•••@example.com', expires_at: '2030-01-01T00:15:00Z', flow: 'browser', same_browser: true };

const desktopContext: api.SignInContext = {
  desktop_link: true, app: 'Desktop', resource: 'demo', audience: 'scenario:demo', installation_id: 'inst', scopes: ['demo:read'],
  redirect_uri: 'http://127.0.0.1:43111/cb', code_challenge: 'abc', code_challenge_method: 'S256', state: 's1',
};

function renderVerify(token = 'valid-token', redirectTo = vi.fn()) {
  renderWithProviders(<MemoryRouter initialEntries={[`/auth/verify?token=${token}`]}><VerifyMagicLink redirectTo={redirectTo} /></MemoryRouter>);
  return redirectTo;
}

describe('VerifyMagicLink', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('rejects an absent token before any network access', async () => {
    renderVerify('');
    expect(await screen.findByRole('heading', { name: 'This link doesn’t work' })).toBeInTheDocument();
    expect(previewSignIn).not.toHaveBeenCalled();
    expect(screen.getByRole('link', { name: /get a new code/i })).toHaveAttribute('href', '/auth/login');
  });

  it('never consumes the link until the person confirms, so link scanners cannot burn it', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockResolvedValue(signedIn);
    renderVerify();

    expect(await screen.findByText('bu•••@example.com')).toBeInTheDocument();
    expect(verifyMagicLink).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('confirm-sign-in'));
    await waitFor(() => { expect(verifyMagicLink).toHaveBeenCalledWith('valid-token', expect.any(String)); });
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
  });

  it('resumes a native app from server-held context and sends only a one-use code to the callback', async () => {
    const context: api.SignInContext = { app: 'Desktop', redirect_uri: 'http://127.0.0.1:43111/callback', code_challenge: 'abc', code_challenge_method: 'S256', state: 'nonce' };
    previewSignIn.mockResolvedValue({ ...browserPreview, flow: 'native_app', context });
    authorizeNativeApp.mockResolvedValue('http://127.0.0.1:43111/callback?code=one-use&state=nonce');
    const redirect = renderVerify();

    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    await waitFor(() => { expect(redirect).toHaveBeenCalledWith('http://127.0.0.1:43111/callback?code=one-use&state=nonce'); });
    expect(authorizeNativeApp).toHaveBeenCalledWith({ token: 'valid-token', browserBinding: expect.any(String) as string }, context);
    expect(verifyMagicLink).not.toHaveBeenCalled();
  });

  it('refuses a server-held callback that leaves this computer', async () => {
    const context: api.SignInContext = { app: 'Evil', redirect_uri: 'https://attacker.example/cb', code_challenge: 'abc', code_challenge_method: 'S256' };
    previewSignIn.mockResolvedValue({ ...browserPreview, flow: 'native_app', context });
    const redirect = renderVerify();

    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByText(/not on this computer/)).toBeInTheDocument();
    expect(authorizeNativeApp).not.toHaveBeenCalled();
    expect(redirect).not.toHaveBeenCalled();
  });

  it('explains how to finish an app sign-in started in another browser and offers the website instead', async () => {
    previewSignIn.mockResolvedValue({ ...browserPreview, flow: 'desktop_link', same_browser: false });
    verifyMagicLink.mockResolvedValue(signedIn);
    renderVerify();

    expect(await screen.findByText(/type the 6-digit code from the email into that window/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /sign in to the website/i }));
    await waitFor(() => { expect(verifyMagicLink).toHaveBeenCalled(); });
    expect(listBusinessAccounts).not.toHaveBeenCalled();
  });

  it('asks which account to connect when the person has several', async () => {
    previewSignIn.mockResolvedValue({ ...browserPreview, flow: 'desktop_link', context: desktopContext });
    verifyMagicLink.mockResolvedValue(signedIn);
    listBusinessAccounts.mockResolvedValue([
      { id: 'acct-1', display_name: 'Personal', role: 'owner' },
      { id: 'acct-2', display_name: 'Studio', role: 'member' },
    ]);
    issueDesktopLink.mockResolvedValue({ code: 'link-code', expires_at: '', installation_id: 'inst', resource: 'demo', audience: 'scenario:demo', scopes: ['demo:read'] });
    const redirect = renderVerify();

    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByTestId('desktop-account-selection')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Personal/ }));
    await waitFor(() => { expect(redirect).toHaveBeenCalledWith('http://127.0.0.1:43111/cb?code=link-code&state=s1'); });
    expect(redirect.mock.calls[0]?.[0]).not.toContain('access-token');
  });

  it('retries a network failure and then completes', async () => {
    previewSignIn.mockRejectedValueOnce(new api.ApiError('offline', 'network')).mockResolvedValue(browserPreview);
    renderVerify();

    fireEvent.click(await screen.findByRole('button', { name: /try again/i }));
    expect(await screen.findByTestId('confirm-sign-in')).toBeInTheDocument();
  });

  it.each([
    ['token_expired', 'This link has expired'],
    ['token_used', 'This link was already used'],
    ['token_invalid', 'This link doesn’t work'],
  ] as const)('explains %s without exposing credentials', async (reason, heading) => {
    previewSignIn.mockRejectedValue(new api.ApiError('secret-detail', 'unauthorized', 401, 'x', false, { reason }));
    renderVerify();
    expect(await screen.findByRole('heading', { name: heading })).toBeInTheDocument();
    expect(screen.queryByText(/secret-detail|valid-token/)).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: /get a new code/i })).toBeInTheDocument();
  });
});
