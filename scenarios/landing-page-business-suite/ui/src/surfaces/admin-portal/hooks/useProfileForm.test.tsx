import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useProfileForm } from './useProfileForm';
import type { AdminProfile } from '../../../shared/api';

// Mock the profile service
const fetchAdminProfileMock = vi.fn();
const updateAdminEmailMock = vi.fn();
const updateAdminPasswordMock = vi.fn();

vi.mock('../services/profile.service', async () => {
  const actual = await vi.importActual<typeof import('../services/profile.service')>(
    '../services/profile.service'
  );
  return {
    ...actual,
    fetchAdminProfile: (...args: unknown[]) => fetchAdminProfileMock(...args),
    updateAdminEmail: (...args: unknown[]) => updateAdminEmailMock(...args),
    updateAdminPassword: (...args: unknown[]) => updateAdminPasswordMock(...args),
  };
});

const mockProfile: AdminProfile = {
  email: 'admin@example.com',
  is_default_email: false,
  is_default_password: false,
};

const mockDefaultProfile: AdminProfile = {
  email: 'admin@template.local',
  is_default_email: true,
  is_default_password: true,
};

const createMockEvent = (): React.FormEvent => ({
  preventDefault: vi.fn(),
} as unknown as React.FormEvent);

describe('useProfileForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchAdminProfileMock.mockResolvedValue(mockProfile);
  });

  describe('initial state', () => {
    it('starts with loading state', () => {
      const { result } = renderHook(() => useProfileForm());

      expect(result.current.loading).toBe(true);
      expect(result.current.profile).toBeNull();
      expect(result.current.loadError).toBeNull();
    });

    it('has empty email form', () => {
      const { result } = renderHook(() => useProfileForm());

      expect(result.current.emailForm).toEqual({
        newEmail: '',
        currentPassword: '',
      });
      expect(result.current.emailStatus).toEqual({ saving: false });
    });

    it('has empty password form', () => {
      const { result } = renderHook(() => useProfileForm());

      expect(result.current.passwordForm).toEqual({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      });
      expect(result.current.passwordStatus).toEqual({ saving: false });
    });
  });

  describe('profile loading', () => {
    it('loads profile on mount', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(fetchAdminProfileMock).toHaveBeenCalledTimes(1);
      expect(result.current.profile).toEqual(mockProfile);
      expect(result.current.loadError).toBeNull();
    });

    it('handles load error', async () => {
      fetchAdminProfileMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.loadError).toBe('API failure');
      expect(result.current.profile).toBeNull();
    });

    it('cleans API error prefix', async () => {
      fetchAdminProfileMock.mockRejectedValue(
        new Error('API call failed (401): Unauthorized')
      );

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.loadError).toBe('Unauthorized');
    });

    it('handles non-Error rejection', async () => {
      fetchAdminProfileMock.mockRejectedValue('Unknown error');

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.loadError).toBe('Request failed');
    });
  });

  describe('default credential risk', () => {
    it('returns false when no default credentials', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.defaultCredentialRisk).toBe(false);
    });

    it('returns true when default email detected', async () => {
      fetchAdminProfileMock.mockResolvedValue({
        ...mockProfile,
        is_default_email: true,
      });

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.defaultCredentialRisk).toBe(true);
    });

    it('returns true when default password detected', async () => {
      fetchAdminProfileMock.mockResolvedValue({
        ...mockProfile,
        is_default_password: true,
      });

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.defaultCredentialRisk).toBe(true);
    });

    it('returns true when both defaults detected', async () => {
      fetchAdminProfileMock.mockResolvedValue(mockDefaultProfile);

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.defaultCredentialRisk).toBe(true);
    });
  });

  describe('email form', () => {
    it('updates email form fields', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEmailForm('newEmail', 'new@example.com');
      });

      expect(result.current.emailForm.newEmail).toBe('new@example.com');

      act(() => {
        result.current.updateEmailForm('currentPassword', 'password123');
      });

      expect(result.current.emailForm.currentPassword).toBe('password123');
    });

    it('validates empty email', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleEmailSubmit(createMockEvent());
      });

      expect(result.current.emailStatus.error).toBe(
        'Enter a new email to update your profile.'
      );
      expect(result.current.emailStatus.saving).toBe(false);
    });

    it('validates empty current password', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEmailForm('newEmail', 'new@example.com');
      });

      await act(async () => {
        await result.current.handleEmailSubmit(createMockEvent());
      });

      expect(result.current.emailStatus.error).toBe(
        'Confirm with your current password before saving changes.'
      );
    });

    it('submits email form successfully', async () => {
      const updatedProfile = { ...mockProfile, email: 'new@example.com' };
      updateAdminEmailMock.mockResolvedValue(updatedProfile);

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEmailForm('newEmail', 'new@example.com');
        result.current.updateEmailForm('currentPassword', 'password123');
      });

      await act(async () => {
        await result.current.handleEmailSubmit(createMockEvent());
      });

      expect(updateAdminEmailMock).toHaveBeenCalledWith('password123', 'new@example.com');
      expect(result.current.emailStatus.message).toBe(
        'Email updated. New sign-in is active immediately.'
      );
      expect(result.current.emailStatus.saving).toBe(false);
      expect(result.current.profile?.email).toBe('new@example.com');
      expect(result.current.emailForm).toEqual({
        newEmail: '',
        currentPassword: '',
      });
    });

    it('handles email submission error', async () => {
      updateAdminEmailMock.mockRejectedValue(new Error('Invalid password'));

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEmailForm('newEmail', 'new@example.com');
        result.current.updateEmailForm('currentPassword', 'wrong');
      });

      await act(async () => {
        await result.current.handleEmailSubmit(createMockEvent());
      });

      expect(result.current.emailStatus.error).toBe('Invalid password');
      expect(result.current.emailStatus.saving).toBe(false);
    });

    it('sets saving state during email submission', async () => {
      let resolveUpdate: (value: AdminProfile) => void;
      updateAdminEmailMock.mockReturnValue(
        new Promise<AdminProfile>((resolve) => {
          resolveUpdate = resolve;
        })
      );

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEmailForm('newEmail', 'new@example.com');
        result.current.updateEmailForm('currentPassword', 'password123');
      });

      act(() => {
        result.current.handleEmailSubmit(createMockEvent());
      });

      expect(result.current.emailStatus.saving).toBe(true);

      await act(async () => {
        resolveUpdate!(mockProfile);
      });

      expect(result.current.emailStatus.saving).toBe(false);
    });
  });

  describe('password form', () => {
    it('updates password form fields', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'newPassword123');
      });

      expect(result.current.passwordForm.newPassword).toBe('newPassword123');

      act(() => {
        result.current.updatePasswordForm('confirmPassword', 'newPassword123');
      });

      expect(result.current.passwordForm.confirmPassword).toBe('newPassword123');

      act(() => {
        result.current.updatePasswordForm('currentPassword', 'oldPassword');
      });

      expect(result.current.passwordForm.currentPassword).toBe('oldPassword');
    });

    it('validates empty new password', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(result.current.passwordStatus.error).toBe(
        'Enter and confirm your new password.'
      );
    });

    it('validates password mismatch', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'password123');
        result.current.updatePasswordForm('confirmPassword', 'password456');
      });

      await act(async () => {
        await result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(result.current.passwordStatus.error).toBe('Passwords do not match.');
    });

    it('validates empty current password', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'newPassword123');
        result.current.updatePasswordForm('confirmPassword', 'newPassword123');
      });

      await act(async () => {
        await result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(result.current.passwordStatus.error).toBe(
        'Enter your current password to authorize this change.'
      );
    });

    it('submits password form successfully', async () => {
      const updatedProfile = { ...mockProfile, is_default_password: false };
      updateAdminPasswordMock.mockResolvedValue(updatedProfile);

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'newPassword123');
        result.current.updatePasswordForm('confirmPassword', 'newPassword123');
        result.current.updatePasswordForm('currentPassword', 'oldPassword');
      });

      await act(async () => {
        await result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(updateAdminPasswordMock).toHaveBeenCalledWith('oldPassword', 'newPassword123');
      expect(result.current.passwordStatus.message).toBe(
        'Password updated. Future logins will require the new secret.'
      );
      expect(result.current.passwordStatus.saving).toBe(false);
      expect(result.current.passwordForm).toEqual({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      });
    });

    it('handles password submission error', async () => {
      updateAdminPasswordMock.mockRejectedValue(new Error('Invalid current password'));

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'newPassword123');
        result.current.updatePasswordForm('confirmPassword', 'newPassword123');
        result.current.updatePasswordForm('currentPassword', 'wrong');
      });

      await act(async () => {
        await result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(result.current.passwordStatus.error).toBe('Invalid current password');
      expect(result.current.passwordStatus.saving).toBe(false);
    });

    it('sets saving state during password submission', async () => {
      let resolveUpdate: (value: AdminProfile) => void;
      updateAdminPasswordMock.mockReturnValue(
        new Promise<AdminProfile>((resolve) => {
          resolveUpdate = resolve;
        })
      );

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updatePasswordForm('newPassword', 'newPassword123');
        result.current.updatePasswordForm('confirmPassword', 'newPassword123');
        result.current.updatePasswordForm('currentPassword', 'oldPassword');
      });

      act(() => {
        result.current.handlePasswordSubmit(createMockEvent());
      });

      expect(result.current.passwordStatus.saving).toBe(true);

      await act(async () => {
        resolveUpdate!(mockProfile);
      });

      expect(result.current.passwordStatus.saving).toBe(false);
    });
  });

  describe('profile reload', () => {
    it('can reload profile manually', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      fetchAdminProfileMock.mockClear();
      const newProfile = { ...mockProfile, email: 'updated@example.com' };
      fetchAdminProfileMock.mockResolvedValue(newProfile);

      await act(async () => {
        await result.current.loadProfile();
      });

      expect(fetchAdminProfileMock).toHaveBeenCalledTimes(1);
      expect(result.current.profile?.email).toBe('updated@example.com');
    });

    it('clears error on reload', async () => {
      fetchAdminProfileMock.mockRejectedValue(new Error('Initial error'));

      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.loadError).toBe('Initial error');

      fetchAdminProfileMock.mockResolvedValue(mockProfile);

      await act(async () => {
        await result.current.loadProfile();
      });

      expect(result.current.loadError).toBeNull();
    });
  });

  describe('constants', () => {
    it('exposes MIN_PASSWORD_LENGTH', () => {
      const { result } = renderHook(() => useProfileForm());

      expect(result.current.MIN_PASSWORD_LENGTH).toBe(12);
    });
  });

  describe('form event handling', () => {
    it('prevents default on email submit', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const mockEvent = createMockEvent();

      await act(async () => {
        await result.current.handleEmailSubmit(mockEvent);
      });

      expect(mockEvent.preventDefault).toHaveBeenCalled();
    });

    it('prevents default on password submit', async () => {
      const { result } = renderHook(() => useProfileForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const mockEvent = createMockEvent();

      await act(async () => {
        await result.current.handlePasswordSubmit(mockEvent);
      });

      expect(mockEvent.preventDefault).toHaveBeenCalled();
    });
  });
});
