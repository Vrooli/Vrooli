import { useMemo, useState, useCallback, lazy, Suspense } from 'react';
import {
  ArrowLeft,
  Settings,
  Film,
  RotateCcw,
  ChevronDown,
  ChevronRight,
  HelpCircle,
  Play,
  Pause,
} from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
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
} from '../execution/replay/replayThemeOptions';
import type {
  CursorSpeedProfile,
  CursorPathStyle,
  ReplayFrame,
} from '../execution/ReplayPlayer';
import Tooltip from '@shared/ui/Tooltip';

const ReplayPlayer = lazy(() => import('../execution/ReplayPlayer'));

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

interface SettingsPageProps {
  onBack: () => void;
}

// Demo frames for the live preview
const createDemoFrames = (): ReplayFrame[] => [
  {
    id: 'demo-1',
    stepIndex: 0,
    stepType: 'navigate',
    status: 'completed',
    success: true,
    durationMs: 1200,
    finalUrl: 'https://example.com',
    screenshot: {
      artifactId: 'demo-screenshot-1',
      url: 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080"%3E%3Crect fill="%231e293b" width="1920" height="1080"/%3E%3Ctext x="960" y="540" font-family="system-ui" font-size="48" fill="%2394a3b8" text-anchor="middle"%3EExample.com%3C/text%3E%3Crect x="760" y="400" width="400" height="60" rx="8" fill="%233b82f6"/%3E%3Ctext x="960" y="440" font-family="system-ui" font-size="24" fill="white" text-anchor="middle"%3EGet Started%3C/text%3E%3C/svg%3E',
      width: 1920,
      height: 1080,
    },
    clickPosition: { x: 960, y: 430 },
  },
  {
    id: 'demo-2',
    stepIndex: 1,
    stepType: 'click',
    status: 'completed',
    success: true,
    durationMs: 800,
    screenshot: {
      artifactId: 'demo-screenshot-2',
      url: 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080"%3E%3Crect fill="%230f172a" width="1920" height="1080"/%3E%3Ctext x="960" y="300" font-family="system-ui" font-size="56" fill="white" text-anchor="middle"%3EWelcome!%3C/text%3E%3Ctext x="960" y="380" font-family="system-ui" font-size="24" fill="%2394a3b8" text-anchor="middle"%3EYour automation is running%3C/text%3E%3Crect x="660" y="500" width="600" height="200" rx="12" fill="%231e293b" stroke="%233b82f6" stroke-width="2"/%3E%3Ctext x="960" y="580" font-family="system-ui" font-size="20" fill="%2394a3b8" text-anchor="middle"%3EEnter your email%3C/text%3E%3Crect x="710" y="620" width="500" height="50" rx="6" fill="%230f172a"/%3E%3C/svg%3E',
      width: 1920,
      height: 1080,
    },
    clickPosition: { x: 960, y: 645 },
    highlightRegions: [
      { boundingBox: { x: 660, y: 500, width: 600, height: 200 }, color: 'rgba(59,130,246,0.3)' },
    ],
  },
  {
    id: 'demo-3',
    stepIndex: 2,
    stepType: 'type',
    status: 'completed',
    success: true,
    durationMs: 1500,
    screenshot: {
      artifactId: 'demo-screenshot-3',
      url: 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080"%3E%3Crect fill="%230f172a" width="1920" height="1080"/%3E%3Ctext x="960" y="300" font-family="system-ui" font-size="56" fill="white" text-anchor="middle"%3EWelcome!%3C/text%3E%3Crect x="660" y="500" width="600" height="200" rx="12" fill="%231e293b" stroke="%2322c55e" stroke-width="2"/%3E%3Ctext x="960" y="580" font-family="system-ui" font-size="20" fill="%2394a3b8" text-anchor="middle"%3EEnter your email%3C/text%3E%3Crect x="710" y="620" width="500" height="50" rx="6" fill="%230f172a" stroke="%2322c55e"/%3E%3Ctext x="730" y="652" font-family="monospace" font-size="18" fill="white"%3Euser@example.com%3C/text%3E%3Ccircle cx="960" cy="800" r="40" fill="%2322c55e"/%3E%3Cpath d="M940 800 L955 815 L985 785" stroke="white" stroke-width="4" fill="none"/%3E%3C/svg%3E',
      width: 1920,
      height: 1080,
    },
    assertion: {
      mode: 'exists',
      selector: 'input[type="email"]',
      success: true,
      message: 'Email input found',
    },
  },
];

