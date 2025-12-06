import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { AdminLogin } from './AdminLogin';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { adminLogin, checkAdminSession } from '../../../shared/api';

const { mockAdminLogin, mockCheckAdminSession } = vi.hoisted(() => ({
  mockAdminLogin: vi.fn(),
  mockCheckAdminSession: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  adminLogin: mockAdminLogin,
  checkAdminSession: mockCheckAdminSession,
}));

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>
  );
};

describe('AdminLogin [REQ:ADMIN-AUTH]', () => {
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();

    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    mockAdminLogin.mockResolvedValue({ authenticated: true, email: 'admin@test.com' });

    // Mock location to avoid session check trigger
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin/login' };
  });

  afterEach(() => {
    window.location = originalLocation;
  });

  it('[REQ:ADMIN-AUTH] should render login form with email and password fields', () => {
    renderWithRouter(<AdminLogin />);

    expect(screen.getByLabelText(/Email Address/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Password/i)).toBeInTheDocument();
    expect(screen.getByTestId('admin-login-submit')).toBeInTheDocument();
  });

  it('[REQ:ADMIN-AUTH] should call login API with email and password on form submit', async () => {
    const user = userEvent.setup();
    mockAdminLogin.mockResolvedValue({ authenticated: true, email: 'admin@test.com' });
    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    await user.type(emailInput, 'admin@test.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockAdminLogin).toHaveBeenCalledWith('admin@test.com', 'password123');
    });
  });

  it('[REQ:ADMIN-AUTH] should navigate to admin home on successful login', async () => {
    const user = userEvent.setup();
    mockAdminLogin.mockResolvedValue({ authenticated: true, email: 'admin@test.com' });

    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    await user.type(emailInput, 'admin@test.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/admin');
    });
  });

  it('[REQ:ADMIN-AUTH] should display error message on login failure', async () => {
    const user = userEvent.setup();
    mockAdminLogin.mockRejectedValue(new Error('Invalid'));

    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    await user.type(emailInput, 'wrong@test.com');
    await user.type(passwordInput, 'wrongpassword');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByTestId('admin-login-error')).toBeInTheDocument();
      expect(screen.getByText('Invalid email or password')).toBeInTheDocument();
    });
  });

  it('should show loading state while logging in', async () => {
    const user = userEvent.setup();
    mockAdminLogin.mockImplementation(() => new Promise(() => {})); // Never resolves

    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    await user.type(emailInput, 'admin@test.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Logging in...')).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });
  });

  it('should clear error message on form resubmit', async () => {
    const user = userEvent.setup();
    mockAdminLogin
      .mockRejectedValueOnce(new Error('Invalid'))
      .mockResolvedValueOnce({ authenticated: true, email: 'admin@test.com' });

    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    // First attempt - fails
    await user.type(emailInput, 'wrong@test.com');
    await user.type(passwordInput, 'wrongpassword');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByTestId('admin-login-error')).toBeInTheDocument();
    });

    // Second attempt - succeeds
    await user.clear(emailInput);
    await user.clear(passwordInput);
    await user.type(emailInput, 'admin@test.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.queryByTestId('admin-login-error')).not.toBeInTheDocument();
    });
  });

  it('should display security notice about bcrypt hashing', () => {
    renderWithRouter(<AdminLogin />);

    expect(
      screen.getByText(/Secured with bcrypt password hashing and httpOnly cookies/i)
    ).toBeInTheDocument();
  });

  it('should have proper input types for email and password', () => {
    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');

    expect(emailInput).toHaveAttribute('type', 'email');
    expect(passwordInput).toHaveAttribute('type', 'password');
  });
});
