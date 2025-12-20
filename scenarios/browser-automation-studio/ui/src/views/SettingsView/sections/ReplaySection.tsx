import { useMemo, useState, useCallback } from 'react';
import {
  Sparkles,
  HelpCircle,
  Bookmark,
  ChevronDown,
  Check,
  Trash2,
  Shuffle,
  Save,
  Monitor,
  Smartphone,
  Tv,
} from 'lucide-react';
import { useSettingsStore, BUILT_IN_PRESETS } from '@stores/settingsStore';
import {
  REPLAY_CHROME_OPTIONS,
  REPLAY_BACKGROUND_OPTIONS,
  BACKGROUND_GROUP_ORDER,
  REPLAY_CURSOR_OPTIONS,
  CURSOR_GROUP_ORDER,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  CURSOR_SCALE_MIN,
  CURSOR_SCALE_MAX,
} from '@/domains/exports/replay/replayThemeOptions';
import type { CursorSpeedProfile, CursorPathStyle } from '@/domains/exports/replay/ReplayPlayer';
import Tooltip from '@shared/ui/Tooltip';
import { RangeSlider } from '@shared/ui';
import { WatermarkSettings } from '@/domains/exports/replay/WatermarkSettings';
import { IntroCardSettings } from '@/domains/exports/replay/IntroCardSettings';
import { OutroCardSettings } from '@/domains/exports/replay/OutroCardSettings';
import { SettingSection, OptionGrid, ToggleSwitch } from './shared';

const SPEED_PROFILE_OPTIONS: Array<{ id: CursorSpeedProfile; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Consistent motion between frames' },
  { id: 'easeIn', label: 'Ease In', description: 'Begin slowly, accelerate toward target' },
  { id: 'easeOut', label: 'Ease Out', description: 'Move quickly, settle into target' },
  { id: 'easeInOut', label: 'Ease In/Out', description: 'Smooth acceleration and deceleration' },
  { id: 'instant', label: 'Instant', description: 'Jump directly at the end' },
];

const PATH_STYLE_OPTIONS: Array<{ id: CursorPathStyle; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Direct line between positions' },
  { id: 'parabolicUp', label: 'Arc Up', description: 'Arcs upward before target' },
  { id: 'parabolicDown', label: 'Arc Down', description: 'Arcs downward before target' },
  { id: 'cubic', label: 'Cubic', description: 'Smooth S-curve motion' },
  { id: 'pseudorandom', label: 'Organic', description: 'Natural random waypoints' },
];

const DIMENSION_PRESETS = [
  {
    id: 'widescreen-720',
    label: '1280 × 720',
    description: 'Best for YouTube, demos, decks',
    width: 1280,
    height: 720,
    icon: <Monitor size={16} />,
    ratioLabel: '16:9',
  },
  {
    id: 'full-hd',
    label: '1920 × 1080',
    description: 'Best for desktop screenshares',
    width: 1920,
    height: 1080,
    icon: <Tv size={16} />,
    ratioLabel: '16:9',
  },
  {
    id: 'instagram-feed',
    label: '1080 × 1350',
    description: 'Best for Instagram feed posts',
    width: 1080,
    height: 1350,
    icon: <Smartphone size={16} />,
    ratioLabel: '4:5',
  },
  {
    id: 'tiktok',
    label: '1080 × 1920',
    description: 'Best for TikTok, Reels, Shorts',
    width: 1080,
    height: 1920,
    icon: <Smartphone size={16} />,
    ratioLabel: '9:16',
  },
] as const;

interface ReplaySectionProps {
  onRandomize: () => void;
  onSavePreset: () => void;
}

