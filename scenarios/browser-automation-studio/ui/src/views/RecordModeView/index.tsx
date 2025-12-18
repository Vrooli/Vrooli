/**
 * RecordModeView - Route wrapper for browser recording mode.
 */
import { lazy, Suspense, useCallback, useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { getConfig } from '@/config';
import { logger } from '@utils/logger';
import toast from 'react-hot-toast';

const RecordModePage = lazy(() =>
  import('@/domains/recording/RecordingSession').then((m) => ({
    default: m.RecordModePage,
  }))
);

export default function RecordModeView() {
  const navigate = useNavigate();
  const { sessionId: routeSessionId } = useParams<{ sessionId?: string }>();
  const [sessionId, setSessionId] = useState<string | null>(routeSessionId ?? null);

  // Sync route param to local state
  useEffect(() => {
    if (routeSessionId && routeSessionId !== 'new') {
      setSessionId(routeSessionId);
    }
  }, [routeSessionId]);

  const handleClose = useCallback(async () => {
    if (sessionId) {
      try {
        const config = await getConfig();
        await fetch(`${config.API_URL}/recordings/live/session/${sessionId}/close`, {
          method: 'POST',
        });
      } catch (error) {
        logger.warn(
          'Failed to close recording session',
          { component: 'RecordModeView', action: 'handleClose' },
          error
        );
      }
    }
    // Go back in history if possible, otherwise navigate home
    // window.history.length > 2 accounts for the initial page load entry
    if (window.history.length > 2) {
      navigate(-1);
    } else {
      navigate('/');
    }
  }, [sessionId, navigate]);

  const handleSessionReady = useCallback(
    (newSessionId: string) => {
      setSessionId(newSessionId);
      // Update URL to include session ID
      navigate(`/record/${newSessionId}`, { replace: true });
    },
    [navigate]
  );

  const handleWorkflowGenerated = useCallback(
    async (workflowId: string, projectId: string) => {
      // Close the recording session on the server
      if (sessionId) {
        try {
          const config = await getConfig();
          await fetch(`${config.API_URL}/recordings/live/session/${sessionId}/close`, {
            method: 'POST',
          });
        } catch (error) {
          logger.warn(
            'Failed to close recording session',
            { component: 'RecordModeView', action: 'handleWorkflowGenerated' },
            error
          );
        }
      }

      // Navigate to the generated workflow
      toast.success('Workflow generated from recording!');
      navigate(`/projects/${projectId}/workflows/${workflowId}`);
    },
    [sessionId, navigate]
  );

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="branded" size={24} message="Loading record mode..." />
          </div>
        }
      >
        <RecordModePage
          sessionId={sessionId}
          onWorkflowGenerated={handleWorkflowGenerated}
          onSessionReady={handleSessionReady}
          onClose={handleClose}
        />
      </Suspense>
    </div>
  );
}
