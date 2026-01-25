import { useCallback, useState } from 'react';
import { safeParse, RecoveryCheckResponseSchema, type RecoveryCheckpoint } from '@/shared/api';

export function useRecoveryCheck(): {
  checkpoint: RecoveryCheckpoint | null;
  isLoading: boolean;
  checkForRecovery: () => Promise<void>;
  resumeRecording: () => Promise<void>;
  startFresh: () => Promise<void>;
} {
  const [checkpoint, setCheckpoint] = useState<RecoveryCheckpoint | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const checkForRecovery = useCallback(async () => {
    try {
      const response = await fetch('/api/recording/recovery/check');
      if (response.ok) {
        const rawData: unknown = await response.json();
        const result = safeParse(RecoveryCheckResponseSchema, rawData, 'RecoveryCheck');
        if (result.success && result.data.checkpoint) {
          setCheckpoint(result.data.checkpoint);
        } else {
          setCheckpoint(null);
        }
      }
    } catch (error) {
      console.error('Failed to check for recovery:', error);
      setCheckpoint(null);
    }
  }, []);

  const resumeRecording = useCallback(async () => {
    if (!checkpoint) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/recording/recovery/${checkpoint.sessionId}/resume`, {
        method: 'POST',
      });
      if (response.ok) {
        setCheckpoint(null);
        // Navigation to recording session would happen here
      }
    } catch (error) {
      console.error('Failed to resume recording:', error);
    } finally {
      setIsLoading(false);
    }
  }, [checkpoint]);

  const startFresh = useCallback(async () => {
    if (!checkpoint) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/recording/recovery/${checkpoint.sessionId}`, {
        method: 'DELETE',
      });
      if (response.ok) {
        setCheckpoint(null);
      }
    } catch (error) {
      console.error('Failed to delete checkpoint:', error);
    } finally {
      setIsLoading(false);
    }
  }, [checkpoint]);

  return {
    checkpoint,
    isLoading,
    checkForRecovery,
    resumeRecording,
    startFresh,
  };
}