export function ReplaySection({ onRandomize, onSavePreset }: ReplaySectionProps) {
  const {
    replay,
    userPresets,
    activePresetId,
    setReplaySetting,
    loadPreset,
    deletePreset,
    getAllPresets,
  } = useSettingsStore();

  const [showPresetDropdown, setShowPresetDropdown] = useState(false);
  const [presetToDelete, setPresetToDelete] = useState<string | null>(null);

  const allPresets = useMemo(() => getAllPresets(), [getAllPresets, userPresets]);

  const activePreset = useMemo(() => {
    if (!activePresetId) return null;
    return allPresets.find((p) => p.id === activePresetId) || null;
  }, [activePresetId, allPresets]);

  const handlePresetSelect = useCallback((presetId: string) => {
    loadPreset(presetId);
    setShowPresetDropdown(false);
  }, [loadPreset]);

  const handleDeletePreset = useCallback((presetId: string) => {
    deletePreset(presetId);
    setPresetToDelete(null);
  }, [deletePreset]);

  // Group backgrounds by kind
  const groupedBackgrounds = useMemo(() => {
    return BACKGROUND_GROUP_ORDER.map((group) => ({
      ...group,
      options: REPLAY_BACKGROUND_OPTIONS.filter((bg) => bg.kind === group.id),
    }));
  }, []);

  // Group cursors
  const groupedCursors = useMemo(() => {
    return CURSOR_GROUP_ORDER.map((group) => ({
      ...group,
      options: REPLAY_CURSOR_OPTIONS.filter((c) => c.group === group.id),
    }));
  }, []);

  const activeDimensionPreset = useMemo(() => {
    return DIMENSION_PRESETS.find(
      (preset) =>
        preset.width === replay.presentationWidth &&
        preset.height === replay.presentationHeight
    ) ?? DIMENSION_PRESETS[0];
  }, [replay.presentationHeight, replay.presentationWidth]);

  return (
    <div className="space-y-4">
      {/* Presets Section */}
      <div className="relative mb-6">
        <div className="flex items-center gap-3 mb-3">
          <Sparkles size={18} className="text-amber-400" />
          <span className="text-sm font-medium text-surface">Quick Presets</span>
          <Tooltip content="Apply a preset to quickly configure all replay settings at once. You can also save your own custom presets.">
            <HelpCircle size={14} className="text-gray-500" />
          </Tooltip>
        </div>

        {/* Preset Selector */}
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowPresetDropdown(!showPresetDropdown)}
            className="w-full flex items-center justify-between px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-surface hover:border-gray-600 transition-colors"
          >
            <div className="flex items-center gap-3">
              <Bookmark size={16} className={activePreset ? 'text-flow-accent' : 'text-gray-500'} />
              <span>{activePreset ? activePreset.name : 'Select a preset...'}</span>
              {activePreset?.isBuiltIn && (
                <span className="text-xs px-2 py-0.5 bg-gray-700 text-gray-400 rounded">Built-in</span>
              )}
            </div>
            <ChevronDown size={16} className={`text-gray-400 transition-transform ${showPresetDropdown ? 'rotate-180' : ''}`} />
          </button>

          {showPresetDropdown && (
            <>
              <div className="fixed inset-0 z-20" onClick={() => setShowPresetDropdown(false)} />
              <div className="absolute left-0 right-0 top-full mt-2 z-30 bg-gray-900 border border-gray-700 rounded-lg shadow-2xl overflow-hidden max-h-80 overflow-y-auto">
                <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide bg-gray-800/50">
                  Built-in Presets
                </div>
                {BUILT_IN_PRESETS.map((preset) => (
                  <button
                    key={preset.id}
                    type="button"
                    onClick={() => handlePresetSelect(preset.id)}
                    className={`w-full flex items-center justify-between px-4 py-3 text-left hover:bg-gray-800 transition-colors ${
                      activePresetId === preset.id ? 'bg-flow-accent/10 text-flow-accent' : 'text-surface'
                    }`}
                  >
                    <div className="flex items-center gap-3">
                      {activePresetId === preset.id ? <Check size={16} className="text-flow-accent" /> : <Bookmark size={16} className="text-gray-500" />}
                      <span>{preset.name}</span>
                    </div>
                  </button>
                ))}

                {userPresets.length > 0 && (
                  <>
                    <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide bg-gray-800/50 border-t border-gray-700">
                      Your Presets
                    </div>
                    {userPresets.map((preset) => (
                      <div
                        key={preset.id}
                        className={`flex items-center justify-between hover:bg-gray-800 transition-colors ${
                          activePresetId === preset.id ? 'bg-flow-accent/10' : ''
                        }`}
                      >
                        <button
                          type="button"
                          onClick={() => handlePresetSelect(preset.id)}
                          className={`flex-1 flex items-center gap-3 px-4 py-3 text-left ${
                            activePresetId === preset.id ? 'text-flow-accent' : 'text-surface'
                          }`}
                        >
                          {activePresetId === preset.id ? <Check size={16} className="text-flow-accent" /> : <Bookmark size={16} className="text-gray-500" />}
                          <span>{preset.name}</span>
                        </button>
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowPresetDropdown(false);
                            setPresetToDelete(preset.id);
                          }}
                          className="p-2 mr-2 text-gray-500 hover:text-red-400 hover:bg-red-900/20 rounded transition-colors"
                          title="Delete preset"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    ))}
                  </>
                )}

                <div className="border-t border-gray-700 p-2 bg-gray-800/30">
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => { setShowPresetDropdown(false); onRandomize(); }}
                      className="flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm text-amber-400 hover:bg-amber-900/20 rounded-lg transition-colors"
                    >
                      <Shuffle size={14} />
                      Random
                    </button>
                    <button
                      type="button"
                      onClick={() => { setShowPresetDropdown(false); onSavePreset(); }}
                      className="flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm text-flow-accent hover:bg-blue-900/20 rounded-lg transition-colors"
                    >
                      <Save size={14} />
                      Save Current
                    </button>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>

        {!activePreset && (
          <p className="mt-2 text-xs text-gray-500 flex items-center gap-1">
            <HelpCircle size={12} />
            Settings have been customized. Save as a preset to reuse them.
          </p>
        )}
      </div>

      <div className="relative my-6">
        <div className="absolute inset-0 flex items-center" aria-hidden="true">
          <div className="w-full border-t border-gray-700"></div>
        </div>
        <div className="relative flex justify-center">
          <span className="bg-flow-bg px-3 text-xs text-gray-500 uppercase tracking-wider">Individual Settings</span>
        </div>
      </div>

      <SettingSection title="Browser Chrome" tooltip="Choose how the browser window frame looks.">
        <OptionGrid
          options={REPLAY_CHROME_OPTIONS}
          value={replay.chromeTheme}
          onChange={(v) => setReplaySetting('chromeTheme', v)}
          columns={4}
        />
      </SettingSection>

      <SettingSection title="Replay Dimensions" tooltip="Set the default canvas size for styled replays.">
        <div className="space-y-4">
          <OptionGrid
            options={DIMENSION_PRESETS.map((preset) => ({
              id: preset.id,
              label: preset.label,
              description: preset.description,
              preview: (
                <div className="flex flex-col items-center gap-1 text-gray-400">
                  <div className="flex items-center gap-1 text-[11px]">
                    {preset.icon}
                    <span>{preset.ratioLabel}</span>
                  </div>
                  <div className="h-6 w-10 rounded border border-gray-600/80 bg-gray-900/60" />
                </div>
              ),
            }))}
            value={activeDimensionPreset.id}
            onChange={(id) => {
              const preset = DIMENSION_PRESETS.find((item) => item.id === id);
              if (!preset) return;
              setReplaySetting('presentationWidth', preset.width);
              setReplaySetting('presentationHeight', preset.height);
              setReplaySetting('useCustomDimensions', false);
            }}
            columns={2}
          />

          <ToggleSwitch
            checked={replay.useCustomDimensions}
            onChange={(v) => setReplaySetting('useCustomDimensions', v)}
            label="Use custom dimensions"
            description="Manually set width and height in pixels"
          />

          {replay.useCustomDimensions && (
            <div className="grid grid-cols-2 gap-3">
              <label className="text-xs text-gray-400 flex flex-col gap-1">
                Width (px)
                <input
                  type="number"
                  min={320}
                  max={3840}
                  step={1}
                  value={replay.presentationWidth}
                  onChange={(e) => {
                    const next = Math.round(Number(e.target.value));
                    const clamped = Number.isFinite(next) ? Math.min(3840, Math.max(320, next)) : 1280;
                    setReplaySetting('presentationWidth', clamped);
                  }}
                  className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                />
              </label>
              <label className="text-xs text-gray-400 flex flex-col gap-1">
                Height (px)
                <input
                  type="number"
                  min={320}
                  max={3840}
                  step={1}
                  value={replay.presentationHeight}
                  onChange={(e) => {
                    const next = Math.round(Number(e.target.value));
                    const clamped = Number.isFinite(next) ? Math.min(3840, Math.max(320, next)) : 720;
                    setReplaySetting('presentationHeight', clamped);
                  }}
                  className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                />
              </label>
            </div>
          )}
        </div>
      </SettingSection>

      <SettingSection title="Background" tooltip="The backdrop behind the browser window.">
        <div className="space-y-4">
          {groupedBackgrounds.map((group) => (
            <div key={group.id}>
              <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">{group.label}</div>
              <OptionGrid
                options={group.options}
                value={replay.backgroundTheme}
                onChange={(v) => setReplaySetting('backgroundTheme', v)}
                columns={4}
              />
            </div>
          ))}
        </div>
      </SettingSection>

      <SettingSection title="Cursor Style" tooltip="The virtual cursor shown during replay.">
        <div className="space-y-4">
          {groupedCursors.map((group) => (
            <div key={group.id}>
              <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">{group.label}</div>
              <OptionGrid
                options={group.options}
                value={replay.cursorTheme}
                onChange={(v) => setReplaySetting('cursorTheme', v)}
                columns={3}
              />
            </div>
          ))}
        </div>
      </SettingSection>

      <SettingSection title="Cursor Size" tooltip="Scale the cursor size.">
        <div className="flex items-center gap-4">
          <RangeSlider
            min={CURSOR_SCALE_MIN}
            max={CURSOR_SCALE_MAX}
            step={0.1}
            value={replay.cursorScale}
            onChange={(next) => setReplaySetting('cursorScale', next)}
            ariaLabel="Cursor size"
            className="flex-1"
          />
          <span className="text-sm text-gray-400 w-12 text-right">{(replay.cursorScale * 100).toFixed(0)}%</span>
        </div>
      </SettingSection>

      <SettingSection title="Click Animation" tooltip="Adds visual emphasis when the cursor clicks.">
        <OptionGrid
          options={REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS}
          value={replay.cursorClickAnimation}
          onChange={(v) => setReplaySetting('cursorClickAnimation', v)}
          columns={3}
        />
      </SettingSection>

      <SettingSection title="Initial Cursor Position" tooltip="Where the cursor appears at the start." defaultOpen={false}>
        <OptionGrid
          options={REPLAY_CURSOR_POSITIONS}
          value={replay.cursorInitialPosition}
          onChange={(v) => setReplaySetting('cursorInitialPosition', v)}
          columns={3}
        />
      </SettingSection>

      <SettingSection title="Cursor Motion" tooltip="Controls how the cursor moves between actions." defaultOpen={false}>
        <div className="space-y-4">
          <div>
            <label className="text-sm text-gray-400 mb-2 block">Speed Profile</label>
            <OptionGrid
              options={SPEED_PROFILE_OPTIONS}
              value={replay.cursorSpeedProfile}
              onChange={(v) => setReplaySetting('cursorSpeedProfile', v)}
              columns={3}
            />
          </div>
          <div>
            <label className="text-sm text-gray-400 mb-2 block">Path Style</label>
            <OptionGrid
              options={PATH_STYLE_OPTIONS}
              value={replay.cursorPathStyle}
              onChange={(v) => setReplaySetting('cursorPathStyle', v)}
              columns={3}
            />
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Playback" tooltip="Control timing and behavior of replay animations." defaultOpen={false}>
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Frame Duration</label>
              <span className="text-sm font-medium text-flow-accent">{(replay.frameDuration / 1000).toFixed(1)}s</span>
            </div>
            <RangeSlider
              min={800}
              max={6000}
              step={100}
              value={replay.frameDuration}
              onChange={(next) => setReplaySetting('frameDuration', next)}
              ariaLabel="Frame duration"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Fast (0.8s)</span>
              <span>Slow (6s)</span>
            </div>
          </div>
          <ToggleSwitch
            checked={replay.autoPlay}
            onChange={(v) => setReplaySetting('autoPlay', v)}
            label="Auto-play replays"
            description="Start playing immediately when opened"
            className="border-t border-gray-800"
          />
          <ToggleSwitch
            checked={replay.loop}
            onChange={(v) => setReplaySetting('loop', v)}
            label="Loop replays"
            description="Restart automatically when finished"
            className="border-t border-gray-800"
          />
        </div>
      </SettingSection>

      <div className="relative my-6">
        <div className="absolute inset-0 flex items-center" aria-hidden="true">
          <div className="w-full border-t border-gray-700"></div>
        </div>
        <div className="relative flex justify-center">
          <span className="bg-flow-bg px-3 text-xs text-gray-500 uppercase tracking-wider">Branding</span>
        </div>
      </div>

      <SettingSection title="Watermark" tooltip="Add a logo overlay to your replays.">
        <WatermarkSettings />
      </SettingSection>

      <SettingSection title="Intro Card" tooltip="Show a title slide before the replay starts." defaultOpen={false}>
        <IntroCardSettings />
      </SettingSection>

      <SettingSection title="Outro Card" tooltip="Show a closing slide with CTA after the replay ends." defaultOpen={false}>
        <OutroCardSettings />
      </SettingSection>

      {/* Delete Preset Confirmation */}
      {presetToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl animate-fade-in">
            <h3 className="text-lg font-semibold text-surface mb-4 flex items-center gap-2">
              <Trash2 size={20} className="text-red-400" />
              Delete Preset
            </h3>
            <p className="text-sm text-gray-400 mb-6">
              Are you sure you want to delete this preset? This action cannot be undone.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setPresetToDelete(null)}
                className="flex-1 px-4 py-2 text-subtle hover:text-surface hover:bg-gray-800 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDeletePreset(presetToDelete)}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors flex items-center justify-center gap-2"
              >
                <Trash2 size={16} />
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
