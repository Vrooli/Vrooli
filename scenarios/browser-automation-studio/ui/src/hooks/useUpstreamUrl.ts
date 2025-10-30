import { useState, useEffect } from 'react';
import { useWorkflowStore } from '../stores/workflowStore';
import { getUpstreamUrl, getUpstreamUrlAsync } from '../utils/nodeUtils';

/**
 * Hook to get the upstream URL for a given node
 * This will automatically update when the workflow state changes
 * For scenario navigation, it resolves to the actual HTTP URL
 */
export function useUpstreamUrl(nodeId: string): string | null {
  const { nodes, edges } = useWorkflowStore();
  const [resolvedUrl, setResolvedUrl] = useState<string | null>(null);

  useEffect(() => {
    // First, get the synchronous result (might be a placeholder for scenarios)
    const syncUrl = getUpstreamUrl(nodeId, nodes, edges);

    // If it's a scenario:// URL, resolve it asynchronously
    if (syncUrl?.startsWith('scenario://')) {
      void getUpstreamUrlAsync(nodeId, nodes, edges).then(url => {
        setResolvedUrl(url);
      });
    } else {
      // For regular URLs, use the sync result
      setResolvedUrl(syncUrl);
    }
  }, [nodeId, nodes, edges]);

  return resolvedUrl;
}