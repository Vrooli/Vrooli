import { useCallback, useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { ViewportDimensions, ActualViewport, ViewportSource } from '../types/viewport';
import {
  type RetryState,
  DEFAULT_RETRY_CONFIG,
  createInitialRetryState,
  getNextRetryState,
  createSuccessState,
  createManualRetryState,
} from '../services';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const safeJson = async (response: Response): Promise<unknown> => {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
};

const isViewportSource = (value: unknown): value is ViewportSource =>
  value === 'requested' ||
  value === 'fingerprint' ||
  value === 'fingerprint_partial' ||
  value === 'default';

const parseActualViewport = (value: unknown): ActualViewport | null => {
  if (!isRecord(value)) return null;
  if (typeof value.width !== 'number' || typeof value.height !== 'number') return null;
  return {
    width: value.width,
    height: value.height,
    source: isViewportSource(value.source) ? value.source : 'requested',
    reason: typeof value.reason === 'string' ? value.reason : '',
  };
};

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

interface UseRecordingSessionReturn {
  sessionId: string | null;
  sessionProfileId: string | null;
  isCreatingSession: boolean;
  sessionError: string | null;
  /** Actual viewport from Playwright with source attribution (may differ from requested due to profile settings) */
  actualViewport: ActualViewport | null;
  /** Initial URL from tab restoration (if tabs were restored during session creation) */
  initialRestoredUrl: string | null;
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
  const [initialRestoredUrl, setInitialRestoredUrl] = useState<string | null>(null);

  // Retry state for exponential backoff
  const [retryState, setRetryState] = useState<RetryState>(createInitialRetryState);

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
    setInitialRestoredUrl(null);
    pendingSessionPromiseRef.current = null;
    // Reset retry state when session ID changes externally
    retryAttemptsRef.current = 0;
    setRetryState(createInitialRetryState());
    if (cooldownTimerRef.current) {
      clearTimeout(cooldownTimerRef.current);
      cooldownTimerRef.current = null;
    }
  }, [initialSessionId, initialSessionProfileId]);

  const ensureSession = useCallback(async (
    viewport?: ViewportDimensions | null,
    profileId?: string | null,
    streamSettings?: StreamSettings | null,
    restoreTabs?: boolean
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
            // Tab restoration - defaults to true on server if not specified
            restore_tabs: restoreTabs,
          }),
        });

        const payload = await safeJson(response);
        if (!response.ok) {
          const message =
            isRecord(payload) && typeof payload.message === 'string'
              ? payload.message
              : `Failed to create recording session: ${response.statusText}`;
          throw new Error(message);
        }

        const newSessionId =
          isRecord(payload) && typeof payload.session_id === 'string'
            ? payload.session_id
            : undefined;

        if (!newSessionId) {
          throw new Error('No session ID returned from server');
        }

        // Success - reset retry state
        retryAttemptsRef.current = 0;
        setRetryState(createSuccessState());

        setSessionId(newSessionId);
        if (isRecord(payload) && typeof payload.session_profile_id === 'string') {
          setSessionProfileId(payload.session_profile_id);
        }
        const actualViewportValue = parseActualViewport(
          isRecord(payload) ? payload.actual_viewport : null
        );
        if (actualViewportValue) {
          setActualViewport(actualViewportValue);
        }
        if (isRecord(payload) && typeof payload.initial_url === 'string') {
          setInitialRestoredUrl(payload.initial_url);
        }
        if (onSessionReady) {
          onSessionReady(newSessionId);
        }
        return newSessionId;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to create recording session';
        setSessionError(message);
        logger.error('Failed to create recording session', { component: 'useRecordingSession', action: 'ensureSession' }, err);

        // Compute next retry state using the service
        const newState = getNextRetryState(retryAttemptsRef.current, DEFAULT_RETRY_CONFIG);
        retryAttemptsRef.current = newState.attempts;
        setRetryState(newState);

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
    initialRestoredUrl,
    ensureSession,
    setSessionProfileId,
    resetSessionError,
    retryState,
    retrySession,
  };
}
