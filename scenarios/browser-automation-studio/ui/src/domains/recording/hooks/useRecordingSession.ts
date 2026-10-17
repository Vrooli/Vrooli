import { useCallback, useEffect, useRef, useState } from 'react';
import { logger } from '@/utils/logger';
import { recordingApi } from '../api';
import type { ActualViewport } from '../api/schemas';
import type { ViewportDimensions } from '../types/viewport';
import {
  type RetryState,
  DEFAULT_RETRY_CONFIG,
  createInitialRetryState,
  getNextRetryState,
  createSuccessState,
  createManualRetryState,
} from '../services';
import { useSessionStore } from '../stores';

// Re-export types for backward compatibility
export type { ViewportSource, ActualViewport } from '../api/schemas';

interface UseRecordingSessionOptions {
  initialSessionId: string | null;
  onSessionReady?: (sessionId: string) => void;
  initialSessionProfileId?: string | null;
}

/** Stream settings passed to session creation */
export interface StreamSettings {
  quality: number;
  fps: number;
  /** 'css' = 1x scale, 'device' = device pixel ratio */
  scale: 'css' | 'device';
}

interface UseRecordingSessionReturn {
  sessionId: string | null;
  sessionProfileId: string | null;
  isCreatingSession: boolean;
  sessionError: string | null;
  /** Actual viewport from Playwright with source attribution (may differ from requested due to profile settings) */
  actualViewport: ActualViewport | null;
  ensureSession: (
    viewport?: ViewportDimensions | null,
    profileId?: string | null,
    streamSettings?: StreamSettings | null,
    /** Whether to restore tabs from the previous session. Default: true */
    restoreTabs?: boolean
  ) => Promise<string | null>;
  setSessionProfileId: (profileId: string | null) => void;
  resetSessionError: () => void;
  /** Retry state for UI feedback */
  retryState: RetryState;
  /** Manual retry function (resets retry count and tries again) */
  retrySession: () => void;
}

