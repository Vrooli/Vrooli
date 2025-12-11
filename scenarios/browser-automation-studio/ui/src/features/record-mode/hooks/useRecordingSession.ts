import { useCallback, useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

interface UseRecordingSessionOptions {
  initialSessionId: string | null;
  onSessionReady?: (sessionId: string) => void;
  initialSessionProfileId?: string | null;
}

interface ViewportSize {
  width: number;
  height: number;
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
  ensureSession: (
    viewport?: ViewportSize | null,
    profileId?: string | null,
    streamSettings?: StreamSettings | null
  ) => Promise<string | null>;
  setSessionProfileId: (profileId: string | null) => void;
  resetSessionError: () => void;
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

  // Track in-flight session creation to prevent duplicate requests.
  // We use a ref because React state updates are async and multiple calls
  // to ensureSession could race past the sessionId check before state updates.
  const pendingSessionPromiseRef = useRef<Promise<string | null> | null>(null);

  useEffect(() => {
    setSessionId(initialSessionId ?? null);
    setSessionProfileId(initialSessionProfileId ?? null);
    setSessionError(null);
    pendingSessionPromiseRef.current = null;
  }, [initialSessionId, initialSessionProfileId]);

  const ensureSession = useCallback(async (
    viewport?: ViewportSize | null,
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

        setSessionId(newSessionId);
        if (data.session_profile_id) {
          setSessionProfileId(data.session_profile_id as string);
        }
        if (onSessionReady) {
          onSessionReady(newSessionId);
        }
        return newSessionId;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to create recording session';
        setSessionError(message);
        logger.error('Failed to create recording session', { component: 'useRecordingSession', action: 'ensureSession' }, err);
        return null;
      } finally {
        setIsCreatingSession(false);
        pendingSessionPromiseRef.current = null;
      }
    };

    // Store the promise so concurrent calls can wait on the same request
    pendingSessionPromiseRef.current = createSession();
    return pendingSessionPromiseRef.current;
  }, [sessionId, sessionProfileId, onSessionReady]);

  const resetSessionError = useCallback(() => setSessionError(null), []);

  return {
    sessionId,
    sessionProfileId,
    isCreatingSession,
    sessionError,
    ensureSession,
    setSessionProfileId,
    resetSessionError,
  };
}
