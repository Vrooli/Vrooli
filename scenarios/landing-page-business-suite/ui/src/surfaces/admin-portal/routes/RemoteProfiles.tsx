import { useMemo, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormField } from '../components/FormField';
import { inputClassName } from '../components/formFieldClasses';
import { PasswordInput } from '../components/PasswordInput';
import { Callout } from '../components/Callout';
import { LAYOUT } from '../config/layout.constants';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { useToast } from '../../../shared/ui/useToast';
import { formatDateTime } from '../../../shared/lib/dateFormatters';
import type { RemoteProfile } from '../../../shared/api';
import { useRemoteProfilesForm } from '../hooks/useRemoteProfilesForm';
import {
  DEFAULT_REMOTE_PROFILE_FORM,
  DEFAULT_REMOTE_PROFILE_LOGIN,
  getRemoteProfileStatusMeta,
  validateRemoteProfileForm,
  validateRemoteProfileLoginForm,
} from '../services/remoteProfiles.service';
import {
  RefreshCw,
  Plus,
  Server,
  LogIn,
  LogOut,
  Trash2,
  Edit,
  Link2,
  ShieldCheck,
  XCircle,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';

const STATUS_STYLES: Record<'success' | 'warning' | 'error' | 'info', string> = {
  success: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  warning: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
  error: 'border-rose-500/30 bg-rose-500/10 text-rose-300',
  info: 'border-slate-500/30 bg-slate-500/10 text-slate-300',
};

const STATUS_ICONS = {
  success: CheckCircle2,
  warning: AlertCircle,
  error: XCircle,
  info: ShieldCheck,
};

export function RemoteProfiles() {
  const { addToast } = useToast();
  const {
    profiles,
    incomingSessions,
    sessionLinksByProfileId,
    loading,
    error,
    actions,
    refresh,
    refreshIncomingSessions,
    handleLoadSessionLinks,
    handleRevokeRemoteSessions,
    handleRevokeIncomingSession,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleLogin,
    handleLogout,
    handleTest,
  } = useRemoteProfilesForm();

  const [showProfileModal, setShowProfileModal] = useState(false);
  const [editingProfile, setEditingProfile] = useState<RemoteProfile | null>(null);
  const [profileForm, setProfileForm] = useState(DEFAULT_REMOTE_PROFILE_FORM);
  const [profileError, setProfileError] = useState<string | null>(null);

  const [showLoginModal, setShowLoginModal] = useState(false);
  const [loginProfile, setLoginProfile] = useState<RemoteProfile | null>(null);
  const [loginForm, setLoginForm] = useState(DEFAULT_REMOTE_PROFILE_LOGIN);
  const [loginError, setLoginError] = useState<string | null>(null);

  const openCreateModal = () => {
    setEditingProfile(null);
    setProfileForm(DEFAULT_REMOTE_PROFILE_FORM);
    setProfileError(null);
    setShowProfileModal(true);
  };

  const openEditModal = (profile: RemoteProfile) => {
    setEditingProfile(profile);
    setProfileForm({
      tag: profile.tag,
      label: profile.label ?? '',
      apiBase: profile.api_base,
    });
    setProfileError(null);
    setShowProfileModal(true);
  };

  const openLoginModal = (profile: RemoteProfile) => {
    setLoginProfile(profile);
    setLoginForm(DEFAULT_REMOTE_PROFILE_LOGIN);
    setLoginError(null);
    setShowLoginModal(true);
  };

  const closeProfileModal = () => {
    setShowProfileModal(false);
    setEditingProfile(null);
    setProfileForm(DEFAULT_REMOTE_PROFILE_FORM);
    setProfileError(null);
  };

  const closeLoginModal = () => {
    setShowLoginModal(false);
    setLoginProfile(null);
    setLoginForm(DEFAULT_REMOTE_PROFILE_LOGIN);
    setLoginError(null);
  };

  const handleSaveProfile = async () => {
    const validationError = validateRemoteProfileForm(profileForm);
    if (validationError) {
      setProfileError(validationError);
      return;
    }

    const result = editingProfile
      ? await handleUpdate(editingProfile.id, profileForm)
      : await handleCreate(profileForm);

    if (result.success) {
      addToast({
        type: 'success',
        message: result.message ?? 'Remote profile saved',
      });
      closeProfileModal();
    } else {
      const message = result.message ?? 'Failed to save remote profile';
      setProfileError(message);
      addToast({ type: 'error', message });
    }
  };

  const handleLoginProfile = async () => {
    if (!loginProfile) return;
    const validationError = validateRemoteProfileLoginForm(loginForm);
    if (validationError) {
      setLoginError(validationError);
      return;
    }

    const result = await handleLogin(loginProfile.id, loginForm);
    if (result.success) {
      addToast({
        type: 'success',
        message: result.message ?? 'Remote session established',
      });
      closeLoginModal();
    } else {
      const message = result.message ?? 'Remote login failed';
      setLoginError(message);
      addToast({ type: 'error', message });
    }
  };

  const handleDeleteProfile = async (profile: RemoteProfile) => {
    if (!confirm(`Delete remote profile "${profile.tag}"? This cannot be undone.`)) {
      return;
    }
    const result = await handleDelete(profile.id);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Remote profile deleted' : 'Failed to delete profile'),
    });
  };

  const handleTestProfile = async (profile: RemoteProfile) => {
    const result = await handleTest(profile.id);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Remote session verified' : 'Remote test failed'),
    });
  };

  const handleLogoutProfile = async (profile: RemoteProfile) => {
    const result = await handleLogout(profile.id);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Remote session revoked' : 'Remote logout failed'),
    });
  };

  const handleInspectRemoteProfile = async (profile: RemoteProfile) => {
    const result = await handleLoadSessionLinks(profile.id);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Remote state loaded' : 'Failed to inspect remote state'),
    });
  };

  const handleRemoteRevokeProfile = async (profile: RemoteProfile) => {
    if (!confirm(`Revoke remote session(s) for "${profile.tag}" and clear local stored session?`)) {
      return;
    }
    const result = await handleRevokeRemoteSessions(profile.id);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Remote sessions revoked' : 'Failed to revoke remote sessions'),
    });
  };

  const handleIncomingRevoke = async (sessionID: string) => {
    if (!confirm(`Revoke incoming remote session "${sessionID}"?`)) {
      return;
    }
    const result = await handleRevokeIncomingSession(sessionID);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message ?? (result.success ? 'Incoming remote session revoked' : 'Failed to revoke incoming remote session'),
    });
  };

  const sortedProfiles = useMemo(() => profiles, [profiles]);

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Remote Profiles"
          description="Connect to deployed Landing Page Business Suite instances for secure admin automation."
          icon={Server}
          iconBgClass="bg-sky-500/10"
          iconColorClass="text-sky-400"
          testId="remote-profiles-header"
          actions={(
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={refresh}
                disabled={actions.refreshing}
                className="gap-2"
              >
                <RefreshCw className={`h-4 w-4 ${actions.refreshing ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button size="sm" onClick={openCreateModal} className="gap-2">
                <Plus className="h-4 w-4" />
                Add Profile
              </Button>
            </>
          )}
        />

        <Callout
          type="info"
          title="Secure remote administration"
          message="Remote profiles store encrypted admin sessions for deployed LPBS instances. Use them to route admin actions through the local API while keeping credentials off the CLI."
        />

        {error && (
          <Callout type="error" message={error} />
        )}

        {loading ? (
          <div className="text-center py-10 text-slate-400">Loading remote profiles...</div>
        ) : (
          <div className={LAYOUT.sectionSpacing}>
            <Card className={LAYOUT.card.base}>
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <CardTitle className="text-lg">Incoming Remote Sessions</CardTitle>
                    <CardDescription>
                      Sessions created by remote-profile connectors logging into this LPBS instance.
                    </CardDescription>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={refreshIncomingSessions}
                    disabled={actions.incomingRefreshing}
                    className="gap-2"
                  >
                    <RefreshCw className={`h-4 w-4 ${actions.incomingRefreshing ? 'animate-spin' : ''}`} />
                    Refresh Incoming
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {incomingSessions.length === 0 ? (
                  <p className="text-sm text-slate-400">No incoming remote-profile sessions detected.</p>
                ) : (
                  <div className="space-y-3">
                    {incomingSessions.map((session) => (
                      <div key={session.session_id} className="rounded-lg border border-slate-700 p-3">
                        <div className="flex items-start justify-between gap-4">
                          <div className="space-y-1 text-sm">
                            <p className="text-slate-200 font-medium">
                              {session.profile_tag ? `#${session.profile_tag}` : session.connector_id}
                            </p>
                            <p className="text-slate-400">
                              Admin: {session.admin_email} • Origin: {session.origin || 'unknown'}
                            </p>
                            <p className="text-slate-500 font-mono text-xs">Session: {session.session_id}</p>
                            <p className="text-slate-500 text-xs">
                              Last activity: {formatDateTime(session.last_activity, 'short')} • Expires: {formatDateTime(session.expires_at, 'short')}
                            </p>
                          </div>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleIncomingRevoke(session.session_id)}
                            disabled={actions.incomingRevokeSessionId === session.session_id}
                            className="gap-1 text-rose-400 hover:text-rose-300 hover:border-rose-500/50"
                          >
                            <LogOut className="h-4 w-4" />
                            Revoke
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
            {sortedProfiles.length === 0 ? (
              <Card className={`${LAYOUT.card.base} border-dashed`}>
                <CardContent className="pt-6 text-center">
                  <Server className="h-12 w-12 text-slate-500 mx-auto mb-4" />
                  <p className="text-slate-400 mb-4">No remote profiles configured yet</p>
                  <Button onClick={openCreateModal} className="gap-2">
                    <Plus className="h-4 w-4" />
                    Add Your First Profile
                  </Button>
                </CardContent>
              </Card>
            ) : (
              sortedProfiles.map((profile) => {
                const links = sessionLinksByProfileId[profile.id];
                const status = getRemoteProfileStatusMeta(profile);
                const StatusIcon = STATUS_ICONS[status.tone];
                const busy = {
                  login: actions.loginId === profile.id,
                  logout: actions.logoutId === profile.id,
                  test: actions.testingId === profile.id,
                  delete: actions.deletingId === profile.id,
                  update: actions.updatingId === profile.id,
                };
                const canTest = profile.has_session && !busy.login;
                return (
                  <Card key={profile.id} className={LAYOUT.card.base}>
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex items-center gap-3">
                          <div className="p-2 rounded-lg bg-slate-800">
                            <Server className="h-5 w-5 text-sky-300" />
                          </div>
                          <div>
                            <CardTitle className="text-lg">
                              {profile.label?.trim() || profile.tag}
                            </CardTitle>
                            <CardDescription className="flex items-center gap-2">
                              <span className="font-mono text-xs text-slate-300">#{profile.tag}</span>
                              <span className="text-slate-500">•</span>
                              <span className="inline-flex items-center gap-1">
                                <Link2 className="h-3 w-3" />
                                {profile.api_base}
                              </span>
                            </CardDescription>
                          </div>
                        </div>
                        <div className={`flex items-center gap-2 rounded-full border px-3 py-1 text-xs ${STATUS_STYLES[status.tone]}`}>
                          <StatusIcon className="h-3 w-3" />
                          <span className="font-semibold">{status.label}</span>
                          <span className="text-slate-400">{status.description}</span>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="grid gap-4 md:grid-cols-3 text-sm text-slate-400">
                        <div>
                          <p className="text-xs uppercase tracking-wide text-slate-500">Session</p>
                          <p className="text-slate-200">
                            {profile.has_session ? 'Stored session' : 'Not logged in'}
                          </p>
                          <p className="text-xs text-slate-500 font-mono">
                            Connector: {profile.connector_id || 'pending'}
                          </p>
                          <p className="text-xs text-slate-500 font-mono">
                            Remote session ID: {profile.remote_session_id || 'unknown'}
                          </p>
                          <p className="text-xs text-slate-500">
                            Expires: {formatDateTime(profile.session_expires_at, 'short')}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-wide text-slate-500">Last login</p>
                          <p className="text-slate-200">{formatDateTime(profile.last_login_at, 'short')}</p>
                        </div>
                        <div>
                          <p className="text-xs uppercase tracking-wide text-slate-500">Last used</p>
                          <p className="text-slate-200">{formatDateTime(profile.last_used_at, 'short')}</p>
                        </div>
                      </div>

                      <div className="mt-4 flex flex-wrap items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openLoginModal(profile)}
                          disabled={busy.login}
                          className="gap-1"
                        >
                          <LogIn className={`h-4 w-4 ${busy.login ? 'animate-pulse' : ''}`} />
                          {profile.has_session ? 'Re-login' : 'Login'}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestProfile(profile)}
                          disabled={!canTest || busy.test}
                          className="gap-1"
                        >
                          <RefreshCw className={`h-4 w-4 ${busy.test ? 'animate-spin' : ''}`} />
                          Test
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleLogoutProfile(profile)}
                          disabled={!profile.has_session || busy.logout}
                          className="gap-1"
                        >
                          <LogOut className={`h-4 w-4 ${busy.logout ? 'animate-pulse' : ''}`} />
                          Logout
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleInspectRemoteProfile(profile)}
                          disabled={actions.loadingLinksId === profile.id || !profile.has_session}
                          className="gap-1"
                        >
                          <RefreshCw className={`h-4 w-4 ${actions.loadingLinksId === profile.id ? 'animate-spin' : ''}`} />
                          Inspect Remote
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRemoteRevokeProfile(profile)}
                          disabled={actions.remoteRevokeId === profile.id || !profile.has_session}
                          className="gap-1 text-rose-400 hover:text-rose-300 hover:border-rose-500/50"
                        >
                          <LogOut className={`h-4 w-4 ${actions.remoteRevokeId === profile.id ? 'animate-pulse' : ''}`} />
                          Revoke Remote
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openEditModal(profile)}
                          disabled={busy.update}
                          className="gap-1"
                        >
                          <Edit className="h-4 w-4" />
                          Edit
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDeleteProfile(profile)}
                          disabled={busy.delete}
                          className="gap-1 text-rose-400 hover:text-rose-300 hover:border-rose-500/50"
                        >
                          <Trash2 className="h-4 w-4" />
                          Delete
                        </Button>
                      </div>
                      {links && (
                        <div className="mt-4 rounded-lg border border-slate-700/60 bg-slate-900/60 p-3">
                          <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">Remote Side Visibility</p>
                          <p className="text-sm text-slate-300">
                            Linked sessions on remote instance: <span className="font-semibold">{links.remote_sessions?.length ?? 0}</span>
                          </p>
                          {(links.remote_sessions?.length ?? 0) > 0 && (
                            <div className="mt-2 space-y-1">
                              {links.remote_sessions!.map((session) => (
                                <p key={session.session_id} className="text-xs text-slate-400 font-mono">
                                  {session.session_id} • {session.origin || 'unknown'} • {formatDateTime(session.last_activity, 'short')}
                                </p>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                );
              })
            )}
          </div>
        )}
      </div>

      {showProfileModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
          <div className="w-full max-w-lg rounded-2xl border border-white/10 bg-slate-900 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="rounded-xl bg-sky-500/10 p-2">
                <Server className="h-5 w-5 text-sky-400" />
              </div>
              <div>
                <h2 className="text-xl font-semibold">
                  {editingProfile ? 'Edit Remote Profile' : 'Add Remote Profile'}
                </h2>
                <p className="text-sm text-slate-400">
                  Provide the remote API base including <code className="text-slate-200">/api/v1</code>.
                </p>
              </div>
            </div>

            <div className={LAYOUT.contentSpacing}>
              <FormField label="Tag" helpText="Lowercase letters, numbers, '-' or '_' (used in CLI commands).">
                <input
                  type="text"
                  value={profileForm.tag}
                  onChange={(e) => setProfileForm((prev) => ({ ...prev, tag: e.target.value }))}
                  className={inputClassName}
                />
              </FormField>
              <FormField label="Label" helpText="Human-friendly display name. Optional.">
                <input
                  type="text"
                  value={profileForm.label}
                  onChange={(e) => setProfileForm((prev) => ({ ...prev, label: e.target.value }))}
                  className={inputClassName}
                />
              </FormField>
              <FormField label="API Base" helpText="Full remote base URL ending with /api/v1. Changing this clears the stored session.">
                <input
                  type="url"
                  value={profileForm.apiBase}
                  onChange={(e) => setProfileForm((prev) => ({ ...prev, apiBase: e.target.value }))}
                  className={inputClassName}
                />
              </FormField>
            </div>

            {profileError && (
              <div className="mt-4 rounded-lg border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">
                {profileError}
              </div>
            )}

            <div className="mt-6 flex justify-end gap-2">
              <Button variant="ghost" onClick={closeProfileModal}>
                Cancel
              </Button>
              <Button
                onClick={handleSaveProfile}
                disabled={actions.creating || actions.updatingId === editingProfile?.id}
                className="gap-2"
              >
                <CheckCircle2 className="h-4 w-4" />
                {editingProfile ? 'Save Changes' : 'Create Profile'}
              </Button>
            </div>
          </div>
        </div>
      )}

      {showLoginModal && loginProfile && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
          <div className="w-full max-w-lg rounded-2xl border border-white/10 bg-slate-900 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="rounded-xl bg-emerald-500/10 p-2">
                <ShieldCheck className="h-5 w-5 text-emerald-400" />
              </div>
              <div>
                <h2 className="text-xl font-semibold">Authenticate Remote Admin</h2>
                <p className="text-sm text-slate-400">
                  Login to <span className="text-slate-200">{loginProfile.api_base}</span>
                </p>
              </div>
            </div>

            <div className={LAYOUT.contentSpacing}>
              <FormField label="Admin Email">
                <input
                  type="email"
                  value={loginForm.email}
                  onChange={(e) => setLoginForm((prev) => ({ ...prev, email: e.target.value }))}
                  className={inputClassName}
                />
              </FormField>
              <FormField label="Admin Password">
                <PasswordInput
                  value={loginForm.password}
                  onChange={(e) => setLoginForm((prev) => ({ ...prev, password: e.target.value }))}
                  placeholder="Enter remote admin password"
                  autoComplete="current-password"
                />
              </FormField>
            </div>

            {loginError && (
              <div className="mt-4 rounded-lg border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">
                {loginError}
              </div>
            )}

            <div className="mt-6 flex justify-end gap-2">
              <Button variant="ghost" onClick={closeLoginModal}>
                Cancel
              </Button>
              <Button
                onClick={handleLoginProfile}
                disabled={actions.loginId === loginProfile.id}
                className="gap-2"
              >
                <LogIn className="h-4 w-4" />
                Authenticate
              </Button>
            </div>
          </div>
        </div>
      )}
    </AdminLayout>
  );
}
