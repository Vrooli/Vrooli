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
  Shuffle,
  Save,
  Trash2,
  Check,
  Bookmark,
  Sparkles,
  Clock,
  Key,
  Eye,
  EyeOff,
  AlertTriangle,
  Wrench,
  Monitor,
  Sun,
  Moon,
  Laptop,
  Type,
  Accessibility,
  Minimize2,
} from 'lucide-react';
import { useSettingsStore, BUILT_IN_PRESETS } from '@stores/settingsStore';
import type { ApiKeySettings, ThemeMode, FontSize, FontFamily } from '@stores/settingsStore';
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

type SettingsTab = 'display' | 'replay' | 'workflow' | 'apikeys';

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
const DEMO_FRAME_1_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="heroGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#1e293b"/>
      <stop offset="100%" style="stop-color:#0f172a"/>
    </linearGradient>
    <linearGradient id="btnGrad" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" style="stop-color:#3b82f6"/>
      <stop offset="100%" style="stop-color:#6366f1"/>
    </linearGradient>
  </defs>
  <rect fill="url(#heroGrad)" width="1920" height="1080"/>
  <circle cx="1600" cy="200" r="300" fill="#3b82f6" opacity="0.05"/>
  <circle cx="200" cy="800" r="250" fill="#6366f1" opacity="0.05"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.9"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <text x="300" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Products</text>
  <text x="420" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Solutions</text>
  <text x="540" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Pricing</text>
  <text x="660" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Docs</text>
  <rect x="1680" y="22" width="160" height="40" rx="8" fill="url(#btnGrad)"/>
  <text x="1760" y="48" font-family="system-ui" font-size="14" fill="white" text-anchor="middle" font-weight="600">Get Started</text>
  <text x="960" y="320" font-family="system-ui" font-size="64" fill="white" text-anchor="middle" font-weight="bold">Automate Your Workflow</text>
  <text x="960" y="390" font-family="system-ui" font-size="24" fill="#94a3b8" text-anchor="middle">Build, test, and deploy browser automations in minutes</text>
  <rect x="780" y="450" width="180" height="56" rx="12" fill="url(#btnGrad)"/>
  <text x="870" y="486" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Start Free</text>
  <rect x="980" y="450" width="180" height="56" rx="12" fill="none" stroke="#475569" stroke-width="2"/>
  <text x="1070" y="486" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Watch Demo</text>
</svg>`;

const DEMO_FRAME_2_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="bgGrad2" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0f172a"/>
      <stop offset="100%" style="stop-color:#1e1b4b"/>
    </linearGradient>
  </defs>
  <rect fill="url(#bgGrad2)" width="1920" height="1080"/>
  <circle cx="200" cy="200" r="400" fill="#3b82f6" opacity="0.03"/>
  <circle cx="1700" cy="900" r="350" fill="#6366f1" opacity="0.03"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <rect x="560" y="160" width="800" height="760" rx="24" fill="#1e293b" stroke="#334155" stroke-width="1"/>
  <text x="960" y="240" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">Create your account</text>
  <text x="960" y="285" font-family="system-ui" font-size="16" fill="#64748b" text-anchor="middle">Start automating in less than 2 minutes</text>
  <text x="640" y="480" font-family="system-ui" font-size="14" fill="#94a3b8">Full Name</text>
  <rect x="640" y="495" width="640" height="52" rx="10" fill="#0f172a" stroke="#334155"/>
  <text x="640" y="585" font-family="system-ui" font-size="14" fill="#94a3b8">Email Address</text>
  <rect x="640" y="600" width="640" height="52" rx="10" fill="#0f172a" stroke="#3b82f6" stroke-width="2"/>
  <rect x="640" y="810" width="640" height="56" rx="12" fill="#3b82f6"/>
  <text x="960" y="846" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Create Account</text>
</svg>`;

const DEMO_FRAME_3_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="bgGrad3" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0f172a"/>
      <stop offset="100%" style="stop-color:#042f2e"/>
    </linearGradient>
    <linearGradient id="successGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#22c55e"/>
      <stop offset="100%" style="stop-color:#16a34a"/>
    </linearGradient>
  </defs>
  <rect fill="url(#bgGrad3)" width="1920" height="1080"/>
  <circle cx="960" cy="540" r="500" fill="#22c55e" opacity="0.02"/>
  <circle cx="960" cy="540" r="350" fill="#22c55e" opacity="0.03"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#22c55e"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <circle cx="960" cy="320" r="80" fill="url(#successGrad)"/>
  <path d="M920 320 L945 345 L1005 285" stroke="white" stroke-width="8" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  <text x="960" y="460" font-family="system-ui" font-size="42" fill="white" text-anchor="middle" font-weight="bold">Account Created Successfully!</text>
  <text x="960" y="510" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Welcome to AutoFlow</text>
