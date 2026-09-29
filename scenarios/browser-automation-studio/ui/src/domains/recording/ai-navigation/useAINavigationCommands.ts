import { useCallback } from 'react';
import { getAIRequestHeadersSync } from '@/utils/apiHeaders';
import { recordingApi, type ApiResult, type RequestOptions } from '../api';
import { logger } from '@/utils/logger';
import type { AINavigationState } from './types';
import { isRecord } from './navigationEvents';
import type { AINavigationCommandRefs } from './runtimeRefs';

export class AINavigationError extends Error {
  code: string;
  details?: Record<string, string>;

  constructor(code: string, message: string, details?: Record<string, string>) {
    super(message);
    this.name = 'AINavigationError';
    this.code = code;
    this.details = details;
  }
}

const parseNavigationError = (rawError: string): AINavigationError => {
  let code = 'UNKNOWN_ERROR';
  let message = rawError;
  let details: Record<string, string> | undefined;

  try {
    const errorData: unknown = JSON.parse(rawError);
    if (!isRecord(errorData)) return new AINavigationError(code, message);

    if (typeof errorData.code === 'string') code = errorData.code;
    if (typeof errorData.message === 'string') message = errorData.message;
    if (isRecord(errorData.details) && Object.values(errorData.details).every((value) => typeof value === 'string')) {
      details = errorData.details as Record<string, string>;
    }
  } catch {
    // Non-JSON errors retain their raw message and the generic code.
  }

  return new AINavigationError(code, message, details);
};

interface UseAINavigationCommandsOptions {
  sessionId: string | null;
  refs: AINavigationCommandRefs;
  observeNavigationStatus: (navigationId: string, generation: number) => void;
}

const startAINavigationRequest = async (
  sessionId: string,
  prompt: string,
  model: string,
  maxSteps: number,
  signal: AbortSignal | undefined,
): Promise<string> => {
  const result = await recordingApi.startAINavigation(
    { sessionId, prompt, model, maxSteps },
    getAIRequestHeadersSync(),
    { signal },
  );
  if (!result.success) throw parseNavigationError(result.error);
  return result.data.navigationId;
};

const commitStartedNavigation = (
  navigationId: string,
  startAttempt: number,
  requestSignal: AbortSignal | undefined,
  refs: AINavigationCommandRefs,
  observeNavigationStatus: (navigationId: string, generation: number) => void,
): string | null => {
  if (requestSignal?.aborted || refs.startAttemptRef.current !== startAttempt) return null;

  // Step numbers are only unique within one navigation session.
  refs.seenStepNumbersRef.current.clear();
  refs.navigationIdRef.current = navigationId;
  refs.startInFlightRef.current = false;
  const commandGeneration = refs.navigationCommandGenerationRef.current;
  refs.setState((prev) => ({ ...prev, navigationId }));
  refs.onStartedRef.current?.(navigationId, startAttempt);
  observeNavigationStatus(navigationId, commandGeneration);
  return navigationId;
};

const handleNavigationStartFailure = (
  error: unknown,
  startAttempt: number,
  requestSignal: AbortSignal | undefined,
  refs: AINavigationCommandRefs,
): never => {
  if (refs.startAttemptRef.current !== startAttempt) throw error;
  refs.startInFlightRef.current = false;
  if (requestSignal?.aborted) throw error;
  refs.navigationStatusRef.current = 'failed';
  const message = error instanceof Error ? error.message : 'Failed to start navigation';
  refs.setState((prev) => ({
    ...prev,
    isNavigating: false,
    status: 'failed',
    error: message,
  }));
  throw error;
};

