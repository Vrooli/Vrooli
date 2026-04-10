import { useCallback, useEffect, useMemo, useState } from 'react';
import type { IncomingRemoteProfileSession, RemoteProfile, RemoteProfileSessionLinks } from '../../../shared/api';
import {
  createRemoteProfile,
  deleteRemoteProfile,
  fetchIncomingRemoteProfileSessions,
  fetchRemoteProfiles,
  getRemoteProfileSessionLinks,
  loginRemoteProfile,
  revokeIncomingRemoteProfileSession,
  revokeRemoteProfileSessions,
  logoutRemoteProfile,
  testRemoteProfile,
  updateRemoteProfile,
  type RemoteProfileFormState,
  type RemoteProfileLoginFormState,
} from '../services/remoteProfiles.service';

export interface RemoteProfileActionResult {
  success: boolean;
  message?: string;
  profile?: RemoteProfile;
}

export interface RemoteProfileActionState {
  creating: boolean;
  updatingId: number | null;
  deletingId: number | null;
  loginId: number | null;
  logoutId: number | null;
  testingId: number | null;
  refreshing: boolean;
  loadingLinksId: number | null;
  remoteRevokeId: number | null;
  incomingRefreshing: boolean;
  incomingRevokeSessionId: string | null;
}

export interface UseRemoteProfilesFormReturn {
  profiles: RemoteProfile[];
  incomingSessions: IncomingRemoteProfileSession[];
  sessionLinksByProfileId: Record<number, RemoteProfileSessionLinks>;
  loading: boolean;
  error: string | null;
  actions: RemoteProfileActionState;
  refresh: () => Promise<void>;
  refreshIncomingSessions: () => Promise<void>;
  handleLoadSessionLinks: (id: number) => Promise<RemoteProfileActionResult & { links?: RemoteProfileSessionLinks }>;
  handleRevokeRemoteSessions: (id: number) => Promise<RemoteProfileActionResult & { links?: RemoteProfileSessionLinks }>;
  handleRevokeIncomingSession: (sessionID: string) => Promise<RemoteProfileActionResult>;
  handleCreate: (form: RemoteProfileFormState) => Promise<RemoteProfileActionResult>;
  handleUpdate: (id: number, form: RemoteProfileFormState) => Promise<RemoteProfileActionResult>;
  handleDelete: (id: number) => Promise<RemoteProfileActionResult>;
  handleLogin: (id: number, form: RemoteProfileLoginFormState) => Promise<RemoteProfileActionResult>;
  handleLogout: (id: number) => Promise<RemoteProfileActionResult>;
  handleTest: (id: number) => Promise<RemoteProfileActionResult>;
}

const defaultActionState: RemoteProfileActionState = {
  creating: false,
  updatingId: null,
  deletingId: null,
  loginId: null,
  logoutId: null,
  testingId: null,
  refreshing: false,
  loadingLinksId: null,
  remoteRevokeId: null,
  incomingRefreshing: false,
  incomingRevokeSessionId: null,
};

