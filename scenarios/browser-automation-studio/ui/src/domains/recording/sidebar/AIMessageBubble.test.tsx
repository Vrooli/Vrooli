/**
 * AIMessageBubble Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AIMessageBubble, type AIMessageBubbleProps } from './AIMessageBubble';
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

      expect(screen.getByText('Starting...')).toBeInTheDocument();
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
      expect(screen.getByText('Completed in 1 step')).toBeInTheDocument();
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

      expect(screen.getByText('Aborted')).toBeInTheDocument();
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

      expect(screen.getByText('1,500 tokens')).toBeInTheDocument();
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

      expect(screen.getByRole('button', { name: /Abort Navigation/i })).toBeInTheDocument();
    });

    it('should not show abort button when completed', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        canAbort: true, // Even with canAbort true
      };
      render(<AIMessageBubble message={message} onAbort={() => {}} />);

      expect(screen.queryByRole('button', { name: /Abort Navigation/i })).not.toBeInTheDocument();
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

      await user.click(screen.getByRole('button', { name: /Abort Navigation/i }));

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

  describe('expandable steps', () => {
    it('should show expand button when completed with steps', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep, { ...mockStep, id: 'step-2', stepNumber: 2 }],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.getByRole('button', { name: /Show 2 steps/i })).toBeInTheDocument();
    });

    it('should not show expand button when running', () => {
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'running',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      expect(screen.queryByRole('button', { name: /Show/i })).not.toBeInTheDocument();
    });

    it('should toggle steps visibility on click', async () => {
      const user = userEvent.setup();
      const message: AIMessage = {
        ...createAssistantMessage('nav-1'),
        status: 'completed',
        steps: [mockStep],
      };
      render(<AIMessageBubble message={message} />);

      // Initially hidden
      expect(screen.queryByText('Clicking the login button')).not.toBeInTheDocument();

      // Click to expand
      await user.click(screen.getByRole('button', { name: /Show 1 step/i }));

      // Now visible
      expect(screen.getByText('Clicking the login button')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Hide 1 step/i })).toBeInTheDocument();

      // Click to collapse
      await user.click(screen.getByRole('button', { name: /Hide 1 step/i }));

      // Hidden again
      expect(screen.queryByText('Clicking the login button')).not.toBeInTheDocument();
    });
  });
});
