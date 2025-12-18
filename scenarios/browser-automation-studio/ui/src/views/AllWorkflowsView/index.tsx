/**
 * AllWorkflowsView - Route wrapper for global workflows list.
 */
import { lazy, Suspense, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { useExecutionStore } from '@/domains/executions';
import toast from 'react-hot-toast';
import { logger } from '@utils/logger';

const GlobalWorkflowsView = lazy(() =>
  import('./GlobalWorkflowsView').then((m) => ({
    default: m.GlobalWorkflowsView,
  }))
);

export default function AllWorkflowsView() {
  const navigate = useNavigate();
  const startExecution = useExecutionStore((state) => state.startExecution);

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
      try {
        await startExecution(workflowId);
        toast.success('Workflow execution started');
      } catch (error) {
        logger.error(
          'Failed to start workflow execution',
          { component: 'AllWorkflowsView', action: 'handleRunWorkflow', workflowId },
          error
        );
        toast.error('Failed to start workflow');
      }
    },
    [startExecution]
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
    </div>
  );
}
