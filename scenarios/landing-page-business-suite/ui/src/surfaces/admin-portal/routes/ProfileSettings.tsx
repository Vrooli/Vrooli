import { AlertTriangle, CheckCircle2, KeyRound, Mail, ShieldCheck } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField } from '../components/FormField';
import { inputClassName } from '../components/formFieldClasses';
import { Button } from '../../../shared/ui/button';
import { LAYOUT } from '../config/layout.constants';
import { useProfileForm } from '../hooks/useProfileForm';

export function ProfileSettings() {
  const {
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
    MIN_PASSWORD_LENGTH,
  } = useProfileForm();

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Harden the default admin identity"
          description="Update the seeded admin email and password so deployments do not rely on defaults. Changes apply immediately to the current session."
          icon={Mail}
          iconBgClass="bg-sky-500/10"
          iconColorClass="text-sky-400"
          testId="profile-header"
          actions={
            profile?.email ? (
              <div className="rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-200" data-testid="profile-current-email">
                <div className="flex items-center gap-2">
                  <Mail className="h-4 w-4 text-blue-200" />
                  <span className="font-medium">{profile.email}</span>
                </div>
                <p className="mt-1 text-xs text-slate-400">Current admin username</p>
              </div>
            ) : undefined
          }
        />

        {defaultCredentialRisk && (
          <div
            className="flex flex-col gap-2 rounded-xl border border-amber-400/30 bg-amber-500/10 px-4 py-3 text-amber-50"
            data-testid="profile-default-warning"
          >
            <div className="flex items-center gap-2 font-semibold">
              <AlertTriangle className="h-4 w-4" />
              Default credentials detected
            </div>
            <p className="text-sm text-amber-100/90">
              The seeded admin account should be replaced. Update the email and set a strong password (12+ chars, mix of letters and numbers).
            </p>
            <div className="flex flex-wrap gap-2 text-xs">
              {profile?.is_default_email && (
                <span className="rounded-full border border-amber-400/50 px-3 py-1">Email still {profile.email}</span>
              )}
              {profile?.is_default_password && (
                <span className="rounded-full border border-amber-400/50 px-3 py-1">Default password hash still active</span>
              )}
            </div>
          </div>
        )}

        {loading ? (
          <div className="text-slate-400">Loading profile...</div>
        ) : loadError ? (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-red-200">Failed to load profile: {loadError}</div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-2">
            <FormSection
              title="Update email"
              description="Replace the seeded admin email with a new owner address."
              icon={ShieldCheck}
              iconColorClass="text-emerald-300"
              testId="profile-email-section"
            >
              <form className="space-y-4" onSubmit={handleEmailSubmit} data-testid="profile-email-form">
                <FormField label="New email">
                  <input
                    type="email"
                    value={emailForm.newEmail}
                    onChange={(event) => updateEmailForm('newEmail', event.target.value)}
                    placeholder="you@company.com"
                    className={inputClassName}
                    data-testid="profile-email-new"
                  />
                </FormField>
                <FormField label="Current password">
                  <input
                    type="password"
                    value={emailForm.currentPassword}
                    onChange={(event) => updateEmailForm('currentPassword', event.target.value)}
                    placeholder="Confirm with current password"
                    className={inputClassName}
                    data-testid="profile-email-current-password"
                    autoComplete="current-password"
                  />
                </FormField>
                <div className="flex items-center justify-between text-xs text-slate-400">
                  <span>Session stays active after changing your email.</span>
                  <Button
                    type="submit"
                    size="sm"
                    className="gap-2"
                    disabled={emailStatus.saving}
                    data-testid="profile-email-submit"
                  >
                    {emailStatus.saving ? 'Saving...' : 'Update email'}
                  </Button>
                </div>
                {emailStatus.error && (
                  <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100" data-testid="profile-email-error">
                    {emailStatus.error}
                  </div>
                )}
                {emailStatus.message && (
                  <div className="flex items-center gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100" data-testid="profile-email-success">
                    <CheckCircle2 className="h-4 w-4" />
                    {emailStatus.message}
                  </div>
                )}
              </form>
            </FormSection>

            <FormSection
              title="Rotate password"
              description="Require a strong password and retire the seeded default."
              icon={KeyRound}
              iconColorClass="text-blue-200"
              testId="profile-password-section"
            >
              <form className="space-y-4" onSubmit={handlePasswordSubmit} data-testid="profile-password-form">
                <FormField label="New password">
                  <input
                    type="password"
                    value={passwordForm.newPassword}
                    onChange={(event) => updatePasswordForm('newPassword', event.target.value)}
                    placeholder="At least 12 characters, letters + numbers"
                    className={inputClassName}
                    data-testid="profile-password-new"
                    autoComplete="new-password"
                  />
                </FormField>
                <FormField label="Confirm new password">
                  <input
                    type="password"
                    value={passwordForm.confirmPassword}
                    onChange={(event) => updatePasswordForm('confirmPassword', event.target.value)}
                    placeholder="Re-enter new password"
                    className={inputClassName}
                    data-testid="profile-password-confirm"
                    autoComplete="new-password"
                  />
                </FormField>
                <FormField label="Current password">
                  <input
                    type="password"
                    value={passwordForm.currentPassword}
                    onChange={(event) => updatePasswordForm('currentPassword', event.target.value)}
                    placeholder="Confirm with current password"
                    className={inputClassName}
                    data-testid="profile-password-current"
                    autoComplete="current-password"
                  />
                </FormField>
                <div className="flex items-center justify-between text-xs text-slate-400">
                  <span>Min {MIN_PASSWORD_LENGTH} chars, include letters and numbers.</span>
                  <Button
                    type="submit"
                    size="sm"
                    className="gap-2"
                    disabled={passwordStatus.saving}
                    data-testid="profile-password-submit"
                  >
                    {passwordStatus.saving ? 'Updating...' : 'Update password'}
                  </Button>
                </div>
                {passwordStatus.error && (
                  <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100" data-testid="profile-password-error">
                    {passwordStatus.error}
                  </div>
                )}
                {passwordStatus.message && (
                  <div className="flex items-center gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100" data-testid="profile-password-success">
                    <CheckCircle2 className="h-4 w-4" />
                    {passwordStatus.message}
                  </div>
                )}
              </form>
            </FormSection>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
