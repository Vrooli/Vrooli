/**
 * RecordModeView - Route wrapper for browser recording mode.
 */
import { lazy, Suspense, useCallback, useState, useEffect, useMemo } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
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
  const [searchParams] = useSearchParams();
  const [sessionId, setSessionId] = useState<string | null>(routeSessionId ?? null);

  // Extract query params for template-based navigation
  const templateParams = useMemo(() => {
    const url = searchParams.get('url');
    const aiPrompt = searchParams.get('ai_prompt');
    const aiModel = searchParams.get('ai_model') || undefined;
    const aiMaxSteps = searchParams.get('ai_max_steps');
    const mode = searchParams.get('mode') as 'recording' | 'execution' | null;
    const executionId = searchParams.get('execution_id') || undefined;
    const workflowId = searchParams.get('workflow_id') || undefined;
    const projectId = searchParams.get('project_id') || undefined;

    return {
      initialUrl: url || undefined,
      aiPrompt: aiPrompt || undefined,
      aiModel,
      aiMaxSteps: aiMaxSteps ? parseInt(aiMaxSteps, 10) : undefined,
      autoStartAI: Boolean(url && aiPrompt),
      mode: mode === 'execution' ? 'execution' : 'recording',
      executionId,
      workflowId,
      projectId,
    };
  }, [searchParams]);

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
      // Update URL to include session ID, preserving query params for template flow
      const currentSearch = searchParams.toString();
      const newPath = currentSearch
        ? `/record/${newSessionId}?${currentSearch}`
        : `/record/${newSessionId}`;
      navigate(newPath, { replace: true });
    },
    [navigate, searchParams]
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
    <div data-testid={selectors.app.shell.ready} className="h-screen">
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="branded" size={24} message="Loading record mode..." />
          </div>
        }
      >
        <RecordModePage
          sessionId={sessionId}
          mode={templateParams.mode as 'recording' | 'execution'}
          executionId={templateParams.executionId}
          initialWorkflowId={templateParams.workflowId}
          initialProjectId={templateParams.projectId}
          onWorkflowGenerated={handleWorkflowGenerated}
          onSessionReady={handleSessionReady}
          onClose={handleClose}
          initialUrl={templateParams.initialUrl}
          aiPrompt={templateParams.aiPrompt}
          aiModel={templateParams.aiModel}
          aiMaxSteps={templateParams.aiMaxSteps}
          autoStartAI={templateParams.autoStartAI}
        />
      </Suspense>
    </div>
  );
}
