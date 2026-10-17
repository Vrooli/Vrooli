import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Navigate, Route, Routes } from 'react-router-dom';
import { UserAuthProvider } from '../../../app/providers/UserAuthProvider';
import { useUserAuth } from '../../../app/providers/useUserAuth';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { isValidEmail, UserLogin } from './UserLogin';
import * as webauthn from '../lib/webauthn';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getUserMe: vi.fn(),
    refreshSessionOnce: vi.fn(),
    requestMagicLink: vi.fn(),
    verifySignInCode: vi.fn(),
    authorizeNativeApp: vi.fn(),
    listBusinessAccounts: vi.fn(),
    issueDesktopLink: vi.fn(),
    getSignInDeliveryStatus: vi.fn(),
  };
});

const getUserMe = vi.mocked(api.getUserMe);
const requestMagicLink = vi.mocked(api.requestMagicLink);
const verifySignInCode = vi.mocked(api.verifySignInCode);
const authorizeNativeApp = vi.mocked(api.authorizeNativeApp);
const listBusinessAccounts = vi.mocked(api.listBusinessAccounts);
const issueDesktopLink = vi.mocked(api.issueDesktopLink);
const getSignInDeliveryStatus = vi.mocked(api.getSignInDeliveryStatus);

const signedIn: api.SignInResult = {
  access_token: 'a', refresh_token: 'r', expires_at: '2030-01-01T00:00:00Z', token_type: 'Bearer',
  user: { id: 'u1', email: 'buyer@example.com', email_verified: true },
};

function renderLogin(route = '/auth/login', redirectTo = vi.fn()) {
  renderWithProviders(<MemoryRouter initialEntries={[route]}><UserAuthProvider><UserLogin redirectTo={redirectTo} /></UserAuthProvider></MemoryRouter>);
  return redirectTo;
}

async function submitEmail(address = 'buyer@example.com') {
  fireEvent.change(screen.getByTestId('email-input'), { target: { value: address } });
  fireEvent.click(screen.getByTestId('submit-button'));
  return screen.findByRole('heading', { name: 'Check your email' });
}

