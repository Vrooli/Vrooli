import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { AdminLogin } from './AdminLogin';
import { AuthProvider } from '../../../contexts/AuthContext';

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
      <AuthProvider>
        {component}
      </AuthProvider>
    </BrowserRouter>
  );
};

describe('AdminLogin [REQ:ADMIN-AUTH]', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();

    // Mock fetch for session checks
    global.fetch = vi.fn();
    vi.mocked(global.fetch).mockResolvedValue({
      ok: false,
    } as Response);

    // Mock location to avoid session check trigger
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin/login' };
  });

  afterEach(() => {
    global.fetch = originalFetch;
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
    const mockLoginFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ email: 'admin@test.com' }),
    } as Response);

    global.fetch = mockLoginFetch;

    renderWithRouter(<AdminLogin />);

    const emailInput = screen.getByTestId('admin-login-email');
    const passwordInput = screen.getByTestId('admin-login-password');
    const submitButton = screen.getByTestId('admin-login-submit');

    await user.type(emailInput, 'admin@test.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockLoginFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/admin/login'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: 'admin@test.com', password: 'password123' }),
          credentials: 'include',
        })
      );
    });
  });

  it('[REQ:ADMIN-AUTH] should navigate to admin home on successful login', async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ email: 'admin@test.com' }),
    } as Response);

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
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
    } as Response);

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
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {})); // Never resolves

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
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: false } as Response) // Session check (from AuthProvider useEffect)
      .mockResolvedValueOnce({ ok: false } as Response) // First login attempt fails
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ email: 'admin@test.com' }),
      } as Response); // Second login attempt succeeds

    global.fetch = mockFetch;

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
