/**
 * useAIConversation Hook Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAIConversation } from './useAIConversation';
import type { AINavigationStep } from '../ai-navigation/types';
import { AINavigationError } from '../ai-navigation/useAINavigation';

// Mock useAINavigation
const mockStartNavigation = vi.fn();
const mockAbortNavigation = vi.fn();
const mockResumeNavigation = vi.fn();
const mockReset = vi.fn();

let mockOnStep: ((step: AINavigationStep, navigationId?: string) => void) | undefined;
let mockOnStarted: ((navigationId: string, startAttempt?: number) => void) | undefined;
let mockStartAttempt = 0;
let mockOnComplete: ((status: string, summary?: string, navigationId?: string) => void) | undefined;

const mockNavState = {
  isNavigating: false,
  navigationId: null as string | null,
  prompt: '',
  model: 'local_first',
  steps: [] as AINavigationStep[],
  status: 'idle' as 'idle' | 'awaiting_human',
  totalTokens: 0,
  error: null as string | null,
  humanIntervention: null as {
    reason: string;
    instructions?: string;
    interventionType: 'captcha' | 'verification' | 'complex_interaction' | 'login_required' | 'other';
    trigger: 'programmatic' | 'ai_requested';
    startedAt: Date;
  } | null,
};

vi.mock('../ai-navigation/useAINavigation', () => ({
  AINavigationError: class AINavigationError extends Error {
    code: string;
    details?: Record<string, string>;

    constructor(code: string, message: string, details?: Record<string, string>) {
      super(message);
      this.name = 'AINavigationError';
      this.code = code;
      this.details = details;
    }
  },
  useAINavigation: vi.fn(({ onStarted, onStep, onComplete }) => {
    // Capture callbacks for testing
    mockOnStarted = onStarted;
    mockOnStep = onStep;
    mockOnComplete = onComplete;

    return {
      state: mockNavState,
      startNavigation: async (...args: [string, string, number?]) => {
        const navigationId = await mockStartNavigation(...args) as string | null;
        if (navigationId) onStarted?.(navigationId, ++mockStartAttempt);
        return navigationId;
      },
      abortNavigation: mockAbortNavigation,
      resumeNavigation: mockResumeNavigation,
      reset: mockReset,
      availableModels: [],
      isNavigating: mockNavState.isNavigating,
      isAwaitingHuman: mockNavState.status === 'awaiting_human',
    };
  }),
}));

describe('useAIConversation', () => {
  const defaultSettings = { model: 'local_first', maxSteps: 20 };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset mock state
    mockNavState.isNavigating = false;
    mockNavState.navigationId = null;
    mockNavState.status = 'idle';
    mockNavState.error = null;
    mockNavState.humanIntervention = null;
    mockNavState.steps = [];
    mockStartAttempt = 0;
    mockResumeNavigation.mockResolvedValue(true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initial state', () => {
    it('should initialize with empty messages', () => {
      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      expect(result.current.messages).toEqual([]);
      expect(result.current.isNavigating).toBe(false);
      expect(result.current.currentNavigationId).toBeNull();
      expect(result.current.isAwaitingHuman).toBe(false);
    });
  });

  describe('sendMessage', () => {
    it('renders entitlement errors through the dedicated message policy', async () => {
      mockStartNavigation.mockRejectedValueOnce(new AINavigationError(
        'AI_NOT_AVAILABLE',
        'plan does not include AI navigation',
      ));

      const { result } = renderHook(() => useAIConversation({
        sessionId: 'test-session',
        settings: defaultSettings,
      }));

      await act(async () => {
        await result.current.sendMessage('Navigate to login');
      });

      expect(result.current.messages).toHaveLength(2);
      expect(result.current.messages[1]).toMatchObject({
        role: 'system',
        errorCode: 'AI_NOT_AVAILABLE',
      });
    });

    it('should create user message when sending', async () => {
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate to login page');
      });

      expect(result.current.messages.length).toBeGreaterThanOrEqual(1);
      const userMessage = result.current.messages[0];
      expect(userMessage.role).toBe('user');
      expect(userMessage.content).toBe('Navigate to login page');
    });

    it('should call startNavigation with correct parameters', async () => {
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: { model: 'remote_only', maxSteps: 30 },
        })
      );

      await act(async () => {
        await result.current.sendMessage('Click the button');
      });

      expect(mockStartNavigation).toHaveBeenCalledWith('Click the button', 'remote_only', 30);
    });

    it('should create assistant message after starting navigation', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate to home');
      });

      // Should have user message and assistant message
      expect(result.current.messages.length).toBe(2);
      expect(result.current.messages[0].role).toBe('user');
      expect(result.current.messages[1].role).toBe('assistant');
      expect(result.current.messages[1].status).toBe('pending');
    });

    it('should admit a navigation only once when start notification is repeated', async () => {
      mockStartNavigation.mockResolvedValue('nav-duplicate');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate once');
      });

      act(() => {
        mockOnStarted?.('nav-duplicate');
      });

      expect(result.current.messages).toHaveLength(2);
      expect(result.current.messages[1].navigationId).toBe('nav-duplicate');
    });

    it('should ignore a late duplicate start notification after completion', async () => {
      mockStartNavigation.mockResolvedValue('nav-late-duplicate');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Complete once');
      });
      act(() => {
        mockOnComplete?.('completed', 'Done', 'nav-late-duplicate');
        mockOnStarted?.('nav-late-duplicate');
      });

      expect(result.current.messages).toHaveLength(2);
      expect(result.current.messages[1].status).toBe('completed');
    });

    it('should ignore a stale start notification after conversation reset', async () => {
      mockStartNavigation.mockResolvedValue('nav-reset-stale');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Reset before admission');
      });
      act(() => {
        result.current.clearConversation();
        mockOnStarted?.('nav-reset-stale');
      });

      expect(result.current.messages).toEqual([]);
    });

    it('should not append a stale start error after the conversation is reset', async () => {
      let rejectStart: ((reason?: unknown) => void) | undefined;
      mockStartNavigation.mockImplementationOnce(() => new Promise((_, reject) => {
        rejectStart = reject;
      }));

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      let pendingSend: Promise<void> | undefined;
      act(() => {
        pendingSend = result.current.sendMessage('Stale start');
      });

      act(() => {
        result.current.clearConversation();
      });

      await act(async () => {
        rejectStart?.(new Error('stale start failure'));
        await pendingSend;
      });

      expect(result.current.messages).toEqual([]);
    });

    it('should admit only one synchronous send while the first start is pending', async () => {
      let resolveStart: ((navigationId: string) => void) | undefined;
      mockStartNavigation.mockImplementationOnce(() => new Promise<string>((resolve) => {
        resolveStart = resolve;
      }));

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      let firstSend: Promise<void> | undefined;
      let secondSend: Promise<void> | undefined;
      act(() => {
        firstSend = result.current.sendMessage('First concurrent send');
        secondSend = result.current.sendMessage('Second concurrent send');
      });

      expect(mockStartNavigation).toHaveBeenCalledTimes(1);
      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].content).toBe('First concurrent send');

      await act(async () => {
        resolveStart?.('nav-concurrent');
        await firstSend;
        await secondSend;
      });

      expect(result.current.messages).toHaveLength(2);
      expect(result.current.messages[1].navigationId).toBe('nav-concurrent');
    });

    it('should not send empty messages', async () => {
      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('   ');
      });

      expect(result.current.messages).toEqual([]);
      expect(mockStartNavigation).not.toHaveBeenCalled();
    });

    it('should add error system message when navigation fails', async () => {
      mockStartNavigation.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate somewhere');
      });

      const messages = result.current.messages;
      expect(messages.length).toBe(2); // user message + error system message
      expect(messages[1].role).toBe('system');
      expect(messages[1].content).toContain('Network error');
    });
  });

  describe('navigation step updates', () => {
    it('should update assistant message with steps', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Click button');
      });

      // Simulate step event
      const step: AINavigationStep = {
        id: 'step-1',
        stepNumber: 1,
        action: { type: 'click', elementId: 5 },
        reasoning: 'Clicking the login button',
        currentUrl: 'https://example.com',
        goalAchieved: false,
        tokensUsed: { promptTokens: 100, completionTokens: 50, totalTokens: 150 },
        durationMs: 500,
        timestamp: new Date(),
      };

      act(() => {
        mockOnStep?.(step, 'nav-123');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('running');
      expect(assistantMessage.steps).toHaveLength(1);
      expect(assistantMessage.steps?.[0].reasoning).toBe('Clicking the login button');
      expect(assistantMessage.totalTokens).toBe(150);
    });

    it('should accumulate tokens across steps', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      const createStep = (stepNumber: number, tokens: number): AINavigationStep => ({
        id: `step-${stepNumber}`,
        stepNumber,
        action: { type: 'click' },
        reasoning: `Step ${stepNumber}`,
        currentUrl: 'https://example.com',
        goalAchieved: false,
        tokensUsed: { promptTokens: tokens, completionTokens: tokens / 2, totalTokens: tokens * 1.5 },
        durationMs: 100,
        timestamp: new Date(),
      });

      act(() => {
        mockOnStep?.(createStep(1, 100), 'nav-123');
      });
      act(() => {
        mockOnStep?.(createStep(2, 100), 'nav-123');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.steps).toHaveLength(2);
      expect(assistantMessage.totalTokens).toBe(300); // 150 + 150
    });

    it('should call onTimelineAction callback when step received', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');
      const onTimelineAction = vi.fn();

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
          onTimelineAction,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      const step: AINavigationStep = {
        id: 'step-1',
        stepNumber: 1,
        action: { type: 'click' },
        reasoning: 'Click',
        currentUrl: 'https://example.com',
        goalAchieved: false,
        tokensUsed: { promptTokens: 100, completionTokens: 50, totalTokens: 150 },
        durationMs: 100,
        timestamp: new Date(),
      };

      act(() => {
        mockOnStep?.(step, 'nav-123');
      });

      expect(onTimelineAction).toHaveBeenCalled();
    });
  });

  describe('navigation completion', () => {
    it('should mark message as completed on success', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      // Rerender to ensure callbacks are updated with current ref values
      rerender();

      act(() => {
        mockOnComplete?.('completed', 'Successfully navigated to login page', 'nav-123');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('completed');
      expect(assistantMessage.content).toBe('Successfully navigated to login page');
      expect(assistantMessage.canAbort).toBe(false);
    });

    it('should mark message as failed on error', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      // Rerender to ensure callbacks are updated with current ref values
      rerender();

      act(() => {
        mockOnComplete?.('failed', undefined, 'nav-123');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('failed');
    });

    it('should mark message as aborted', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      // Rerender to ensure callbacks are updated with current ref values
      rerender();

      act(() => {
        mockOnComplete?.('aborted', undefined, 'nav-123');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('aborted');
    });

    it('keeps one recoverable assistant message across observation loss and terminal recovery', async () => {
      mockNavState.navigationId = 'nav-observation-loss';
      mockStartNavigation.mockResolvedValue('nav-observation-loss');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Recover this navigation');
      });

      mockNavState.status = 'observation_unavailable' as typeof mockNavState.status;
      mockNavState.error = 'status transport lost';
      rerender();

      await waitFor(() => {
        expect(result.current.messages).toHaveLength(2);
        expect(result.current.messages[1]).toMatchObject({
          status: 'observation_unavailable',
          error: 'status transport lost',
          canAbort: true,
        });
      });

      // A recovered terminal event must settle the existing assistant message,
      // not append a duplicate or leave the user message orphaned.
      mockNavState.status = 'aborted' as typeof mockNavState.status;
      mockNavState.error = null;
      rerender();
      act(() => {
        mockOnComplete?.('aborted', undefined, 'nav-observation-loss');
      });

      expect(result.current.messages).toHaveLength(2);
      expect(result.current.messages[1]).toMatchObject({
        status: 'aborted',
        canAbort: false,
      });
    });

    it('should not let a stale navigation completion finalize a successor message', async () => {
      mockNavState.navigationId = 'nav-first';
      mockStartNavigation.mockResolvedValueOnce('nav-first').mockResolvedValueOnce('nav-second');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('First navigation');
      });

      mockNavState.navigationId = 'nav-second';
      rerender();
      await act(async () => {
        await result.current.sendMessage('Successor navigation');
      });

      act(() => {
        mockOnComplete?.('failed', 'stale completion', 'nav-first');
      });

      expect(result.current.messages[1].status).toBe('pending');
      expect(result.current.messages[3].status).toBe('pending');
    });

    it('should ignore an older start callback after a successor starts', async () => {
      mockNavState.navigationId = 'nav-first';
      mockStartNavigation.mockResolvedValueOnce('nav-first').mockResolvedValueOnce('nav-second');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('First navigation');
      });

      mockNavState.navigationId = 'nav-second';
      rerender();
      await act(async () => {
        await result.current.sendMessage('Successor navigation');
      });

      act(() => {
        mockOnStarted?.('nav-first', 1);
      });

      expect(result.current.messages[1].navigationId).toBe('nav-first');
      expect(result.current.messages[3].navigationId).toBe('nav-second');
      expect(result.current.messages).toHaveLength(4);
    });

    it('should ignore a step callback without the current navigation identity', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      const step: AINavigationStep = {
        id: 'stale-step',
        stepNumber: 1,
        action: { type: 'click' },
        reasoning: 'Stale step',
        currentUrl: 'https://example.com',
        goalAchieved: false,
        tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
        durationMs: 1,
        timestamp: new Date(),
      };

      act(() => {
        mockOnStep?.(step);
      });

      expect(result.current.messages[1].steps).toEqual([]);
    });
  });

  describe('abort functionality', () => {
    it('should call abortNavigation on underlying hook', async () => {
      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.abortNavigation();
      });

      expect(mockAbortNavigation).toHaveBeenCalled();
    });
  });

  describe('human intervention', () => {
    it('should update message status when awaiting human', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      // Simulate human intervention state
      mockNavState.humanIntervention = {
        reason: 'CAPTCHA detected',
        instructions: 'Please solve the CAPTCHA',
        interventionType: 'captcha',
        trigger: 'programmatic',
        startedAt: new Date(),
      };

      // Trigger re-render to pick up state change
      rerender();

      await waitFor(() => {
        const assistantMessage = result.current.messages[1];
        expect(assistantMessage.status).toBe('awaiting_human');
        expect(assistantMessage.humanIntervention?.reason).toBe('CAPTCHA detected');
      });
    });

    it('should keep handoff retryable when a resume command fails', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      mockNavState.status = 'awaiting_human';
      mockNavState.humanIntervention = {
        reason: 'Verification required',
        interventionType: 'verification',
        trigger: 'ai_requested',
        startedAt: new Date(),
      };
      rerender();

      mockNavState.error = 'resume unavailable';
      rerender();

      mockResumeNavigation.mockResolvedValueOnce(false);
      await act(async () => {
        await result.current.resumeNavigation();
      });

      await waitFor(() => {
        const assistantMessage = result.current.messages[1];
        expect(assistantMessage.status).toBe('awaiting_human');
        expect(assistantMessage.error).toBe('resume unavailable');
        expect(assistantMessage.humanIntervention?.reason).toBe('Verification required');
      });
    });

    it('should call resumeNavigation and update message', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result, rerender } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Navigate');
      });

      // Set up intervention state
      mockNavState.humanIntervention = {
        reason: 'Login required',
        interventionType: 'login_required',
        trigger: 'ai_requested',
        startedAt: new Date(),
      };
      rerender();

      await act(async () => {
        await result.current.resumeNavigation();
      });

      expect(mockResumeNavigation).toHaveBeenCalled();

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('running');
      expect(assistantMessage.humanIntervention).toBeUndefined();
    });
  });

  describe('clearConversation', () => {
    it('should clear all messages', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue('nav-123');

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      await act(async () => {
        await result.current.sendMessage('Message 1');
      });

      expect(result.current.messages.length).toBeGreaterThan(0);

      act(() => {
        result.current.clearConversation();
      });

      expect(result.current.messages).toEqual([]);
      expect(mockReset).toHaveBeenCalled();
    });
  });

  describe('addSystemMessage', () => {
    it('should add system message to conversation', () => {
      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: defaultSettings,
        })
      );

      act(() => {
        result.current.addSystemMessage('Session started');
      });

      expect(result.current.messages).toHaveLength(1);
      expect(result.current.messages[0].role).toBe('system');
      expect(result.current.messages[0].content).toBe('Session started');
    });
  });

  describe('session change', () => {
    it('should clear messages when session changes', () => {
      const { result, rerender } = renderHook(
        ({ sessionId }) =>
          useAIConversation({
            sessionId,
            settings: defaultSettings,
          }),
        { initialProps: { sessionId: 'session-1' } }
      );

      act(() => {
        result.current.addSystemMessage('Test message');
      });

      expect(result.current.messages).toHaveLength(1);

      // Change session
      rerender({ sessionId: 'session-2' });

      expect(result.current.messages).toEqual([]);
    });
  });
});
