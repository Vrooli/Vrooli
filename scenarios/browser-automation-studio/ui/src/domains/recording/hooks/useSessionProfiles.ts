import { useCallback, useEffect, useMemo, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { BrowserProfile, RecordingSessionProfile } from '../types/types';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const safeJson = async (response: Response): Promise<unknown> => {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
};

const parseProfile = (value: unknown): RecordingSessionProfile | null => {
  if (!isRecord(value)) return null;
  if (typeof value.id !== 'string') return null;
  if (typeof value.name !== 'string') return null;
  return value as RecordingSessionProfile;
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
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions`);
      if (!response.ok) {
        throw new Error(`Failed to fetch sessions (${response.status})`);
      }
      const payload = await safeJson(response);
      const list = isRecord(payload) && Array.isArray(payload.profiles)
        ? payload.profiles.map(parseProfile).filter((p): p is RecordingSessionProfile => p !== null)
        : [];
      setProfiles(list);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load sessions';
      setError(message);
      logger.error(message, { component: 'useSessionProfiles', action: 'fetchProfiles' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const createProfile = useCallback(
    async (name?: string): Promise<RecordingSessionProfile | null> => {
      setCreating(true);
      setError(null);
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/recordings/sessions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name }),
        });
        if (!response.ok) {
          throw new Error(`Failed to create session (${response.status})`);
        }
        const payload = await safeJson(response);
        const profile = parseProfile(payload);
        if (!profile) {
          throw new Error('Invalid session profile response');
        }
        setProfiles((prev) => [profile, ...prev.filter((p) => p.id !== profile.id)]);
        return profile;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to create session';
        setError(message);
        logger.error(message, { component: 'useSessionProfiles', action: 'createProfile' }, err);
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
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      if (!response.ok) {
        throw new Error(`Failed to rename session (${response.status})`);
      }
      const payload = await safeJson(response);
      const updated = parseProfile(payload);
      if (!updated) {
        throw new Error('Invalid session profile response');
      }
      setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to rename session';
      setError(message);
      logger.error(message, { component: 'useSessionProfiles', action: 'renameProfile' }, err);
    }
  }, []);

  const deleteProfile = useCallback(async (id: string) => {
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Failed to delete session (${response.status})`);
      }
      setProfiles((prev) => prev.filter((p) => p.id !== id));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete session';
      setError(message);
      logger.error(message, { component: 'useSessionProfiles', action: 'deleteProfile' }, err);
    }
  }, []);

  const updateBrowserProfile = useCallback(async (id: string, browserProfile: BrowserProfile) => {
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ browser_profile: browserProfile }),
      });
      if (!response.ok) {
        throw new Error(`Failed to update browser profile (${response.status})`);
      }
      const payload = await safeJson(response);
      const updated = parseProfile(payload);
      if (!updated) {
        throw new Error('Invalid session profile response');
      }
      setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update browser profile';
      setError(message);
      logger.error(message, { component: 'useSessionProfiles', action: 'updateBrowserProfile' }, err);
      throw err;
    }
  }, []);

  useEffect(() => {
    void fetchProfiles();
  }, [fetchProfiles]);

  const getDefaultProfileId = useCallback((): string | null => {
    if (profiles.length === 0) return null;
    // Profiles are returned sorted by last_used_at desc from API
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
