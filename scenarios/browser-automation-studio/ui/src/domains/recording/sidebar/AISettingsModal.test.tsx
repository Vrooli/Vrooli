/**
 * AISettingsModal Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@/test-utils';
import userEvent from '@testing-library/user-event';
import { AISettingsModal, type AISettingsModalProps } from './AISettingsModal';
import { VISION_MODELS } from '../ai-navigation/types';
import { DEFAULT_AI_SETTINGS } from './types';

describe('AISettingsModal', () => {
  const defaultProps: AISettingsModalProps = {
    isOpen: true,
    onClose: vi.fn(),
    currentSettings: DEFAULT_AI_SETTINGS,
    onSaveSettings: vi.fn(),
    availableModels: VISION_MODELS,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should not render when isOpen is false', () => {
      render(<AISettingsModal {...defaultProps} isOpen={false} />);

      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', () => {
      render(<AISettingsModal {...defaultProps} />);

      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('AI Navigation Settings')).toBeInTheDocument();
    });

    it('should display all available models', () => {
      render(<AISettingsModal {...defaultProps} />);

      VISION_MODELS.forEach((model) => {
        expect(screen.getByText(model.displayName)).toBeInTheDocument();
      });
    });

    it('should display tier badges for models', () => {
      render(<AISettingsModal {...defaultProps} />);

      expect(screen.getAllByText('LOCAL').length).toBeGreaterThan(0);
      expect(screen.getAllByText('REMOTE').length).toBeGreaterThan(0);
    });

    it('should show recommended badge for recommended models', () => {
      render(<AISettingsModal {...defaultProps} />);

      const recommendedBadges = screen.getAllByText('Recommended');
      expect(recommendedBadges.length).toBeGreaterThan(0);
    });

    it('should display max steps slider', () => {
      render(<AISettingsModal {...defaultProps} />);

      expect(screen.getByText('Maximum Steps')).toBeInTheDocument();
      expect(screen.getByRole('slider')).toBeInTheDocument();
    });

    it('should explain gateway-reported usage', () => {
      render(<AISettingsModal {...defaultProps} />);

      expect(screen.getByText(/usage and any applicable charge are reported by AI Gateway/i)).toBeInTheDocument();
    });
  });

  describe('initial state', () => {
    it('should initialize with current settings', () => {
      const customSettings = { model: 'remote_only', maxSteps: 35 };
      render(<AISettingsModal {...defaultProps} currentSettings={customSettings} />);

      const hostedButton = screen.getByRole('button', { name: /Hosted vision/i });
      expect(hostedButton).toHaveClass('border-purple-500');

      // Max steps should show 35
      expect(screen.getByText('35 steps')).toBeInTheDocument();
    });
  });

  describe('model selection', () => {
    it('should update selected model on click', async () => {
      const user = userEvent.setup();
      render(<AISettingsModal {...defaultProps} />);

      const hostedButton = screen.getByRole('button', { name: /Hosted vision/i });
      await user.click(hostedButton);

      // Should now have purple border (selected)
      expect(hostedButton).toHaveClass('border-purple-500');
    });
  });

  describe('max steps slider', () => {
    it('should update max steps on change', () => {
      render(<AISettingsModal {...defaultProps} />);

      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '40' } });

      expect(screen.getByText('40 steps')).toBeInTheDocument();
    });
  });

  describe('save functionality', () => {
    it('should call onSaveSettings with updated values on save', async () => {
      const user = userEvent.setup();
      const onSaveSettings = vi.fn();
      render(<AISettingsModal {...defaultProps} onSaveSettings={onSaveSettings} />);

      await user.click(screen.getByRole('button', { name: /Hosted vision/i }));

      // Change max steps
      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '30' } });

      // Save
      const saveButton = screen.getByRole('button', { name: /Save Settings/i });
      await user.click(saveButton);

      expect(onSaveSettings).toHaveBeenCalledWith({
        model: 'remote_only',
        maxSteps: 30,
      });
    });

    it('should call onClose after save', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(<AISettingsModal {...defaultProps} onClose={onClose} />);

      const saveButton = screen.getByRole('button', { name: /Save Settings/i });
      await user.click(saveButton);

      expect(onClose).toHaveBeenCalled();
    });
  });

  describe('close functionality', () => {
    it('should call onClose when Cancel is clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(<AISettingsModal {...defaultProps} onClose={onClose} />);

      const cancelButton = screen.getByRole('button', { name: /Cancel/i });
      await user.click(cancelButton);

      expect(onClose).toHaveBeenCalled();
    });

    it('should call onClose when X button is clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(<AISettingsModal {...defaultProps} onClose={onClose} />);

      const closeButton = screen.getByRole('button', { name: /Close/i });
      await user.click(closeButton);

      expect(onClose).toHaveBeenCalled();
    });

    it('should call onClose when backdrop is clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(<AISettingsModal {...defaultProps} onClose={onClose} />);

      const backdrop = screen.getByRole('dialog');
      await user.click(backdrop);

      expect(onClose).toHaveBeenCalled();
    });

    it('should call onClose when Escape is pressed', () => {
      const onClose = vi.fn();
      render(<AISettingsModal {...defaultProps} onClose={onClose} />);

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onClose).toHaveBeenCalled();
    });
  });

  describe('reset on reopen', () => {
    it('should reset to current settings when modal reopens', () => {
      const { rerender } = render(
        <AISettingsModal
          {...defaultProps}
          currentSettings={{ model: 'local_first', maxSteps: 20 }}
        />
      );

      // Change settings in modal
      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '50' } });

      expect(screen.getByText('50 steps')).toBeInTheDocument();

      // Close and reopen with different settings
      rerender(
        <AISettingsModal
          {...defaultProps}
          isOpen={false}
          currentSettings={{ model: 'remote_only', maxSteps: 30 }}
        />
      );

      rerender(
        <AISettingsModal
          {...defaultProps}
          isOpen={true}
          currentSettings={{ model: 'remote_only', maxSteps: 30 }}
        />
      );

      // Should show the new current settings
      expect(screen.getByText('30 steps')).toBeInTheDocument();
    });
  });
});
