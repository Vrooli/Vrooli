import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactElement } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ProfileSettings } from './ProfileSettings';

const { mockGetAdminProfile, mockUpdateAdminProfile } = vi.hoisted(() => ({
  mockGetAdminProfile: vi.fn(),
  mockUpdateAdminProfile: vi.fn(),
}));

vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-mock" />,
}));

vi.mock('../../../shared/api', () => ({
  getAdminProfile: mockGetAdminProfile,
  updateAdminProfile: mockUpdateAdminProfile,
  adminLogout: vi.fn(),
}));

const renderWithRouter = (component: ReactElement) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('ProfileSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAdminProfile.mockResolvedValue({
      email: 'admin@localhost',
      is_default_email: true,
      is_default_password: true,
    });
    mockUpdateAdminProfile.mockResolvedValue({
      email: 'owner@test.com',
      is_default_email: false,
      is_default_password: false,
    });
  });

  it('shows warnings when default credentials are still active', async () => {
    renderWithRouter(<ProfileSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('profile-default-warning')).toBeInTheDocument();
    });
    expect(screen.getByText('admin@localhost')).toBeInTheDocument();
    expect(screen.getByText(/Default credentials detected/i)).toBeInTheDocument();
  });

  it('submits an email update with current password confirmation', async () => {
    const user = userEvent.setup();
    renderWithRouter(<ProfileSettings />);

    const emailInput = await screen.findByTestId('profile-email-new');
    const passwordInput = screen.getByTestId('profile-email-current-password');
    const submitButton = screen.getByTestId('profile-email-submit');

    await user.type(emailInput, 'owner@test.com');
    await user.type(passwordInput, 'changeme123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockUpdateAdminProfile).toHaveBeenCalledWith({
        current_password: 'changeme123',
        new_email: 'owner@test.com',
      });
      expect(screen.getByTestId('profile-email-success')).toBeInTheDocument();
    });
  });

  it('submits a password rotation request', async () => {
    const user = userEvent.setup();
    renderWithRouter(<ProfileSettings />);

    const newPassword = await screen.findByTestId('profile-password-new');
    const confirmPassword = screen.getByTestId('profile-password-confirm');
    const currentPassword = screen.getByTestId('profile-password-current');
    const submitButton = screen.getByTestId('profile-password-submit');

    await user.type(newPassword, 'StrongerPass123!');
    await user.type(confirmPassword, 'StrongerPass123!');
    await user.type(currentPassword, 'changeme123');
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockUpdateAdminProfile).toHaveBeenCalledWith({
        current_password: 'changeme123',
        new_password: 'StrongerPass123!',
      });
      expect(screen.getByTestId('profile-password-success')).toBeInTheDocument();
    });
  });
});