interface SettingSectionProps {
  title: string;
  tooltip?: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function SettingSection({ title, tooltip, defaultOpen = true, children }: SettingSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          <span className="font-medium text-white">{title}</span>
          {tooltip && (
            <Tooltip content={tooltip}>
              <HelpCircle size={14} className="text-gray-500" />
            </Tooltip>
          )}
        </div>
        {isOpen ? <ChevronDown size={16} className="text-gray-400" /> : <ChevronRight size={16} className="text-gray-400" />}
      </button>
      {isOpen && <div className="p-4 bg-gray-900/50 space-y-4">{children}</div>}
    </div>
  );
}

interface OptionGridProps<T extends string> {
  options: Array<{ id: T; label: string; subtitle?: string; description?: string; preview?: React.ReactNode; previewStyle?: React.CSSProperties; previewNode?: React.ReactNode }>;
  value: T;
  onChange: (value: T) => void;
  columns?: 2 | 3 | 4;
}

function OptionGrid<T extends string>({ options, value, onChange, columns = 3 }: OptionGridProps<T>) {
  const gridCols = columns === 2 ? 'grid-cols-2' : columns === 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-2 sm:grid-cols-3';

  return (
    <div className={`grid ${gridCols} gap-2`}>
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          onClick={() => onChange(option.id)}
          className={`flex flex-col items-center p-3 rounded-lg border transition-all text-center ${
            value === option.id
              ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
              : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
          }`}
        >
          {option.preview && <div className="mb-2">{option.preview}</div>}
          {option.previewNode && <div className="relative w-10 h-10 mb-2 rounded-lg overflow-hidden" style={option.previewStyle}>{option.previewNode}</div>}
          {!option.preview && !option.previewNode && option.previewStyle && (
            <div className="w-10 h-10 mb-2 rounded-lg" style={option.previewStyle} />
          )}
          <span className="text-sm font-medium text-white">{option.label}</span>
          {(option.subtitle || option.description) && (
            <span className="text-xs text-gray-500 mt-0.5">{option.subtitle || option.description}</span>
          )}
        </button>
      ))}
    </div>
  );
}

