import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@/test-utils';
import userEvent from '@testing-library/user-event';
import { PreviewSettingsPanel } from '../PreviewSettingsPanel';

// Mock the settings store
vi.mock('@stores/settingsStore', () => ({
  useSettingsStore: () => ({
    replay: {
      presentation: { showDesktop: true, showBrowserFrame: true, showDeviceFrame: false },
      background: { type: 'theme', theme: 'aurora' },
      chromeTheme: 'aurora',
      browserScale: 1,
      deviceFrameTheme: 'minimal',
      cursorTheme: 'default',
      cursorInitialPosition: 'center',
      cursorClickAnimation: 'pulse',
      cursorScale: 1,
      cursorSpeedProfile: 'easeInOut',
      cursorPathStyle: 'cubic',
      presentationWidth: 1280,
      presentationHeight: 720,
      useCustomDimensions: false,
      frameDuration: 2000,
      autoPlay: true,
      loop: false,
      watermark: null,
      introCard: null,
      outroCard: null,
    },
    userPresets: [],
    activePresetId: null,
    setReplaySetting: vi.fn(),
    randomizeSettings: vi.fn(),
    saveAsPreset: vi.fn(),
    loadPreset: vi.fn(),
    deletePreset: vi.fn(),
    getAllPresets: () => [],
  }),
  BUILT_IN_PRESETS: [
    { id: 'default', name: 'Default', isBuiltIn: true },
    { id: 'cinematic', name: 'Cinematic', isBuiltIn: true },
  ],
}));

// Mock the stream settings hook
vi.mock('@/domains/recording/capture/streamSettingsState', () => ({
  useStreamSettings: () => ({
    preset: 'balanced',
    settings: { quality: 55, fps: 20, scale: 'css' },
    customSettings: { quality: 55, fps: 20, scale: 'css' },
    showStats: false,
    setPreset: vi.fn(),
    setCustomSettings: vi.fn(),
    setShowStats: vi.fn(),
  }),
}));

describe('PreviewSettingsPanel', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    sessionId: null,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('connects the inline settings panel active tab to its named content', async () => {
    const user = userEvent.setup();
    render(<PreviewSettingsPanel {...defaultProps} />);
    const quality = screen.getByRole('tab', { name: 'Quality' });
    expect(quality).toHaveAttribute('aria-controls', screen.getByRole('tabpanel', { name: 'Quality' }).id);
    await user.click(screen.getByRole('tab', { name: 'Visual' }));
    const visual = screen.getByRole('tab', { name: 'Visual' });
    expect(visual).toHaveAttribute('aria-selected', 'true');
    expect(visual).toHaveAttribute('aria-controls', screen.getByRole('tabpanel', { name: 'Visual' }).id);
    expect(quality).toHaveAttribute('aria-selected', 'false');
  });

});
