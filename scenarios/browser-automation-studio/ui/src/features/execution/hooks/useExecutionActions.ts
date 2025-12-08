import { useExecutionStore } from '@stores/executionStore';

export const useExecutionActions = () =>
  useExecutionStore((state) => ({
    refreshTimeline: state.refreshTimeline,
    stopExecution: state.stopExecution,
    startExecution: state.startExecution,
    loadExecution: state.loadExecution,
    loadExecutions: state.loadExecutions,
  }));
