import { useState, useCallback, useMemo } from 'react';
import type {
  BrowserProfile,
  ProfilePreset,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
  ProxySettings,
} from '@/domains/recording/types/types';
import { PRESET_CONFIGS } from '../../../settings';

export interface UseProfileSettingsProps {
  /** Initial profile to populate settings from */
  initialProfile?: BrowserProfile;
}

export interface UseProfileSettingsReturn {
  // State
  preset: ProfilePreset;
  fingerprint: FingerprintSettings;
  behavior: BehaviorSettings;
  antiDetection: AntiDetectionSettings;
  proxy: ProxySettings;
  extraHeaders: Record<string, string>;

  // Actions
  applyPreset: (preset: ProfilePreset) => void;
  updateFingerprint: <K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => void;
  updateBehavior: <K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => void;
  updateAntiDetection: <K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => void;
  updateProxy: <K extends keyof ProxySettings>(key: K, value: ProxySettings[K]) => void;
  addExtraHeader: (key: string, value: string) => void;
  updateExtraHeader: (oldKey: string, newKey: string, value: string) => void;
  removeExtraHeader: (key: string) => void;
  resetToInitial: () => void;

  // Derived
  getCurrentProfile: () => BrowserProfile;
  hasChanges: boolean;
}

/**
 * Shallow equality check for objects (one level deep).
 */
function shallowEqual<T extends object>(a: T, b: T): boolean {
  const keysA = Object.keys(a) as Array<keyof T>;
  const keysB = Object.keys(b) as Array<keyof T>;

  if (keysA.length !== keysB.length) return false;

  for (const key of keysA) {
    if (a[key] !== b[key]) return false;
  }

  return true;
}

/**
 * Manages all profile settings state and update operations.
 *
 * Responsibilities:
 * - Hold all settings state (preset, fingerprint, behavior, antiDetection, proxy, extraHeaders)
 * - Apply preset configurations
 * - Individual setting updates
 * - Track if changes exist vs initial profile
 * - Provide current profile for saving
 */
export function useProfileSettings({
  initialProfile,
}: UseProfileSettingsProps = {}): UseProfileSettingsReturn {
  // Settings state
  const [preset, setPreset] = useState<ProfilePreset>(initialProfile?.preset ?? 'none');
  const [fingerprint, setFingerprint] = useState<FingerprintSettings>(initialProfile?.fingerprint ?? {});
  const [behavior, setBehavior] = useState<BehaviorSettings>(initialProfile?.behavior ?? {});
  const [antiDetection, setAntiDetection] = useState<AntiDetectionSettings>(initialProfile?.anti_detection ?? {});
  const [proxy, setProxy] = useState<ProxySettings>(initialProfile?.proxy ?? {});
  const [extraHeaders, setExtraHeaders] = useState<Record<string, string>>(initialProfile?.extra_headers ?? {});

  // Apply preset configurations
  const applyPreset = useCallback((newPreset: ProfilePreset) => {
    setPreset(newPreset);
    const config = PRESET_CONFIGS[newPreset];
    setBehavior((prev) => ({ ...prev, ...config.behavior }));
    setAntiDetection((prev) => ({ ...prev, ...config.antiDetection }));
  }, []);

  // Individual setting updates
  const updateFingerprint = useCallback(<K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => {
    setFingerprint((prev) => ({ ...prev, [key]: value }));
  }, []);

  const updateBehavior = useCallback(<K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => {
    setBehavior((prev) => ({ ...prev, [key]: value }));
  }, []);

  const updateAntiDetection = useCallback(<K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => {
    setAntiDetection((prev) => ({ ...prev, [key]: value }));
  }, []);

  const updateProxy = useCallback(<K extends keyof ProxySettings>(key: K, value: ProxySettings[K]) => {
    setProxy((prev) => ({ ...prev, [key]: value }));
  }, []);

  // Extra headers operations
  const addExtraHeader = useCallback((key: string, value: string) => {
    setExtraHeaders((prev) => ({ ...prev, [key]: value }));
  }, []);

  const updateExtraHeader = useCallback((oldKey: string, newKey: string, value: string) => {
    setExtraHeaders((prev) => {
      const { [oldKey]: _, ...rest } = prev;
      return { ...rest, [newKey]: value };
    });
  }, []);

  const removeExtraHeader = useCallback((key: string) => {
    setExtraHeaders((prev) => {
      const { [key]: _, ...rest } = prev;
      return rest;
    });
  }, []);

  // Reset to initial values
  const resetToInitial = useCallback(() => {
    setPreset(initialProfile?.preset ?? 'none');
    setFingerprint(initialProfile?.fingerprint ?? {});
    setBehavior(initialProfile?.behavior ?? {});
    setAntiDetection(initialProfile?.anti_detection ?? {});
    setProxy(initialProfile?.proxy ?? {});
    setExtraHeaders(initialProfile?.extra_headers ?? {});
  }, [initialProfile]);

  // Get current profile for saving
  const getCurrentProfile = useCallback((): BrowserProfile => {
    return {
      preset,
      fingerprint: Object.keys(fingerprint).length > 0 ? fingerprint : undefined,
      behavior: Object.keys(behavior).length > 0 ? behavior : undefined,
      anti_detection: Object.keys(antiDetection).length > 0 ? antiDetection : undefined,
      proxy: Object.keys(proxy).length > 0 ? proxy : undefined,
      extra_headers: Object.keys(extraHeaders).length > 0 ? extraHeaders : undefined,
    };
  }, [preset, fingerprint, behavior, antiDetection, proxy, extraHeaders]);

  // Derive hasChanges by comparing current state to initial
  const hasChanges = useMemo(() => {
    const initialPreset = initialProfile?.preset ?? 'none';
    const initialFingerprint = initialProfile?.fingerprint ?? {};
    const initialBehavior = initialProfile?.behavior ?? {};
    const initialAntiDetection = initialProfile?.anti_detection ?? {};
    const initialProxy = initialProfile?.proxy ?? {};
    const initialExtraHeaders = initialProfile?.extra_headers ?? {};

    return (
      preset !== initialPreset ||
      !shallowEqual(fingerprint, initialFingerprint) ||
      !shallowEqual(behavior, initialBehavior) ||
      !shallowEqual(antiDetection, initialAntiDetection) ||
      !shallowEqual(proxy, initialProxy) ||
      !shallowEqual(extraHeaders, initialExtraHeaders)
    );
  }, [preset, fingerprint, behavior, antiDetection, proxy, extraHeaders, initialProfile]);

  return {
    // State
    preset,
    fingerprint,
    behavior,
    antiDetection,
    proxy,
    extraHeaders,

    // Actions
    applyPreset,
    updateFingerprint,
    updateBehavior,
    updateAntiDetection,
    updateProxy,
    addExtraHeader,
    updateExtraHeader,
    removeExtraHeader,
    resetToInitial,

    // Derived
    getCurrentProfile,
    hasChanges,
  };
}