function SettingsPage({ onBack }: SettingsPageProps) {
  const { replay, setReplaySetting, resetReplaySettings } = useSettingsStore();
  const [isPreviewPlaying, setIsPreviewPlaying] = useState(true);

  const demoFrames = useMemo(() => createDemoFrames(), []);

  const handleReset = useCallback(() => {
    if (window.confirm('Reset all replay settings to defaults?')) {
      resetReplaySettings();
    }
  }, [resetReplaySettings]);

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

  return (
    <div className="flex flex-col h-screen bg-flow-bg">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={onBack}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                title="Back to Dashboard"
                aria-label="Back to Dashboard"
              >
                <ArrowLeft size={20} />
              </button>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-gray-800 rounded-lg">
                  <Settings size={20} className="text-flow-accent" />
                </div>
                <div>
                  <h1 className="text-xl font-bold text-white">Settings</h1>
                  <p className="text-sm text-gray-400">Configure your replay preferences</p>
                </div>
              </div>
            </div>
            <button
              onClick={handleReset}
              className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
              title="Reset to defaults"
            >
              <RotateCcw size={16} />
              <span className="hidden sm:inline">Reset</span>
            </button>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col lg:flex-row">
        {/* Settings Panel */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6 lg:max-w-2xl">
          <div className="space-y-4">
            {/* Replay Settings Header */}
            <div className="flex items-center gap-3 mb-6">
              <Film size={24} className="text-flow-accent" />
              <div>
                <h2 className="text-lg font-semibold text-white">Replay Appearance</h2>
                <p className="text-sm text-gray-400">Customize how workflow replays look and behave</p>
              </div>
            </div>

            {/* Chrome Theme */}
            <SettingSection title="Browser Chrome" tooltip="The browser window frame style surrounding the screenshot">
              <OptionGrid
                options={REPLAY_CHROME_OPTIONS}
                value={replay.chromeTheme}
                onChange={(v) => setReplaySetting('chromeTheme', v)}
                columns={4}
              />
            </SettingSection>

            {/* Background Theme */}
            <SettingSection title="Background">
              <div className="space-y-4">
                {groupedBackgrounds.map((group) => (
                  <div key={group.id}>
                    <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                      {group.label}
                    </div>
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

            {/* Cursor Theme */}
            <SettingSection title="Cursor Style">
              <div className="space-y-4">
                {groupedCursors.map((group) => (
                  <div key={group.id}>
                    <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                      {group.label}
                    </div>
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

            {/* Cursor Scale */}
            <SettingSection title="Cursor Size" tooltip="Adjust the size of the virtual cursor">
              <div className="flex items-center gap-4">
                <input
                  type="range"
                  min={CURSOR_SCALE_MIN}
                  max={CURSOR_SCALE_MAX}
                  step={0.1}
                  value={replay.cursorScale}
                  onChange={(e) => setReplaySetting('cursorScale', parseFloat(e.target.value))}
                  className="flex-1 accent-flow-accent"
                />
                <span className="text-sm text-gray-400 w-12 text-right">
                  {(replay.cursorScale * 100).toFixed(0)}%
                </span>
              </div>
            </SettingSection>

            {/* Click Animation */}
            <SettingSection title="Click Animation" tooltip="Visual feedback when the cursor clicks">
              <OptionGrid
                options={REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS}
                value={replay.cursorClickAnimation}
                onChange={(v) => setReplaySetting('cursorClickAnimation', v)}
                columns={3}
              />
            </SettingSection>

            {/* Cursor Position */}
            <SettingSection title="Initial Cursor Position" tooltip="Where the cursor starts at the beginning of a replay" defaultOpen={false}>
              <OptionGrid
                options={REPLAY_CURSOR_POSITIONS}
                value={replay.cursorInitialPosition}
                onChange={(v) => setReplaySetting('cursorInitialPosition', v)}
                columns={3}
              />
            </SettingSection>

            {/* Motion Settings */}
            <SettingSection title="Cursor Motion" tooltip="How the cursor moves between positions" defaultOpen={false}>
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

            {/* Playback Settings */}
            <SettingSection title="Playback" tooltip="Default playback behavior" defaultOpen={false}>
              <div className="space-y-4">
                <div>
                  <label className="text-sm text-gray-400 mb-2 block">
                    Frame Duration: {(replay.frameDuration / 1000).toFixed(1)}s
                  </label>
                  <input
                    type="range"
                    min={800}
                    max={6000}
                    step={100}
                    value={replay.frameDuration}
                    onChange={(e) => setReplaySetting('frameDuration', parseInt(e.target.value, 10))}
                    className="w-full accent-flow-accent"
                  />
                  <div className="flex justify-between text-xs text-gray-500 mt-1">
                    <span>Fast (0.8s)</span>
                    <span>Slow (6s)</span>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm text-gray-300">Auto-play replays</label>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={replay.autoPlay}
                    onClick={() => setReplaySetting('autoPlay', !replay.autoPlay)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      replay.autoPlay ? 'bg-flow-accent' : 'bg-gray-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        replay.autoPlay ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm text-gray-300">Loop replays</label>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={replay.loop}
                    onClick={() => setReplaySetting('loop', !replay.loop)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      replay.loop ? 'bg-flow-accent' : 'bg-gray-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        replay.loop ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>
              </div>
            </SettingSection>
          </div>
        </div>

        {/* Live Preview */}
        <div className="lg:w-1/2 xl:w-3/5 border-t lg:border-t-0 lg:border-l border-gray-800 bg-gray-900/50 flex flex-col">
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
            <div className="flex items-center gap-2">
              <Film size={16} className="text-gray-400" />
              <span className="text-sm font-medium text-white">Live Preview</span>
            </div>
            <button
              type="button"
              onClick={() => setIsPreviewPlaying(!isPreviewPlaying)}
              className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
            >
              {isPreviewPlaying ? <Pause size={14} /> : <Play size={14} />}
              {isPreviewPlaying ? 'Pause' : 'Play'}
            </button>
          </div>
          <div className="flex-1 p-4 min-h-[300px] lg:min-h-0 flex items-center justify-center">
            <div className="w-full max-w-4xl aspect-video">
              <Suspense
                fallback={
                  <div className="w-full h-full flex items-center justify-center bg-gray-800 rounded-lg">
                    <span className="text-gray-400">Loading preview...</span>
                  </div>
                }
              >
                <ReplayPlayer
                  frames={demoFrames}
                  autoPlay={isPreviewPlaying}
                  loop={replay.loop}
                  chromeTheme={replay.chromeTheme}
                  backgroundTheme={replay.backgroundTheme}
                  cursorTheme={replay.cursorTheme}
                  cursorInitialPosition={replay.cursorInitialPosition}
                  cursorScale={replay.cursorScale}
                  cursorClickAnimation={replay.cursorClickAnimation}
                  cursorDefaultSpeedProfile={replay.cursorSpeedProfile}
                  cursorDefaultPathStyle={replay.cursorPathStyle}
                />
              </Suspense>
            </div>
          </div>
          <div className="px-4 py-3 border-t border-gray-800 text-center">
            <p className="text-xs text-gray-500">
              Preview shows a sample 3-step workflow with your current settings
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default SettingsPage;
