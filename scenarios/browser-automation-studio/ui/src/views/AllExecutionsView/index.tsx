/**
 * AllExecutionsView - Route wrapper for global executions list.
 */
import { lazy, Suspense, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { useExecutionStore } from '@/domains/executions';

const GlobalExecutionsView = lazy(() =>
  import('./GlobalExecutionsView').then((m) => ({
    default: m.GlobalExecutionsView,
  }))
);

export default function AllExecutionsView() {
  const navigate = useNavigate();
  const loadExecution = useExecutionStore((state) => state.loadExecution);

  const handleBack = () => {
    // Go back in history if possible, otherwise navigate home
    if (window.history.length > 2) {
      navigate(-1);
    } else {
      navigate('/');
    }
  };

  const handleViewExecution = useCallback(
    async (executionId: string, workflowId: string) => {
      // Load the execution details
      await loadExecution(executionId);

      // Find the workflow's project and navigate to the workflow view
      const workflowsResponse = await fetch(`/api/v1/workflows/${workflowId}`);
      if (workflowsResponse.ok) {
        const workflowData = await workflowsResponse.json();
        const projectId = workflowData.project_id ?? workflowData.projectId;
        if (projectId) {
          navigate(`/projects/${projectId}/workflows/${workflowId}`);
        }
      }
    },
    [loadExecution, navigate]
  );

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="default" size={24} message="Loading executions..." />
          </div>
        }
      >
        <GlobalExecutionsView onBack={handleBack} onViewExecution={handleViewExecution} />
      </Suspense>
    </div>
  );
}