export function useAINavigationCommands({
  sessionId,
  refs,
  observeNavigationStatus,
}: UseAINavigationCommandsOptions) {
  const observeCurrentCommand = useCallback((
    navigationId: string,
    generation: number,
    status?: AINavigationState['status'],
  ): void => {
    if (!refs.isCurrentCommand(navigationId, generation)) return;
    if (status) refs.navigationStatusRef.current = status;
    observeNavigationStatus(navigationId, generation);
  }, [observeNavigationStatus, refs]);

  const runHandoffRequest = useCallback(async (
    navigationId: string,
    operation: 'abort' | 'resume',
    request: (id: string, options: RequestOptions) => Promise<ApiResult<void>>,
  ): Promise<number | null> => {
    const commandGeneration = refs.navigationCommandGenerationRef.current + 1;
    refs.navigationCommandGenerationRef.current = commandGeneration;
    refs.handoffCommandInFlightRef.current = true;
    const result = await request(navigationId, {
      signal: refs.abortControllerRef.current?.signal,
    });
    if (result.success) return commandGeneration;

    if (refs.navigationCommandGenerationRef.current === commandGeneration) {
      refs.handoffCommandInFlightRef.current = false;
    }
    if (operation === 'abort') {
      logger.error('Abort failed', { component: 'useAINavigation' }, new Error(result.error));
    }
    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({ ...prev, error: result.error }));
    return null;
  }, [refs]);

  const startNavigation = useCallback(
    async (prompt: string, model: string, maxSteps = 20): Promise<string | null> => {
      if (!sessionId) {
        refs.setState((prev) => ({ ...prev, error: 'No session available' }));
        return null;
      }
      if (refs.startInFlightRef.current || refs.navigationIdRef.current) {
        refs.setState((prev) => ({ ...prev, error: 'Navigation already in progress' }));
        return null;
      }

      refs.startInFlightRef.current = true;
      refs.navigationStatusRef.current = 'navigating';
      const startAttempt = refs.startAttemptRef.current + 1;
      refs.startAttemptRef.current = startAttempt;
      const requestSignal = refs.abortControllerRef.current?.signal;
      refs.setState((prev) => ({
        ...prev,
        isNavigating: true,
        prompt,
        model,
        steps: [],
        status: 'navigating',
        totalTokens: 0,
        error: null,
      }));

      try {
        const navigationId = await startAINavigationRequest(
          sessionId,
          prompt,
          model,
          maxSteps,
          requestSignal,
        );
        return commitStartedNavigation(
          navigationId,
          startAttempt,
          requestSignal,
          refs,
          observeNavigationStatus,
        );
      } catch (error) {
        return handleNavigationStartFailure(error, startAttempt, requestSignal, refs);
      }
    }, [observeNavigationStatus, refs, sessionId]);

  const abortNavigation = useCallback(async () => {
    const navigationId = refs.navigationIdRef.current;
    if (!navigationId) {
      logger.warn('Cannot abort: no navigationId in ref', { component: 'useAINavigation' });
      return;
    }
    if (refs.handoffCommandInFlightRef.current) return;

    logger.info('Aborting navigation', { component: 'useAINavigation', navigationId });
    const commandGeneration = await runHandoffRequest(navigationId, 'abort', recordingApi.abortAINavigation.bind(recordingApi));
    if (commandGeneration === null) return;

    logger.info('Abort request sent, waiting for completion', { component: 'useAINavigation' });
    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({
      ...prev,
      isNavigating: true,
      status: 'aborting',
      humanIntervention: null,
      error: null,
    }));
    observeCurrentCommand(navigationId, commandGeneration, 'aborting');
  }, [observeCurrentCommand, refs, runHandoffRequest]);

  const resumeNavigation = useCallback(async () => {
    const navigationId = refs.navigationIdRef.current;
    if (!navigationId) return false;
    if (refs.navigationStatusRef.current !== 'awaiting_human') {
      refs.setState((prev) => ({ ...prev, error: 'Navigation is not awaiting human intervention' }));
      return false;
    }
    if (refs.handoffCommandInFlightRef.current) return false;

    const commandGeneration = await runHandoffRequest(navigationId, 'resume', recordingApi.resumeAINavigation.bind(recordingApi));
    if (commandGeneration === null) return false;

    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({ ...prev, error: null }));
    observeCurrentCommand(navigationId, commandGeneration);
    return refs.navigationIdRef.current === navigationId;
  }, [observeCurrentCommand, refs, runHandoffRequest]);

  return { startNavigation, abortNavigation, resumeNavigation };
}
