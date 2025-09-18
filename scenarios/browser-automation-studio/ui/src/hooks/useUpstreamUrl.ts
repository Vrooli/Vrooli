import { useMemo } from 'react';
import { useWorkflowStore } from '../stores/workflowStore';
import { getUpstreamUrl } from '../utils/nodeUtils';

/**
 * Hook to get the upstream URL for a given node
 * This will automatically update when the workflow state changes
 */
export function useUpstreamUrl(nodeId: string): string | null {
  const { nodes, edges } = useWorkflowStore();
  
  return useMemo(() => {
    return getUpstreamUrl(nodeId, nodes, edges);
  }, [nodeId, nodes, edges]);
}