export function useRecordingSession({
  initialSessionId,
  onSessionReady,
  initialSessionProfileId = null,
}: UseRecordingSessionOptions): UseRecordingSessionReturn {
  const [sessionId, setSessionId] = useState<string | null>(initialSessionId ?? null);
  const [sessionProfileId, setSessionProfileId] = useState<string | null>(initialSessionProfileId ?? null);
  const [isCreatingSession, setIsCreatingSession] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [actualViewport, setActualViewport] = useState<ActualViewport | null>(null);

  // Retry state for exponential backoff
  const [retryState, setRetryState] = useState<RetryState>(createInitialRetryState);

  // Get store state and actions for syncing state
  const storeSessionId = useSessionStore((s) => s.sessionId);
  const storeIsValidated = useSessionStore((s) => s.isValidated);
  const storeSetSession = useSessionStore((s) => s.setSession);
  const storeSetIsCreating = useSessionStore((s) => s.setIsCreating);
  const storeSetError = useSessionStore((s) => s.setError);
  const storeSetRetryState = useSessionStore((s) => s.setRetryState);
  const storeClearSession = useSessionStore((s) => s.clearSession);
  const storeValidateSession = useSessionStore((s) => s.validateSession);

  // Track in-flight session creation to prevent duplicate requests.
  // We use a ref because React state updates are async and multiple calls
  // to ensureSession could race past the sessionId check before state updates.
  const pendingSessionPromiseRef = useRef<Promise<string | null> | null>(null);
  const creationOwnerRef = useRef<object | null>(null);
  const cooldownTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryAttemptsRef = useRef(0);

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(false);
  const ownerRef = useRef({ initialSessionId, profileId: initialSessionProfileId, generation: 0 });
  if (ownerRef.current.initialSessionId !== initialSessionId || ownerRef.current.profileId !== initialSessionProfileId) {
    ownerRef.current = { initialSessionId, profileId: initialSessionProfileId, generation: ownerRef.current.generation + 1 };
  }
  const controllerGenerationRef = useRef(-1);
  const validationGenerationRef = useRef(0);

  // Clean up on unmount
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      ownerRef.current = { ...ownerRef.current, generation: ownerRef.current.generation + 1 };
      validationGenerationRef.current += 1;
      if (cooldownTimerRef.current) {
        clearTimeout(cooldownTimerRef.current);
      }
      abortControllerRef.current?.abort();
      if (creationOwnerRef.current) storeSetIsCreating(false, creationOwnerRef.current);
      creationOwnerRef.current = null;
    };
  }, [storeSetIsCreating]);

  // Validate URL-based session ID on mount/change
  useEffect(() => {
    const generation = ownerRef.current.generation;
    const validationGeneration = ++validationGenerationRef.current;
    const isCurrent = () => mountedRef.current && ownerRef.current.generation === generation;
    if (controllerGenerationRef.current !== generation) {
      abortControllerRef.current?.abort();
      abortControllerRef.current = new AbortController();
      controllerGenerationRef.current = generation;
      pendingSessionPromiseRef.current = null;
      if (creationOwnerRef.current) storeSetIsCreating(false, creationOwnerRef.current);
      creationOwnerRef.current = null;
      setIsCreatingSession(false);
      retryAttemptsRef.current = 0;
      setRetryState(createInitialRetryState());
      if (cooldownTimerRef.current) {
        clearTimeout(cooldownTimerRef.current);
        cooldownTimerRef.current = null;
      }
    }
    // If the URL session matches what's already validated in the store, skip validation.
    // This prevents unnecessary WebSocket subscription flapping when the URL updates
    // after session creation (e.g., redirect from /record to /record/<session_id>).
    if (initialSessionId && initialSessionId === storeSessionId && storeIsValidated) {
      logger.debug('[useRecordingSession] URL session matches validated store session, skipping validation', {
        sessionId: initialSessionId,
      });
      // Still sync local state
      // A created session's own URL redirect retains its profile/viewport.
      // A different validated session must not inherit that presentation.
      const isCreatedSessionRedirect = initialSessionId === sessionId;
      setSessionId(initialSessionId);
      setSessionProfileId(current => initialSessionProfileId ?? (isCreatedSessionRedirect ? current : null));
      if (!isCreatedSessionRedirect) setActualViewport(null);
      setSessionError(null);
      return;
    }

    // If there's no URL session ID but the store has a validated session,
    // this is likely a timing issue where the URL hasn't been updated yet.
    // Don't clear the store - let the URL redirect happen.
    if (!initialSessionId && storeSessionId && storeIsValidated) {
      logger.debug('[useRecordingSession] No URL session but store has validated session, waiting for URL update', {
        storeSessionId,
      });
      return;
    }

    // Reset local state
    setSessionId(initialSessionId ?? null);
    setSessionProfileId(initialSessionProfileId ?? null);
    setSessionError(null);
    setActualViewport(null);

    // If switching to a different session (or no session), clear the store first
    if (storeSessionId && storeSessionId !== initialSessionId) {
      logger.debug('[useRecordingSession] Switching sessions, clearing store', {
        from: storeSessionId,
        to: initialSessionId,
      });
      storeClearSession();
    }

    // If we have an initial session ID from URL, validate it exists on server
    if (initialSessionId) {
      logger.debug('[useRecordingSession] Validating URL session:', { sessionId: initialSessionId });
      storeValidateSession(initialSessionId, { profileId: initialSessionProfileId, signal: abortControllerRef.current?.signal }).then((isValid) => {
        if (!isCurrent() || validationGenerationRef.current !== validationGeneration) return;
        if (!isValid) {
          logger.info('[useRecordingSession] URL session invalid, will create new one', {
            sessionId: initialSessionId,
          });
          // Clear local session ID to trigger new session creation
          setSessionId(null);
        } else {
          logger.debug('[useRecordingSession] URL session validated:', { sessionId: initialSessionId });
          // Validation commits its identity/profile at the store owner. A
          // second hook commit could revive a retired validation attempt.
        }
      });
    }
  }, [initialSessionId, initialSessionProfileId, storeSessionId, storeIsValidated, storeClearSession, storeValidateSession, storeSetIsCreating]);

  const ownerGeneration = ownerRef.current.generation;
  const ensureSession = useCallback(async (
    viewport?: ViewportDimensions | null,
    profileId?: string | null,
    streamSettings?: StreamSettings | null,
    restoreTabs?: boolean
  ): Promise<string | null> => {
    if (!mountedRef.current || ownerRef.current.generation !== ownerGeneration) return null;
    if (sessionId) {
      return sessionId;
    }

    // If there's already a pending session creation, wait for it instead of starting another
    if (pendingSessionPromiseRef.current) {
      return pendingSessionPromiseRef.current;
    }

    // Check if we're in cooldown or exceeded max retries
    if (retryState.inCooldown) {
      logger.debug('Session creation blocked: in cooldown', { component: 'useRecordingSession' });
      return null;
    }
    if (retryState.maxRetriesExceeded) {
      logger.debug('Session creation blocked: max retries exceeded', { component: 'useRecordingSession' });
      return null;
    }

    setIsCreatingSession(true);
    setSessionError(null);
    const creationOwner = {};
    creationOwnerRef.current = creationOwner;
    storeSetIsCreating(true, creationOwner);
    storeSetError(null);
    const generation = ownerRef.current.generation;
    const controller = abortControllerRef.current;
    const isCurrent = () => mountedRef.current && ownerRef.current.generation === generation && !controller?.signal.aborted;

    const createSession = async (): Promise<string | null> => {
      try {
        const result = await recordingApi.createSession(
          {
            viewportWidth: viewport?.width && viewport.width > 0 ? Math.round(viewport.width) : undefined,
            viewportHeight: viewport?.height && viewport.height > 0 ? Math.round(viewport.height) : undefined,
            sessionProfileId: profileId ?? sessionProfileId ?? undefined,
            streamQuality: streamSettings?.quality,
            streamFps: streamSettings?.fps,
            streamScale: streamSettings?.scale,
            restoreTabs,
          },
          { signal: controller?.signal }
        );

        if (!isCurrent()) return null;

        if (!result.success) {
          throw new Error(result.error);
        }

        const { session_id: newSessionId, session_profile_id, actual_viewport } = result.data;

        // Success - reset retry state
        retryAttemptsRef.current = 0;
        setRetryState(createSuccessState());
        storeSetRetryState(createSuccessState());

        setSessionId(newSessionId);
        if (session_profile_id) {
          setSessionProfileId(session_profile_id);
        }
        if (actual_viewport) {
          setActualViewport(actual_viewport);
        }

        // Sync session to store
        storeSetSession({
          sessionId: newSessionId,
          profileId: session_profile_id ?? null,
          actualViewport: actual_viewport ?? null,
        });

        if (onSessionReady) {
          onSessionReady(newSessionId);
        }
        return newSessionId;
      } catch (err) {
        if (!isCurrent()) return null;
        // Handle abort
        if (err instanceof Error && err.name === 'AbortError') {
          return null;
        }

        const message = err instanceof Error ? err.message : 'Failed to create recording session';
        setSessionError(message);
        storeSetError(message);
        logger.error('Failed to create recording session', { component: 'useRecordingSession', action: 'ensureSession' }, err);

        // Compute next retry state using the service
        const newState = getNextRetryState(retryAttemptsRef.current, DEFAULT_RETRY_CONFIG);
        retryAttemptsRef.current = newState.attempts;
        setRetryState(newState);
        storeSetRetryState(newState);

        if (newState.maxRetriesExceeded) {
          logger.warn('Max session creation retries exceeded', {
            component: 'useRecordingSession',
            attempts: newState.attempts,
            maxRetries: DEFAULT_RETRY_CONFIG.maxRetries,
          });
        } else if (newState.inCooldown && newState.nextRetryAt) {
          const delay = newState.nextRetryAt - Date.now();
          logger.info('Session creation failed, will retry', {
            component: 'useRecordingSession',
            attempts: newState.attempts,
            nextRetryInMs: delay,
          });

          // Clear cooldown after delay
          cooldownTimerRef.current = setTimeout(() => {
            if (!isCurrent()) return;
            setRetryState((prev) => ({
              ...prev,
              inCooldown: false,
              nextRetryAt: null,
            }));
            cooldownTimerRef.current = null;
          }, delay);
        }

        return null;
      } finally {
        if (isCurrent()) {
          setIsCreatingSession(false);
          storeSetIsCreating(false, creationOwner);
          if (creationOwnerRef.current === creationOwner) {
            creationOwnerRef.current = null;
            pendingSessionPromiseRef.current = null;
          }
        }
      }
    };

    // Store the promise so concurrent calls can wait on the same request
    pendingSessionPromiseRef.current = createSession();
    return pendingSessionPromiseRef.current;
  }, [ownerGeneration, sessionId, sessionProfileId, onSessionReady, retryState.inCooldown, retryState.maxRetriesExceeded, storeSetIsCreating, storeSetError, storeSetSession, storeSetRetryState]);

  // Manual retry function - resets retry state and allows another attempt
  const retrySession = useCallback(() => {
    // Clear any pending cooldown timer
    if (cooldownTimerRef.current) {
      clearTimeout(cooldownTimerRef.current);
      cooldownTimerRef.current = null;
    }

    // Reset retry state
    retryAttemptsRef.current = 0;
    setRetryState(createManualRetryState());
    setSessionError(null);

    logger.info('Manual session retry triggered', { component: 'useRecordingSession' });
  }, []);

  const resetSessionError = useCallback(() => setSessionError(null), []);

  return {
    sessionId,
    sessionProfileId,
    isCreatingSession,
    sessionError,
    actualViewport,
    ensureSession,
    setSessionProfileId,
    resetSessionError,
    retryState,
    retrySession,
  };
}
