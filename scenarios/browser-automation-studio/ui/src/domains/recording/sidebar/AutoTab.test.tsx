/**
 * AutoTab Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AutoTab, type AutoTabProps } from './AutoTab';
import { createUserMessage, createAssistantMessage, createSystemMessage, DEFAULT_AI_SETTINGS } from './types';
import { VISION_MODELS } from '../ai-navigation/types';
import type { AIMessage } from './types';

// Mock scrollIntoView which isn't available in jsdom
Element.prototype.scrollIntoView = vi.fn();

// Mock AISettingsModal to simplify tests
vi.mock('./AISettingsModal', () => ({
  AISettingsModal: vi.fn(({ isOpen, onClose, onSaveSettings, currentSettings }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="settings-modal" role="dialog">
        <button onClick={onClose}>Close</button>
        <button
          onClick={() => onSaveSettings({ ...currentSettings, maxSteps: 50 })}
          data-testid="save-settings"
        >
          Save
        </button>
      </div>
    );
  }),
}));

// Mock AIMessageBubble to simplify tests
vi.mock('./AIMessageBubble', () => ({
  AIMessageBubble: vi.fn(({ message, onAbort, onHumanDone }) => (
    <div data-testid={`message-${message.id}`} data-role={message.role}>
      <span>{message.content || message.status}</span>
      {onAbort && (
        <button onClick={onAbort} data-testid="abort-button">
          Abort
        </button>
      )}
      {onHumanDone && (
        <button onClick={onHumanDone} data-testid="human-done-button">
          Done
        </button>
      )}
    </div>
  )),
}));

describe('AutoTab', () => {
  const defaultProps: AutoTabProps = {
    messages: [],
    isNavigating: false,
    settings: DEFAULT_AI_SETTINGS,
    availableModels: VISION_MODELS,
    onSendMessage: vi.fn(),
    onAbort: vi.fn(),
    onHumanDone: vi.fn(),
    onSettingsChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render empty state when no messages', () => {
      render(<AutoTab {...defaultProps} />);

      // "AI Navigator" appears in header and empty state
      expect(screen.getAllByText('AI Navigator')).toHaveLength(2);
      expect(
        screen.getByText(/Describe where you want to navigate/)
      ).toBeInTheDocument();
    });

    it('should render messages', () => {
      const messages: AIMessage[] = [
        createUserMessage('Go to login'),
        createAssistantMessage('nav-1'),
      ];

      render(<AutoTab {...defaultProps} messages={messages} />);

      expect(screen.getByTestId(`message-${messages[0].id}`)).toBeInTheDocument();
      expect(screen.getByTestId(`message-${messages[1].id}`)).toBeInTheDocument();
    });

    it('should render input field', () => {
      render(<AutoTab {...defaultProps} />);

      expect(screen.getByRole('textbox', { name: /Navigation prompt/i })).toBeInTheDocument();
    });

    it('should render settings button', () => {
      render(<AutoTab {...defaultProps} />);

      expect(screen.getByRole('button', { name: /AI Settings/i })).toBeInTheDocument();
    });

    it('should show placeholder text when not navigating', () => {
      render(<AutoTab {...defaultProps} />);

      expect(screen.getByPlaceholderText('Where do you want to go?')).toBeInTheDocument();
    });

    it('should show navigating placeholder when navigating', () => {
      render(<AutoTab {...defaultProps} isNavigating={true} />);

      expect(screen.getByPlaceholderText('Navigation in progress...')).toBeInTheDocument();
    });
  });

  describe('input handling', () => {
    it('should update input value on change', async () => {
      const user = userEvent.setup();
      render(<AutoTab {...defaultProps} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, 'Navigate to dashboard');

      expect(input).toHaveValue('Navigate to dashboard');
    });

    it('should call onSendMessage when form submitted', async () => {
      const user = userEvent.setup();
      const onSendMessage = vi.fn();
      render(<AutoTab {...defaultProps} onSendMessage={onSendMessage} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, 'Go to settings');
      await user.click(screen.getByRole('button', { name: /Send message/i }));

      expect(onSendMessage).toHaveBeenCalledWith('Go to settings');
    });

    it('should clear input after sending', async () => {
      const user = userEvent.setup();
      render(<AutoTab {...defaultProps} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, 'Test message');
      await user.click(screen.getByRole('button', { name: /Send message/i }));

      expect(input).toHaveValue('');
    });

    it('should submit on Enter key (without Shift)', async () => {
      const user = userEvent.setup();
      const onSendMessage = vi.fn();
      render(<AutoTab {...defaultProps} onSendMessage={onSendMessage} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, 'Navigate{Enter}');

      expect(onSendMessage).toHaveBeenCalledWith('Navigate');
    });

    it('should not submit on Shift+Enter', async () => {
      const user = userEvent.setup();
      const onSendMessage = vi.fn();
      render(<AutoTab {...defaultProps} onSendMessage={onSendMessage} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, 'Line 1{Shift>}{Enter}{/Shift}Line 2');

      expect(onSendMessage).not.toHaveBeenCalled();
    });

    it('should not submit empty messages', async () => {
      const user = userEvent.setup();
      const onSendMessage = vi.fn();
      render(<AutoTab {...defaultProps} onSendMessage={onSendMessage} />);

      const input = screen.getByRole('textbox', { name: /Navigation prompt/i });
      await user.type(input, '   ');
      await user.click(screen.getByRole('button', { name: /Send message/i }));

      expect(onSendMessage).not.toHaveBeenCalled();
    });

    it('should disable input when navigating', () => {
      render(<AutoTab {...defaultProps} isNavigating={true} />);

      expect(screen.getByRole('textbox', { name: /Navigation prompt/i })).toBeDisabled();
    });

    it('should disable send button when navigating', () => {
      render(<AutoTab {...defaultProps} isNavigating={true} />);

      expect(screen.getByRole('button', { name: /Send message/i })).toBeDisabled();
    });

    it('should disable send button when input is empty', () => {
      render(<AutoTab {...defaultProps} />);

      expect(screen.getByRole('button', { name: /Send message/i })).toBeDisabled();
    });
  });

  describe('settings modal', () => {
    it('should open settings modal when button clicked', async () => {
      const user = userEvent.setup();
      render(<AutoTab {...defaultProps} />);

      await user.click(screen.getByRole('button', { name: /AI Settings/i }));

      expect(screen.getByTestId('settings-modal')).toBeInTheDocument();
    });

    it('should close settings modal', async () => {
      const user = userEvent.setup();
      render(<AutoTab {...defaultProps} />);

      await user.click(screen.getByRole('button', { name: /AI Settings/i }));
      await user.click(screen.getByText('Close'));

      expect(screen.queryByTestId('settings-modal')).not.toBeInTheDocument();
    });

    it('should call onSettingsChange when settings saved', async () => {
      const user = userEvent.setup();
      const onSettingsChange = vi.fn();
      render(<AutoTab {...defaultProps} onSettingsChange={onSettingsChange} />);

      await user.click(screen.getByRole('button', { name: /AI Settings/i }));
      await user.click(screen.getByTestId('save-settings'));

      expect(onSettingsChange).toHaveBeenCalledWith(expect.objectContaining({ maxSteps: 50 }));
    });
  });

  describe('clear button', () => {
    it('should show clear button when onClear provided and messages exist', () => {
      const messages: AIMessage[] = [createUserMessage('Test')];
      render(<AutoTab {...defaultProps} messages={messages} onClear={() => {}} />);

      expect(screen.getByRole('button', { name: /Clear conversation/i })).toBeInTheDocument();
    });

    it('should not show clear button when no messages', () => {
      render(<AutoTab {...defaultProps} onClear={() => {}} />);

      expect(screen.queryByRole('button', { name: /Clear conversation/i })).not.toBeInTheDocument();
    });

    it('should not show clear button when onClear not provided', () => {
      const messages: AIMessage[] = [createUserMessage('Test')];
      render(<AutoTab {...defaultProps} messages={messages} />);

      expect(screen.queryByRole('button', { name: /Clear conversation/i })).not.toBeInTheDocument();
    });

    it('should call onClear when clicked', async () => {
      const user = userEvent.setup();
      const onClear = vi.fn();
      const messages: AIMessage[] = [createUserMessage('Test')];
      render(<AutoTab {...defaultProps} messages={messages} onClear={onClear} />);

      await user.click(screen.getByRole('button', { name: /Clear conversation/i }));

      expect(onClear).toHaveBeenCalled();
    });
  });

  describe('message interactions', () => {
    it('should pass onAbort to messages with canAbort', () => {
      const onAbort = vi.fn();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        canAbort: true,
      };

      render(<AutoTab {...defaultProps} messages={[message]} onAbort={onAbort} />);

      expect(screen.getByTestId('abort-button')).toBeInTheDocument();
    });

    it('should not pass onAbort to messages without canAbort', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        canAbort: false,
      };

      render(<AutoTab {...defaultProps} messages={[message]} />);

      expect(screen.queryByTestId('abort-button')).not.toBeInTheDocument();
    });

    it('should call onAbort when abort button clicked', async () => {
      const user = userEvent.setup();
      const onAbort = vi.fn();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        canAbort: true,
      };

      render(<AutoTab {...defaultProps} messages={[message]} onAbort={onAbort} />);

      await user.click(screen.getByTestId('abort-button'));

      expect(onAbort).toHaveBeenCalled();
    });

    it('should pass onHumanDone to messages awaiting human', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'awaiting_human',
        humanIntervention: {
          reason: 'CAPTCHA',
          interventionType: 'captcha',
          trigger: 'programmatic',
          startedAt: new Date(),
        },
      };

      render(<AutoTab {...defaultProps} messages={[message]} />);

      expect(screen.getByTestId('human-done-button')).toBeInTheDocument();
    });

    it('should call onHumanDone when human done button clicked', async () => {
      const user = userEvent.setup();
      const onHumanDone = vi.fn();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'awaiting_human',
        humanIntervention: {
          reason: 'Login',
          interventionType: 'login_required',
          trigger: 'ai_requested',
          startedAt: new Date(),
        },
      };

      render(<AutoTab {...defaultProps} messages={[message]} onHumanDone={onHumanDone} />);

      await user.click(screen.getByTestId('human-done-button'));

      expect(onHumanDone).toHaveBeenCalled();
    });
  });

  describe('className prop', () => {
    it('should apply custom className', () => {
      const { container } = render(<AutoTab {...defaultProps} className="custom-class" />);

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });
});
