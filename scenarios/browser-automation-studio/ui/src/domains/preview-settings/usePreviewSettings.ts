/**
 * usePreviewSettings Hook
 *
 * Orchestrates state management for the Preview Settings dialog.
 * Combines stream settings (from useStreamSettings) and replay settings
 * (from useSettingsStore) into a unified interface.
 */

import { useState, useCallback, useEffect } from 'react';
import { useStreamSettings, type StreamPreset, type StreamSettingsValues } from '@/domains/recording/capture/StreamSettings';
import { useSettingsStore } from '@stores/settingsStore';
import type { SectionId } from './types';

interface UsePreviewSettingsProps {
  /** Session ID for live stream settings updates */
  sessionId: string | null;
  /** Initial section to show */
  initialSection?: SectionId;
}

export function usePreviewSettings({
  sessionId,
  initialSection = 'stream',
}: UsePreviewSettingsProps) {
  // Section navigation
  const [activeSection, setActiveSection] = useState<SectionId>(initialSection);

  // Stream settings from existing hook
  const {
    preset: streamPreset,
    settings: streamSettings,
    customSettings: streamCustomSettings,
    showStats,
    setPreset: setStreamPreset,
    setCustomSettings: setStreamCustomSettings,
    setShowStats,
  } = useStreamSettings();

  // Replay settings from Zustand store
  const {
    replay,
    setReplaySetting,
    randomizeSettings,
    saveAsPreset,
  } = useSettingsStore();

  // Track if we need to send stream settings to server
  const [pendingStreamUpdate, setPendingStreamUpdate] = useState(false);

  // Send stream settings to server when they change (live update)
  useEffect(() => {
    if (!sessionId || !pendingStreamUpdate) return;

    const updateStreamSettings = async () => {
      try {
        const response = await fetch(`/api/v1/recordings/live/${sessionId}/stream-settings`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            quality: streamSettings.quality,
            fps: streamSettings.fps,
            perfMode: showStats,
          }),
        });
        if (!response.ok) {
          console.warn('Failed to update stream settings:', await response.text());
        }
      } catch (err) {
        console.warn('Failed to update stream settings:', err);
      }
    };

    void updateStreamSettings();
    setPendingStreamUpdate(false);
  }, [sessionId, streamSettings, showStats, pendingStreamUpdate]);

  // Stream settings handlers
  const handleStreamPresetChange = useCallback((preset: StreamPreset) => {
    setStreamPreset(preset);
    setPendingStreamUpdate(true);
  }, [setStreamPreset]);

  const handleStreamSettingsChange = useCallback((settings: StreamSettingsValues) => {
    setStreamCustomSettings(settings);
    setPendingStreamUpdate(true);
  }, [setStreamCustomSettings]);

  const handleShowStatsChange = useCallback((show: boolean) => {
    setShowStats(show);
    setPendingStreamUpdate(true);
  }, [setShowStats]);

  // Replay settings handlers (these auto-persist via Zustand)
  const handleRandomize = useCallback(() => {
    randomizeSettings();
  }, [randomizeSettings]);

  const handleSavePreset = useCallback((name: string) => {
    saveAsPreset(name);
  }, [saveAsPreset]);

  return {
    // Section navigation
    activeSection,
    setActiveSection,

    // Stream settings
    streamPreset,
    streamSettings,
    streamCustomSettings,
    showStats,
    hasActiveSession: !!sessionId,
    onStreamPresetChange: handleStreamPresetChange,
    onStreamSettingsChange: handleStreamSettingsChange,
    onShowStatsChange: handleShowStatsChange,

    // Replay settings (direct access to store values)
    replay,
    setReplaySetting,

    // Actions
    onRandomize: handleRandomize,
    onSavePreset: handleSavePreset,
  };
}
