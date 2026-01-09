import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, CheckCircle2, KeyRound, Mail, ShieldCheck } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { getAdminProfile, updateAdminProfile, type AdminProfile } from '../../../shared/api';

type FormStatus = {
  saving: boolean;
  message?: string;
  error?: string;
};

const MIN_PASSWORD_LENGTH = 12;

const cleanError = (error: unknown): string => {
  if (error instanceof Error) {
    return error.message.replace(/^API call failed \(\d+\):\s*/, '');
  }
  return 'Request failed';
};

export function ProfileSettings() {
  const [profile, setProfile] = useState<AdminProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [emailForm, setEmailForm] = useState({ newEmail: '', currentPassword: '' });
  const [emailStatus, setEmailStatus] = useState<FormStatus>({ saving: false });

  const [passwordForm, setPasswordForm] = useState({ currentPassword: '', newPassword: '', confirmPassword: '' });
  const [passwordStatus, setPasswordStatus] = useState<FormStatus>({ saving: false });

  const defaultCredentialRisk = useMemo(() => {
    if (!profile) return false;
    return profile.is_default_email || profile.is_default_password;
  }, [profile]);

  useEffect(() => {
    const loadProfile = async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const data = await getAdminProfile();
        setProfile(data);
      } catch (error) {
        setLoadError(cleanError(error));
      } finally {
        setLoading(false);
      }
    };

    void loadProfile();
  }, []);

  const handleEmailSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setEmailStatus({ saving: true });

    if (!emailForm.newEmail.trim()) {
      setEmailStatus({ saving: false, error: 'Enter a new email to update your profile.' });
      return;
    }
    if (!emailForm.currentPassword.trim()) {
      setEmailStatus({ saving: false, error: 'Confirm with your current password before saving changes.' });
      return;
    }

    try {
      const updated = await updateAdminProfile({
        current_password: emailForm.currentPassword.trim(),
        new_email: emailForm.newEmail.trim(),
      });
      setProfile(updated);
      setEmailStatus({ saving: false, message: 'Email updated. New sign-in is active immediately.' });
      setEmailForm({ newEmail: '', currentPassword: '' });
    } catch (error) {
      setEmailStatus({ saving: false, error: cleanError(error) });
    }
  };

  const handlePasswordSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setPasswordStatus({ saving: true });

    if (!passwordForm.newPassword.trim() || !passwordForm.confirmPassword.trim()) {
      setPasswordStatus({ saving: false, error: 'Enter and confirm your new password.' });
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordStatus({ saving: false, error: 'Passwords do not match.' });
      return;
    }
    if (!passwordForm.currentPassword.trim()) {
      setPasswordStatus({ saving: false, error: 'Enter your current password to authorize this change.' });
      return;
    }

    try {
      const updated = await updateAdminProfile({
        current_password: passwordForm.currentPassword.trim(),
        new_password: passwordForm.newPassword.trim(),
      });
      setProfile(updated);
      setPasswordStatus({ saving: false, message: 'Password updated. Future logins will require the new secret.' });
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
    } catch (error) {
      setPasswordStatus({ saving: false, error: cleanError(error) });
    }
  };

  return (
    <AdminLayout>
      <div className="space-y-8">
        <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/50 to-slate-950 p-6 shadow-xl">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Profile & Security</p>
              <h1 className="mt-1 text-2xl font-bold text-white">Harden the default admin identity</h1>
              <p className="mt-2 text-sm text-slate-400">
                Update the seeded admin email and password so deployments do not rely on defaults. Changes apply immediately to the current session.
              </p>
            </div>
            {profile?.email && (
              <div className="rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-200" data-testid="profile-current-email">
                <div className="flex items-center gap-2">
                  <Mail className="h-4 w-4 text-blue-200" />
                  <span className="font-medium">{profile.email}</span>
                </div>
                <p className="mt-1 text-xs text-slate-400">Current admin username</p>
              </div>
            )}
          </div>

          {defaultCredentialRisk && (
            <div
              className="mt-4 flex flex-col gap-2 rounded-xl border border-amber-400/30 bg-amber-500/10 px-4 py-3 text-amber-50"
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
        </div>

        {loading ? (
          <div className="text-slate-400">Loading profile...</div>
        ) : loadError ? (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-red-200">Failed to load profile: {loadError}</div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-2">
            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <ShieldCheck className="h-5 w-5 text-emerald-300" />
                  Update email
                </CardTitle>
                <CardDescription>Replace the seeded admin email with a new owner address.</CardDescription>
              </CardHeader>
              <CardContent>
                <form className="space-y-4" onSubmit={handleEmailSubmit} data-testid="profile-email-form">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">New email</label>
                    <input
                      type="email"
                      value={emailForm.newEmail}
                      onChange={(event) => setEmailForm((prev) => ({ ...prev, newEmail: event.target.value }))}
                      placeholder="you@company.com"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      data-testid="profile-email-new"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Current password</label>
                    <input
                      type="password"
                      value={emailForm.currentPassword}
                      onChange={(event) => setEmailForm((prev) => ({ ...prev, currentPassword: event.target.value }))}
                      placeholder="Confirm with current password"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      data-testid="profile-email-current-password"
                      autoComplete="current-password"
                    />
                  </div>
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
              </CardContent>
            </Card>

            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <KeyRound className="h-5 w-5 text-blue-200" />
                  Rotate password
                </CardTitle>
                <CardDescription>Require a strong password and retire the seeded default.</CardDescription>
              </CardHeader>
              <CardContent>
                <form className="space-y-4" onSubmit={handlePasswordSubmit} data-testid="profile-password-form">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">New password</label>
                    <input
                      type="password"
                      value={passwordForm.newPassword}
                      onChange={(event) => setPasswordForm((prev) => ({ ...prev, newPassword: event.target.value }))}
                      placeholder="At least 12 characters, letters + numbers"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      data-testid="profile-password-new"
                      autoComplete="new-password"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Confirm new password</label>
                    <input
                      type="password"
                      value={passwordForm.confirmPassword}
                      onChange={(event) => setPasswordForm((prev) => ({ ...prev, confirmPassword: event.target.value }))}
                      placeholder="Re-enter new password"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      data-testid="profile-password-confirm"
                      autoComplete="new-password"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Current password</label>
                    <input
                      type="password"
                      value={passwordForm.currentPassword}
                      onChange={(event) => setPasswordForm((prev) => ({ ...prev, currentPassword: event.target.value }))}
                      placeholder="Confirm with current password"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      data-testid="profile-password-current"
                      autoComplete="current-password"
                    />
                  </div>
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
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
