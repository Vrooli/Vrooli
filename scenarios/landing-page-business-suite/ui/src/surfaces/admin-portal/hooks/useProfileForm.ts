import { useCallback, useEffect, useMemo, useState } from 'react';
import type { AdminProfile } from '../../../shared/api';
import {
  fetchAdminProfile,
  updateAdminEmail,
  updateAdminPassword,
  cleanError,
  hasDefaultCredentialRisk,
  validateEmailForm,
  validatePasswordForm,
  MIN_PASSWORD_LENGTH,
  DEFAULT_EMAIL_FORM,
  DEFAULT_PASSWORD_FORM,
  type FormStatus,
  type EmailFormState,
  type PasswordFormState,
} from '../services/profile.service';

/**
 * Return type for the useProfileForm hook
 */
export interface UseProfileFormReturn {
  /** Admin profile data */
  profile: AdminProfile | null;
  /** Whether profile is loading */
  loading: boolean;
  /** Profile load error */
  loadError: string | null;

  /** Email form state */
  emailForm: EmailFormState;
  /** Email form status (saving, message, error) */
  emailStatus: FormStatus;
  /** Update email form field */
  updateEmailForm: (field: keyof EmailFormState, value: string) => void;
  /** Handle email form submission */
  handleEmailSubmit: (event: React.FormEvent) => Promise<void>;

  /** Password form state */
  passwordForm: PasswordFormState;
  /** Password form status (saving, message, error) */
  passwordStatus: FormStatus;
  /** Update password form field */
  updatePasswordForm: (field: keyof PasswordFormState, value: string) => void;
  /** Handle password form submission */
  handlePasswordSubmit: (event: React.FormEvent) => Promise<void>;

  /** Whether default credentials are detected */
  defaultCredentialRisk: boolean;

  /** Reload profile data */
  loadProfile: () => Promise<void>;

  /** Minimum password length constant */
  MIN_PASSWORD_LENGTH: number;
}

/**
 * Custom hook for managing admin profile form
 *
 * Encapsulates all state management for the profile settings page,
 * including profile loading, email/password form handling, and validation.
 *
 * @returns Object containing form state and handlers
 */
export function useProfileForm(): UseProfileFormReturn {
  // Profile state
  const [profile, setProfile] = useState<AdminProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Email form state
  const [emailForm, setEmailForm] = useState<EmailFormState>(DEFAULT_EMAIL_FORM);
  const [emailStatus, setEmailStatus] = useState<FormStatus>({ saving: false });

  // Password form state
  const [passwordForm, setPasswordForm] = useState<PasswordFormState>(DEFAULT_PASSWORD_FORM);
  const [passwordStatus, setPasswordStatus] = useState<FormStatus>({ saving: false });

  /**
   * Load admin profile
   */
  const loadProfile = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await fetchAdminProfile();
      setProfile(data);
    } catch (error) {
      setLoadError(cleanError(error));
    } finally {
      setLoading(false);
    }
  }, []);

  // Load profile on mount
  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  /**
   * Compute default credential risk status
   */
  const defaultCredentialRisk = useMemo(
    () => hasDefaultCredentialRisk(profile),
    [profile]
  );

  /**
   * Update email form field
   */
  const updateEmailForm = useCallback((field: keyof EmailFormState, value: string) => {
    setEmailForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  /**
   * Update password form field
   */
  const updatePasswordForm = useCallback((field: keyof PasswordFormState, value: string) => {
    setPasswordForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  /**
   * Handle email form submission
   */
  const handleEmailSubmit = useCallback(
    async (event: React.FormEvent) => {
      event.preventDefault();
      setEmailStatus({ saving: true });

      const validationError = validateEmailForm(emailForm);
      if (validationError) {
        setEmailStatus({ saving: false, error: validationError });
        return;
      }

      try {
        const updated = await updateAdminEmail(
          emailForm.currentPassword,
          emailForm.newEmail
        );
        setProfile(updated);
        setEmailStatus({
          saving: false,
          message: 'Email updated. New sign-in is active immediately.',
        });
        setEmailForm(DEFAULT_EMAIL_FORM);
      } catch (error) {
        setEmailStatus({ saving: false, error: cleanError(error) });
      }
    },
    [emailForm]
  );

  /**
   * Handle password form submission
   */
  const handlePasswordSubmit = useCallback(
    async (event: React.FormEvent) => {
      event.preventDefault();
      setPasswordStatus({ saving: true });

      const validationError = validatePasswordForm(passwordForm);
      if (validationError) {
        setPasswordStatus({ saving: false, error: validationError });
        return;
      }

      try {
        const updated = await updateAdminPassword(
          passwordForm.currentPassword,
          passwordForm.newPassword
        );
        setProfile(updated);
        setPasswordStatus({
          saving: false,
          message: 'Password updated. Future logins will require the new secret.',
        });
        setPasswordForm(DEFAULT_PASSWORD_FORM);
      } catch (error) {
        setPasswordStatus({ saving: false, error: cleanError(error) });
      }
    },
    [passwordForm]
  );

  return {
    profile,
    loading,
    loadError,
    emailForm,
    emailStatus,
    updateEmailForm,
    handleEmailSubmit,
    passwordForm,
    passwordStatus,
    updatePasswordForm,
    handlePasswordSubmit,
    defaultCredentialRisk,
    loadProfile,
    MIN_PASSWORD_LENGTH,
  };
}
