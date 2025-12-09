import { useCallback, useEffect, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

interface UseRecordingSessionOptions {
  initialSessionId: string | null;
  onSessionReady?: (sessionId: string) => void;
}

interface ViewportSize {
  width: number;
  height: number;
}

interface UseRecordingSessionReturn {
  sessionId: string | null;
  isCreatingSession: boolean;
  sessionError: string | null;
  ensureSession: (viewport?: ViewportSize | null) => Promise<string | null>;
  resetSessionError: () => void;
}

export function useRecordingSession({
  initialSessionId,
  onSessionReady,
}: UseRecordingSessionOptions): UseRecordingSessionReturn {
  const [sessionId, setSessionId] = useState<string | null>(initialSessionId ?? null);
  const [isCreatingSession, setIsCreatingSession] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);

  useEffect(() => {
    setSessionId(initialSessionId ?? null);
    setSessionError(null);
  }, [initialSessionId]);

  const ensureSession = useCallback(async (viewport?: ViewportSize | null): Promise<string | null> => {
    if (sessionId) {
      return sessionId;
    }

    setIsCreatingSession(true);
    setSessionError(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          viewport_width: viewport?.width && viewport.width > 0 ? Math.round(viewport.width) : 1280,
          viewport_height: viewport?.height && viewport.height > 0 ? Math.round(viewport.height) : 720,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to create recording session: ${response.statusText}`);
      }

      const data = await response.json();
      const newSessionId = data.session_id;

      if (!newSessionId) {
        throw new Error('No session ID returned from server');
      }

      setSessionId(newSessionId);
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
    }
  }, [sessionId, onSessionReady]);

  const resetSessionError = useCallback(() => setSessionError(null), []);

  return {
    sessionId,
    isCreatingSession,
    sessionError,
    ensureSession,
    resetSessionError,
  };
}
