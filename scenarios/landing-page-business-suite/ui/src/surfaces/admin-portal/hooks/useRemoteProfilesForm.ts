import { useCallback, useEffect, useMemo, useState } from 'react';
import type { RemoteProfile } from '../../../shared/api';
import {
  createRemoteProfile,
  deleteRemoteProfile,
  fetchRemoteProfiles,
  loginRemoteProfile,
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
}

export interface UseRemoteProfilesFormReturn {
  profiles: RemoteProfile[];
  loading: boolean;
  error: string | null;
  actions: RemoteProfileActionState;
  refresh: () => Promise<void>;
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
};

export function useRemoteProfilesForm(): UseRemoteProfilesFormReturn {
  const [profiles, setProfiles] = useState<RemoteProfile[]>([]);
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

  useEffect(() => {
    loadProfiles(false);
  }, [loadProfiles]);

  const refresh = useCallback(async () => {
    await loadProfiles(true);
  }, [loadProfiles]);

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

  const stableProfiles = useMemo(() => profiles, [profiles]);

  return {
    profiles: stableProfiles,
    loading,
    error,
    actions,
    refresh,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleLogin,
    handleLogout,
    handleTest,
  };
}
