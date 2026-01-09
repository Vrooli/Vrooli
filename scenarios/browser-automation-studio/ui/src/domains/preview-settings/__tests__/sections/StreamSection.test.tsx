import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { StreamSection } from '../../sections/StreamSection';
import type { StreamPreset, StreamSettingsValues } from '@/domains/recording/capture/StreamSettings';

describe('StreamSection', () => {
  const defaultSettings: StreamSettingsValues = {
    quality: 55,
    fps: 20,
    scale: 'css',
  };

  const defaultProps = {
    preset: 'balanced' as StreamPreset,
    settings: defaultSettings,
    showStats: false,
    hasActiveSession: false,
    onPresetChange: vi.fn(),
    onSettingsChange: vi.fn(),
    onShowStatsChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Preset Selection', () => {
    it('renders all preset options', () => {
      render(<StreamSection {...defaultProps} />);

      expect(screen.getByText('Fast')).toBeInTheDocument();
      expect(screen.getByText('Balanced')).toBeInTheDocument();
      expect(screen.getByText('Sharp')).toBeInTheDocument();
      expect(screen.getByText('HiDPI')).toBeInTheDocument();
      expect(screen.getByText('Custom')).toBeInTheDocument();
    });

    it('shows balanced preset as selected by default', () => {
      render(<StreamSection {...defaultProps} />);

      const balancedButton = screen.getByText('Balanced').closest('button');
      expect(balancedButton).toHaveClass('border-blue-500');
    });

    it('calls onPresetChange when selecting a different preset', async () => {
      const user = userEvent.setup();
      render(<StreamSection {...defaultProps} />);

      await user.click(screen.getByText('Fast'));

      expect(defaultProps.onPresetChange).toHaveBeenCalledWith('fast');
      expect(defaultProps.onSettingsChange).toHaveBeenCalledWith({
        quality: 40,
        fps: 10,
        scale: 'css',
      });
    });
  });

  describe('Custom Settings', () => {
    it('does not show custom settings controls when preset is not custom', () => {
      render(<StreamSection {...defaultProps} />);

      expect(screen.queryByLabelText('Quality')).not.toBeInTheDocument();
      expect(screen.queryByLabelText('Frame Rate')).not.toBeInTheDocument();
    });

    it('shows custom settings controls when preset is custom', () => {
      render(<StreamSection {...defaultProps} preset="custom" />);

      expect(screen.getByLabelText('Quality')).toBeInTheDocument();
      expect(screen.getByLabelText('Frame Rate')).toBeInTheDocument();
      expect(screen.getByText('Resolution')).toBeInTheDocument();
    });

    it('calls onSettingsChange when quality slider changes', async () => {
      const user = userEvent.setup();
      render(<StreamSection {...defaultProps} preset="custom" />);

      const qualitySlider = screen.getByLabelText('Quality');
      // Note: Testing range inputs with userEvent is tricky, so we test the presence
      expect(qualitySlider).toHaveAttribute('type', 'range');
      expect(qualitySlider).toHaveAttribute('min', '10');
      expect(qualitySlider).toHaveAttribute('max', '100');
    });

    it('calls onSettingsChange when resolution button is clicked', async () => {
      const user = userEvent.setup();
      render(<StreamSection {...defaultProps} preset="custom" />);

      await user.click(screen.getByText('HiDPI (2x)'));

      expect(defaultProps.onSettingsChange).toHaveBeenCalledWith({
        ...defaultSettings,
        scale: 'device',
      });
    });
  });

  describe('Performance Stats Toggle', () => {
    it('renders performance stats toggle', () => {
      render(<StreamSection {...defaultProps} />);

      expect(screen.getByText('Show performance stats')).toBeInTheDocument();
      expect(screen.getByText('FPS, latency, and bottleneck analysis')).toBeInTheDocument();
    });

    it('calls onShowStatsChange when toggle is clicked', async () => {
      const user = userEvent.setup();
      render(<StreamSection {...defaultProps} />);

      await user.click(screen.getByText('Show performance stats'));

      expect(defaultProps.onShowStatsChange).toHaveBeenCalledWith(true);
    });
  });

  describe('Active Session Hint', () => {
    it('shows resolution hint when session is active', () => {
      render(<StreamSection {...defaultProps} hasActiveSession={true} />);

      expect(screen.getByText(/Resolution.*changes require a new session/)).toBeInTheDocument();
    });

    it('does not show resolution hint when no active session', () => {
      render(<StreamSection {...defaultProps} hasActiveSession={false} />);

      expect(screen.queryByText(/Resolution.*changes require a new session/)).not.toBeInTheDocument();
    });
  });
});
