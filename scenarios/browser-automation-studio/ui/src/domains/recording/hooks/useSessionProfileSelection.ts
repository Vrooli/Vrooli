/**
 * useSessionProfileSelection Hook
 *
 * Manages session profile selection state and handlers for the recording session.
 * Extracted from RecordingSession.tsx to reduce component complexity.
 *
 * Features:
 * - Profile selection with localStorage persistence
 * - Profile initialization from localStorage or API default
 * - Fallback handling when selected profile is deleted
 * - Configuration modal state management
 */

import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { RecordingSessionProfile, BrowserProfile } from '../types/types';

/** LocalStorage key for persisting the last selected session profile ID */
const LAST_SELECTED_PROFILE_KEY = 'bas_last_selected_session_profile_id';

/** Return type of useSessionProfiles hook */
interface SessionProfilesHook {
  profiles: RecordingSessionProfile[];
  loading: boolean;
  error: string | null;
  create: () => Promise<RecordingSessionProfile | null>;
  remove: (id: string) => Promise<void>;
  refresh: () => Promise<void>;
  getDefaultProfileId: () => string | null;
  updateBrowserProfile: (profileId: string, browserProfile: BrowserProfile) => Promise<void>;
}

interface UseSessionProfileSelectionOptions {
  /** Session profile ID from the recording session hook */
  sessionProfileId: string | null;
  /** Callback to set session profile ID on the recording session */
  setSessionProfileId: (id: string | null) => void;
  /** Session profiles hook return value */
  sessionProfiles: SessionProfilesHook;
}

interface UseSessionProfileSelectionReturn {
  /** Currently selected profile ID */
  selectedProfileId: string | null;
  /** Set the selected profile ID (also updates session and localStorage) */
  handleSelectSessionProfile: (profileId: string | null) => void;
  /** Profile being configured in the modal */
  configuringProfile: RecordingSessionProfile | null;
  /** Section to open in session settings dialog */
  configuringSection: 'history' | undefined;
  /** Create a new session profile */
  handleCreateSessionProfile: () => Promise<void>;
  /** Open configuration modal for a profile */
  handleConfigureSession: (profileId: string) => void;
  /** Open history settings for current profile */
  handleOpenHistorySettings: () => void;
  /** Navigate to session settings page */
  handleNavigateToSessionSettings: () => void;
  /** Save browser profile updates */
  handleSaveBrowserProfile: (browserProfile: BrowserProfile) => Promise<void>;
  /** Close the configuration modal */
  closeConfigureModal: () => void;
}

