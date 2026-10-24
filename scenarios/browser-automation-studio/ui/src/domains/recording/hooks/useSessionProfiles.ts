import { useCallback, useEffect, useMemo, useState } from 'react';
import { ConnectError } from '@connectrpc/connect';
import type { SessionProfile as ProtoSessionProfile } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';
import { sessionProfilesClient } from '@/api/sessionProfiles';
import { protoTimestampToISOString } from '@/utils/timestamps';
import { logger } from '@/utils/logger';
import type { BrowserProfile, RecordingSessionProfile } from '../types/types';

const toRecordingSessionProfile = (p: ProtoSessionProfile): RecordingSessionProfile => ({
  id: p.id,
  name: p.name,
  created_at: protoTimestampToISOString(p.createdAt) ?? '',
  updated_at: protoTimestampToISOString(p.updatedAt) ?? '',
  last_used_at: protoTimestampToISOString(p.lastUsedAt) ?? '',
  has_storage_state: p.hasStorageState,
  browser_profile: p.browserProfile ? (p.browserProfile as unknown as BrowserProfile) : undefined,
});

const reportError = (err: unknown, action: string, fallback: string): string => {
  const message =
    err instanceof ConnectError ? err.rawMessage || err.message :
    err instanceof Error ? err.message : fallback;
  logger.error(message, { component: 'useSessionProfiles', action }, err);
  return message;
};

interface UseSessionProfilesResult {
  profiles: RecordingSessionProfile[];
  loading: boolean;
  error: string | null;
  creating: boolean;
  rename: (id: string, name: string) => Promise<void>;
  create: (name?: string) => Promise<RecordingSessionProfile | null>;
  remove: (id: string) => Promise<void>;
  refresh: () => Promise<void>;
  getDefaultProfileId: () => string | null;
  updateBrowserProfile: (id: string, browserProfile: BrowserProfile) => Promise<void>;
}

export function useSessionProfiles(): UseSessionProfilesResult {
  const [profiles, setProfiles] = useState<RecordingSessionProfile[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProfiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await sessionProfilesClient.list({});
      setProfiles(res.profiles.map(toRecordingSessionProfile));
    } catch (err) {
      setError(reportError(err, 'fetchProfiles', 'Failed to load sessions'));
    } finally {
      setLoading(false);
    }
  }, []);

  const createProfile = useCallback(
    async (name?: string): Promise<RecordingSessionProfile | null> => {
      setCreating(true);
      setError(null);
      try {
        const res = await sessionProfilesClient.create({ name: name ?? '' });
        if (!res.profile) {
          throw new Error('Invalid session profile response');
        }
        const profile = toRecordingSessionProfile(res.profile);
        setProfiles((prev) => [profile, ...prev.filter((p) => p.id !== profile.id)]);
        return profile;
      } catch (err) {
        setError(reportError(err, 'createProfile', 'Failed to create session'));
        return null;
      } finally {
        setCreating(false);
      }
    },
    []
  );

  const renameProfile = useCallback(async (id: string, name: string) => {
    setError(null);
    try {
      const res = await sessionProfilesClient.update({ id, name });
      if (!res.profile) {
        throw new Error('Invalid session profile response');
      }
      const updated = toRecordingSessionProfile(res.profile);
      setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
    } catch (err) {
      setError(reportError(err, 'renameProfile', 'Failed to rename session'));
    }
  }, []);

  const deleteProfile = useCallback(async (id: string) => {
    setError(null);
    try {
      await sessionProfilesClient.delete({ id });
      setProfiles((prev) => prev.filter((p) => p.id !== id));
    } catch (err) {
      setError(reportError(err, 'deleteProfile', 'Failed to delete session'));
    }
  }, []);

  const updateBrowserProfile = useCallback(async (id: string, browserProfile: BrowserProfile) => {
    setError(null);
    try {
      const res = await sessionProfilesClient.update({
        id,
        browserProfile: browserProfile as never,
      });
      if (!res.profile) {
        throw new Error('Invalid session profile response');
      }
      const updated = toRecordingSessionProfile(res.profile);
      setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
    } catch (err) {
      const message = reportError(err, 'updateBrowserProfile', 'Failed to update browser profile');
      setError(message);
      throw err;
    }
  }, []);

  useEffect(() => {
    void fetchProfiles();
  }, [fetchProfiles]);

  const getDefaultProfileId = useCallback((): string | null => {
    if (profiles.length === 0) return null;
    return profiles[0]?.id ?? null;
  }, [profiles]);

  return useMemo(
    () => ({
      profiles,
      loading,
      creating,
      error,
      rename: renameProfile,
      create: createProfile,
      remove: deleteProfile,
      refresh: fetchProfiles,
      getDefaultProfileId,
      updateBrowserProfile,
    }),
    [profiles, loading, creating, error, renameProfile, createProfile, deleteProfile, fetchProfiles, getDefaultProfileId, updateBrowserProfile]
  );
}
