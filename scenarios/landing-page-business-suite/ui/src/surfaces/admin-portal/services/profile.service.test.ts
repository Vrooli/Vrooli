import { describe, it, expect } from 'vitest';
import type { AdminProfile } from '../../../shared/api';
import {
  MIN_PASSWORD_LENGTH,
  DEFAULT_EMAIL_FORM,
  DEFAULT_PASSWORD_FORM,
  cleanError,
  hasDefaultCredentialRisk,
  validateEmailForm,
  validatePasswordForm,
  isPasswordStrong,
  getPasswordStrengthFeedback,
  type EmailFormState,
  type PasswordFormState,
} from './profile.service';

const createMockProfile = (overrides: Partial<AdminProfile> = {}): AdminProfile => ({
  email: 'admin@example.com',
  is_default_email: false,
  is_default_password: false,
  ...overrides,
});

describe('profile.service', () => {
  describe('MIN_PASSWORD_LENGTH', () => {
    it('is 12 characters', () => {
      expect(MIN_PASSWORD_LENGTH).toBe(12);
    });
  });

  describe('DEFAULT_EMAIL_FORM', () => {
    it('has empty fields', () => {
      expect(DEFAULT_EMAIL_FORM.newEmail).toBe('');
      expect(DEFAULT_EMAIL_FORM.currentPassword).toBe('');
    });
  });

  describe('DEFAULT_PASSWORD_FORM', () => {
    it('has empty fields', () => {
      expect(DEFAULT_PASSWORD_FORM.currentPassword).toBe('');
      expect(DEFAULT_PASSWORD_FORM.newPassword).toBe('');
      expect(DEFAULT_PASSWORD_FORM.confirmPassword).toBe('');
    });
  });

  describe('cleanError', () => {
    it('removes API prefix from error message', () => {
      const error = new Error('API call failed (401): Invalid credentials');
      expect(cleanError(error)).toBe('Invalid credentials');
    });

    it('returns error message if no prefix', () => {
      const error = new Error('Network error');
      expect(cleanError(error)).toBe('Network error');
    });

    it('returns default message for non-Error', () => {
      expect(cleanError('string error')).toBe('Request failed');
      expect(cleanError(null)).toBe('Request failed');
      expect(cleanError(undefined)).toBe('Request failed');
    });
  });

  describe('hasDefaultCredentialRisk', () => {
    it('returns false for null profile', () => {
      expect(hasDefaultCredentialRisk(null)).toBe(false);
    });

    it('returns false when both are not default', () => {
      const profile = createMockProfile({
        is_default_email: false,
        is_default_password: false,
      });
      expect(hasDefaultCredentialRisk(profile)).toBe(false);
    });

    it('returns true when email is default', () => {
      const profile = createMockProfile({
        is_default_email: true,
        is_default_password: false,
      });
      expect(hasDefaultCredentialRisk(profile)).toBe(true);
    });

    it('returns true when password is default', () => {
      const profile = createMockProfile({
        is_default_email: false,
        is_default_password: true,
      });
      expect(hasDefaultCredentialRisk(profile)).toBe(true);
    });

    it('returns true when both are default', () => {
      const profile = createMockProfile({
        is_default_email: true,
        is_default_password: true,
      });
      expect(hasDefaultCredentialRisk(profile)).toBe(true);
    });
  });

  describe('validateEmailForm', () => {
    it('returns error for empty email', () => {
      const form: EmailFormState = {
        newEmail: '',
        currentPassword: 'password123',
      };
      expect(validateEmailForm(form)).toBe('Enter a new email to update your profile.');
    });

    it('returns error for whitespace-only email', () => {
      const form: EmailFormState = {
        newEmail: '   ',
        currentPassword: 'password123',
      };
      expect(validateEmailForm(form)).toBe('Enter a new email to update your profile.');
    });

    it('returns error for empty password', () => {
      const form: EmailFormState = {
        newEmail: 'new@example.com',
        currentPassword: '',
      };
      expect(validateEmailForm(form)).toBe('Confirm with your current password before saving changes.');
    });

    it('returns null for valid form', () => {
      const form: EmailFormState = {
        newEmail: 'new@example.com',
        currentPassword: 'password123',
      };
      expect(validateEmailForm(form)).toBeNull();
    });
  });

  describe('validatePasswordForm', () => {
    it('returns error for empty new password', () => {
      const form: PasswordFormState = {
        currentPassword: 'current123',
        newPassword: '',
        confirmPassword: 'newpassword1',
      };
      expect(validatePasswordForm(form)).toBe('Enter and confirm your new password.');
    });

    it('returns error for empty confirm password', () => {
      const form: PasswordFormState = {
        currentPassword: 'current123',
        newPassword: 'newpassword1',
        confirmPassword: '',
      };
      expect(validatePasswordForm(form)).toBe('Enter and confirm your new password.');
    });

    it('returns error when passwords do not match', () => {
      const form: PasswordFormState = {
        currentPassword: 'current123',
        newPassword: 'newpassword1',
        confirmPassword: 'differentpw1',
      };
      expect(validatePasswordForm(form)).toBe('Passwords do not match.');
    });

    it('returns error for empty current password', () => {
      const form: PasswordFormState = {
        currentPassword: '',
        newPassword: 'newpassword1',
        confirmPassword: 'newpassword1',
      };
      expect(validatePasswordForm(form)).toBe('Enter your current password to authorize this change.');
    });

    it('returns null for valid form', () => {
      const form: PasswordFormState = {
        currentPassword: 'current123',
        newPassword: 'newpassword1',
        confirmPassword: 'newpassword1',
      };
      expect(validatePasswordForm(form)).toBeNull();
    });
  });

  describe('isPasswordStrong', () => {
    it('returns false for short password', () => {
      expect(isPasswordStrong('short1')).toBe(false);
    });

    it('returns false for password without letter', () => {
      expect(isPasswordStrong('123456789012')).toBe(false);
    });

    it('returns false for password without number', () => {
      expect(isPasswordStrong('abcdefghijkl')).toBe(false);
    });

    it('returns true for strong password', () => {
      expect(isPasswordStrong('password1234')).toBe(true);
      expect(isPasswordStrong('abcdefgh1234')).toBe(true);
    });
  });

  describe('getPasswordStrengthFeedback', () => {
    it('returns length error for short password', () => {
      const result = getPasswordStrengthFeedback('short1');
      expect(result).toBe(`Password must be at least ${String(MIN_PASSWORD_LENGTH)} characters`);
    });

    it('returns letter error for password without letter', () => {
      const result = getPasswordStrengthFeedback('123456789012');
      expect(result).toBe('Password must contain at least one letter');
    });

    it('returns number error for password without number', () => {
      const result = getPasswordStrengthFeedback('abcdefghijkl');
      expect(result).toBe('Password must contain at least one number');
    });

    it('returns null for strong password', () => {
      expect(getPasswordStrengthFeedback('password1234')).toBeNull();
    });
  });
});
