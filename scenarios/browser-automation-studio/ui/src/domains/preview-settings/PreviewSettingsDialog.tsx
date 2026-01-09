/**
 * PreviewSettingsDialog Component
 *
 * Unified settings dialog for the Record page preview.
 * Combines stream quality settings and replay styling into
 * a tabbed interface with sidebar navigation.
 */

import { useState, useCallback } from 'react';
import { SlidersHorizontal } from 'lucide-react';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { PreviewSettingsSidebar } from './PreviewSettingsSidebar';
import { PreviewSettingsHeader } from './PreviewSettingsHeader';
import { usePreviewSettings } from './usePreviewSettings';
import {
  StreamSection,
  VisualSection,
  CursorSection,
  PlaybackSection,
  BrandingSection,
} from './sections';
import type { SectionId } from './types';

interface PreviewSettingsDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Callback to close the dialog */
  onClose: () => void;
  /** Session ID for live stream settings updates */
  sessionId: string | null;
}

export function PreviewSettingsDialog({
  isOpen,
  onClose,
  sessionId,
}: PreviewSettingsDialogProps) {
  const [showSavePresetDialog, setShowSavePresetDialog] = useState(false);
  const [newPresetName, setNewPresetName] = useState('');

  const {
    activeSection,
    setActiveSection,
    streamPreset,
    streamSettings,
    showStats,
    hasActiveSession,
    onStreamPresetChange,
    onStreamSettingsChange,
    onShowStatsChange,
    replay,
    setReplaySetting,
    onRandomize,
    onSavePreset,
  } = usePreviewSettings({ sessionId });

  const handleSavePreset = useCallback(() => {
    const trimmedName = newPresetName.trim();
    if (!trimmedName) return;
    onSavePreset(trimmedName);
    setNewPresetName('');
    setShowSavePresetDialog(false);
  }, [newPresetName, onSavePreset]);

  const renderSection = (section: SectionId) => {
    switch (section) {
      case 'stream':
        return (
          <StreamSection
            preset={streamPreset}
            settings={streamSettings}
            showStats={showStats}
            hasActiveSession={hasActiveSession}
            onPresetChange={onStreamPresetChange}
            onSettingsChange={onStreamSettingsChange}
            onShowStatsChange={onShowStatsChange}
          />
        );

      case 'visual':
        return (
          <VisualSection
            presentation={replay.presentation}
            background={replay.background}
            chromeTheme={replay.chromeTheme}
            browserScale={replay.browserScale}
            deviceFrameTheme={replay.deviceFrameTheme}
            onPresentationChange={(v) => setReplaySetting('presentation', v)}
            onBackgroundChange={(v) => setReplaySetting('background', v)}
            onChromeThemeChange={(v) => setReplaySetting('chromeTheme', v)}
            onBrowserScaleChange={(v) => setReplaySetting('browserScale', v)}
            onDeviceFrameThemeChange={(v) => setReplaySetting('deviceFrameTheme', v)}
          />
        );

      case 'cursor':
        return (
          <CursorSection
            cursorTheme={replay.cursorTheme}
            cursorInitialPosition={replay.cursorInitialPosition}
            cursorClickAnimation={replay.cursorClickAnimation}
            cursorScale={replay.cursorScale}
            cursorSpeedProfile={replay.cursorSpeedProfile}
            cursorPathStyle={replay.cursorPathStyle}
            onCursorThemeChange={(v) => setReplaySetting('cursorTheme', v)}
            onCursorInitialPositionChange={(v) => setReplaySetting('cursorInitialPosition', v)}
            onCursorClickAnimationChange={(v) => setReplaySetting('cursorClickAnimation', v)}
            onCursorScaleChange={(v) => setReplaySetting('cursorScale', v)}
            onCursorSpeedProfileChange={(v) => setReplaySetting('cursorSpeedProfile', v)}
            onCursorPathStyleChange={(v) => setReplaySetting('cursorPathStyle', v)}
          />
        );

      case 'playback':
        return (
          <PlaybackSection
            presentationWidth={replay.presentationWidth}
            presentationHeight={replay.presentationHeight}
            useCustomDimensions={replay.useCustomDimensions}
            frameDuration={replay.frameDuration}
            autoPlay={replay.autoPlay}
            loop={replay.loop}
            onPresentationWidthChange={(v) => setReplaySetting('presentationWidth', v)}
            onPresentationHeightChange={(v) => setReplaySetting('presentationHeight', v)}
            onUseCustomDimensionsChange={(v) => setReplaySetting('useCustomDimensions', v)}
            onFrameDurationChange={(v) => setReplaySetting('frameDuration', v)}
            onAutoPlayChange={(v) => setReplaySetting('autoPlay', v)}
            onLoopChange={(v) => setReplaySetting('loop', v)}
          />
        );

      case 'branding':
        return <BrandingSection />;

      default:
        return null;
    }
  };

  return (
    <>
      <ResponsiveDialog
        isOpen={isOpen}
        ariaLabel="Preview settings"
        size="xl"
        className="!p-0 w-full"
      >
        <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-h-[90vh] overflow-hidden flex flex-col">
          <PreviewSettingsHeader
            onClose={onClose}
            onRandomize={onRandomize}
            onSavePreset={() => setShowSavePresetDialog(true)}
          />

          {/* Body - Sidebar + Content */}
          <div className="flex-1 flex overflow-hidden">
            <PreviewSettingsSidebar
              activeSection={activeSection}
              onSectionChange={setActiveSection}
            />

            <div className="flex-1 overflow-y-auto p-6">
              {renderSection(activeSection)}
            </div>
          </div>
        </div>
      </ResponsiveDialog>

      {/* Save Preset Dialog */}
      <ResponsiveDialog
        isOpen={showSavePresetDialog}
        onDismiss={() => setShowSavePresetDialog(false)}
        ariaLabel="Save replay preset"
      >
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-6 w-full max-w-md">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
              <SlidersHorizontal size={16} className="text-blue-500" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Save Preset</h2>
              <p className="text-xs text-gray-500 dark:text-gray-400">Name this replay style for reuse.</p>
            </div>
          </div>
          <input
            type="text"
            value={newPresetName}
            onChange={(e) => setNewPresetName(e.target.value)}
            placeholder="My replay preset"
            className="w-full px-3 py-2 text-sm bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500/50 mb-4"
            autoFocus
          />
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setShowSavePresetDialog(false)}
              className="flex-1 px-4 py-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSavePreset}
              disabled={!newPresetName.trim()}
              className="flex-1 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Save Preset
            </button>
          </div>
        </div>
      </ResponsiveDialog>
    </>
  );
}
