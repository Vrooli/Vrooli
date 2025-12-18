import { useMemo, useState, useCallback, lazy, Suspense } from 'react';
import {
  ArrowLeft,
  Settings,
  Film,
  RotateCcw,
  Shuffle,
  Save,
  Check,
  Bookmark,
  Monitor,
  Wrench,
  Key,
  Database,
  Clock,
  Image as ImageIcon,
  CreditCard,
  CalendarClock,
  Play,
  Pause,
} from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { BrandingTab } from './sections/branding';
import { SessionProfilesTab } from './sections/SessionProfilesSection';
import { SubscriptionTab } from './sections/subscription';
import { SchedulesTab } from './sections/schedules';
import {
  DisplaySection,
  ReplaySection,
  WorkflowSection,
  ApiKeysSection,
  DataSection,
  createDemoFrames,
} from './sections';

const ReplayPlayer = lazy(() => import('@/domains/exports/replay/ReplayPlayer'));

type SettingsTab = 'display' | 'replay' | 'branding' | 'workflow' | 'apikeys' | 'data' | 'sessions' | 'subscription' | 'schedules';

const SETTINGS_TABS: Array<{ id: SettingsTab; label: string; icon: React.ReactNode; description: string }> = [
  { id: 'display', label: 'Display', icon: <Monitor size={18} />, description: 'Appearance and accessibility' },
  { id: 'subscription', label: 'Subscription', icon: <CreditCard size={18} />, description: 'Manage your subscription' },
  { id: 'replay', label: 'Replay', icon: <Film size={18} />, description: 'Customize replay appearance' },
  { id: 'branding', label: 'Branding', icon: <ImageIcon size={18} />, description: 'Logos and backgrounds' },
  { id: 'workflow', label: 'Workflow Defaults', icon: <Wrench size={18} />, description: 'Default workflow settings' },
  { id: 'apikeys', label: 'API Keys', icon: <Key size={18} />, description: 'Manage API integrations' },
  { id: 'sessions', label: 'Sessions', icon: <Clock size={18} />, description: 'Persist Playwright sessions' },
  { id: 'schedules', label: 'Schedules', icon: <CalendarClock size={18} />, description: 'Automate workflow runs' },
  { id: 'data', label: 'Data', icon: <Database size={18} />, description: 'Manage and clear data' },
];

interface SettingsViewProps {
  onBack: () => void;
}

export function SettingsView({ onBack }: SettingsViewProps) {
  const {
    replay,
    resetReplaySettings,
    resetWorkflowDefaults,
    resetDisplaySettings,
    clearApiKeys,
    randomizeSettings,
    saveAsPreset,
  } = useSettingsStore();

  const [activeTab, setActiveTab] = useState<SettingsTab>('display');
  const [isPreviewPlaying, setIsPreviewPlaying] = useState(true);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [newPresetName, setNewPresetName] = useState('');

  // Use 1.5x frame duration for preview so users can better see their customizations
  const previewFrameDuration = Math.round(replay.frameDuration * 1.5);
  const demoFrames = useMemo(() => createDemoFrames(previewFrameDuration), [previewFrameDuration]);

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

  return (
    <div className="flex flex-col h-screen bg-flow-bg">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={onBack}
                className="p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
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
                  <h1 className="text-xl font-bold text-surface">Settings</h1>
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
                className="flex items-center gap-2 px-3 py-2 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
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
                    ? 'border-flow-accent text-surface'
                    : 'border-transparent text-subtle hover:text-surface hover:border-gray-600'
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
            <h3 className="text-lg font-semibold text-surface mb-4 flex items-center gap-2">
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
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/30"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSavePreset();
                if (e.key === 'Escape') setShowSaveDialog(false);
              }}
            />
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowSaveDialog(false)}
                className="flex-1 px-4 py-2 text-subtle hover:text-surface hover:bg-gray-800 rounded-lg transition-colors"
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

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col lg:flex-row">
        {/* Settings Panel */}
        <div className={`flex-1 overflow-y-auto p-4 sm:p-6 ${activeTab === 'replay' ? 'lg:max-w-2xl' : 'max-w-5xl w-full mx-auto'}`}>
          {activeTab === 'display' && <DisplaySection />}
          {activeTab === 'subscription' && <SubscriptionTab />}
          {activeTab === 'replay' && (
            <ReplaySection
              onRandomize={handleRandomize}
              onSavePreset={() => setShowSaveDialog(true)}
            />
          )}
          {activeTab === 'branding' && <BrandingTab />}
          {activeTab === 'workflow' && <WorkflowSection />}
          {activeTab === 'apikeys' && <ApiKeysSection />}
          {activeTab === 'sessions' && <SessionProfilesTab />}
          {activeTab === 'schedules' && <SchedulesTab />}
          {activeTab === 'data' && <DataSection />}
        </div>

        {/* Live Preview - Only show for replay tab */}
        {activeTab === 'replay' && (
          <div
            data-theme="dark"
            className="lg:w-1/2 xl:w-3/5 border-t lg:border-t-0 lg:border-l border-gray-800 bg-gray-900/50 flex flex-col min-h-0 overflow-hidden"
          >
            <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 border-b border-gray-800">
              <div className="flex items-center gap-2">
                <Film size={16} className="text-gray-400" />
                <span className="text-sm font-medium text-surface">Live Preview</span>
              </div>
              <button
                type="button"
                onClick={() => setIsPreviewPlaying(!isPreviewPlaying)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-surface bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              >
                {isPreviewPlaying ? <Pause size={14} /> : <Play size={14} />}
                {isPreviewPlaying ? 'Pause' : 'Play'}
              </button>
            </div>
            <div className="flex-1 min-h-0 p-4 overflow-y-auto">
              <div className="w-full max-w-4xl mx-auto">
                <Suspense
                  fallback={
                    <div className="w-full aspect-video flex items-center justify-center bg-gray-800 rounded-lg">
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
                    watermark={replay.watermark}
                    introCard={replay.introCard}
                    outroCard={replay.outroCard}
                  />
                </Suspense>
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

export default SettingsView;
