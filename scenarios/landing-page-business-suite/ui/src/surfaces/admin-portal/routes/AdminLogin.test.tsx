import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { AdminLogin } from './AdminLogin';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';

const { mockAdminLogin, mockCheckAdminSession } = vi.hoisted(() => ({
  mockAdminLogin: vi.fn(),
  mockCheckAdminSession: vi.fn(),
}));

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual('../../../shared/api');
  return { ...actual, adminLogin: mockAdminLogin, checkAdminSession: mockCheckAdminSession };
});

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

async function renderLogin() {
  render(<BrowserRouter><AdminAuthProvider><AdminLogin /></AdminAuthProvider></BrowserRouter>);
  await waitFor(() => { expect(mockCheckAdminSession).toHaveBeenCalled(); });
}

async function submitCredentials(email = 'admin@test.com', password = 'password123') {
  const user = userEvent.setup();
  await user.type(screen.getByTestId('admin-login-email'), email);
  await user.type(screen.getByTestId('admin-login-password'), password);
  await user.click(screen.getByTestId('admin-login-submit'));
  return user;
}

describe('AdminLogin [REQ:ADMIN-AUTH]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    mockAdminLogin.mockResolvedValue({ authenticated: true, email: 'admin@test.com' });
    window.history.replaceState({}, '', '/admin/login');
  });

  afterEach(() => {
    window.history.replaceState({}, '', '/');
  });

  it('[REQ:ADMIN-AUTH] renders labelled email and password fields with password-manager hints', async () => {
    await renderLogin();
    expect(screen.getByLabelText('Email address')).toHaveAttribute('autocomplete', 'username');
    expect(screen.getByLabelText('Password')).toHaveAttribute('autocomplete', 'current-password');
    expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password');
    expect(screen.getByTestId('admin-login-submit')).toBeInTheDocument();
  });

  it('[REQ:ADMIN-AUTH] signs in with email and password and opens the admin home', async () => {
    await renderLogin();
    await submitCredentials();
    await waitFor(() => { expect(mockAdminLogin).toHaveBeenCalledWith('admin@test.com', 'password123', ''); });
    expect(mockNavigate).toHaveBeenCalledWith('/admin');
  });

  it('reveals the password on request without submitting', async () => {
    await renderLogin();
    fireEvent.change(screen.getByTestId('admin-login-password'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Show password' }));
    expect(screen.getByTestId('admin-login-password')).toHaveAttribute('type', 'text');
    expect(mockAdminLogin).not.toHaveBeenCalled();
  });

  it('[REQ:ADMIN-SECURITY] asks for an authenticator code when two-factor is on, then signs in', async () => {
    mockAdminLogin
      .mockRejectedValueOnce(new ConnectError('Enter the code', Code.FailedPrecondition))
      .mockResolvedValueOnce({ authenticated: true, email: 'admin@test.com' });
    await renderLogin();
    await submitCredentials();

    expect(await screen.findByRole('heading', { name: 'Two-factor verification' })).toBeInTheDocument();
    fireEvent.change(screen.getByTestId('admin-totp-code'), { target: { value: '123456' } });
    await waitFor(() => { expect(mockAdminLogin).toHaveBeenLastCalledWith('admin@test.com', 'password123', '123456'); });
    expect(mockNavigate).toHaveBeenCalledWith('/admin');
  });

  it('accepts a recovery code instead of an authenticator code', async () => {
    mockAdminLogin
      .mockRejectedValueOnce(new ConnectError('Enter the code', Code.FailedPrecondition))
      .mockResolvedValueOnce({ authenticated: true, email: 'admin@test.com' });
    await renderLogin();
    const user = await submitCredentials();
    await user.click(await screen.findByRole('button', { name: /use a recovery code/i }));
    await user.type(screen.getByTestId('admin-recovery-code'), 'abcde-fghjk');
    await user.click(screen.getByTestId('admin-mfa-submit'));
    await waitFor(() => { expect(mockAdminLogin).toHaveBeenLastCalledWith('admin@test.com', 'password123', 'abcde-fghjk'); });
  });

  it.each([
    [Code.Unauthenticated, 'That email and password don’t match an administrator account.'],
    [Code.ResourceExhausted, 'Too many failed attempts. For your security, sign-in is paused for 15 minutes.'],
    [Code.Internal, 'The server couldn’t complete sign-in. Please try again.'],
  ])('maps connect code %s to operator guidance without server details', async (code, message) => {
    mockAdminLogin.mockRejectedValue(new ConnectError('internal stack detail', code));
    await renderLogin();
    await submitCredentials();
    expect(await screen.findByTestId('admin-login-error')).toHaveTextContent(message);
    expect(screen.queryByText(/internal stack detail/)).not.toBeInTheDocument();
  });

  it('disables sign-in while locked out', async () => {
    mockAdminLogin.mockRejectedValue(new ConnectError('locked', Code.ResourceExhausted));
    await renderLogin();
    await submitCredentials();
    await screen.findByTestId('admin-login-error');
    expect(screen.getByTestId('admin-login-submit')).toBeDisabled();
  });

  it('offers a retry for network failures and resubmits the same credentials', async () => {
    mockAdminLogin.mockRejectedValueOnce(new TypeError('Failed to fetch')).mockResolvedValueOnce({ authenticated: true, email: 'admin@test.com' });
    await renderLogin();
    const user = await submitCredentials();
    await user.click(await screen.findByRole('button', { name: 'Retry' }));
    await waitFor(() => { expect(mockAdminLogin).toHaveBeenCalledTimes(2); });
    expect(mockAdminLogin).toHaveBeenLastCalledWith('admin@test.com', 'password123', '');
  });
});
