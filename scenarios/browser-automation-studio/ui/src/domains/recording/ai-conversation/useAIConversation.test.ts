/**
 * useAIConversation Hook Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAIConversation } from './useAIConversation';
import type { AINavigationStep } from '../ai-navigation/types';

// Mock useAINavigation
const mockStartNavigation = vi.fn();
const mockAbortNavigation = vi.fn();
const mockResumeNavigation = vi.fn();
const mockReset = vi.fn();

let mockOnStep: ((step: AINavigationStep) => void) | undefined;
let mockOnComplete: ((status: string, summary?: string) => void) | undefined;

const mockNavState = {
  isNavigating: false,
  navigationId: null as string | null,
  prompt: '',
  model: 'qwen3-vl-30b',
  steps: [] as AINavigationStep[],
  status: 'idle' as const,
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
  useAINavigation: vi.fn(({ onStep, onComplete }) => {
    // Capture callbacks for testing
    mockOnStep = onStep;
    mockOnComplete = onComplete;

    return {
      state: mockNavState,
      startNavigation: mockStartNavigation,
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
  const defaultSettings = { model: 'qwen3-vl-30b', maxSteps: 20 };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset mock state
    mockNavState.isNavigating = false;
    mockNavState.navigationId = null;
    mockNavState.status = 'idle';
    mockNavState.error = null;
    mockNavState.humanIntervention = null;
    mockNavState.steps = [];
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
    it('should create user message when sending', async () => {
      mockStartNavigation.mockResolvedValue(undefined);

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
      mockStartNavigation.mockResolvedValue(undefined);

      const { result } = renderHook(() =>
        useAIConversation({
          sessionId: 'test-session',
          settings: { model: 'gpt-4o', maxSteps: 30 },
        })
      );

      await act(async () => {
        await result.current.sendMessage('Click the button');
      });

      expect(mockStartNavigation).toHaveBeenCalledWith('Click the button', 'gpt-4o', 30);
    });

    it('should create assistant message after starting navigation', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
      mockStartNavigation.mockResolvedValue(undefined);

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
        mockOnStep?.(step);
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('running');
      expect(assistantMessage.steps).toHaveLength(1);
      expect(assistantMessage.steps?.[0].reasoning).toBe('Clicking the login button');
      expect(assistantMessage.totalTokens).toBe(150);
    });

    it('should accumulate tokens across steps', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
        mockOnStep?.(createStep(1, 100));
      });
      act(() => {
        mockOnStep?.(createStep(2, 100));
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.steps).toHaveLength(2);
      expect(assistantMessage.totalTokens).toBe(300); // 150 + 150
    });

    it('should call onTimelineAction callback when step received', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);
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
        mockOnStep?.(step);
      });

      expect(onTimelineAction).toHaveBeenCalled();
    });
  });

  describe('navigation completion', () => {
    it('should mark message as completed on success', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
        mockOnComplete?.('completed', 'Successfully navigated to login page');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('completed');
      expect(assistantMessage.content).toBe('Successfully navigated to login page');
      expect(assistantMessage.canAbort).toBe(false);
    });

    it('should mark message as failed on error', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
        mockOnComplete?.('failed');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('failed');
    });

    it('should mark message as aborted', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
        mockOnComplete?.('aborted');
      });

      const assistantMessage = result.current.messages[1];
      expect(assistantMessage.status).toBe('aborted');
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
      mockStartNavigation.mockResolvedValue(undefined);

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

    it('should call resumeNavigation and update message', async () => {
      mockNavState.navigationId = 'nav-123';
      mockStartNavigation.mockResolvedValue(undefined);

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
      mockStartNavigation.mockResolvedValue(undefined);

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
