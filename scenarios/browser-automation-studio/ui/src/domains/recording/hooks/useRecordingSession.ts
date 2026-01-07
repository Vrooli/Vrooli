import { useCallback, useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { ViewportDimensions, ActualViewport } from '../types/viewport';

// Re-export types for backward compatibility
export type { ViewportSource, ActualViewport } from '../types/viewport';

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

/** Retry state for session creation */
interface RetryState {
  /** Number of retry attempts made */
  attempts: number;
  /** Whether we're in a cooldown period before next retry */
  inCooldown: boolean;
  /** When the next retry is allowed (for cooldown display) */
  nextRetryAt: number | null;
  /** Whether max retries exceeded (requires manual retry) */
  maxRetriesExceeded: boolean;
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
    streamSettings?: StreamSettings | null
  ) => Promise<string | null>;
  setSessionProfileId: (profileId: string | null) => void;
  resetSessionError: () => void;
  /** Retry state for UI feedback */
  retryState: RetryState;
  /** Manual retry function (resets retry count and tries again) */
  retrySession: () => void;
}

// Retry configuration
const MAX_AUTO_RETRIES = 3;
const BASE_RETRY_DELAY_MS = 1000; // 1 second
const MAX_RETRY_DELAY_MS = 30000; // 30 seconds

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
  const [retryState, setRetryState] = useState<RetryState>({
    attempts: 0,
    inCooldown: false,
    nextRetryAt: null,
    maxRetriesExceeded: false,
  });

  // Track in-flight session creation to prevent duplicate requests.
  // We use a ref because React state updates are async and multiple calls
  // to ensureSession could race past the sessionId check before state updates.
  const pendingSessionPromiseRef = useRef<Promise<string | null> | null>(null);
  const cooldownTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryAttemptsRef = useRef(0);

  // Clean up cooldown timer on unmount
  useEffect(() => {
    return () => {
      if (cooldownTimerRef.current) {
        clearTimeout(cooldownTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    setSessionId(initialSessionId ?? null);
    setSessionProfileId(initialSessionProfileId ?? null);
    setSessionError(null);
    setActualViewport(null);
    pendingSessionPromiseRef.current = null;
    // Reset retry state when session ID changes externally
    retryAttemptsRef.current = 0;
    setRetryState({
      attempts: 0,
      inCooldown: false,
      nextRetryAt: null,
      maxRetriesExceeded: false,
    });
    if (cooldownTimerRef.current) {
      clearTimeout(cooldownTimerRef.current);
      cooldownTimerRef.current = null;
    }
  }, [initialSessionId, initialSessionProfileId]);

  const ensureSession = useCallback(async (
    viewport?: ViewportDimensions | null,
    profileId?: string | null,
    streamSettings?: StreamSettings | null
  ): Promise<string | null> => {
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

    const createSession = async (): Promise<string | null> => {
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/recordings/live/session`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            viewport_width: viewport?.width && viewport.width > 0 ? Math.round(viewport.width) : 1280,
            viewport_height: viewport?.height && viewport.height > 0 ? Math.round(viewport.height) : 720,
            session_profile_id: profileId ?? sessionProfileId ?? undefined,
            // Stream settings for frame streaming configuration
            stream_quality: streamSettings?.quality,
            stream_fps: streamSettings?.fps,
            stream_scale: streamSettings?.scale,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.message || `Failed to create recording session: ${response.statusText}`);
        }

        const data = await response.json();
        const newSessionId = data.session_id as string | undefined;

        if (!newSessionId) {
          throw new Error('No session ID returned from server');
        }

        // Success - reset retry state
        retryAttemptsRef.current = 0;
        setRetryState({
          attempts: 0,
          inCooldown: false,
          nextRetryAt: null,
          maxRetriesExceeded: false,
        });

        setSessionId(newSessionId);
        if (data.session_profile_id) {
          setSessionProfileId(data.session_profile_id as string);
        }
        // Store actual viewport from Playwright with source attribution (may differ from requested due to profile)
        if (data.actual_viewport) {
          setActualViewport({
            width: data.actual_viewport.width,
            height: data.actual_viewport.height,
            source: data.actual_viewport.source ?? 'requested',
            reason: data.actual_viewport.reason ?? '',
          });
        }
        if (onSessionReady) {
          onSessionReady(newSessionId);
        }
        return newSessionId;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to create recording session';
        setSessionError(message);
        logger.error('Failed to create recording session', { component: 'useRecordingSession', action: 'ensureSession' }, err);

        // Increment retry count and apply exponential backoff
        retryAttemptsRef.current += 1;
        const attempts = retryAttemptsRef.current;

        if (attempts >= MAX_AUTO_RETRIES) {
          // Max retries exceeded - require manual retry
          setRetryState({
            attempts,
            inCooldown: false,
            nextRetryAt: null,
            maxRetriesExceeded: true,
          });
          logger.warn('Max session creation retries exceeded', {
            component: 'useRecordingSession',
            attempts,
            maxRetries: MAX_AUTO_RETRIES,
          });
        } else {
          // Calculate backoff delay: 1s, 2s, 4s, 8s... capped at MAX_RETRY_DELAY_MS
          const delay = Math.min(
            BASE_RETRY_DELAY_MS * Math.pow(2, attempts - 1),
            MAX_RETRY_DELAY_MS
          );
          const nextRetryAt = Date.now() + delay;

          setRetryState({
            attempts,
            inCooldown: true,
            nextRetryAt,
            maxRetriesExceeded: false,
          });

          logger.info('Session creation failed, will retry', {
            component: 'useRecordingSession',
            attempts,
            nextRetryInMs: delay,
          });

          // Clear cooldown after delay
          cooldownTimerRef.current = setTimeout(() => {
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
        setIsCreatingSession(false);
        pendingSessionPromiseRef.current = null;
      }
    };

    // Store the promise so concurrent calls can wait on the same request
    pendingSessionPromiseRef.current = createSession();
    return pendingSessionPromiseRef.current;
  }, [sessionId, sessionProfileId, onSessionReady, retryState.inCooldown, retryState.maxRetriesExceeded]);

  // Manual retry function - resets retry state and allows another attempt
  const retrySession = useCallback(() => {
    // Clear any pending cooldown timer
    if (cooldownTimerRef.current) {
      clearTimeout(cooldownTimerRef.current);
      cooldownTimerRef.current = null;
    }

    // Reset retry state
    retryAttemptsRef.current = 0;
    setRetryState({
      attempts: 0,
      inCooldown: false,
      nextRetryAt: null,
      maxRetriesExceeded: false,
    });
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