</svg>`;

const createDemoFrames = (): ReplayFrame[] => [
  {
    id: 'demo-1',
    stepIndex: 0,
    stepType: 'navigate',
    status: 'completed',
    success: true,
    durationMs: 1200,
    finalUrl: 'https://autoflow.app',
    screenshot: {
      artifactId: 'demo-screenshot-1',
      url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_1_SVG)}`,
      width: 1920,
      height: 1080,
    },
    clickPosition: { x: 1760, y: 42 },
  },
  {
    id: 'demo-2',
    stepIndex: 1,
    stepType: 'click',
    status: 'completed',
    success: true,
    durationMs: 800,
    finalUrl: 'https://autoflow.app/signup',
    screenshot: {
      artifactId: 'demo-screenshot-2',
      url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_2_SVG)}`,
      width: 1920,
      height: 1080,
    },
    clickPosition: { x: 960, y: 626 },
  },
  {
    id: 'demo-3',
    stepIndex: 2,
    stepType: 'type',
    status: 'completed',
    success: true,
    durationMs: 1500,
    finalUrl: 'https://autoflow.app/welcome',
    screenshot: {
      artifactId: 'demo-screenshot-3',
      url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_3_SVG)}`,
      width: 1920,
      height: 1080,
    },
    assertion: {
      mode: 'exists',
      selector: '.success-message',
      success: true,
      message: 'Account creation successful',
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

// Tabs configuration
const SETTINGS_TABS: Array<{ id: SettingsTab; label: string; icon: React.ReactNode; description: string }> = [
  { id: 'display', label: 'Display', icon: <Monitor size={18} />, description: 'Appearance and accessibility' },
  { id: 'replay', label: 'Replay', icon: <Film size={18} />, description: 'Customize replay appearance' },
  { id: 'workflow', label: 'Workflow Defaults', icon: <Wrench size={18} />, description: 'Default workflow settings' },
  { id: 'apikeys', label: 'API Keys', icon: <Key size={18} />, description: 'Manage API integrations' },
];

function SettingsPage({ onBack }: SettingsPageProps) {
  const {
    replay,
    workflowDefaults,
    apiKeys,
    display,
    userPresets,
    activePresetId,
    setReplaySetting,
    resetReplaySettings,
    loadPreset,
    saveAsPreset,
    deletePreset,
    randomizeSettings,
    getAllPresets,
    setWorkflowDefault,
    resetWorkflowDefaults,
    setApiKey,
    clearApiKeys,
    setDisplaySetting,
    resetDisplaySettings,
    getEffectiveTheme,
  } = useSettingsStore();

  const [activeTab, setActiveTab] = useState<SettingsTab>('display');
  const [isPreviewPlaying, setIsPreviewPlaying] = useState(true);
  const [showPresetDropdown, setShowPresetDropdown] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [newPresetName, setNewPresetName] = useState('');
  const [presetToDelete, setPresetToDelete] = useState<string | null>(null);
  const [showApiKeys, setShowApiKeys] = useState<Record<keyof ApiKeySettings, boolean>>({
    browserlessApiKey: false,
    openaiApiKey: false,
    anthropicApiKey: false,
    customApiEndpoint: false,
  });

  const demoFrames = useMemo(() => createDemoFrames(), []);
  const allPresets = useMemo(() => getAllPresets(), [getAllPresets, userPresets]);

  const activePreset = useMemo(() => {
    if (!activePresetId) return null;
    return allPresets.find((p) => p.id === activePresetId) || null;
  }, [activePresetId, allPresets]);

  const handleReset = useCallback(() => {
    if (activeTab === 'display') {
      if (window.confirm('Reset all display settings to defaults?')) {
        resetDisplaySettings();
      }
    } else if (activeTab === 'replay') {
      if (window.confirm('Reset all replay settings to defaults?')) {
        resetReplaySettings();
      }
    } else if (activeTab === 'workflow') {
      if (window.confirm('Reset all workflow defaults to their original values?')) {
        resetWorkflowDefaults();
      }
    } else if (activeTab === 'apikeys') {
      if (window.confirm('Clear all API keys? This cannot be undone.')) {
        clearApiKeys();
      }
    }
  }, [activeTab, resetDisplaySettings, resetReplaySettings, resetWorkflowDefaults, clearApiKeys]);

  const handleRandomize = useCallback(() => {
    randomizeSettings();
  }, [randomizeSettings]);

  const handleSavePreset = useCallback(() => {
    const trimmedName = newPresetName.trim();
    if (!trimmedName) return;
    saveAsPreset(trimmedName);
    setNewPresetName('');
    setShowSaveDialog(false);
  }, [newPresetName, saveAsPreset]);

  const handleDeletePreset = useCallback((presetId: string) => {
    deletePreset(presetId);
    setPresetToDelete(null);
  }, [deletePreset]);

  const handlePresetSelect = useCallback((presetId: string) => {
    loadPreset(presetId);
    setShowPresetDropdown(false);
  }, [loadPreset]);

  const toggleApiKeyVisibility = (key: keyof ApiKeySettings) => {
    setShowApiKeys(prev => ({ ...prev, [key]: !prev[key] }));
  };

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

  const renderReplaySettings = () => (
    <div className="space-y-4">
      {/* Presets Section */}
      <div className="relative mb-6">
        <div className="flex items-center gap-3 mb-3">
          <Sparkles size={18} className="text-amber-400" />
          <span className="text-sm font-medium text-white">Quick Presets</span>
          <Tooltip content="Apply a preset to quickly configure all replay settings at once. You can also save your own custom presets.">
            <HelpCircle size={14} className="text-gray-500" />
          </Tooltip>
        </div>

        {/* Preset Selector */}
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowPresetDropdown(!showPresetDropdown)}
            className="w-full flex items-center justify-between px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white hover:border-gray-600 transition-colors"
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
                      activePresetId === preset.id ? 'bg-flow-accent/10 text-flow-accent' : 'text-white'
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
                            activePresetId === preset.id ? 'text-flow-accent' : 'text-white'
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
                      onClick={() => { setShowPresetDropdown(false); handleRandomize(); }}
                      className="flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm text-amber-400 hover:bg-amber-900/20 rounded-lg transition-colors"
                    >
                      <Shuffle size={14} />
                      Random
                    </button>
                    <button
                      type="button"
                      onClick={() => { setShowPresetDropdown(false); setShowSaveDialog(true); }}
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
          <input
            type="range"
            min={CURSOR_SCALE_MIN}
            max={CURSOR_SCALE_MAX}
            step={0.1}
            value={replay.cursorScale}
            onChange={(e) => setReplaySetting('cursorScale', parseFloat(e.target.value))}
            className="flex-1 accent-flow-accent"
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
          <div className="flex items-center justify-between py-2 border-t border-gray-800">
            <div>
              <label className="text-sm text-gray-300 block">Auto-play replays</label>
              <span className="text-xs text-gray-500">Start playing immediately when opened</span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={replay.autoPlay}
              onClick={() => setReplaySetting('autoPlay', !replay.autoPlay)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                replay.autoPlay ? 'bg-flow-accent' : 'bg-gray-700'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${replay.autoPlay ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>
          <div className="flex items-center justify-between py-2 border-t border-gray-800">
            <div>
              <label className="text-sm text-gray-300 block">Loop replays</label>
              <span className="text-xs text-gray-500">Restart automatically when finished</span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={replay.loop}
              onClick={() => setReplaySetting('loop', !replay.loop)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                replay.loop ? 'bg-flow-accent' : 'bg-gray-700'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${replay.loop ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>
        </div>
      </SettingSection>
    </div>
  );

  const renderWorkflowSettings = () => (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Clock size={24} className="text-green-400" />
        <div>
          <h2 className="text-lg font-semibold text-white">Workflow Defaults</h2>
          <p className="text-sm text-gray-400">Default settings applied to new workflows</p>
        </div>
      </div>

      <SettingSection title="Timeouts" tooltip="Control how long workflows and steps wait before timing out.">
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Workflow Timeout</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.defaultTimeout}s</span>
            </div>
            <input
              type="range"
              min={10}
              max={300}
              step={5}
              value={workflowDefaults.defaultTimeout}
              onChange={(e) => setWorkflowDefault('defaultTimeout', parseInt(e.target.value, 10))}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>10s</span>
              <span>5 min</span>
            </div>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Step Timeout</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.stepTimeout}s</span>
            </div>
            <input
              type="range"
              min={5}
              max={60}
              step={1}
              value={workflowDefaults.stepTimeout}
              onChange={(e) => setWorkflowDefault('stepTimeout', parseInt(e.target.value, 10))}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>5s</span>
              <span>60s</span>
            </div>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Retry Behavior" tooltip="Configure automatic retry behavior for failed steps.">
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Retry Attempts</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.retryAttempts}</span>
            </div>
            <input
              type="range"
              min={0}
              max={5}
              step={1}
              value={workflowDefaults.retryAttempts}
              onChange={(e) => setWorkflowDefault('retryAttempts', parseInt(e.target.value, 10))}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>No retries</span>
              <span>5 retries</span>
            </div>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Retry Delay</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.retryDelay}ms</span>
            </div>
            <input
              type="range"
              min={100}
              max={5000}
              step={100}
              value={workflowDefaults.retryDelay}
              onChange={(e) => setWorkflowDefault('retryDelay', parseInt(e.target.value, 10))}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>100ms</span>
              <span>5s</span>
            </div>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Screenshots" tooltip="Configure when to capture screenshots during execution.">
        <div className="space-y-3">
          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm text-gray-300 block">Capture on Failure</label>
              <span className="text-xs text-gray-500">Take a screenshot when a step fails</span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={workflowDefaults.screenshotOnFailure}
              onClick={() => setWorkflowDefault('screenshotOnFailure', !workflowDefaults.screenshotOnFailure)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                workflowDefaults.screenshotOnFailure ? 'bg-flow-accent' : 'bg-gray-700'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${workflowDefaults.screenshotOnFailure ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>
          <div className="flex items-center justify-between py-2 border-t border-gray-800">
            <div>
              <label className="text-sm text-gray-300 block">Capture on Success</label>
              <span className="text-xs text-gray-500">Take a screenshot after each successful step</span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={workflowDefaults.screenshotOnSuccess}
              onClick={() => setWorkflowDefault('screenshotOnSuccess', !workflowDefaults.screenshotOnSuccess)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                workflowDefaults.screenshotOnSuccess ? 'bg-flow-accent' : 'bg-gray-700'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${workflowDefaults.screenshotOnSuccess ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Browser Options" tooltip="Control browser behavior during execution.">
        <div className="space-y-5">
          <div className="flex items-center justify-between py-2">
            <div>
              <label className="text-sm text-gray-300 block">Headless Mode</label>
              <span className="text-xs text-gray-500">Run browser without visible window</span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={workflowDefaults.headless}
              onClick={() => setWorkflowDefault('headless', !workflowDefaults.headless)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                workflowDefaults.headless ? 'bg-flow-accent' : 'bg-gray-700'
              }`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${workflowDefaults.headless ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Slow Motion</label>
              <span className="text-sm font-medium text-flow-accent">{workflowDefaults.slowMo}ms</span>
            </div>
            <input
              type="range"
              min={0}
              max={1000}
              step={50}
              value={workflowDefaults.slowMo}
              onChange={(e) => setWorkflowDefault('slowMo', parseInt(e.target.value, 10))}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Normal</span>
              <span>Very Slow (1s)</span>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Add a delay between each action for debugging
            </p>
          </div>
        </div>
      </SettingSection>
    </div>
  );

  const renderApiKeysSettings = () => (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Key size={24} className="text-amber-400" />
        <div>
          <h2 className="text-lg font-semibold text-white">API Keys & Integrations</h2>
          <p className="text-sm text-gray-400">Override default API keys with your own</p>
        </div>
      </div>

      <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4 mb-6">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-amber-200 font-medium">Security Notice</p>
            <p className="text-xs text-amber-300/80 mt-1">
              API keys are stored locally in your browser. Never share your API keys with others.
              Keys are used to authenticate with external services.
            </p>
          </div>
        </div>
      </div>

      <SettingSection title="Browser Service" tooltip="API key for the headless browser service.">
        <div className="space-y-3">
          <div>
            <label className="text-sm text-gray-300 block mb-2">Browserless API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.browserlessApiKey ? 'text' : 'password'}
                value={apiKeys.browserlessApiKey}
                onChange={(e) => setApiKey('browserlessApiKey', e.target.value)}
                placeholder="Enter your Browserless API key"
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('browserlessApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.browserlessApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.browserlessApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Leave empty to use the default shared key
            </p>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="AI Services" tooltip="API keys for AI-powered features.">
        <div className="space-y-5">
          <div>
            <label className="text-sm text-gray-300 block mb-2">OpenAI API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.openaiApiKey ? 'text' : 'password'}
                value={apiKeys.openaiApiKey}
                onChange={(e) => setApiKey('openaiApiKey', e.target.value)}
                placeholder="sk-..."
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('openaiApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.openaiApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.openaiApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Used for AI workflow generation
            </p>
          </div>
          <div>
            <label className="text-sm text-gray-300 block mb-2">Anthropic API Key</label>
            <div className="relative">
              <input
                type={showApiKeys.anthropicApiKey ? 'text' : 'password'}
                value={apiKeys.anthropicApiKey}
                onChange={(e) => setApiKey('anthropicApiKey', e.target.value)}
                placeholder="sk-ant-..."
                className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
              />
              <button
                type="button"
                onClick={() => toggleApiKeyVisibility('anthropicApiKey')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label={showApiKeys.anthropicApiKey ? 'Hide API key' : 'Show API key'}
              >
                {showApiKeys.anthropicApiKey ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Alternative AI provider for workflow generation
            </p>
          </div>
        </div>
      </SettingSection>

      <SettingSection title="Custom Endpoint" tooltip="Override the default API endpoint.">
        <div>
          <label className="text-sm text-gray-300 block mb-2">Custom API Endpoint</label>
          <input
            type="text"
            value={apiKeys.customApiEndpoint}
            onChange={(e) => setApiKey('customApiEndpoint', e.target.value)}
            placeholder="https://api.example.com"
            className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
          />
          <p className="text-xs text-gray-500 mt-2">
            For self-hosted or proxy API endpoints
          </p>
        </div>
      </SettingSection>
    </div>
  );

  const renderDisplaySettings = () => {
    const themeOptions: Array<{ id: ThemeMode; label: string; icon: React.ReactNode; description: string }> = [
      { id: 'light', label: 'Light', icon: <Sun size={20} />, description: 'Bright theme for well-lit environments' },
      { id: 'dark', label: 'Dark', icon: <Moon size={20} />, description: 'Dark theme that reduces eye strain' },
      { id: 'system', label: 'System', icon: <Laptop size={20} />, description: 'Follow your system preferences' },
    ];

    const fontSizeOptions: Array<{ id: FontSize; label: string; description: string }> = [
      { id: 'small', label: 'Small', description: 'Compact interface, more content visible' },
      { id: 'medium', label: 'Medium', description: 'Default balanced size' },
      { id: 'large', label: 'Large', description: 'Larger text for better readability' },
    ];

    const fontFamilyOptions: Array<{ id: FontFamily; label: string; preview: string; description: string }> = [
      { id: 'sans', label: 'Sans Serif', preview: 'Aa Bb Cc', description: 'Clean and modern appearance' },
      { id: 'mono', label: 'Monospace', preview: 'Aa Bb Cc', description: 'Technical feel, fixed-width characters' },
      { id: 'system', label: 'System', preview: 'Aa Bb Cc', description: 'Match your OS default font' },
    ];

    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3 mb-6">
          <Monitor size={24} className="text-purple-400" />
          <div>
            <h2 className="text-lg font-semibold text-white">Display Settings</h2>
            <p className="text-sm text-gray-400">Customize appearance and accessibility</p>
          </div>
        </div>

        <SettingSection title="Theme" tooltip="Choose your preferred color scheme.">
          <div className="grid grid-cols-3 gap-3">
            {themeOptions.map((option) => (
              <button
                key={option.id}
                type="button"
                onClick={() => setDisplaySetting('themeMode', option.id)}
                className={`flex flex-col items-center p-4 rounded-lg border transition-all ${
                  display.themeMode === option.id
                    ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
                    : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
                }`}
              >
                <div className={`mb-2 ${display.themeMode === option.id ? 'text-flow-accent' : 'text-gray-400'}`}>
                  {option.icon}
                </div>
                <span className="text-sm font-medium text-white">{option.label}</span>
                <span className="text-xs text-gray-500 mt-1 text-center">{option.description}</span>
              </button>
            ))}
          </div>
          {display.themeMode === 'system' && (
            <p className="mt-3 text-xs text-gray-500 flex items-center gap-1">
              <Laptop size={12} />
              Currently using: {getEffectiveTheme()} theme based on system preference
            </p>
          )}
        </SettingSection>

        <SettingSection title="Font Size" tooltip="Adjust the text size throughout the application.">
          <div className="space-y-3">
            {fontSizeOptions.map((option) => (
              <button
                key={option.id}
                type="button"
                onClick={() => setDisplaySetting('fontSize', option.id)}
                className={`w-full flex items-center justify-between p-3 rounded-lg border transition-all ${
                  display.fontSize === option.id
                    ? 'border-flow-accent bg-flow-accent/10'
                    : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
                }`}
              >
                <div className="flex items-center gap-3">
                  <Type size={18} className={display.fontSize === option.id ? 'text-flow-accent' : 'text-gray-400'} />
                  <div className="text-left">
                    <span className="text-sm font-medium text-white block">{option.label}</span>
                    <span className="text-xs text-gray-500">{option.description}</span>
                  </div>
                </div>
                {display.fontSize === option.id && (
                  <Check size={16} className="text-flow-accent" />
                )}
              </button>
            ))}
          </div>
        </SettingSection>

        <SettingSection title="Font Family" tooltip="Choose your preferred typeface." defaultOpen={false}>
          <div className="space-y-3">
            {fontFamilyOptions.map((option) => (
              <button
                key={option.id}
                type="button"
                onClick={() => setDisplaySetting('fontFamily', option.id)}
                className={`w-full flex items-center justify-between p-3 rounded-lg border transition-all ${
                  display.fontFamily === option.id
                    ? 'border-flow-accent bg-flow-accent/10'
                    : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
                }`}
              >
                <div className="flex items-center gap-3">
                  <span
                    className={`text-lg ${option.id === 'mono' ? 'font-mono' : option.id === 'sans' ? 'font-sans' : 'font-system'} ${
                      display.fontFamily === option.id ? 'text-flow-accent' : 'text-gray-400'
                    }`}
                  >
                    {option.preview}
                  </span>
                  <div className="text-left">
                    <span className="text-sm font-medium text-white block">{option.label}</span>
                    <span className="text-xs text-gray-500">{option.description}</span>
                  </div>
                </div>
                {display.fontFamily === option.id && (
                  <Check size={16} className="text-flow-accent" />
                )}
              </button>
            ))}
          </div>
        </SettingSection>

        <SettingSection title="Accessibility" tooltip="Options to improve usability and reduce visual stress.">
          <div className="space-y-3">
            <div className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                <Accessibility size={18} className="text-gray-400" />
                <div>
                  <label className="text-sm text-gray-300 block">Reduced Motion</label>
                  <span className="text-xs text-gray-500">Minimize animations and transitions</span>
                </div>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={display.reducedMotion}
                onClick={() => setDisplaySetting('reducedMotion', !display.reducedMotion)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                  display.reducedMotion ? 'bg-flow-accent' : 'bg-gray-700'
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${display.reducedMotion ? 'translate-x-6' : 'translate-x-1'}`} />
              </button>
            </div>
            <div className="flex items-center justify-between py-2 border-t border-gray-800">
              <div className="flex items-center gap-3">
                <Eye size={18} className="text-gray-400" />
                <div>
                  <label className="text-sm text-gray-300 block">High Contrast</label>
                  <span className="text-xs text-gray-500">Increase text and border contrast</span>
                </div>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={display.highContrast}
                onClick={() => setDisplaySetting('highContrast', !display.highContrast)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                  display.highContrast ? 'bg-flow-accent' : 'bg-gray-700'
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${display.highContrast ? 'translate-x-6' : 'translate-x-1'}`} />
              </button>
            </div>
            <div className="flex items-center justify-between py-2 border-t border-gray-800">
              <div className="flex items-center gap-3">
                <Minimize2 size={18} className="text-gray-400" />
                <div>
                  <label className="text-sm text-gray-300 block">Compact Mode</label>
                  <span className="text-xs text-gray-500">Reduce spacing for denser information</span>
                </div>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={display.compactMode}
                onClick={() => setDisplaySetting('compactMode', !display.compactMode)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
                  display.compactMode ? 'bg-flow-accent' : 'bg-gray-700'
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${display.compactMode ? 'translate-x-6' : 'translate-x-1'}`} />
              </button>
            </div>
          </div>
        </SettingSection>
      </div>
    );
  };

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
                  <p className="text-sm text-gray-400 hidden sm:block">Manage your preferences and integrations</p>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {activeTab === 'replay' && (
                <button
                  onClick={handleRandomize}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-amber-400 hover:text-amber-300 hover:bg-amber-900/30 rounded-lg transition-colors"
                  title="Randomize all settings"
                >
                  <Shuffle size={16} />
                  <span className="hidden sm:inline">Random</span>
                </button>
              )}
              {activeTab === 'replay' && (
                <button
                  onClick={() => setShowSaveDialog(true)}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-flow-accent hover:text-blue-300 hover:bg-blue-900/30 rounded-lg transition-colors"
                  title="Save current settings as a preset"
                >
                  <Save size={16} />
                  <span className="hidden sm:inline">Save</span>
                </button>
              )}
              <button
                onClick={handleReset}
                className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                title={activeTab === 'apikeys' ? 'Clear all API keys' : 'Reset to defaults'}
              >
                <RotateCcw size={16} />
                <span className="hidden sm:inline">{activeTab === 'apikeys' ? 'Clear' : 'Reset'}</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <div className="border-b border-gray-800 bg-flow-bg/95 backdrop-blur">
        <div className="px-4 sm:px-6">
          <nav className="flex gap-1 -mb-px overflow-x-auto">
            {SETTINGS_TABS.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                  activeTab === tab.id
                    ? 'border-flow-accent text-white'
                    : 'border-transparent text-gray-400 hover:text-white hover:border-gray-600'
                }`}
              >
                {tab.icon}
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>
        </div>
      </div>

      {/* Save Preset Dialog */}
      {showSaveDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl animate-fade-in">
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <Bookmark size={20} className="text-flow-accent" />
              Save Preset
            </h3>
            <p className="text-sm text-gray-400 mb-4">
              Save your current settings as a reusable preset.
            </p>
            <input
              type="text"
              value={newPresetName}
              onChange={(e) => setNewPresetName(e.target.value)}
              placeholder="My Custom Preset"
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/30"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSavePreset();
                if (e.key === 'Escape') setShowSaveDialog(false);
              }}
            />
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowSaveDialog(false)}
                className="flex-1 px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSavePreset}
                disabled={!newPresetName.trim()}
                className="flex-1 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                <Check size={16} />
                Save Preset
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Preset Confirmation */}
      {presetToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl animate-fade-in">
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <Trash2 size={20} className="text-red-400" />
              Delete Preset
            </h3>
            <p className="text-sm text-gray-400 mb-6">
              Are you sure you want to delete this preset? This action cannot be undone.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setPresetToDelete(null)}
                className="flex-1 px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
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

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col lg:flex-row">
        {/* Settings Panel - constrain width only when preview is shown (Replay tab) */}
        <div className={`flex-1 overflow-y-auto p-4 sm:p-6 ${activeTab === 'replay' ? 'lg:max-w-2xl' : ''}`}>
          {activeTab === 'display' && renderDisplaySettings()}
          {activeTab === 'replay' && renderReplaySettings()}
          {activeTab === 'workflow' && renderWorkflowSettings()}
          {activeTab === 'apikeys' && renderApiKeysSettings()}
        </div>

        {/* Live Preview - Only show for replay tab
            Force dark mode on preview container since replay content is designed for dark backgrounds */}
        {activeTab === 'replay' && (
          <div
            data-theme="dark"
            className="lg:w-1/2 xl:w-3/5 border-t lg:border-t-0 lg:border-l border-gray-800 bg-gray-900/50 flex flex-col min-h-0 overflow-hidden"
          >
            <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 border-b border-gray-800">
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
            <div className="flex-1 min-h-0 p-4 flex items-center justify-center overflow-hidden">
              <div className="w-full h-full max-w-4xl flex items-center justify-center">
                <div className="w-full max-h-full aspect-video overflow-hidden">
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
            </div>
            <div className="flex-shrink-0 px-4 py-3 border-t border-gray-800 text-center">
              <p className="text-xs text-gray-500">
                Preview shows a sample 3-step workflow with your current settings
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default SettingsPage;
