import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@/test-utils';
import userEvent from '@testing-library/user-event';
import { PreviewSettingsDialog } from '../PreviewSettingsDialog';
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

describe('PreviewSettingsDialog', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    sessionId: null,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Dialog Visibility', () => {
    it('renders when isOpen is true', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      expect(screen.getByRole('dialog', { name: 'Preview settings' })).toBeInTheDocument();
    });

    it('does not render when isOpen is false', () => {
      render(<PreviewSettingsDialog {...defaultProps} isOpen={false} />);

      expect(screen.queryByRole('dialog', { name: 'Preview settings' })).not.toBeInTheDocument();
    });
  });

  describe('Header', () => {
    it('exposes a named settings dialog and its header controls', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      expect(screen.getByRole('dialog', { name: 'Preview settings' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Close settings' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Presets' })).toBeInTheDocument();
    });

    it('renders presets dropdown button', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      expect(screen.getByText('Presets')).toBeInTheDocument();
    });
  });

  describe('Sidebar Navigation', () => {
    it('renders all navigation sections', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      // Stream group
      expect(screen.getByText('Quality')).toBeInTheDocument();

      // Replay group
      expect(screen.getByText('Visual')).toBeInTheDocument();
      expect(screen.getByText('Cursor')).toBeInTheDocument();
      expect(screen.getByText('Playback')).toBeInTheDocument();
      expect(screen.getByText('Branding')).toBeInTheDocument();
    });

    it('shows Stream section as active by default', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      // Stream section should be highlighted
      const qualityButton = screen.getByRole('tab', { name: 'Quality' });
      expect(qualityButton).toHaveAttribute('aria-selected', 'true');
    });

    it('switches section when sidebar item is clicked', async () => {
      const user = userEvent.setup();
      render(<PreviewSettingsDialog {...defaultProps} />);

      // Click on Visual section
      await user.click(screen.getByRole('tab', { name: 'Visual' }));

      // Visual should now be highlighted
      const visualButton = screen.getByRole('tab', { name: 'Visual' });
      expect(visualButton).toHaveAttribute('aria-selected', 'true');

      // Stream should no longer be highlighted
      const qualityButton = screen.getByRole('tab', { name: 'Quality' });
      expect(qualityButton).toHaveAttribute('aria-selected', 'false');
    });
  });

  describe('Section Content', () => {
    it('shows Stream section content by default', () => {
      render(<PreviewSettingsDialog {...defaultProps} />);

      // Stream section shows preset options
      expect(screen.getByText('Fast')).toBeInTheDocument();
      expect(screen.getByText('Balanced')).toBeInTheDocument();
      expect(screen.getByText('Sharp')).toBeInTheDocument();
    });

    it('shows Visual section content when selected', async () => {
      const user = userEvent.setup();
      render(<PreviewSettingsDialog {...defaultProps} />);

      await user.click(screen.getByRole('tab', { name: 'Visual' }));

      // The active tab controls its accessible content panel.
      const visualTab = screen.getByRole('tab', { name: 'Visual' });
      const panel = screen.getByRole('tabpanel', { name: 'Visual' });
      expect(visualTab).toHaveAttribute('aria-controls', panel.id);
      expect(panel).toHaveAttribute('aria-labelledby', visualTab.id);
      expect(screen.getByText('Presentation Mode')).toBeInTheDocument();
    });
  });

  describe('Close Behavior', () => {
    it('calls onClose when close button is clicked', async () => {
      const user = userEvent.setup();
      render(<PreviewSettingsDialog {...defaultProps} />);

      await user.click(screen.getByRole('button', { name: 'Close settings' }));
      expect(defaultProps.onClose).toHaveBeenCalledOnce();
    });
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
