/**
 * AllWorkflowsView - Route wrapper for global workflows list.
 */
import { lazy, Suspense, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner, PromptDialog } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { useStartWorkflow } from '@/domains/executions';
import toast from 'react-hot-toast';
import { logger } from '@utils/logger';

const GlobalWorkflowsView = lazy(() =>
  import('./GlobalWorkflowsView').then((m) => ({
    default: m.GlobalWorkflowsView,
  }))
);

export default function AllWorkflowsView() {
  const navigate = useNavigate();

  // Execution hook with start URL prompt
  const { startWorkflow, promptDialogProps } = useStartWorkflow({
    onSuccess: () => {
      toast.success('Workflow execution started');
    },
    onError: (error) => {
      logger.error(
        'Failed to start workflow execution',
        { component: 'AllWorkflowsView', action: 'handleRunWorkflow' },
        error
      );
      toast.error('Failed to start workflow');
    },
  });

  const handleBack = () => {
    // Go back in history if possible, otherwise navigate home
    if (window.history.length > 2) {
      navigate(-1);
    } else {
      navigate('/');
    }
  };

  const handleNavigateToWorkflow = useCallback(
    async (projectId: string, workflowId: string) => {
      navigate(`/projects/${projectId}/workflows/${workflowId}`);
    },
    [navigate]
  );

  const handleRunWorkflow = useCallback(
    async (workflowId: string) => {
      // startWorkflow handles success/error via hook options
      await startWorkflow({ workflowId });
    },
    [startWorkflow]
  );

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="default" size={24} message="Loading workflows..." />
          </div>
        }
      >
        <GlobalWorkflowsView
          onBack={handleBack}
          onNavigateToWorkflow={handleNavigateToWorkflow}
          onRunWorkflow={handleRunWorkflow}
        />
      </Suspense>

      {/* Start URL prompt for workflows without navigate step */}
      <PromptDialog {...promptDialogProps} />
    </div>
  );
}
