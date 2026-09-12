import type { AdminProfile } from '../../../shared/api';
import {
  getAdminProfile as apiGetAdminProfile,
  updateAdminProfile as apiUpdateAdminProfile,
} from '../../../shared/api';

/**
 * Minimum password length required
 */
export const MIN_PASSWORD_LENGTH = 12;

/**
 * Form status for tracking save state
 */
export interface FormStatus {
  saving: boolean;
  message?: string;
  error?: string;
}

/**
 * Email form state
 */
export interface EmailFormState {
  newEmail: string;
  currentPassword: string;
}

/**
 * Password form state
 */
export interface PasswordFormState {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

/**
 * Default email form state
 */
export const DEFAULT_EMAIL_FORM: EmailFormState = {
  newEmail: '',
  currentPassword: '',
};

/**
 * Default password form state
 */
export const DEFAULT_PASSWORD_FORM: PasswordFormState = {
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
};

/**
 * Clean error message by removing API prefix
 */
export function cleanError(error: unknown): string {
  if (error instanceof Error) {
    return error.message.replace(/^API call failed \(\d+\):\s*/, '');
  }
  return 'Request failed';
}

/**
 * Check if profile has default credential risk
 */
export function hasDefaultCredentialRisk(profile: AdminProfile | null): boolean {
  if (!profile) return false;
  return profile.is_default_email || profile.is_default_password;
}

/**
 * Validate email form
 */
export function validateEmailForm(form: EmailFormState): string | null {
  if (!form.newEmail.trim()) {
    return 'Enter a new email to update your profile.';
  }
  if (!form.currentPassword.trim()) {
    return 'Confirm with your current password before saving changes.';
  }
  return null;
}

/**
 * Validate password form
 */
export function validatePasswordForm(form: PasswordFormState): string | null {
  if (!form.newPassword.trim() || !form.confirmPassword.trim()) {
    return 'Enter and confirm your new password.';
  }
  if (form.newPassword !== form.confirmPassword) {
    return 'Passwords do not match.';
  }
  if (!form.currentPassword.trim()) {
    return 'Enter your current password to authorize this change.';
  }
  return null;
}

/**
 * Check if password meets minimum requirements
 */
export function isPasswordStrong(password: string): boolean {
  if (password.length < MIN_PASSWORD_LENGTH) {
    return false;
  }
  // Check for at least one letter and one number
  const hasLetter = /[a-zA-Z]/.test(password);
  const hasNumber = /[0-9]/.test(password);
  return hasLetter && hasNumber;
}

/**
 * Get password strength feedback
 */
export function getPasswordStrengthFeedback(password: string): string | null {
  if (password.length < MIN_PASSWORD_LENGTH) {
    return `Password must be at least ${String(MIN_PASSWORD_LENGTH)} characters`;
  }
  if (!/[a-zA-Z]/.test(password)) {
    return 'Password must contain at least one letter';
  }
  if (!/[0-9]/.test(password)) {
    return 'Password must contain at least one number';
  }
  return null;
}

// API wrapper functions

/**
 * Fetch admin profile
 */
export async function fetchAdminProfile(): Promise<AdminProfile> {
  return apiGetAdminProfile();
}

/**
 * Update admin email
 */
export async function updateAdminEmail(
  currentPassword: string,
  newEmail: string
): Promise<AdminProfile> {
  return apiUpdateAdminProfile({
    current_password: currentPassword.trim(),
    new_email: newEmail.trim(),
  });
}

/**
 * Update admin password
 */
export async function updateAdminPassword(
  currentPassword: string,
  newPassword: string
): Promise<AdminProfile> {
  return apiUpdateAdminProfile({
    current_password: currentPassword.trim(),
    new_password: newPassword.trim(),
  });
}