export function useRemoteProfilesForm(): UseRemoteProfilesFormReturn {
  const [profiles, setProfiles] = useState<RemoteProfile[]>([]);
  const [incomingSessions, setIncomingSessions] = useState<IncomingRemoteProfileSession[]>([]);
  const [sessionLinksByProfileId, setSessionLinksByProfileId] = useState<Record<number, RemoteProfileSessionLinks>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actions, setActions] = useState<RemoteProfileActionState>(defaultActionState);

  const upsertProfile = useCallback((profile: RemoteProfile) => {
    setProfiles((prev) => {
      const index = prev.findIndex((item) => item.id === profile.id);
      if (index === -1) {
        return [profile, ...prev];
      }
      const next = [...prev];
      next[index] = profile;
      return next;
    });
  }, []);

  const removeProfile = useCallback((id: number) => {
    setProfiles((prev) => prev.filter((item) => item.id !== id));
  }, []);

  const loadProfiles = useCallback(async (silent = false) => {
    if (silent) {
      setActions((prev) => ({ ...prev, refreshing: true }));
    } else {
      setLoading(true);
    }
    try {
      const data = await fetchRemoteProfiles();
      setProfiles(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load remote profiles');
    } finally {
      if (silent) {
        setActions((prev) => ({ ...prev, refreshing: false }));
      } else {
        setLoading(false);
      }
    }
  }, []);

  const loadIncomingSessions = useCallback(async (silent = false) => {
    if (silent) {
      setActions((prev) => ({ ...prev, incomingRefreshing: true }));
    }
    try {
      const data = await fetchIncomingRemoteProfileSessions();
      setIncomingSessions(data);
    } finally {
      if (silent) {
        setActions((prev) => ({ ...prev, incomingRefreshing: false }));
      }
    }
  }, []);

  useEffect(() => {
    void Promise.all([
      loadProfiles(false),
      loadIncomingSessions(false),
    ]);
  }, [loadProfiles, loadIncomingSessions]);

  const refresh = useCallback(async () => {
    await Promise.all([
      loadProfiles(true),
      loadIncomingSessions(true),
    ]);
  }, [loadProfiles, loadIncomingSessions]);

  const refreshIncomingSessions = useCallback(async () => {
    await loadIncomingSessions(true);
  }, [loadIncomingSessions]);

  const handleCreate = useCallback(async (form: RemoteProfileFormState): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, creating: true }));
    try {
      const profile = await createRemoteProfile(form);
      upsertProfile(profile);
      return { success: true, profile, message: 'Remote profile created' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to create profile' };
    } finally {
      setActions((prev) => ({ ...prev, creating: false }));
    }
  }, [upsertProfile]);

  const handleUpdate = useCallback(async (id: number, form: RemoteProfileFormState): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, updatingId: id }));
    try {
      const profile = await updateRemoteProfile(id, form);
      upsertProfile(profile);
      return { success: true, profile, message: 'Remote profile updated' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to update profile' };
    } finally {
      setActions((prev) => ({ ...prev, updatingId: null }));
    }
  }, [upsertProfile]);

  const handleDelete = useCallback(async (id: number): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, deletingId: id }));
    try {
      await deleteRemoteProfile(id);
      removeProfile(id);
      return { success: true, message: 'Remote profile deleted' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to delete profile' };
    } finally {
      setActions((prev) => ({ ...prev, deletingId: null }));
    }
  }, [removeProfile]);

  const handleLogin = useCallback(async (id: number, form: RemoteProfileLoginFormState): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, loginId: id }));
    try {
      const profile = await loginRemoteProfile(id, form);
      upsertProfile(profile);
      return { success: true, profile, message: 'Remote session established' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Remote login failed' };
    } finally {
      setActions((prev) => ({ ...prev, loginId: null }));
    }
  }, [upsertProfile]);

  const handleLogout = useCallback(async (id: number): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, logoutId: id }));
    try {
      const profile = await logoutRemoteProfile(id);
      upsertProfile(profile);
      return { success: true, profile, message: 'Remote session revoked' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Remote logout failed' };
    } finally {
      setActions((prev) => ({ ...prev, logoutId: null }));
    }
  }, [upsertProfile]);

  const handleTest = useCallback(async (id: number): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, testingId: id }));
    try {
      const profile = await testRemoteProfile(id);
      upsertProfile(profile);
      return { success: true, profile, message: 'Remote session verified' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Remote session test failed' };
    } finally {
      setActions((prev) => ({ ...prev, testingId: null }));
    }
  }, [upsertProfile]);

  const handleLoadSessionLinks = useCallback(async (id: number): Promise<RemoteProfileActionResult & { links?: RemoteProfileSessionLinks }> => {
    setActions((prev) => ({ ...prev, loadingLinksId: id }));
    try {
      const links = await getRemoteProfileSessionLinks(id);
      setSessionLinksByProfileId((prev) => ({ ...prev, [id]: links }));
      return { success: true, links, message: 'Remote session state loaded' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to load remote session state' };
    } finally {
      setActions((prev) => ({ ...prev, loadingLinksId: null }));
    }
  }, []);

  const handleRevokeRemoteSessions = useCallback(async (id: number): Promise<RemoteProfileActionResult & { links?: RemoteProfileSessionLinks }> => {
    setActions((prev) => ({ ...prev, remoteRevokeId: id }));
    try {
      const links = await revokeRemoteProfileSessions(id);
      setSessionLinksByProfileId((prev) => ({ ...prev, [id]: links }));
      const refreshed = await fetchRemoteProfiles();
      setProfiles(refreshed);
      await loadIncomingSessions(true);
      return { success: true, links, message: 'Remote sessions revoked and local session cleared' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to revoke remote sessions' };
    } finally {
      setActions((prev) => ({ ...prev, remoteRevokeId: null }));
    }
  }, [loadIncomingSessions]);

  const handleRevokeIncomingSession = useCallback(async (sessionID: string): Promise<RemoteProfileActionResult> => {
    setActions((prev) => ({ ...prev, incomingRevokeSessionId: sessionID }));
    try {
      await revokeIncomingRemoteProfileSession(sessionID);
      setIncomingSessions((prev) => prev.filter((session) => session.session_id !== sessionID));
      return { success: true, message: 'Incoming session revoked' };
    } catch (err) {
      return { success: false, message: err instanceof Error ? err.message : 'Failed to revoke incoming session' };
    } finally {
      setActions((prev) => ({ ...prev, incomingRevokeSessionId: null }));
    }
  }, []);

  const stableProfiles = useMemo(() => profiles, [profiles]);
  const stableIncomingSessions = useMemo(() => incomingSessions, [incomingSessions]);
  const stableSessionLinks = useMemo(() => sessionLinksByProfileId, [sessionLinksByProfileId]);

  return {
    profiles: stableProfiles,
    incomingSessions: stableIncomingSessions,
    sessionLinksByProfileId: stableSessionLinks,
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
  };
}
