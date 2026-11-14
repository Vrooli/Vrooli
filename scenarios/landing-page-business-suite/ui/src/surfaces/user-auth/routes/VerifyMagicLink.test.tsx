import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { UserAuthProvider } from '../../../app/providers/UserAuthProvider';
import { useUserAuth } from '../../../app/providers/useUserAuth';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { VerifyMagicLink } from './VerifyMagicLink';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getUserMe: vi.fn(),
    previewSignIn: vi.fn(),
    verifyMagicLink: vi.fn(),
    authorizeNativeApp: vi.fn(),
    issueDesktopLink: vi.fn(),
    listBusinessAccounts: vi.fn(),
  };
});

const getUserMe = vi.mocked(api.getUserMe);
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

function SessionWitness() {
  const auth = useUserAuth();
  return <output aria-label="Shared buyer session">{auth.isAuthenticated ? auth.user?.email : 'signed out'}</output>;
}

function renderVerify(token = 'valid-token', redirectTo = vi.fn()) {
  renderWithProviders(<MemoryRouter initialEntries={[`/auth/verify?token=${token}`]}><UserAuthProvider><VerifyMagicLink redirectTo={redirectTo} /><SessionWitness /></UserAuthProvider></MemoryRouter>);
  return redirectTo;
}

describe('VerifyMagicLink', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    getUserMe.mockResolvedValue({ user: signedIn.user });
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

  it('synchronizes the shared browser session after confirming a magic link', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockResolvedValue(signedIn);
    renderVerify();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('signed out');
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    await waitFor(() => { expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('buyer@example.com'); });
    expect(getUserMe).toHaveBeenCalledTimes(1);
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
  });

  it('retries only the existing session when a consumed magic link cannot load it', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockResolvedValue(signedIn);
    getUserMe.mockRejectedValueOnce(new Error('session read interrupted')).mockResolvedValueOnce({ user: signedIn.user });
    renderVerify();
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByRole('heading', { name: 'Sign-in didn’t finish' })).toBeInTheDocument();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('signed out');
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    await waitFor(() => { expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('buyer@example.com'); });
    expect(getUserMe).toHaveBeenCalledTimes(2);
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
    expect(previewSignIn).toHaveBeenCalledTimes(1);
  });

  it('rejects a different session after successful link verification and preserves the verified email on session-only retries', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockResolvedValue(signedIn);
    const otherUser = { ...signedIn.user, id: 'other-user', email: 'other@example.com' };
    getUserMe.mockResolvedValueOnce({ user: otherUser }).mockResolvedValueOnce({ user: otherUser }).mockResolvedValueOnce({ user: signedIn.user });
    renderVerify();
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByRole('heading', { name: 'Sign-in didn’t finish' })).toBeInTheDocument();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('signed out');
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('heading', { name: 'Sign-in didn’t finish' })).toBeInTheDocument();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('signed out');
    expect(getUserMe).toHaveBeenCalledTimes(2);
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('buyer@example.com');
    expect(getUserMe).toHaveBeenCalledTimes(3);
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
  });

  it('offers an explicit current-account check after a lost link response without claiming link success', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockRejectedValueOnce(new api.ApiError('response lost', 'network'));
    getUserMe.mockResolvedValueOnce({ user: { ...signedIn.user, id: 'other-user', email: 'other@example.com' } });
    renderVerify();
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByRole('link', { name: 'Check your current account' })).toHaveAttribute('href', '/account');
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    expect(screen.getByLabelText('Shared buyer session')).toHaveTextContent('other@example.com');
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
    expect(getUserMe).toHaveBeenCalledTimes(1);
  });

  it.each(['/account/security', 'https://other.example/'])('honors a safe magic-link destination and rejects an external one: %s', async (next) => {
    previewSignIn.mockResolvedValue(browserPreview);
    verifyMagicLink.mockResolvedValue(signedIn);
    renderWithProviders(<MemoryRouter initialEntries={[`/auth/verify?token=valid-token&next=${encodeURIComponent(next)}`]}>
      <UserAuthProvider><Routes>
        <Route path="/auth/verify" element={<VerifyMagicLink />} />
        <Route path="/account/security" element={<h1>Requested security</h1>} />
        <Route path="/account" element={<h1>Default account</h1>} />
      </Routes></UserAuthProvider>
    </MemoryRouter>);
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    expect(await screen.findByRole('heading', { name: next.startsWith('/') ? 'Requested security' : 'Default account' }, { timeout: 2500 })).toBeInTheDocument();
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
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

  it('does not navigate back after a pending confirmation is abandoned', async () => {
    previewSignIn.mockResolvedValue(browserPreview);
    let resolveProof!: (value: api.SignInResult) => void;
    verifyMagicLink.mockReturnValueOnce(new Promise((resolve) => { resolveProof = resolve; }));
    renderWithProviders(<MemoryRouter initialEntries={['/auth/verify?token=valid-token&next=%2Faccount%2Fsecurity']}>
      <UserAuthProvider><Routes>
        <Route path="/auth/verify" element={<VerifyMagicLink />} />
        <Route path="/auth/login" element={<h1>Chosen email sign-in</h1>} />
        <Route path="/account/security" element={<h1>Unwanted delayed destination</h1>} />
      </Routes></UserAuthProvider>
    </MemoryRouter>);
    fireEvent.click(await screen.findByTestId('confirm-sign-in'));
    fireEvent.click(screen.getByRole('link', { name: 'Use a different email' }));
    await act(async () => { resolveProof(signedIn); await Promise.resolve(); });
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 1000)); });
    expect(screen.getByRole('heading', { name: 'Chosen email sign-in' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Unwanted delayed destination' })).not.toBeInTheDocument();
    expect(verifyMagicLink).toHaveBeenCalledTimes(1);
  });
});
