import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { isValidEmail, UserLogin } from './UserLogin';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    requestMagicLink: vi.fn(),
    verifySignInCode: vi.fn(),
    authorizeNativeApp: vi.fn(),
    listBusinessAccounts: vi.fn(),
    issueDesktopLink: vi.fn(),
  };
});

const requestMagicLink = vi.mocked(api.requestMagicLink);
const verifySignInCode = vi.mocked(api.verifySignInCode);
const authorizeNativeApp = vi.mocked(api.authorizeNativeApp);
const listBusinessAccounts = vi.mocked(api.listBusinessAccounts);
const issueDesktopLink = vi.mocked(api.issueDesktopLink);

const signedIn: api.SignInResult = {
  access_token: 'a', refresh_token: 'r', expires_at: '2030-01-01T00:00:00Z', token_type: 'Bearer',
  user: { id: 'u1', email: 'buyer@example.com', email_verified: true },
};

function renderLogin(route = '/auth/login', redirectTo = vi.fn()) {
  renderWithProviders(<MemoryRouter initialEntries={[route]}><UserLogin redirectTo={redirectTo} /></MemoryRouter>);
  return redirectTo;
}

async function submitEmail(address = 'buyer@example.com') {
  fireEvent.change(screen.getByTestId('email-input'), { target: { value: address } });
  fireEvent.click(screen.getByTestId('submit-button'));
  return screen.findByRole('heading', { name: 'Check your email' });
}

describe('UserLogin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    requestMagicLink.mockResolvedValue({ message: 'sent', expires_at: '2030-01-01T00:15:00Z' });
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
    expect(await screen.findByText(/couldn’t send the email right now/)).toBeInTheDocument();
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
});