export function useSessionProfileSelection({
  sessionProfileId,
  setSessionProfileId,
  sessionProfiles,
}: UseSessionProfileSelectionOptions): UseSessionProfileSelectionReturn {
  const navigate = useNavigate();

  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [configuringProfile, setConfiguringProfile] = useState<RecordingSessionProfile | null>(null);
  const [configuringSection, setConfiguringSection] = useState<'history' | undefined>(undefined);

  // Initialize profile selection - prefer localStorage, then API, then default
  useEffect(() => {
    // Skip if we already have a selection
    if (selectedProfileId) return;
    // Wait for profiles to load
    if (sessionProfiles.profiles.length === 0) return;

    // Priority 1: Check localStorage for last explicitly selected profile
    const storedProfileId = localStorage.getItem(LAST_SELECTED_PROFILE_KEY);
    if (storedProfileId && sessionProfiles.profiles.some((p) => p.id === storedProfileId)) {
      setSelectedProfileId(storedProfileId);
      setSessionProfileId(storedProfileId);
      return;
    }

    // Priority 2: Use sessionProfileId from hook or API default
    const maybeDefault = sessionProfileId ?? sessionProfiles.getDefaultProfileId();
    if (maybeDefault) {
      setSelectedProfileId(maybeDefault);
      setSessionProfileId(maybeDefault);
    }
  }, [selectedProfileId, sessionProfileId, sessionProfiles.profiles, sessionProfiles.getDefaultProfileId, setSessionProfileId]);

  // Sync when sessionProfileId changes externally
  useEffect(() => {
    if (sessionProfileId && sessionProfileId !== selectedProfileId) {
      setSelectedProfileId(sessionProfileId);
    }
  }, [sessionProfileId, selectedProfileId]);

  // Handle case where selected profile no longer exists (was deleted)
  useEffect(() => {
    if (
      selectedProfileId &&
      sessionProfiles.profiles.length > 0 &&
      !sessionProfiles.profiles.some((p) => p.id === selectedProfileId)
    ) {
      const fallback = sessionProfiles.getDefaultProfileId();
      setSelectedProfileId(fallback);
      setSessionProfileId(fallback);
      // Update localStorage with the fallback or clear it
      if (fallback) {
        localStorage.setItem(LAST_SELECTED_PROFILE_KEY, fallback);
      } else {
        localStorage.removeItem(LAST_SELECTED_PROFILE_KEY);
      }
    }
  }, [selectedProfileId, sessionProfiles.getDefaultProfileId, sessionProfiles.profiles, setSessionProfileId]);

  // Select a session profile (with localStorage persistence)
  const handleSelectSessionProfile = useCallback(
    (profileId: string | null) => {
      setSelectedProfileId(profileId);
      setSessionProfileId(profileId);
      // Persist to localStorage for next visit
      if (profileId) {
        localStorage.setItem(LAST_SELECTED_PROFILE_KEY, profileId);
      } else {
        localStorage.removeItem(LAST_SELECTED_PROFILE_KEY);
      }
    },
    [setSessionProfileId]
  );

  // Create a new session profile
  const handleCreateSessionProfile = useCallback(async () => {
    const created = await sessionProfiles.create();
    if (created) {
      setSelectedProfileId(created.id);
      setSessionProfileId(created.id);
      // Persist newly created profile to localStorage
      localStorage.setItem(LAST_SELECTED_PROFILE_KEY, created.id);
    }
  }, [sessionProfiles, setSessionProfileId]);

  // Open configuration modal for a profile
  const handleConfigureSession = useCallback(
    (profileId: string) => {
      const profile = sessionProfiles.profiles.find((p) => p.id === profileId);
      if (profile) {
        setConfiguringSection(undefined);
        setConfiguringProfile(profile);
      }
    },
    [sessionProfiles.profiles]
  );

  // Open history settings for current profile
  const handleOpenHistorySettings = useCallback(() => {
    const profileId = sessionProfileId || selectedProfileId;
    const currentProfile = sessionProfiles.profiles.find((p) => p.id === profileId);
    if (currentProfile) {
      setConfiguringSection('history');
      setConfiguringProfile(currentProfile);
    }
  }, [sessionProfiles.profiles, sessionProfileId, selectedProfileId]);

  // Navigate to session settings page
  const handleNavigateToSessionSettings = useCallback(() => {
    navigate('/settings?tab=sessions');
  }, [navigate]);

  // Save browser profile updates
  const handleSaveBrowserProfile = useCallback(
    async (browserProfile: BrowserProfile) => {
      if (!configuringProfile) return;
      await sessionProfiles.updateBrowserProfile(configuringProfile.id, browserProfile);
      await sessionProfiles.refresh();
    },
    [configuringProfile, sessionProfiles]
  );

  // Close the configuration modal
  const closeConfigureModal = useCallback(() => {
    setConfiguringProfile(null);
    setConfiguringSection(undefined);
  }, []);

  return {
    selectedProfileId,
    handleSelectSessionProfile,
    configuringProfile,
    configuringSection,
    handleCreateSessionProfile,
    handleConfigureSession,
    handleOpenHistorySettings,
    handleNavigateToSessionSettings,
    handleSaveBrowserProfile,
    closeConfigureModal,
  };
}
