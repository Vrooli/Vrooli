import { useCallback, useEffect, useState } from 'react';
import type {
  BrowserProfile,
  ProfilePreset,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
} from '@/domains/recording/types/types';
import { useStorageState } from '@/domains/recording';
import { PRESET_CONFIGS } from '../settings';
import type { SectionId } from './SessionSidebar';

interface UseSessionManagerProps {
  profileId: string;
  initialProfile?: BrowserProfile;
  onSave: (profile: BrowserProfile) => Promise<void>;
}

export function useSessionManager({ profileId, initialProfile, onSave }: UseSessionManagerProps) {
  // Active section
  const [activeSection, setActiveSection] = useState<SectionId>('presets');

  // Saving state
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);

  // Profile settings state
  const [preset, setPreset] = useState<ProfilePreset>(initialProfile?.preset ?? 'none');
  const [fingerprint, setFingerprint] = useState<FingerprintSettings>(initialProfile?.fingerprint ?? {});
  const [behavior, setBehavior] = useState<BehaviorSettings>(initialProfile?.behavior ?? {});
  const [antiDetection, setAntiDetection] = useState<AntiDetectionSettings>(initialProfile?.anti_detection ?? {});

  // Storage state (read-only for now)
  const { storageState, loading: storageLoading, error: storageError, fetchStorageState, clear: clearStorage } = useStorageState();

  // Fetch storage when switching to storage sections
  useEffect(() => {
    if (activeSection === 'cookies' || activeSection === 'local-storage') {
      void fetchStorageState(profileId);
    }
    return () => {
      if (activeSection === 'cookies' || activeSection === 'local-storage') {
        clearStorage();
      }
    };
  }, [activeSection, profileId, fetchStorageState, clearStorage]);

  // Initialize from initialProfile
  useEffect(() => {
    if (initialProfile) {
      setPreset(initialProfile.preset ?? 'none');
      setFingerprint(initialProfile.fingerprint ?? {});
      setBehavior(initialProfile.behavior ?? {});
      setAntiDetection(initialProfile.anti_detection ?? {});
      setIsDirty(false);
    }
  }, [initialProfile]);

  // Apply preset defaults when preset changes
  const applyPreset = useCallback((newPreset: ProfilePreset) => {
    setPreset(newPreset);
    const config = PRESET_CONFIGS[newPreset];
    setBehavior((prev) => ({ ...prev, ...config.behavior }));
    setAntiDetection((prev) => ({ ...prev, ...config.antiDetection }));
    setIsDirty(true);
  }, []);

  // Update helpers
  const updateFingerprint = useCallback(<K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => {
    setFingerprint((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  }, []);

  const updateBehavior = useCallback(<K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => {
    setBehavior((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  }, []);

  const updateAntiDetection = useCallback(<K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => {
    setAntiDetection((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  }, []);

  // Save handler
  const handleSave = useCallback(async () => {
    setSaving(true);
    setError(null);
    try {
      const profile: BrowserProfile = {
        preset,
        fingerprint: Object.keys(fingerprint).length > 0 ? fingerprint : undefined,
        behavior: Object.keys(behavior).length > 0 ? behavior : undefined,
        anti_detection: Object.keys(antiDetection).length > 0 ? antiDetection : undefined,
      };
      await onSave(profile);
      setIsDirty(false);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save profile');
      return false;
    } finally {
      setSaving(false);
    }
  }, [preset, fingerprint, behavior, antiDetection, onSave]);

  // Reset to initial values
  const reset = useCallback(() => {
    setPreset(initialProfile?.preset ?? 'none');
    setFingerprint(initialProfile?.fingerprint ?? {});
    setBehavior(initialProfile?.behavior ?? {});
    setAntiDetection(initialProfile?.anti_detection ?? {});
    setIsDirty(false);
    setError(null);
  }, [initialProfile]);

  return {
    // Section navigation
    activeSection,
    setActiveSection,

    // Settings state
    preset,
    fingerprint,
    behavior,
    antiDetection,

    // Settings actions
    applyPreset,
    updateFingerprint,
    updateBehavior,
    updateAntiDetection,

    // Storage state
    storageState,
    storageLoading,
    storageError,

    // Save state
    saving,
    error,
    isDirty,
    handleSave,
    reset,
  };
}