describe('UserLogin', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
    localStorage.clear();
    getUserMe.mockResolvedValue({ user: signedIn.user });
    requestMagicLink.mockResolvedValue({ message: 'sent', expires_at: '2030-01-01T00:15:00Z' });
    getSignInDeliveryStatus.mockResolvedValue({ status: 'pending', reasonClass: '', expiresAt: undefined } as any);
  });

  it('rejects blank and malformed addresses at the shared validation boundary', () => {
    expect(isValidEmail('   ')).toBe(false);
    expect(isValidEmail('a@b')).toBe(false);
    expect(isValidEmail('a b@c.co')).toBe(false);
    expect(isValidEmail('buyer@example.com')).toBe(true);
  });

  it('validates missing and malformed email without calling the API', async () => {
    renderLogin();
    fireEvent.click(screen.getByTestId('submit-button'));
    expect(await screen.findByText('Email is required')).toBeInTheDocument();

    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'invalid-address' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    expect(await screen.findByText('Please enter a valid email address')).toBeInTheDocument();
    expect(requestMagicLink).not.toHaveBeenCalled();
  });

  it('requests a code with a normalized address and a persistent browser binding', async () => {
    renderLogin();
    await submitEmail(' Buyer@Example.COM ');
    expect(requestMagicLink).toHaveBeenCalledTimes(1);
    const [address, options] = requestMagicLink.mock.calls[0] ?? [];
    expect(address).toBe('buyer@example.com');
    expect(options?.browserBinding).toMatch(/^[0-9a-f]{48}$/);
    expect(options?.context).toBeUndefined();
    expect(screen.getByText('buyer@example.com')).toBeInTheDocument();
  });

  it('signs in with a pasted code and returns to a same-site destination', async () => {
    verifySignInCode.mockResolvedValue(signedIn);
    renderLogin('/auth/login?next=%2Fdownloads');
    await submitEmail();

    fireEvent.paste(screen.getByTestId('code-input'), { clipboardData: { getData: () => '123 456' } });

    await waitFor(() => { expect(verifySignInCode).toHaveBeenCalledWith('buyer@example.com', '123456', expect.any(String)); });
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
  });

  it('updates shared session state before entering the requested protected page', async () => {
    verifySignInCode.mockResolvedValue(signedIn);
    function ProtectedDestination() {
      const auth = useUserAuth();
      return auth.isAuthenticated
        ? <h1>Buyer security for {auth.user?.email}</h1>
        : <Navigate to="/auth/login" replace />;
    }
    renderWithProviders(<MemoryRouter initialEntries={['/auth/login?next=%2Faccount%2Fsecurity']}>
      <UserAuthProvider><Routes>
        <Route path="/auth/login" element={<UserLogin />} />
        <Route path="/account/security" element={<ProtectedDestination />} />
      </Routes></UserAuthProvider>
    </MemoryRouter>);
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    expect(await screen.findByRole('heading', { name: 'Buyer security for buyer@example.com' }, { timeout: 2500 })).toBeInTheDocument();
    expect(getUserMe).toHaveBeenCalledTimes(1);
    expect(verifySignInCode).toHaveBeenCalledTimes(1);
    expect(requestMagicLink).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId('email-input')).not.toBeInTheDocument();
  });

  it('holds success until the authoritative session read completes', async () => {
    let resolveSession!: (value: { user: api.UserAuthUser }) => void;
    getUserMe.mockReturnValueOnce(new Promise((resolve) => { resolveSession = resolve; }));
    verifySignInCode.mockResolvedValue(signedIn);
    renderLogin();
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    await waitFor(() => { expect(getUserMe).toHaveBeenCalledTimes(1); });
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    await act(async () => { resolveSession({ user: signedIn.user }); await Promise.resolve(); });
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
  });

  it('retries the existing session after a consumed code without a second verification or email', async () => {
    verifySignInCode.mockResolvedValue(signedIn);
    getUserMe.mockRejectedValueOnce(new Error('session read interrupted')).mockResolvedValueOnce({ user: signedIn.user });
    renderLogin();
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    expect(await screen.findByRole('heading', { name: 'Finish signing in' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
    expect(getUserMe).toHaveBeenCalledTimes(2);
    expect(verifySignInCode).toHaveBeenCalledTimes(1);
    expect(requestMagicLink).toHaveBeenCalledTimes(1);
  });

  it('checks the session instead of repeating verification after a lost verification response', async () => {
    verifySignInCode.mockRejectedValueOnce(new api.ApiError('response lost', 'network'));
    renderLogin();
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    expect(await screen.findByRole('heading', { name: 'You’re signed in' })).toBeInTheDocument();
    expect(getUserMe).toHaveBeenCalledTimes(1);
    expect(verifySignInCode).toHaveBeenCalledTimes(1);
    expect(requestMagicLink).toHaveBeenCalledTimes(1);
  });

  it('does not hand off a lost code response to a different existing account', async () => {
    verifySignInCode.mockRejectedValueOnce(new api.ApiError('response lost', 'network'));
    getUserMe.mockResolvedValueOnce({ user: { ...signedIn.user, id: 'other-user', email: 'other@example.com' } });
    renderLogin('/auth/login?next=%2Faccount%2Fsecurity');
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    expect(await screen.findByRole('heading', { name: 'Finish signing in' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    expect(verifySignInCode).toHaveBeenCalledTimes(1);
    expect(getUserMe).toHaveBeenCalledTimes(1);
  });

  it('lets a signed-in buyer dismiss the optional passkey prompt without enrollment', async () => {
    vi.spyOn(webauthn, 'isPasskeySupported').mockReturnValue(true);
    vi.spyOn(webauthn, 'isConditionalAvailable').mockResolvedValue(false);
    const register = vi.spyOn(api, 'beginPasskeyRegistration');
    verifySignInCode.mockResolvedValue(signedIn);
    renderLogin('/auth/login?next=%2Faccount%2Fsecurity');
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    fireEvent.click(await screen.findByRole('button', { name: 'Not now' }));
    expect(localStorage.getItem('lpbs.passkey.not-now-at')).toBeTruthy();
    expect(register).not.toHaveBeenCalled();
    expect(getUserMe).toHaveBeenCalledTimes(1);
  });

  it('shows provider delivery feedback in the code step and polls with the browser binding', async () => {
    getSignInDeliveryStatus.mockResolvedValue({ status: 'delivered', reasonClass: '', expiresAt: undefined } as any);
    renderLogin();
    await submitEmail();
    expect(await screen.findByText('Delivered to your mail server')).toBeInTheDocument();
    expect(getSignInDeliveryStatus).toHaveBeenCalledWith('buyer@example.com');
    expect(screen.getByText('Delivered to your mail server').closest('[aria-live="polite"]')).not.toBeNull();
  });

  it('clears the code and explains a wrong code without leaving the step', async () => {
    verifySignInCode.mockRejectedValue(new api.ApiError('bad', 'unauthorized', 401, 'x', false, { reason: 'code_invalid' }));
    renderLogin();
    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '999999' } });

    expect(await screen.findByText(/That code isn’t right/)).toBeInTheDocument();
    expect(screen.getByTestId('code-input')).toHaveValue('');
    expect(screen.getByRole('heading', { name: 'Check your email' })).toBeInTheDocument();
  });

  it('holds resend behind a cooldown, then sends again and keeps earlier codes valid', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      renderLogin();
      await submitEmail();
      const resend = screen.getByTestId('resend-button');
      expect(resend).toBeDisabled();
      expect(resend).toHaveTextContent('0:45');
      for (let second = 0; second < 46; second++) {
        await act(async () => { await vi.advanceTimersByTimeAsync(1_000); });
      }
      expect(screen.getByTestId('resend-button')).toBeEnabled();
      fireEvent.click(screen.getByTestId('resend-button'));
      expect(await screen.findByText(/Earlier codes still work/)).toBeInTheDocument();
      expect(requestMagicLink).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it('reports a delivery outage truthfully instead of claiming an email was sent', async () => {
    requestMagicLink.mockRejectedValue(new api.ApiError('down', 'server_error', 503, 'x', true, { reason: 'delivery_unavailable' }));
    renderLogin();
    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'buyer@example.com' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    expect(await screen.findByText(/couldn’t send an email, so no sign-in code was issued/)).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Check your email' })).not.toBeInTheDocument();
  });

  it.each([
    ['rate_limited', 'Too many sign-in requests. Wait a few minutes, then try again.'],
    ['network', 'We couldn’t reach the server. Check your connection and try again.'],
    ['unknown', 'Something went wrong on our side. Please try again.'],
  ] as const)('maps %s request failures to customer-safe guidance', async (type, message) => {
    requestMagicLink.mockRejectedValue(new api.ApiError('internal detail', type));
    renderLogin();
    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'buyer@example.com' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.queryByText('internal detail')).not.toBeInTheDocument();
  });

  it('stores native-app parameters with the request and completes through PKCE, never tokens', async () => {
    authorizeNativeApp.mockResolvedValue('http://127.0.0.1:43111/callback?code=one-use&state=nonce');
    const redirect = renderLogin('/auth/login?redirect_uri=http%3A%2F%2F127.0.0.1%3A43111%2Fcallback&app=Desktop&state=nonce&code_challenge=abc&code_challenge_method=S256');
    expect(screen.getByRole('heading', { name: 'Continue to Desktop' })).toBeInTheDocument();
    await submitEmail();
    expect(requestMagicLink.mock.calls[0]?.[1]?.context).toMatchObject({ redirect_uri: 'http://127.0.0.1:43111/callback', code_challenge: 'abc', state: 'nonce' });

    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    await waitFor(() => { expect(redirect).toHaveBeenCalledWith('http://127.0.0.1:43111/callback?code=one-use&state=nonce'); });
    expect(verifySignInCode).not.toHaveBeenCalled();
    expect(redirect.mock.calls[0]?.[0]).not.toContain('access');
  });

  it('refuses app callbacks that leave this computer before asking for an email', () => {
    renderLogin('/auth/login?redirect_uri=https%3A%2F%2Fattacker.example%2Fcb&code_challenge=abc&code_challenge_method=S256');
    expect(screen.getByRole('heading', { name: 'We can’t continue this sign-in' })).toBeInTheDocument();
    expect(screen.queryByTestId('email-input')).not.toBeInTheDocument();
  });

  it('shows desktop capabilities in plain language, then connects the chosen account', async () => {
    verifySignInCode.mockResolvedValue(signedIn);
    listBusinessAccounts.mockResolvedValue([
      { id: 'acct-1', display_name: 'Personal', role: 'owner' },
      { id: 'acct-2', display_name: 'Studio', role: 'member' },
    ]);
    issueDesktopLink.mockResolvedValue({ code: 'link-code', expires_at: '', installation_id: 'inst', resource: 'demo', audience: 'scenario:demo', scopes: ['demo:read'] });
    const redirect = renderLogin('/auth/login?desktop_link=true&app=Desktop&resource=demo&audience=scenario%3Ademo&installation_id=inst&scopes=demo%3Aread%2Cdemo%3Awrite&redirect_uri=http%3A%2F%2F127.0.0.1%3A43111%2Fcb&code_challenge=abc&code_challenge_method=S256&state=s1');

    const consent = screen.getByTestId('desktop-link-consent');
    expect(consent).toHaveTextContent('Desktop wants to connect to your account on this computer.');
    expect(consent).toHaveTextContent('Read demo');
    expect(consent).toHaveTextContent('Change demo');

    await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    fireEvent.click(await screen.findByRole('button', { name: /Studio/ }));

    await waitFor(() => { expect(redirect).toHaveBeenCalledWith('http://127.0.0.1:43111/cb?code=link-code&state=s1'); });
    expect(issueDesktopLink).toHaveBeenCalledWith(expect.objectContaining({ business_account_id: 'acct-2', code_challenge: 'abc' }));
  });

  it('lets a customer go back and use a different email', async () => {
    renderLogin();
    await submitEmail();
    fireEvent.click(screen.getByRole('button', { name: /use a different email/i }));
    expect(await screen.findByTestId('email-input')).toHaveValue('buyer@example.com');
  });

  it('keeps the email choice after a delayed consumed code resolves', async () => {
    let resolveProof!: (value: api.SignInResult) => void;
    verifySignInCode.mockReturnValueOnce(new Promise((resolve) => { resolveProof = resolve; }));
    renderLogin();await submitEmail();
    fireEvent.change(screen.getByTestId('code-input'), { target: { value: '123456' } });
    fireEvent.click(screen.getByRole('button', { name: 'Use a different email' }));
    await act(async () => { resolveProof(signedIn); await Promise.resolve(); });
    expect(screen.getByTestId('email-input')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'You’re signed in' })).not.toBeInTheDocument();
    expect(verifySignInCode).toHaveBeenCalledTimes(1);
    expect(requestMagicLink).toHaveBeenCalledTimes(1);
  });
});
