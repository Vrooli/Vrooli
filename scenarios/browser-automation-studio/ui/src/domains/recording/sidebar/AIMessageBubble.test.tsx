/**
 * AIMessageBubble Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@/test-utils';
import { expectNoA11yViolations } from '@vrooli/api-base/testing';
import userEvent from '@testing-library/user-event';
import { AIMessageBubble } from './AIMessageBubble';
import { createUserMessage, createAssistantMessage, createSystemMessage, type AIMessage } from './types';
import type { AINavigationStep } from '../ai-navigation/types';

describe('AIMessageBubble', () => {
  const mockStep: AINavigationStep = {
    id: 'step-1',
    stepNumber: 1,
    action: { type: 'click', elementId: 1 },
    reasoning: 'Clicking the login button',
    currentUrl: 'https://example.com',
    goalAchieved: false,
    tokensUsed: { promptTokens: 100, completionTokens: 50, totalTokens: 150 },
    durationMs: 500,
    timestamp: new Date(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('user messages', () => {
    it('should render user message with content', () => {
      const message = createUserMessage('Navigate to the login page');
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Navigate to the login page')).toBeInTheDocument();
    });

    it('should render user message with purple background', () => {
      const message = createUserMessage('Test message');
      const { container } = render(<AIMessageBubble message={message} />);

      const bubble = container.querySelector('.bg-purple-600');
      expect(bubble).toBeInTheDocument();
    });

    it('should show timestamp', () => {
      const message = createUserMessage('Test message');
      render(<AIMessageBubble message={message} />);

      // Check for time format (e.g., "2:30 PM")
      const timeElement = screen.getByText(/\d{1,2}:\d{2}\s*(AM|PM)?/i);
      expect(timeElement).toBeInTheDocument();
    });
  });

  describe('system messages', () => {
    it('should render system message centered', () => {
      const message = createSystemMessage('Session started');
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Session started')).toBeInTheDocument();
    });

    it('should render system message with gray background', () => {
      const message = createSystemMessage('Test system message');
      const { container } = render(<AIMessageBubble message={message} />);

      const bubble = container.querySelector('.bg-gray-100');
      expect(bubble).toBeInTheDocument();
    });
  });

  describe('assistant messages', () => {
    it('should render pending status', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'pending',
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Starting')).toBeInTheDocument();
      expect(screen.getByText('Starting navigation...')).toBeInTheDocument();
    });

    it('should render running status with step count', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Navigating')).toBeInTheDocument();
      expect(screen.getByText('(1)')).toBeInTheDocument();
    });

    it('should render completed status', () => {
      const completedStep = { ...mockStep, goalAchieved: true };
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [completedStep],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('Goal achieved in 1 step')).toBeInTheDocument();
    });

    it('should render failed status with error', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'failed',
        error: 'Element not found',
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Failed')).toBeInTheDocument();
      expect(screen.getByText('Element not found')).toBeInTheDocument();
    });

    it('should render aborted status', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'aborted',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Stopped')).toBeInTheDocument();
    });

    it('should render awaiting_human status', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'awaiting_human',
        humanIntervention: {
          reason: 'CAPTCHA detected',
          instructions: 'Please solve the CAPTCHA',
          interventionType: 'captcha',
          trigger: 'programmatic',
          startedAt: new Date(),
        },
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('Waiting for you')).toBeInTheDocument();
      expect(screen.getByText('CAPTCHA detected')).toBeInTheDocument();
      expect(screen.getByText('Please solve the CAPTCHA')).toBeInTheDocument();
    });

    it('should show token usage for completed messages', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep],
        totalTokens: 1500,
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByText('1,500 tokens used')).toBeInTheDocument();
    });
  });

  describe('abort functionality', () => {
    it('should show abort button when running and canAbort is true', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        canAbort: true,
      };
      render(<AIMessageBubble message={message} onAbort={() => {}} />);

      expect(screen.getAllByRole('button', { name: /Stop navigation/i })).toHaveLength(2);
    });

    it('should not show abort button when completed', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        canAbort: true, // Even with canAbort true
      };
      render(<AIMessageBubble message={message} onAbort={() => {}} />);

      expect(screen.queryByRole('button', { name: /Stop navigation/i })).not.toBeInTheDocument();
    });

    it('should call onAbort when abort button is clicked', async () => {
      const user = userEvent.setup();
      const onAbort = vi.fn();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        canAbort: true,
      };
      render(<AIMessageBubble message={message} onAbort={onAbort} />);

      await user.click(screen.getAllByRole('button', { name: /Stop navigation/i })[0]);

      expect(onAbort).toHaveBeenCalled();
    });
  });

  describe('human intervention', () => {
    it('should show "I\'m Done" button when awaiting human', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'awaiting_human',
        humanIntervention: {
          reason: 'Login required',
          interventionType: 'login_required',
          trigger: 'ai_requested',
          startedAt: new Date(),
        },
      };
      render(<AIMessageBubble message={message} onHumanDone={() => {}} />);

      expect(screen.getByRole('button', { name: /I'm Done/i })).toBeInTheDocument();
    });

    it('should call onHumanDone when button is clicked', async () => {
      const user = userEvent.setup();
      const onHumanDone = vi.fn();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'awaiting_human',
        humanIntervention: {
          reason: 'Login required',
          interventionType: 'login_required',
          trigger: 'ai_requested',
          startedAt: new Date(),
        },
      };
      render(<AIMessageBubble message={message} onHumanDone={onHumanDone} />);

      await user.click(screen.getByRole('button', { name: /I'm Done/i }));

      expect(onHumanDone).toHaveBeenCalled();
    });
  });

  describe('timeline steps', () => {
    it('has no axe violations in the expanded timeline state', async () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        content: 'Fill the sign-in form',
        status: 'completed',
        steps: [{
          ...mockStep,
          reasoning: 'The sign-in form is ready.',
          action: {
            type: 'type',
            selector: '#email',
            value: 'user@example.com',
            text: 'user@example.com',
            success: true,
          },
        }],
      };
      const { container } = render(<AIMessageBubble message={message} />);

      await expectNoA11yViolations(container);
    });

    it('labels provider-specific action types in the timeline', () => {
      const providerActions = ['find', 'read', 'evaluate', 'tabs', 'drag', 'zoom'] as const;
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        steps: providerActions.map((type, index) => ({
          ...mockStep,
          id: `step-${index + 1}`,
          stepNumber: index + 1,
          action: { type },
        })),
      };
      render(<AIMessageBubble message={message} />);

      for (const label of ['Find', 'Read', 'Evaluate', 'Tabs', 'Drag', 'Zoom']) {
        expect(screen.getByRole('button', { name: new RegExp(label) })).toBeInTheDocument();
      }
    });

    it('shows preserved provider action details when a step is expanded', async () => {
      const user = userEvent.setup();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [{
          ...mockStep,
          action: {
            type: 'evaluate',
            selector: '#email',
            value: 'typed@example.com',
            text: 'typed@example.com',
            result: 'field updated',
            success: true,
          },
        }],
      };
      render(<AIMessageBubble message={message} />);

      await user.click(screen.getByRole('button', { name: /Evaluate/i }));

      expect(screen.getByText('#email')).toBeInTheDocument();
      expect(screen.getAllByText('typed@example.com')).toHaveLength(2);
      expect(screen.getByText('field updated')).toBeInTheDocument();
      expect(screen.getByText('Success')).toBeInTheDocument();
    });

    it('does not render synthetic secrets from sensitive AI actions', async () => {
      const user = userEvent.setup();
      const secret = 'BAS_SYNTHETIC_PASSWORD_7f3e';
      const message: AIMessage = {
        ...createAssistantMessage('nav-secret'),
        status: 'completed',
        steps: [{
          ...mockStep,
          action: {
            type: 'type',
            selector: 'input[name="password"]',
            value: secret,
            text: secret,
            result: `password=${secret}`,
          },
        }],
      };
      render(<AIMessageBubble message={message} />);

      await user.click(screen.getByRole('button', { name: /Type/i }));

      expect(screen.queryByText(secret)).not.toBeInTheDocument();
      expect(screen.getAllByText('[REDACTED]')).toHaveLength(2);
      expect(screen.getByText('password=[REDACTED]')).toBeInTheDocument();
    });

    it('keeps intentional empty provider values visible', async () => {
      const user = userEvent.setup();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [{
          ...mockStep,
          action: { type: 'type', selector: '#email', value: '', text: '' },
        }],
      };
      render(<AIMessageBubble message={message} />);

      await user.click(screen.getByRole('button', { name: /Type/i }));

      expect(screen.getAllByText('(empty)')).toHaveLength(2);
    });

    it('renders completed steps as timeline cards', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep, { ...mockStep, id: 'step-2', stepNumber: 2 }],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getAllByRole('button', { name: /Click/i })).toHaveLength(2);
    });

    it('renders running steps without hiding the timeline', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByRole('button', { name: /Click/i })).toBeInTheDocument();
    });

    it('exposes disclosure state for the navigation and step controls', async () => {
      const user = userEvent.setup();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      const navigationToggle = screen.getByRole('button', { name: 'AI Navigation' });
      const stepToggle = screen.getByRole('button', { name: /Click/i });

      expect(navigationToggle).toHaveAttribute('aria-expanded', 'true');
      expect(navigationToggle).toHaveAttribute('aria-controls', 'ai-navigation-content-assistant-nav-1');
      expect(stepToggle).toHaveAttribute('aria-expanded', 'false');
      expect(stepToggle).toHaveAttribute('aria-controls', 'ai-navigation-step-details-step-1');

      await user.click(navigationToggle);
      expect(navigationToggle).toHaveAttribute('aria-expanded', 'false');

      await user.click(navigationToggle);
      const reopenedStepToggle = screen.getByRole('button', { name: /Click/i });
      await user.click(reopenedStepToggle);
      expect(reopenedStepToggle).toHaveAttribute('aria-expanded', 'true');
    });

    it('toggles an individual step\'s details on click', async () => {
      const user = userEvent.setup();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      // The step reasoning is initially hidden.
      expect(screen.queryByText('Clicking the login button')).not.toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /Click/i }));

      expect(screen.getByText('Clicking the login button')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /Click/i }));

      expect(screen.queryByText('Clicking the login button')).not.toBeInTheDocument();
    });
  });
});
