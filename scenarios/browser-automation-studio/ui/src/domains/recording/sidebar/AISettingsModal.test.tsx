/**
 * AISettingsModal Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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

      expect(screen.getAllByText('BUDGET').length).toBeGreaterThan(0);
      expect(screen.getAllByText('STANDARD').length).toBeGreaterThan(0);
      expect(screen.getAllByText('PREMIUM').length).toBeGreaterThan(0);
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

    it('should display cost estimate', () => {
      render(<AISettingsModal {...defaultProps} />);

      expect(screen.getByText('Estimated cost per navigation:')).toBeInTheDocument();
    });
  });

  describe('initial state', () => {
    it('should initialize with current settings', () => {
      const customSettings = { model: 'gpt-4o', maxSteps: 35 };
      render(<AISettingsModal {...defaultProps} currentSettings={customSettings} />);

      // The GPT-4o model should be selected (radio should be checked visually)
      // Use exact match to avoid matching "GPT-4o Mini"
      const modelButtons = screen.getAllByRole('button').filter(
        (btn) => btn.textContent?.includes('GPT-4o') && !btn.textContent?.includes('Mini')
      );
      expect(modelButtons[0]).toHaveClass('border-purple-500');

      // Max steps should show 35
      expect(screen.getByText('35 steps')).toBeInTheDocument();
    });
  });

  describe('model selection', () => {
    it('should update selected model on click', async () => {
      const user = userEvent.setup();
      render(<AISettingsModal {...defaultProps} />);

      // Click on GPT-4o (use exact match to avoid GPT-4o Mini)
      const modelButtons = screen.getAllByRole('button').filter(
        (btn) => btn.textContent?.includes('GPT-4o') && !btn.textContent?.includes('Mini')
      );
      const gpt4oButton = modelButtons[0];
      await user.click(gpt4oButton);

      // Should now have purple border (selected)
      expect(gpt4oButton).toHaveClass('border-purple-500');
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

      // Change model (use exact match to avoid GPT-4o Mini)
      const modelButtons = screen.getAllByRole('button').filter(
        (btn) => btn.textContent?.includes('GPT-4o') && !btn.textContent?.includes('Mini')
      );
      await user.click(modelButtons[0]);

      // Change max steps
      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '30' } });

      // Save
      const saveButton = screen.getByRole('button', { name: /Save Settings/i });
      await user.click(saveButton);

      expect(onSaveSettings).toHaveBeenCalledWith({
        model: 'gpt-4o',
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

  describe('cost estimation', () => {
    it('should update cost estimate when model changes', async () => {
      const user = userEvent.setup();
      render(<AISettingsModal {...defaultProps} />);

      // Get initial cost
      const costElement = screen.getByText(/Estimated cost per navigation:/i).parentElement;
      const initialCost = costElement?.textContent;

      // Change to premium model (more expensive) - Claude Sonnet 4 is unique
      const premiumButton = screen.getByText('Claude Sonnet 4').closest('button');
      if (!premiumButton) throw new Error('Claude Sonnet 4 button not found');
      await user.click(premiumButton);

      // Cost should have changed
      const newCost = costElement?.textContent;
      expect(newCost).not.toBe(initialCost);
    });

    it('should update cost estimate when max steps changes', () => {
      render(<AISettingsModal {...defaultProps} />);

      const costElement = screen.getByText(/Estimated cost per navigation:/i).parentElement;
      const initialCost = costElement?.textContent;

      // Increase max steps
      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '50' } });

      // Cost should have increased
      const newCost = costElement?.textContent;
      expect(newCost).not.toBe(initialCost);
    });
  });

  describe('reset on reopen', () => {
    it('should reset to current settings when modal reopens', () => {
      const { rerender } = render(
        <AISettingsModal
          {...defaultProps}
          currentSettings={{ model: 'qwen3-vl-30b', maxSteps: 20 }}
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
          currentSettings={{ model: 'gpt-4o', maxSteps: 30 }}
        />
      );

      rerender(
        <AISettingsModal
          {...defaultProps}
          isOpen={true}
          currentSettings={{ model: 'gpt-4o', maxSteps: 30 }}
        />
      );

      // Should show the new current settings
      expect(screen.getByText('30 steps')).toBeInTheDocument();
    });
  });
});
