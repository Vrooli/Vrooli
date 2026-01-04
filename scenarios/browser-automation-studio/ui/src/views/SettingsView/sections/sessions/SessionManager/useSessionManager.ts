import { useCallback, useEffect, useState } from 'react';
import type {
  BrowserProfile,
  ProfilePreset,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
  ProxySettings,
} from '@/domains/recording/types/types';
import { useStorageState, useServiceWorkers } from '@/domains/recording';
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
  const [proxy, setProxy] = useState<ProxySettings>(initialProfile?.proxy ?? {});
  const [extraHeaders, setExtraHeaders] = useState<Record<string, string>>(initialProfile?.extra_headers ?? {});

  // Storage state with delete operations
  const {
    storageState,
    loading: storageLoading,
    error: storageError,
    deleting: storageDeleting,
    fetchStorageState,
    clear: clearStorage,
    clearAllCookies,
    deleteCookiesByDomain,
    deleteCookie,
    clearAllLocalStorage,
    deleteLocalStorageByOrigin,
    deleteLocalStorageItem,
  } = useStorageState();

  // Service workers state
  const {
    serviceWorkers,
    loading: swLoading,
    error: swError,
    deleting: swDeleting,
    fetchServiceWorkers,
    clear: clearServiceWorkers,
    unregisterAll: unregisterAllServiceWorkers,
    unregisterWorker: unregisterServiceWorker,
  } = useServiceWorkers();

  // Fetch storage when switching to storage sections
  useEffect(() => {
    if (activeSection === 'cookies' || activeSection === 'local-storage') {
      void fetchStorageState(profileId);
    }
    if (activeSection === 'service-workers') {
      void fetchServiceWorkers(profileId);
    }
    return () => {
      if (activeSection === 'cookies' || activeSection === 'local-storage') {
        clearStorage();
      }
      if (activeSection === 'service-workers') {
        clearServiceWorkers();
      }
    };
  }, [activeSection, profileId, fetchStorageState, clearStorage, fetchServiceWorkers, clearServiceWorkers]);

  // Initialize from initialProfile
  useEffect(() => {
    if (initialProfile) {
      setPreset(initialProfile.preset ?? 'none');
      setFingerprint(initialProfile.fingerprint ?? {});
      setBehavior(initialProfile.behavior ?? {});
      setAntiDetection(initialProfile.anti_detection ?? {});
      setProxy(initialProfile.proxy ?? {});
      setExtraHeaders(initialProfile.extra_headers ?? {});
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

  const updateProxy = useCallback(<K extends keyof ProxySettings>(key: K, value: ProxySettings[K]) => {
    setProxy((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  }, []);

  // Extra headers handlers
  const addExtraHeader = useCallback((key: string, value: string) => {
    setExtraHeaders((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  }, []);

  const updateExtraHeader = useCallback((oldKey: string, newKey: string, value: string) => {
    setExtraHeaders((prev) => {
      const { [oldKey]: _, ...rest } = prev;
      return { ...rest, [newKey]: value };
    });
    setIsDirty(true);
  }, []);

  const removeExtraHeader = useCallback((key: string) => {
    setExtraHeaders((prev) => {
      const { [key]: _, ...rest } = prev;
      return rest;
    });
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
        proxy: Object.keys(proxy).length > 0 ? proxy : undefined,
        extra_headers: Object.keys(extraHeaders).length > 0 ? extraHeaders : undefined,
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
  }, [preset, fingerprint, behavior, antiDetection, proxy, extraHeaders, onSave]);

  // Reset to initial values
  const reset = useCallback(() => {
    setPreset(initialProfile?.preset ?? 'none');
    setFingerprint(initialProfile?.fingerprint ?? {});
    setBehavior(initialProfile?.behavior ?? {});
    setAntiDetection(initialProfile?.anti_detection ?? {});
    setProxy(initialProfile?.proxy ?? {});
    setExtraHeaders(initialProfile?.extra_headers ?? {});
    setIsDirty(false);
    setError(null);
  }, [initialProfile]);

  // Wrapped delete handlers that pass profileId
  const handleClearAllCookies = useCallback(async () => {
    return clearAllCookies(profileId);
  }, [clearAllCookies, profileId]);

  const handleDeleteCookiesByDomain = useCallback(async (domain: string) => {
    return deleteCookiesByDomain(profileId, domain);
  }, [deleteCookiesByDomain, profileId]);

  const handleDeleteCookie = useCallback(async (domain: string, name: string) => {
    return deleteCookie(profileId, domain, name);
  }, [deleteCookie, profileId]);

  const handleClearAllLocalStorage = useCallback(async () => {
    return clearAllLocalStorage(profileId);
  }, [clearAllLocalStorage, profileId]);

  const handleDeleteLocalStorageByOrigin = useCallback(async (origin: string) => {
    return deleteLocalStorageByOrigin(profileId, origin);
  }, [deleteLocalStorageByOrigin, profileId]);

  const handleDeleteLocalStorageItem = useCallback(async (origin: string, name: string) => {
    return deleteLocalStorageItem(profileId, origin, name);
  }, [deleteLocalStorageItem, profileId]);

  // Service worker handlers
  const handleUnregisterAllServiceWorkers = useCallback(async () => {
    return unregisterAllServiceWorkers(profileId);
  }, [unregisterAllServiceWorkers, profileId]);

  const handleUnregisterServiceWorker = useCallback(async (scopeURL: string) => {
    return unregisterServiceWorker(profileId, scopeURL);
  }, [unregisterServiceWorker, profileId]);

  return {
    // Section navigation
    activeSection,
    setActiveSection,

    // Settings state
    preset,
    fingerprint,
    behavior,
    antiDetection,
    proxy,
    extraHeaders,

    // Settings actions
    applyPreset,
    updateFingerprint,
    updateBehavior,
    updateAntiDetection,
    updateProxy,
    addExtraHeader,
    updateExtraHeader,
    removeExtraHeader,

    // Storage state
    storageState,
    storageLoading,
    storageError,
    storageDeleting,

    // Storage delete actions
    clearAllCookies: handleClearAllCookies,
    deleteCookiesByDomain: handleDeleteCookiesByDomain,
    deleteCookie: handleDeleteCookie,
    clearAllLocalStorage: handleClearAllLocalStorage,
    deleteLocalStorageByOrigin: handleDeleteLocalStorageByOrigin,
    deleteLocalStorageItem: handleDeleteLocalStorageItem,

    // Service workers state
    serviceWorkers,
    swLoading,
    swError,
    swDeleting,

    // Service worker actions
    unregisterAllServiceWorkers: handleUnregisterAllServiceWorkers,
    unregisterServiceWorker: handleUnregisterServiceWorker,

    // Save state
    saving,
    error,
    isDirty,
    handleSave,
    reset,
  };
}
